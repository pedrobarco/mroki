-- Modify "diffs" table
ALTER TABLE "diffs" ADD COLUMN "config" jsonb NULL;
-- Modify "gates" table
ALTER TABLE "gates" ADD COLUMN "diff_sort_arrays" boolean NULL DEFAULT false;
