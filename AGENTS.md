# AGENTS.md

Guidance for AI coding agents working in this repository. Humans should read the
[Development Guide](docs/development/DEVELOPMENT.md) and
[Contributing Guide](docs/development/CONTRIBUTING.md); this file distills the same conventions
into rules an agent must follow.

## Overview

**mroki** mirrors live HTTP traffic to a shadow service, diffs the JSON responses, and surfaces the
differences. It is a Go + Vue monorepo with four components:

- **mroki-proxy** (`cmd/mroki-proxy`, `pkg/proxy`) — forwards each request to live + shadow, returns
  the live response, sends raw responses to the API. Never fails live traffic (best-effort).
- **mroki-api** (`cmd/mroki-api`, `internal/`) — REST API, server-side JSON diffing, persistence.
- **mroki-hub** (`web/mroki-hub`) — Vue 3 web UI.
- **caddy-mroki** (`cmd/caddy-mroki`, `pkg/caddymodule`) — Caddy module, standalone diffing.

See [Architecture Overview](docs/architecture/OVERVIEW.md) for data flow and design decisions.

## Repository map

```
cmd/                 Executables (mroki-api, mroki-proxy, caddy-mroki)
internal/            Private API code (DDD + CQRS layering, see below)
pkg/                 Public libraries: proxy, diff, client, dto, logger, ratelimit, jsontree, caddymodule
ent/                 GENERATED Ent code + schema/ + migrate/migrations/ (see Database)
web/mroki-hub/       Vue 3 + TypeScript SPA (pnpm)
docs/                Documentation (getting-started, production, api, architecture, development)
build/               Dockerfiles (build/package/) and dev stack (build/dev/compose.yaml)
deployments/         Docker Compose + Kubernetes/Helm manifests
.github/workflows/   CI (ci.yaml orchestrates reusable _go/_pnpm/_docker/_helm/_release)
```

## Setup & common commands

Use the `Makefile` (run `make help` for the full list). Prefer these over ad-hoc commands.

```bash
make dev-up        # Start full dev stack (db + api + proxy + hub) via Docker Compose
make build         # Build all components
make test          # Run all tests (api + proxy + hub unit + hub e2e)
make lint          # golangci-lint + hub lint
make api-test      # go test ./cmd/mroki-api/... ./internal/... ./pkg/...
make proxy-test    # go test ./cmd/mroki-proxy/...
```

Prerequisites: Go 1.26+, Docker + Docker Compose, **Node.js 22 (LTS)** with **pnpm 10**, Make. Node 22
matches CI (`.github/actions/setup-node`) and the hub Docker image (`node:22-alpine`); the minimum is
Node 20.19+ (Vite 7 / ESLint 10 / Vitest 4 — the 22.x line needs 22.13+). Install pre-commit hooks
once with `pre-commit install`.

## Architecture & layering (internal/)

mroki-api follows **DDD + CQRS + hexagonal** architecture. Respect the dependency direction —
dependencies point **inward**, the domain depends on nothing:

- `internal/domain/traffictesting/` — Domain layer. Aggregates (`Gate`, `Request`, `Response`,
  `Diff`), value objects with `Parse*`/`New*` constructors (e.g. `GateID`, `GateName`, `GateURL`,
  `Path`, `HTTPMethod`, `Headers`), repository **interfaces** (`GateRepository`,
  `RequestRepository`, `StatsRepository`), and domain errors. No imports from other layers.
- `internal/application/{commands,queries,services}/` — CQRS handlers (e.g. `CreateGateHandler`,
  `ListGatesHandler`, `ResponseComparer`). Depend only on domain interfaces. Do validation by
  constructing value objects; wrap errors with `%w`.
- `internal/infrastructure/persistence/ent/` — Implements the domain repository interfaces using
  `*ent.Client`. `mapper.go` converts between generated `ent.*` types and domain models. Assert
  conformance with `var _ traffictesting.GateRepository = (*gateRepository)(nil)`.
- `internal/interfaces/http/{handlers,middleware}/` — HTTP layer (stdlib `net/http`, Go 1.22+
  routing). Handlers call application handlers and (de)serialize via `pkg/dto`.
- `internal/config/`, `internal/infrastructure/jobs/` — config + background jobs (TTL cleanup).
- Wiring / dependency injection lives in `cmd/mroki-api/main.go`.

When adding a feature that touches the database, expect to change all layers: schema → domain →
application → repository/mapper → handler → DTO, plus tests at each layer.

## Working on an issue / non-trivial change

For **non-trivial or multi-layer** work (a new feature/behavior, a change spanning more than one
`internal/` layer, or touching multiple files), follow this loop:

1. Restate the issue/goal and identify the layers it affects.
2. Present a concise **per-layer summary of the planned changes** and wait for approval before
   implementing.
3. Implement layer by layer (schema → domain → application → repository/mapper → handler → DTO).
4. Add or update tests; run `make lint` and `make test` (or the scoped targets) until green.
5. Do a final review of code, tests, and docs together for consistency before handing back.

Trivial changes (typos, one-line fixes, isolated doc tweaks) skip the approval gate.

## Go conventions

- Format with `gofmt`; lint with `golangci-lint` (v2.11.4 — match CI). Run `make api-lint`.
- Table-driven tests, `*_test.go` next to code, `testify` for assertions.
- Mocks are generated with `mockgen` via `//go:generate` directives into `internal/mocks/...`.
  Regenerate with `go generate ./...` — do not hand-write mock files.
- Document exported symbols. Keep functions short and focused. Wrap errors with context.

## Hub (Vue / TypeScript) conventions

Work inside `web/mroki-hub` with `pnpm` (see its [README](web/mroki-hub/README.md)).

- **TypeScript is required** in every Vue component: `<script setup lang="ts">` (enforced by ESLint
  `vue/block-lang`). Plain `<script setup>` fails lint.
- Composition API with `<script setup>`. Tailwind CSS v4 + shadcn-vue.
- **Use semantic CSS color tokens** (`bg-background`, `text-foreground`, `bg-primary`,
  `text-muted-foreground`, …) — never hardcoded colors like `bg-white`/`text-gray-900`.
- Run `pnpm lint` and `pnpm format` before committing. Unit tests: `pnpm test:unit` (vitest);
  e2e: `pnpm test:e2e` (Playwright, spins up the backend stack).

## Database: Ent schema + migrations

- Schema lives in `ent/schema/` (`gate.go`, `request.go`, `response.go`, `diff.go`).
- After editing schema, regenerate with `go generate ./ent/...`.
- Generate a versioned migration with `make api-migrate name=<description>` (written to
  `ent/migrate/migrations/`). Migrations are applied by the `mroki-db-migrator` (Atlas) image —
  **not** by the API at startup.
- Storage: headers JSONB, bodies JSONB, diff content TEXT (RFC 6902 JSON Patch). All relations
  `ON DELETE CASCADE`.

## Generated code — do not hand-edit

- All of `ent/` **except** `ent/schema/`, `ent/generate.go`, and `ent/migrate/main.go` is generated.
- Mock files under `internal/mocks/`.
- `web/mroki-hub/src/components/ui/` (shadcn-vue) is generated.
Change the source (schema, `//go:generate` target, component generator) and regenerate instead.

## Testing

- Go: `make test` or `go test -race ./...`. CI runs `gotestsum ./...` + `golangci-lint` + build.
- Always add/adjust tests for behavior changes; update existing tests affected by your change.
- Hub: `pnpm test:unit` and `pnpm test:e2e`.

## Commits & pull requests

- **Conventional Commits** are enforced (commitizen `commit-msg` hook). Format:
  `<type>(<scope>): <description>`.
- Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`, `build`, `style`.
- Common scopes: `proxy`, `api`, `hub`, `caddy`, `diff`, `docs`, `gate`, `domain`, `config`,
  `security`, `helm`, `infra`.
- One focused change per PR; include tests; update docs when behavior changes. Branch from `main`
  as `feat/...`, `fix/...`, `docs/...`.

## CI/CD

`.github/workflows/ci.yaml` runs on PRs and pushes to `main`: lint + test + build for Go
(`_go.yaml`) and the hub (`_pnpm.yaml`). On `main`/tags it builds and pushes Docker images
(`_docker.yaml`), packages Helm (`_helm.yaml`), and on `v*` tags cuts a release (`_release.yaml`,
changelog via `cliff.toml`). Make sure `make lint` and `make test` pass locally before opening a PR.

## Documentation map

- Getting started: `docs/getting-started/` (FULL_STACK, STANDALONE_PROXY, CADDY_MODULE)
- API: `docs/api/` (REFERENCE, WALKTHROUGH)
- Architecture: `docs/architecture/` (OVERVIEW, DIFF_ANALYSIS)
- Production: `docs/production/` (CONFIGURATION for all env vars, SECURITY, MONITORING, deploys)
- Development: `docs/development/` (DEVELOPMENT, CONTRIBUTING)

## Safety rules & gotchas

- Manage dependencies with the right package manager (`go get` / `go mod tidy`, `pnpm add`) — do not
  hand-edit `go.mod`, `go.sum`, or `package.json`/lockfiles.
- Never hand-edit generated code (see above) — regenerate.
- The proxy must never fail live traffic; API/shadow errors are best-effort and only logged.
- Only JSON responses are diffed; diffing is server-side in mroki-api (proxy/caddy diff only in
  standalone mode).
- Do not commit secrets. API keys are config (`MROKI_APP_API_KEY`, min 16 chars); see
  [Configuration](docs/production/CONFIGURATION.md) and [Security](docs/production/SECURITY.md).
- Keep changes scoped to the request; surface related downstream changes rather than expanding scope
  silently.
