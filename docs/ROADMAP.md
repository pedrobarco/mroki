# mroki Roadmap

**Last Updated:** 2026-04-27

All completed, pending, and planned work for mroki. Items use a consistent format:
- `[x]` Complete · `[ ]` Not started
- **P0** Blocker · **P1** High · **P2** Medium · **P3** Nice-to-have

---

## v1 — Complete

Everything below shipped as part of the v1 milestone.

### Core Platform

- [x] **Security & Stability** — RFC 7807 errors, HTTP timeouts, body size limits, graceful shutdown, API key auth, rate limiting (1000 req/min), input validation via value objects
- [x] **Developer Experience** — Diff engine rewrite (gjson/sjson + go-cmp, 30%+ faster), field filtering (whitelist/blacklist + wildcards), TTL cleanup job, CORS support
- [x] **Server-Side Diff Computation** — Moved diff computation from proxy to mroki-api. Proxy sends raw responses; API computes diffs on ingest. Standalone proxy mode retains local diff computation. Backward compatible (API accepts pre-computed diffs).
- [x] **Proxy/Caddy Feature Parity** — Brought caddy-mroki to feature parity with standalone proxy: sampling rate, max body size check, diff options (ignored/included fields, float tolerance). Caddy operates in standalone mode only (local diff + print).

### Gate CRUD & Model

- [x] Gate create, update (`PATCH /gates/{id}`), delete (`DELETE /gates/{id}` with cascade)
- [x] `name` field (unique, mutable), `created_at` (immutable default timestamp)
- [x] Unique + immutable URL pair — `(live_url, shadow_url)` composite unique index, 409 Conflict on duplicates

### Per-Gate Diff Configuration

- [x] `DiffConfig` value object stored as JSON fields on gate schema (ignored fields, included fields, float tolerance)
- [x] `PATCH /gates/{id}` accepts `diff_config`; server-side diff computation applies per-gate config on ingest

### Gate Statistics

- [x] Per-gate stats embedded in `GET /gates` and `GET /gates/{id}` — `request_count_24h`, `diff_count_24h`, `diff_rate`, `last_active`
- [x] Global `GET /stats` — total gates, requests (24h), diff rate

### Latency Tracking

- [x] Proxy captures round-trip time per response via `time.Since()`
- [x] `latency_ms` (required `int64`) on Response entity, DTOs, list/detail views

### Backend Infrastructure

- [x] Request ID middleware (`X-Request-ID` generation + propagation)
- [x] Circuit breaker in proxy (`failsafe-go` retry + circuit breaker RoundTripper stack)
- [x] HTTP connection pooling (`MaxIdleConns`, `IdleConnTimeout`)
- [x] Structured error logging (request context in all error/warn logs)
- [x] Configurable server timeouts (`ReadTimeout`, `WriteTimeout`, `IdleTimeout` via env vars)
- [x] TLS timeout alignment (reduced `TLSHandshakeTimeout` to 5s to match `LIVE_TIMEOUT`)
- [x] API timeout budget validation (warn at startup if retry config exceeds `API_TIMEOUT`)
- [x] Caddy module maintenance (standalone-only docs, zap→slog bridge, Dockerfile fix)

### mroki-hub v1

- [x] Vue 3 + TypeScript SPA — gates (list, create, detail), request browser (filtering, sorting, pagination), diff viewer (side-by-side + unified), e2e test suite
- [x] Gate statistics wired — total gates count, requests 24h, diff count, diff rate, last active
- [x] Request list metadata — status codes, diff indicator, latency (all via eager-loaded edges)
- [x] "Configure" dialog — update name and diff config (ignored/included fields, float tolerance)
- [x] "Copy cURL" button — dropdown with Live/Shadow options, generates cURL with method/URL/headers/body
- [x] "Export JSON" button — downloads full request detail as `request-{id}.json`
- [x] "Showing N of M requests" label — wired to pagination
- [x] Gate delete button — AlertDialog confirmation with cascade delete
- [x] Removed dead UI — "API Connected" badge, dead nav links, "Active" badge, "Pause" button

### Documentation (v1)

- [x] Update MROKI_PROXY.md — circuit breaker behavior, connection pooling, auth setup
- [x] Update API_CONTRACTS.md — auth, rate limiting, and pagination fully documented

---

## v2 Roadmap

The next development phase, organized into four prioritized tracks.

### Phase 1: Production Readiness

Low-effort, high-value items for real-world deployments.

- [ ] **P2** Create PRODUCTION_READINESS.md — Pre-deployment checklist, monitoring requirements, runbook
- [ ] **P2** Update MROKI_API.md — Production deployment, security config, performance tuning
- [ ] **P2** Compression middleware — Gzip responses > 1KB
- [ ] **P2** Proxy HTTP client configurability — Expose `MaxIdleConns`, `MaxIdleConnsPerHost`, `IdleConnTimeout` via env vars in `newDefaultHTTPClient()`
- [ ] **P3** Config hot-reload — Reload safe settings on SIGHUP without restart

### Phase 2: Observability & Analytics

Make mroki more useful for ongoing monitoring and performance analysis.

- [ ] **P1** Latency analysis — P50/P95/P99 comparison between live and shadow (API endpoint + hub visualization)
- [ ] **P2** WebSocket live feed — Real-time request stream per gate
- [ ] **P2** Diff alerts — Configurable thresholds for diff rate alerts
- [ ] **P3** Webhook notifications — Notify external systems on diffs or error spikes

### Phase 3: Export & Tooling

Power-user features for debugging workflows.

- [ ] **P2** Bulk export — `GET /gates/{id}/requests/export?format=json`
- [ ] **P2** HAR export — Export in HTTP Archive format
- [ ] **P3** Request replay — Resend captured requests to live or shadow endpoints

### Phase 4: Auth & Multi-tenancy

Required if mroki becomes a shared or hosted tool.

- [ ] **P2** User authentication — Login/signup with session management
- [ ] **P2** Role-based access control — Admin, viewer, operator roles
- [ ] **P2** Settings page — Vue page at `/settings` (API key management, data retention config)
- [ ] **P3** Team/organization support — Multi-tenant gate isolation
- [ ] **P3** User avatar — Dynamic initials in `Header.vue` (currently hardcoded "DK")
