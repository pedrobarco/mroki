---
type: auto
description: Ent schema and Atlas migration workflow — apply when changing ent/schema or ent/migrate.
---

# Database (Ent + Atlas)

Applies when changing the database schema or migrations. Full detail in `AGENTS.md`.

- Schema lives in `ent/schema/` (`gate.go`, `request.go`, `response.go`, `diff.go`).
- After editing schema, regenerate generated code: `go generate ./ent/...`.
- Generate a versioned migration: `make api-migrate name=<description>` (written to
  `ent/migrate/migrations/`).
- Migrations are applied by the `mroki-db-migrator` (Atlas) image — **not** by the API at startup.
- Everything in `ent/` except `ent/schema/`, `ent/generate.go`, and `ent/migrate/main.go` is
  generated — never hand-edit it.
- Storage: headers JSONB, bodies JSONB, diff content TEXT (RFC 6902 JSON Patch); relations cascade.
