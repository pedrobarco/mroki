---
type: always
description: Core mroki repository conventions and hard guardrails for all work.
---

# mroki — project rules

The canonical engineering guide for this repo is **`AGENTS.md`** at the repository root. Read and
follow it. This rule restates only the non-negotiable guardrails so they always apply.

## Hard guardrails

- **Never hand-edit generated code.** This includes `ent/` (except `ent/schema/`, `ent/generate.go`,
  `ent/migrate/main.go`), `internal/mocks/`, and `web/mroki-hub/src/components/ui/` (shadcn-vue).
  Change the source and regenerate instead.
- **Respect `internal/` layering** (DDD + CQRS). Dependencies point inward; the domain
  (`internal/domain/traffictesting`) depends on nothing. See AGENTS.md → Architecture & layering.
- **Use package managers** — never hand-edit `go.mod`/`go.sum`/`package.json`/lockfiles
  (`go get`, `go mod tidy`, `pnpm add`).
- **Conventional Commits** for all commit messages (`<type>(<scope>): <description>`).
- **The proxy must never fail live traffic** — API/shadow errors are best-effort and only logged.
- **Never commit secrets** (e.g. `MROKI_APP_API_KEY`); they are configuration.
- Run `make lint` and `make test` before opening a PR; add/update tests for behavior changes.
