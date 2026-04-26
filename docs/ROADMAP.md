# mroki Roadmap

**Last Updated:** 2026-04-26

All completed, pending, and planned work for mroki. Items use a consistent format:
- `[x]` Complete · `[ ]` Not started
- **P0** Blocker · **P1** High · **P2** Medium · **P3** Nice-to-have

---

## Completed

- [x] **Security & Stability** — RFC 7807 errors, HTTP timeouts, body size limits, graceful shutdown, API key auth, rate limiting (1000 req/min), input validation via value objects
- [x] **Developer Experience** — Diff engine rewrite (gjson/sjson + go-cmp, 30%+ faster), field filtering (whitelist/blacklist + wildcards), TTL cleanup job, CORS support
- [x] **mroki-hub v1** — Vue 3 + TypeScript SPA with gates (list, create, detail), request browser (filtering, sorting, pagination), diff viewer (side-by-side + unified), gate filtering/sorting/pagination, e2e test suite
- [x] **Server-Side Diff Computation** — Moved diff computation from proxy to mroki-api. Proxy sends raw responses; API computes diffs on ingest. Standalone proxy mode retains local diff computation. Backward compatible (API accepts pre-computed diffs).
- [x] **Proxy/Caddy Feature Parity** — Brought caddy-mroki to feature parity with standalone proxy: sampling rate, max body size check, diff options (ignored/included fields, float tolerance). Added sampling rate support to proxy. Caddy operates in standalone mode only (local diff + print).

---

## TODO: Wiring Gaps

Concrete items where the **UI already exists** but shows hardcoded/dummy data or has non-functional elements.

### Gate Model

- [x] **P1** Add `name` field — Unique, mutable name. Added to schema, domain model, create API, DTO, and wired in `GateCard.vue`, `GateDetail.vue`, `GateForm.vue`.
- [x] **P1** Add `created_at` field — Immutable default timestamp. Wired in `GateDetail.vue`. Added `created_at` sort field.
- [x] **P1** Unique + immutable URL pair — `(live_url, shadow_url)` composite unique index. Both fields immutable after creation. 409 Conflict on duplicates.

### Gate Statistics

- [x] **P1** Wire total gates count — Stats bar wired to `GET /stats` → `total_gates` (`Gates.vue`).
- [x] **P1** Requests in last 24h — Wired to `gate.stats.request_count_24h` (`GateCard.vue`, `GateDetail.vue`).
- [x] **P1** Diff count per gate — Wired to `gate.stats.diff_count_24h` (`GateCard.vue`).
- [x] **P1** Diff rate — Wired to `gate.stats.diff_rate` / `GET /stats` → `total_diff_rate` (`GateCard.vue`, `Gates.vue`).
- [x] **P2** Last active timestamp — Wired to `gate.stats.last_active` with relative formatting (`GateCard.vue`).

### Request List Metadata

- [x] **P1** Status codes in list view — Wired to `request.live_response.status_code` / `request.shadow_response.status_code` via eager-loaded responses.
- [x] **P1** Diff indicator per request — Wired to `request.has_diff` via eager-loaded diff edge.
- [x] **P2** Latency per request — Captured in proxy, stored as `latency_ms` on response schema. Wired to `request.live_response.latency_ms` / `request.shadow_response.latency_ms` in list and detail views.

### Dead UI Elements

- [x] **P1** "Configure" button (backend) — `PATCH /gates/{id}` endpoint implemented. Supports updating `name` and `diff_config` (ignored fields, included fields, float tolerance).
- [x] **P2** "Configure" button (frontend) — Wired in `GateDetail.vue` via `GateConfigDialog.vue` dialog. Calls `PATCH /gates/{id}` with name and diff config (ignored/included fields, float tolerance). Added `DiffConfig` type, `UpdateGatePayload`, and `updateGate()` API function. Removed dead "Pause" button (gate status feature removed).
- [ ] **P2** "Copy cURL" button — No click handler (`RequestDetail.vue`). Client-side: generate cURL from request data.
- [ ] **P2** "Export JSON" button — No click handler (`RequestDetail.vue`). Client-side: serialize + download.
- [ ] **P2** "Showing N of M requests" label — Hardcoded count (`GateDetail.vue`). Wire to pagination `total`.
- [ ] **P2** Gate delete button — Backend `DELETE /gates/{id}` exists but no UI button or confirmation dialog.
- [ ] **P2** Response headers viewer — Response headers are captured but not displayed in `RequestDetail.vue`.
- [ ] **P2** Request body viewer — Request body is captured but not viewable in `RequestDetail.vue`.

### Hardcoded UI State

- [ ] **P3** "API Connected" badge — Always green (`Header.vue`). Wire to `GET /health/ready`.
- [ ] **P3** User avatar "DK" — Hardcoded initials (`Header.vue`). No auth system yet.

---

## TODO: Backend Infrastructure

Pending infrastructure tasks for production readiness.

### Observability & Resilience

- [x] **P1** Request ID middleware — `X-Request-ID` header generation + propagation through logs and proxy.
- [x] **P1** Circuit breaker in proxy — Resilient HTTP client with `failsafe-go` retry + circuit breaker RoundTripper stack.
- [x] **P1** HTTP connection pooling — Configure `MaxIdleConns`, `IdleConnTimeout` in proxy client.
- [x] **P2** Structured error logging — All error/warn logs now include `request.id`, `request.method`, `request.path`. Normalized to typed `slog.String`/`slog.Int`/`slog.Duration` style throughout.

### Production Hardening

- [x] **P2** Configurable server timeouts — `ReadTimeout`, `WriteTimeout`, `IdleTimeout` exposed via env vars with previous hardcoded values as defaults.
- [x] **P2** Align transport TLS timeout with context — Reduced `TLSHandshakeTimeout` from 10s to 5s to match default `LIVE_TIMEOUT`.
- [x] **P2** Validate API timeout budget — Warn at startup if retry config (retries × backoff) could exceed `API_TIMEOUT`. Implemented as a `SeverityWarning` in the validation system.
- [ ] **P3** Compression middleware — Gzip responses > 1KB.
- [ ] **P3** Config hot-reload — Reload safe settings on SIGHUP without restart.

### Documentation

- [ ] **P2** Create PRODUCTION_READINESS.md — Pre-deployment checklist, monitoring requirements, runbook.
- [ ] **P2** Update MROKI_API.md — Production deployment, security config, performance tuning.
- [x] **P2** Update MROKI_PROXY.md — Circuit breaker behavior, connection pooling, auth setup.
- [ ] **P2** Update API_CONTRACTS.md — Document auth, rate limiting, pagination (currently marked "Planned v2").

### Technical Debt

- [x] **P2** Caddy module stale — Documented as standalone-only, bridged Caddy zap logger to slog, fixed ServeHTTP as terminating handler, fixed Dockerfile build.
- [ ] **P3** Proxy HTTP client not configurable — Live/shadow `http.Client` connection pool settings (`MaxIdleConns`, `MaxIdleConnsPerHost`, `IdleConnTimeout`) are hardcoded in `newDefaultHTTPClient()`. Could be exposed via env vars for production tuning.

---

## Roadmap: Future Features

Larger capabilities not yet started, organized by priority.

### Core CRUD Completeness

- [x] **P1** Delete gate — `DELETE /gates/{id}` with cascade delete.
- [x] **P1** Update gate — `PATCH /gates/{id}` to modify name and diff config (live_url and shadow_url are immutable).

### Gate Summary Endpoint

- [x] **P1** Embed per-gate stats in gate responses — Stats (`request_count_24h`, `diff_count_24h`, `diff_rate`, `last_active`) embedded in `GET /gates` and `GET /gates/{id}` responses. Replaces the originally planned `GET /gates/{id}/summary` to avoid N+1 calls.
- [x] **P2** `GET /stats` — Global dashboard: total gates, requests (24h), diff rate.

### Latency Tracking

- [x] **P2** Agent latency capture — Proxy measures round-trip time per response via `time.Since()`.
- [x] **P2** Latency in schema/API — `latency_ms` (required `int64`) added to Response entity, DTOs, and list/detail views.
- [ ] **P3** Latency analysis — P50/P95/P99 comparison between live and shadow.

### Per-Gate Diff Configuration

- [x] **P2** Per-gate DiffConfig — `DiffConfig` value object stored as JSON fields on gate schema (ignored fields, included fields, float tolerance).
- [x] **P2** Diff config API — `PATCH /gates/{id}` accepts `diff_config` in the request body.
- [x] **P2** API-side diff options — Server-side diff computation applies per-gate `DiffConfig` (ignored fields, included fields, float tolerance) when computing diffs on request ingest.

### Export & Tooling

- [ ] **P3** Bulk export — `GET /gates/{id}/requests/export?format=json`.
- [ ] **P3** HAR export — Export in HTTP Archive format.
- [ ] **P3** Request replay — Resend captured requests to live or shadow endpoints.

### Settings & Configuration

- [ ] **P3** Settings page — Vue page at `/settings`.
- [ ] **P3** API key management — Create, rotate, revoke keys from UI.
- [ ] **P3** Data retention config — Configure auto-deletion from UI.

### Real-time & Notifications

- [ ] **P3** WebSocket live feed — Real-time request stream per gate.
- [ ] **P3** Diff alerts — Configurable thresholds for diff rate alerts.
- [ ] **P3** Webhook notifications — Notify external systems on diffs or error spikes.

### Authentication & Multi-tenancy

- [ ] **P3** User authentication — Login/signup with session management.
- [ ] **P3** Role-based access control — Admin, viewer, operator roles.
- [ ] **P3** Team/organization support — Multi-tenant gate isolation.
