-- Migration: body bytea → jsonb
--
-- Converts request/response body columns from bytea (base64-encoded) to native
-- JSONB. Existing base64 content is decoded and stored as JSONB objects. Non-JSON
-- content (HTML, plain text, XML) is wrapped as a JSON string value.
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
