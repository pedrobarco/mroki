# mroki Roadmap

**Last Updated:** 2026-03-28

All completed, pending, and planned work for mroki. Items use a consistent format:
- `[x]` Complete · `[ ]` Not started
- **P0** Blocker · **P1** High · **P2** Medium · **P3** Nice-to-have

---

## Completed

- [x] **Security & Stability** — RFC 7807 errors, HTTP timeouts, body size limits, graceful shutdown, API key auth, rate limiting (1000 req/min), input validation via value objects
- [x] **Developer Experience** — Diff engine rewrite (gjson/sjson + go-cmp, 30%+ faster), field filtering (whitelist/blacklist + wildcards), TTL cleanup job, CORS support
- [x] **mroki-hub v1** — Vue 3 + TypeScript SPA with gates (list, create, detail), request browser (filtering, sorting, pagination), diff viewer (side-by-side + unified), gate filtering/sorting/pagination, e2e test suite

---

## TODO: Wiring Gaps

Concrete items where the **UI already exists** but shows hardcoded/dummy data or has non-functional elements.

### Gate Model

- [ ] **P1** Add `name` field — UI shows hardcoded names like "checkout-api" (`GateCard.vue`, `GateDetail.vue`). Add to schema, domain model, create/update API, DTO.
- [ ] **P1** Add `created_at` field — UI shows hardcoded "Mar 12, 2026" (`GateDetail.vue`). Add immutable default timestamp to schema.
- [ ] **P2** Add `status` field — "Pause" button exists with no handler (`GateDetail.vue`). Add active/paused status + `PATCH /gates/{id}/status`.

### Gate Statistics

- [ ] **P1** Wire total gates count — Stats bar shows hardcoded `"4"` (`Gates.vue`). Use pagination `total` from API.
- [ ] **P1** Requests in last 24h — Shows hardcoded `"5,241"` (`GateCard.vue`, `GateDetail.vue`). Add aggregate query.
- [ ] **P1** Diff count per gate — Shows hardcoded `"162"` (`GateCard.vue`). Add aggregate query.
- [ ] **P1** Diff rate — Shows hardcoded `"3.1%"` / `"4.2%"` (`GateCard.vue`, `Gates.vue`). Compute `diffs / requests`.
- [ ] **P2** Active agents count — Shows hardcoded `"3"` with dummy names (`Gates.vue`, `GateCard.vue`). Count distinct agents.
- [ ] **P2** Last active timestamp — Shows hardcoded `"2 min ago"` (`GateCard.vue`). Derive from latest request.

### Request List Metadata

- [ ] **P1** Status codes in list view — Hardcoded per row (`RequestList.vue`). Add `live_status_code` / `shadow_status_code` to summary DTO.
- [ ] **P1** Diff count per request — Hardcoded per row (`RequestList.vue`). Add `diff_count` to summary DTO.
- [ ] **P2** Latency per request — Hardcoded `"142ms"` (`RequestList.vue`, `RequestDetail.vue`). Requires agent capture + schema field.

### Dead UI Elements

- [ ] **P1** "Configure" button — No handler, no endpoint (`GateDetail.vue`). Needs `PUT /gates/{id}` + form.
- [ ] **P2** "Copy cURL" button — No click handler (`RequestDetail.vue`). Client-side: generate cURL from request data.
- [ ] **P2** "Export JSON" button — No click handler (`RequestDetail.vue`). Client-side: serialize + download.
- [ ] **P2** "Showing N of M requests" label — Hardcoded count (`GateDetail.vue`). Wire to pagination `total`.

### Dead Navigation Links

- [ ] **P2** "Agents" nav link — Points to `#` (`Header.vue`). No route or page.
- [ ] **P2** "Requests" nav link — Points to `#` (`Header.vue`). No route or page.
- [ ] **P2** "Settings" nav link — Points to `#` (`Header.vue`). No route or page.

### Hardcoded UI State

- [ ] **P3** "API Connected" badge — Always green (`Header.vue`). Wire to `GET /health/ready`.
- [ ] **P3** User avatar "DK" — Hardcoded initials (`Header.vue`). No auth system yet.

---

## TODO: Backend Infrastructure

Pending infrastructure tasks for production readiness.

### Observability & Resilience

- [ ] **P1** Request ID middleware — `X-Request-ID` header generation + propagation through logs and agent.
- [ ] **P1** Circuit breaker in agent — Stop retrying when API is down. Use `gobreaker` with 5-failure threshold.
- [ ] **P1** HTTP connection pooling — Configure `MaxIdleConns`, `IdleConnTimeout` in agent client.
- [ ] **P2** Structured error logging — Add request context (method, path, request ID) to all error logs.
- [ ] **P2** Update API_CONTRACTS.md — Document auth, rate limiting, pagination (currently marked "Planned v2").

### Production Hardening

- [ ] **P2** TLS/HTTPS support — Optional `ListenAndServeTLS` with cert/key config.
- [ ] **P2** Request deduplication — Return 200 for duplicate request IDs instead of error.
- [ ] **P3** Compression middleware — Gzip responses > 1KB.
- [ ] **P3** Config hot-reload — Reload safe settings on SIGHUP without restart.

### Documentation

- [ ] **P2** Create PRODUCTION_READINESS.md — Pre-deployment checklist, monitoring requirements, runbook.
- [ ] **P2** Update MROKI_API.md — Production deployment, security config, performance tuning.
- [ ] **P2** Update MROKI_AGENT.md — Circuit breaker behavior, connection pooling, auth setup.

---

## Roadmap: Future Features

Larger capabilities not yet started, organized by priority.

### Core CRUD Completeness

- [ ] **P1** Delete gate — `DELETE /gates/{id}` with cascade delete.
- [ ] **P1** Update gate — `PUT /gates/{id}` to modify name, live_url, shadow_url.
- [ ] **P2** Delete request — `DELETE /gates/{id}/requests/{request_id}`.
- [ ] **P3** Bulk delete requests — `DELETE /gates/{id}/requests?older_than=30d`.

### Gate Summary Endpoint

- [ ] **P1** `GET /gates/{id}/summary` — Request count (24h), diff count, diff rate, active agents, last active. Unblocks most gate statistic wiring gaps above.
- [ ] **P2** `GET /stats` — Global dashboard: total gates, total agents, requests (24h), diff rate.

### Agents

- [ ] **P2** List agents endpoint — `GET /agents` with distinct agents, last seen, gate associations.
- [ ] **P2** Agent detail endpoint — `GET /agents/{id}` with activity history.
- [ ] **P2** Agents page — Vue page at `/agents` with list and status.
- [ ] **P3** Agent health monitoring — Heartbeats, stale/disconnected detection.

### Cross-Gate Request Explorer

- [ ] **P2** Cross-gate search — `GET /requests?method=POST&path=/api/*&has_diff=true`.
- [ ] **P2** Requests page — Vue page at `/requests` with cross-gate filtering.
- [ ] **P3** Full-text body search — Search within request/response bodies.

### Latency Tracking

- [ ] **P2** Agent latency capture — Record `latency_ms` for live and shadow responses.
- [ ] **P2** Latency in schema/API — Add `latency_ms` to Response entity and DTOs.
- [ ] **P3** Latency analysis — P50/P95/P99 comparison between live and shadow.

### Per-Gate Diff Configuration

- [ ] **P2** Per-gate DiffConfig — Store field filtering rules per gate in database.
- [ ] **P2** Diff config API — `PUT /gates/{id}/diff-config` to manage per-gate rules.
- [ ] **P2** Agent integration — Fetch and apply per-gate diff config.

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
