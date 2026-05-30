---
type: auto
description: Conventions for Go code in mroki-api, proxy, and shared packages — apply when editing Go under internal/, cmd/, or pkg/.
---

# Go (mroki-api / proxy / pkg)

Applies when editing Go under `internal/`, `cmd/`, or `pkg/`. Full detail in `AGENTS.md`.

## Layering (internal/)

- `domain/traffictesting` — aggregates, value objects with `Parse*`/`New*` constructors, repository
  **interfaces**, and domain errors. No imports from other layers.
- `application/{commands,queries,services}` — CQRS handlers; depend only on domain interfaces;
  validate by constructing value objects; wrap errors with `%w`.
- `infrastructure/persistence/ent` — implements repo interfaces via `*ent.Client`; `mapper.go` maps
  ent ⇄ domain. Assert conformance: `var _ traffictesting.GateRepository = (*gateRepository)(nil)`.
- `interfaces/http/{handlers,middleware}` — stdlib `net/http`; (de)serialize via `pkg/dto`.
- Wiring / dependency injection lives in `cmd/mroki-api/main.go`.

A DB-touching feature usually spans: schema → domain → application → repository/mapper → handler → DTO.

## Conventions

- `gofmt`; `golangci-lint` v2.11.4 (`make api-lint`). Document exported symbols; keep functions focused.
- Table-driven tests with `testify`; `*_test.go` beside code. Run `make api-test` / `go test -race ./...`.
- Mocks are generated via `//go:generate` mockgen into `internal/mocks/` — regenerate with
  `go generate ./...`, never hand-write them.
