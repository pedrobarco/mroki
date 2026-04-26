-- Modify "gates" table
ALTER TABLE "gates" ADD COLUMN "diff_ignored_fields" jsonb NULL, ADD COLUMN "diff_included_fields" jsonb NULL, ADD COLUMN "diff_float_tolerance" double precision NULL;
