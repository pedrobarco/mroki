-- Modify "diffs" table: add has_content nullable, backfill from existing
-- content, then enforce NOT NULL. New rows set the value explicitly from the
-- domain (see request_repository.saveDiff), so no column default is needed.
ALTER TABLE "diffs" ADD COLUMN "has_content" boolean;
UPDATE "diffs" SET "has_content" = jsonb_array_length("content") > 0;
ALTER TABLE "diffs" ALTER COLUMN "has_content" SET NOT NULL;
