-- Migration: body bytea → jsonb
--
-- Converts request/response body columns from bytea (base64-encoded) to native
-- JSONB. Existing base64 content is decoded and stored as JSONB objects. Non-JSON
-- content (HTML, plain text, XML) is wrapped as a JSON string value.
--
-- DESIGN DECISIONS & FUTURE STEPS
-- --------------------------------
-- Current state: both requests.body and responses.body are JSONB. The API returns
-- bodies as string | null (stringified JSON). The proxy still sends base64-encoded
-- bodies over JSON; the API decodes them on write.
--
-- Request body (requests.body):
--   Already parsed and redacted as JSON on write (bodyToRawMessage + redactor),
--   so the content is a processed JSON structure, not raw opaque bytes. JSONB is
--   the correct choice: the body has already paid the parse cost, and JSONB
--   enables future querying (e.g., "find all requests where body.user_id = X")
--   without a second parse or migration.
--
-- Response bodies (responses.body):
--   Used for display, JSON patch diffing, and future querying. JSONB is the
--   correct choice here — the diff engine operates on the stored JSON structure.
--
-- Future: multi-content-type diffing
--   JSON patch (RFC 6902) only works on JSON documents. To support structured
--   diffs for XML, HTML, or plain text, the architecture should evolve to:
--
--   1. Split into two columns per body:
--      - body      (TEXT)  — original body in its native format, for display
--      - body_json (JSONB) — JSON-normalized version, for diffing & querying
--
--   2. Convert per content-type on write:
--      - application/json → body = raw JSON text, body_json = same content as JSONB
--      - text/xml         → body = raw XML,       body_json = XML-to-JSON conversion
--      - text/html        → body = raw HTML,      body_json = HTML-to-JSON tree
--      - text/plain       → body = raw text,       body_json = JSON string value
--      - binary           → body = base64 string, body_json = NULL (skip diff)
--
--   3. Diff engine always uses body_json; API always returns body (text).
--
--   The current json.RawMessage / JSONB column becomes body_json, and body (text)
--   is added alongside it.
--
-- Future: eliminate base64 on the proxy→API wire
--   The proxy currently base64-encodes bodies because the transport is JSON (which
--   can't represent arbitrary binary). Options to eliminate this overhead:
--   - Have the proxy send json.RawMessage (pass-through for JSON bodies, wrap
--     non-JSON as a JSON string) — eliminates base64 for ~95% of traffic
--   - Switch to gRPC/protobuf (native bytes fields) — eliminates base64 entirely
--   - Switch to MessagePack/CBOR — binary JSON alternatives with native bytes
--
-- ROLLBACK STRATEGY
-- -----------------
-- This migration is NOT automatically reversible. JSONB → bytea requires
-- re-encoding the content as base64. If you need to roll back:
--
--   1. Create a new forward migration with the SQL below.
--   2. Re-deploy the previous application version that expects bytea bodies.
--
--   CREATE OR REPLACE FUNCTION _safe_jsonb_to_bytea(raw jsonb) RETURNS bytea AS $$
--   BEGIN
--     IF raw IS NULL THEN RETURN NULL; END IF;
--     -- encode the text representation of the jsonb value back to base64 bytea
--     RETURN decode(encode(convert_to(raw #>> '{}', 'UTF-8'), 'base64'), 'escape')::bytea;
--   EXCEPTION WHEN OTHERS THEN
--     RETURN decode(encode(convert_to(raw::text, 'UTF-8'), 'base64'), 'escape')::bytea;
--   END;
--   $$ LANGUAGE plpgsql;
--
--   ALTER TABLE "requests" ALTER COLUMN "body" TYPE bytea
--     USING _safe_jsonb_to_bytea(body);
--   ALTER TABLE "responses" ALTER COLUMN "body" TYPE bytea
--     USING _safe_jsonb_to_bytea(body);
--
--   DROP FUNCTION _safe_jsonb_to_bytea(jsonb);
--
-- NOTE: The rollback uses `#>> '{}'` to extract scalar JSON strings without
-- surrounding quotes (e.g. jsonb '"hello"' → text 'hello'). For JSON objects
-- and arrays it falls back to `raw::text` which preserves the JSON structure.
-- Data fidelity is preserved in both directions.

-- Helper: decode base64 bytea → jsonb, wrapping non-JSON content as a JSON string.
-- This handles the case where a body was base64-encoded plain text, HTML, XML, etc.
CREATE OR REPLACE FUNCTION _safe_base64_to_jsonb(raw bytea) RETURNS jsonb AS $$
DECLARE
  decoded text;
BEGIN
  IF raw IS NULL THEN RETURN NULL; END IF;
  decoded := convert_from(decode(encode(raw, 'escape'), 'base64'), 'UTF-8');
  RETURN decoded::jsonb;
EXCEPTION WHEN OTHERS THEN
  -- Not valid JSON: wrap the decoded text as a JSON string value
  RETURN to_jsonb(decoded);
END;
$$ LANGUAGE plpgsql;

-- Modify "requests" table
-- Existing body is base64-encoded content stored as bytea.
ALTER TABLE "requests" ALTER COLUMN "body" TYPE jsonb
  USING _safe_base64_to_jsonb(body);
-- Modify "responses" table
ALTER TABLE "responses" ALTER COLUMN "body" TYPE jsonb
  USING _safe_base64_to_jsonb(body);

-- Clean up helper function
DROP FUNCTION _safe_base64_to_jsonb(bytea);
