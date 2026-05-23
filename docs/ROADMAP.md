# mroki Roadmap

**Last Updated:** 2026-05-18

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
- [x] Gate Settings page — dedicated `/gates/:id/settings` page for name, field redaction, diff config (ignored/included fields, float tolerance), and gate deletion
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

The next development phase, organized into five prioritized tracks.

### Phase 1: Release & Deployment Infrastructure

Ship v1 to users — CI/CD, container images, release pipeline, and Helm charts.

- [x] **P1** CI pipeline — GitHub Actions: lint, test, build on push/PR for all components
- [x] **P1** Docker image builds — Multi-arch builds + push to GHCR for mroki-api, mroki-proxy, mroki-hub
- [x] **P1** mroki-hub Dockerfile — Production container (nginx) serving the Vue SPA
- [x] **P1** Release pipeline — Semantic versioning, git-cliff changelog, GitHub Releases
- [x] **P1** Helm charts — Umbrella chart with subcharts (api, proxy, hub), conditional enablement, published to GHCR OCI registry
- [ ] **P1** API versioning — Add `/v1/` prefix to all API routes (`/v1/gates`, `/v1/stats`, etc.). Must happen before wider adoption — adding versioning later is a breaking change. Proxy client updated to use versioned paths. Old unversioned routes can redirect or coexist temporarily

### Phase 2: Security & Data Safety

Fixes and hardening required before any production deployment with real traffic.

- [x] **P0** Constant-time API key comparison — Replace `token != cfg.validKey` with `subtle.ConstantTimeCompare()` in `apikey.go` (timing side-channel vector, one-line fix)
- [x] **P0** Field redaction — Default redacted fields (`headers.Authorization`, `headers.Cookie`, `headers.Set-Cookie`, `headers.X-Api-Key`) applied to stored request/response data automatically. Per-gate `redacted_fields` config to add additional fields (headers and body) to redact. Redacted fields also excluded from diff computation
- [ ] **P1** Per-gate retention — Optional `retention` field on gate schema. Cleanup job uses gate-specific value if set, falls back to global `MROKI_APP_RETENTION` default

### Phase 3: Diff Engine & Accuracy

Fix bugs and close gaps in what the diff engine compares and how results are surfaced.

- [ ] **P0** Remove statusCode from diff wrapper — Remove `statusCode` from the synthetic JSON used for diffing (`{"headers": {...}, "body": ...}` only). Status code is already displayed as standalone metadata in the UI; including it in the diff tree is redundant
- [ ] **P1** Non-JSON body handling — When response body isn't valid JSON, embed it as a raw string in the diff wrapper (`{"headers": {...}, "body": "<raw text>"}`). Diff engine compares it as a single string value, producing a `replace` at `/body` if different. No second diff algorithm needed — stays JSON-only throughout
- [ ] **P1** Content-type auto-detection — Detect JSON vs non-JSON from `Content-Type` header to decide how to embed body in diff wrapper (parsed JSON object vs raw string). Binary content types skipped with metadata note
- [ ] **P2** Regex field matching — Extend `FieldNormalizer` to accept regex patterns alongside gjson paths for `ignored_fields`/`included_fields` (e.g., `"regex:.*_at$"`, `"regex:.*timestamp.*"`). Walk JSON field paths and match against patterns, then delete/keep using existing sjson infrastructure

### Phase 4: Observability

Instrument mroki itself so operators can monitor, trend, and alert using standard tooling (Prometheus + Grafana + AlertManager).

- [ ] **P1** Prometheus metrics — Expose `/metrics` endpoint on both proxy and API. Key metrics: `mroki_requests_total` (counter, labels: gate_id), `mroki_diffs_total` (counter, labels: gate_id), `mroki_response_latency_seconds` (histogram, labels: gate_id, response_type=live|shadow), `mroki_diff_computation_seconds` (histogram), proxy active goroutines, circuit breaker state. Enables diff rate trends, latency percentile comparison (P50/P95/P99), and alerting — all handled by external monitoring stack rather than built into mroki
- [ ] **P2** Top changed paths — Aggregate most-diffed JSON paths across requests for a gate (e.g., "`/body/user/email` changed in 847 requests"). Application-layer aggregation parsing `Diff.Content`

### Phase 5: Diff Workflow & Collaboration

Enable teams to review, triage, and collaborate on diffs.

- [ ] **P1** Diff review status — Add `status` enum field to Request entity (initial values: `new`, `reviewed`; enum for future extensibility). Schema change + API update (`PATCH /gates/{id}/requests/{id}`) + hub UI toggle + filter by status
- [ ] **P2** Batch status update — Bulk update review status by filter (e.g., "mark all unreviewed requests as reviewed"). New API endpoint for bulk update
- [ ] **P2** Diff comments — Per-operation comments tied to `diff_path` + `diff_op` (matches RFC 6902 PatchOp). New `diff_comments` table (request_id, diff_path, diff_op, comment, created_at). Hub: comment icon on each diff line in DiffViewer, click to add/view. Enables team context (e.g., "this timestamp diff is expected")

### Phase 6: Production Hardening

Operational items for real-world deployments at scale.

- [ ] **P2** Proxy health endpoints — Add `/health/live` and `/health/ready` endpoints to mroki-proxy for Kubernetes probes
- [ ] **P2** Bounded callback concurrency — Semaphore in proxy to limit concurrent callback goroutines (`MAX_CONCURRENT_CALLBACKS`, default 200). Drop shadow comparison with warning when full
- [ ] **P2** Compression middleware — Gzip responses > 1KB
- [ ] **P2** Proxy HTTP client configurability — Expose `MaxIdleConns`, `MaxIdleConnsPerHost`, `IdleConnTimeout` via env vars in `newDefaultHTTPClient()`
- [ ] **P2** Create PRODUCTION_READINESS.md — Pre-deployment checklist, monitoring requirements, runbook
- [ ] **P2** Update MROKI_API.md — Production deployment, security config, performance tuning
- [ ] **P2** Non-root containers — Run all Docker images as non-root user. Small Dockerfile change, standard security practice for production
- [ ] **P2** Container image scanning — Integrate Trivy in CI Docker workflow to fail builds on critical/high CVEs in base images
- [ ] **P2** Full-stack integration test in CI — Docker Compose-based test in GitHub Actions that spins up proxy + API + DB, sends traffic, and validates the end-to-end diff flow works. Current CI only runs unit tests
- [ ] **P3** Config hot-reload — Reload safe settings on SIGHUP without restart

### Phase 7: Export & Replay

Debugging and replay workflows.

- [ ] **P2** HAR export — Export individual request as `.har` file (HTTP Archive format) from hub. New API endpoint + hub button alongside existing "Export JSON". Portable to Chrome DevTools, Postman, Charles Proxy
- [ ] **P2** Request replay — Hub "Replay" button prompts user for proxy URL, then sends stored request (method, path, headers, body) to that URL. Creates a new captured request through the normal proxy flow. No schema change needed — proxy URL provided at replay time

### Phase 8: Auth & Multi-tenancy

Required if mroki becomes a shared or hosted tool.

- [ ] **P2** User authentication — Login/signup with session management
- [ ] **P2** Role-based access control — Admin, viewer, operator roles
- [ ] **P2** Settings page — Vue page at `/settings` (API key management, data retention config)
- [ ] **P3** Team/organization support — Multi-tenant gate isolation
- [ ] **P3** User avatar — Dynamic initials in `Header.vue` (currently hardcoded "DK")
