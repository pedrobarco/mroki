-- Create "gates" table
CREATE TABLE "gates" ("id" uuid NOT NULL, "live_url" character varying NOT NULL, "shadow_url" character varying NOT NULL, PRIMARY KEY ("id"));
-- Create "requests" table
CREATE TABLE "requests" ("id" uuid NOT NULL, "agent_id" character varying NULL, "method" character varying NOT NULL, "path" character varying NOT NULL, "headers" jsonb NULL, "body" bytea NULL, "created_at" timestamptz NOT NULL, "gate_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "requests_gates_requests" FOREIGN KEY ("gate_id") REFERENCES "gates" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "request_agent_id" to table: "requests"
CREATE INDEX "request_agent_id" ON "requests" ("agent_id");
-- Create index "request_gate_id" to table: "requests"
CREATE INDEX "request_gate_id" ON "requests" ("gate_id");
-- Create index "request_gate_id_created_at" to table: "requests"
CREATE INDEX "request_gate_id_created_at" ON "requests" ("gate_id", "created_at");
-- Create index "request_gate_id_method" to table: "requests"
CREATE INDEX "request_gate_id_method" ON "requests" ("gate_id", "method");
-- Create index "request_gate_id_path" to table: "requests"
CREATE INDEX "request_gate_id_path" ON "requests" ("gate_id", "path");
-- Create "responses" table
CREATE TABLE "responses" ("id" uuid NOT NULL, "type" character varying NOT NULL, "status_code" integer NOT NULL, "headers" jsonb NULL, "body" bytea NULL, "created_at" timestamptz NOT NULL, "request_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "responses_requests_responses" FOREIGN KEY ("request_id") REFERENCES "requests" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "response_request_id" to table: "responses"
CREATE INDEX "response_request_id" ON "responses" ("request_id");
-- Create "diffs" table
CREATE TABLE "diffs" ("id" uuid NOT NULL, "content" jsonb NOT NULL, "created_at" timestamptz NOT NULL, "request_id" uuid NOT NULL, "from_response_id" uuid NOT NULL, "to_response_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "diffs_requests_diff" FOREIGN KEY ("request_id") REFERENCES "requests" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "diffs_responses_diffs_from" FOREIGN KEY ("from_response_id") REFERENCES "responses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "diffs_responses_diffs_to" FOREIGN KEY ("to_response_id") REFERENCES "responses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "diffs_request_id_key" to table: "diffs"
CREATE UNIQUE INDEX "diffs_request_id_key" ON "diffs" ("request_id");
