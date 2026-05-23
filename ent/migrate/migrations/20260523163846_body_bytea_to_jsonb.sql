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
