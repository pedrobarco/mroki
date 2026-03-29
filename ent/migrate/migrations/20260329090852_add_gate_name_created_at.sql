-- Modify "gates" table
ALTER TABLE "gates" ADD COLUMN "name" character varying NOT NULL, ADD COLUMN "created_at" timestamptz NOT NULL;
-- Create index "gates_name_key" to table: "gates"
CREATE UNIQUE INDEX "gates_name_key" ON "gates" ("name");
