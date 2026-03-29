-- Create index "gate_live_url_shadow_url" to table: "gates"
CREATE UNIQUE INDEX "gate_live_url_shadow_url" ON "gates" ("live_url", "shadow_url");
