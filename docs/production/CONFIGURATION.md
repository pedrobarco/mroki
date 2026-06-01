# Configuration Reference

This is the single source of truth for all mroki configuration. Each component is configured independently — refer to the relevant section for your deployment.

## Configuration Methods

| Method | Components | Notes |
|--------|-----------|-------|
| Environment variables | mroki-api, mroki-proxy, mroki-hub | Prefixed with `MROKI_APP_` (api/proxy) or `VITE_` (hub) |
| `.env` files | mroki-api, mroki-proxy, mroki-hub | Loaded automatically when present in working directory |
| Caddyfile directives | caddy-mroki | Uses `mroki_gate` block syntax inside Caddyfile |

---

## mroki-api

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `MROKI_APP_API_KEY` | Yes | — | API key for authentication |
| `MROKI_APP_PORT` | No | `8090` | Server port |
| `MROKI_APP_RATE_LIMIT` | No | `1000` | Requests per minute per IP |
| `MROKI_APP_MAX_BODY_SIZE` | No | `10485760` | Request body size limit in bytes (10 MB) |
| `MROKI_APP_CORS_ORIGINS` | No | _(disabled)_ | Comma-separated allowed origins |
| `MROKI_APP_RETENTION` | No | `0` | Request retention duration (Go duration format, `0` = keep forever) |
| `MROKI_APP_CLEANUP_INTERVAL` | No | `1h` | Cleanup job interval (Go duration format) |
| `MROKI_APP_READ_TIMEOUT` | No | `15s` | Server read timeout |
| `MROKI_APP_WRITE_TIMEOUT` | No | `30s` | Server write timeout (must be ≥ read timeout) |
| `MROKI_APP_IDLE_TIMEOUT` | No | `60s` | Server idle timeout (must be ≥ write timeout) |
| `MROKI_APP_DATABASE_MAX_CONNS` | No | `25` | Connection pool max connections |
| `MROKI_APP_DATABASE_MIN_CONNS` | No | `5` | Connection pool min connections |
| `MROKI_APP_DATABASE_MAX_CONN_IDLE` | No | `5m` | Max idle time for a pooled connection |
| `MROKI_APP_DATABASE_MAX_CONN_LIFE` | No | `1h` | Max lifetime of a pooled connection |

> **Schema migrations** are not configured via environment variables. They are applied by the `mroki-db-migrator` image (Atlas) — a Helm `pre-install`/`pre-upgrade` Job on Kubernetes (`api.migration.*`, including `baseline` for pre-existing databases) and a one-shot service on Docker Compose. See [Kubernetes → Database migrations](KUBERNETES.md#database-migrations).

---

## mroki-proxy

The proxy supports two mutually exclusive operating modes: **API mode** and **Standalone mode**. You must configure exactly one.

### Common

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_PORT` | No | `8080` | Proxy server port |
| `MROKI_APP_ADMIN_PORT` | No | `8081` | Admin server port for health endpoints — must differ from `MROKI_APP_PORT` |
| `MROKI_APP_LIVE_TIMEOUT` | No | `5s` | Live request timeout — blocks client response, keep tight |
| `MROKI_APP_SHADOW_TIMEOUT` | No | `10s` | Shadow request timeout — does not block client |
| `MROKI_APP_MAX_BODY_SIZE` | No | `10485760` | Skip shadow for requests above this size in bytes (`0` = unlimited) |
| `MROKI_APP_SAMPLING_RATE` | No | `1.0` | Shadow traffic sampling rate (`0.0`–`1.0`, `1.0` = 100%) |
| `MROKI_APP_SHADOW_RULES` | No | _(deny non-idempotent)_ | Shadow matching rules — see [Shadow Matching Rules](#shadow-matching-rules) |
| `MROKI_APP_READ_TIMEOUT` | No | `30s` | Server read timeout |
| `MROKI_APP_WRITE_TIMEOUT` | No | `60s` | Server write timeout (must be ≥ live timeout) |
| `MROKI_APP_IDLE_TIMEOUT` | No | `120s` | Server idle timeout |
| `MROKI_APP_HTTP_CLIENT_MAX_IDLE_CONNS` | No | `100` | Outbound idle connection pool size across all hosts (`0` = unlimited) |
| `MROKI_APP_HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST` | No | `10` | Outbound idle connections kept per host (`0` = Go default of 2) |
| `MROKI_APP_HTTP_CLIENT_MAX_CONNS_PER_HOST` | No | `100` | Limit on total outbound connections per host (`0` = unlimited) |
| `MROKI_APP_HTTP_CLIENT_IDLE_CONN_TIMEOUT` | No | `90s` | How long an idle outbound connection is kept before closing (`0` = no timeout) |

#### Health endpoints

The proxy exposes health endpoints on the admin port (`MROKI_APP_ADMIN_PORT`, default `8081`), kept separate from the main proxy port so they never collide with proxied traffic forwarded to the upstream service.

| Endpoint | Purpose | Responses |
|----------|---------|-----------|
| `GET /health/live` | Liveness — process is running | `200 OK` |
| `GET /health/ready` | Readiness — accepting traffic | `200 OK` when ready, `503 Service Unavailable` during startup or shutdown |

### API Mode (Recommended)

Fetches gate configuration (live/shadow URLs) from mroki-api on startup.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_API_URL` | Yes | — | mroki-api server URL |
| `MROKI_APP_GATE_ID` | Yes | — | Gate ID from mroki-api (UUID) |
| `MROKI_APP_API_KEY` | Yes | — | API key for authentication |
| `MROKI_APP_API_TIMEOUT` | No | `30s` | Overall deadline for API calls including all retries |
| `MROKI_APP_MAX_RETRIES` | No | `3` | Maximum retry attempts for API requests |
| `MROKI_APP_RETRY_DELAY` | No | `1s` | Initial delay between retries, doubles each attempt |
| `MROKI_APP_CB_FAILURE_THRESHOLD` | No | `5` | Circuit breaker: consecutive failures before opening |
| `MROKI_APP_CB_DELAY` | No | `1m` | Circuit breaker: delay before transitioning from open to half-open |
| `MROKI_APP_CB_SUCCESS_THRESHOLD` | No | `2` | Circuit breaker: successes in half-open state before closing |

### Standalone Mode

Uses hardcoded URLs — no communication with mroki-api.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_LIVE_URL` | Yes | — | Live/production service URL |
| `MROKI_APP_SHADOW_URL` | Yes | — | Shadow/experimental service URL |

### Field Redaction

Sensitive field values (headers and JSON body) are replaced with `[REDACTED]` before storage or diff computation. A default set (`Authorization`, `Cookie`, `Set-Cookie`, `X-Api-Key` headers) is always redacted. In API mode, extra redacted fields come from the gate configuration in mroki-api.

> **Note:** Requests forwarded to the shadow service include a fixed `X-Mroki-Mode: shadow` header so downstream systems can identify shadow traffic. It is added to shadow requests only — live requests are never modified — and is intentionally **not** redacted so its value stays visible for reference in stored request data. The header name is not configurable.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_REDACTED_FIELDS` | No | _(none)_ | Comma-separated additional fields to redact (gjson path notation). Redacted fields are also excluded from diff computation. |

### Shadow Matching Rules

Selectively shadow requests based on HTTP method and path. By default the proxy mirrors every request to the shadow service, including infrastructure routes (`/metrics`, `/health`) and non-idempotent requests that may cause side effects.

`MROKI_APP_SHADOW_RULES` is a comma-separated list of `ACTION METHOD:path` entries:

- **ACTION** — `allow` (shadow it) or `deny` (skip shadow, live-only)
- **METHOD** — an HTTP method (e.g. `POST`) or `*` for any method
- **path** — a path pattern (e.g. `/health/*`, `*.json`, `*`)

Path patterns use the same semantics as Caddy's [`path` matcher](https://caddyserver.com/docs/caddyfile/matchers#path), so the standalone proxy and the `caddy-mroki` module behave identically. The meaning of the `*` wildcard depends on where it sits:

- A bare `*` matches **any** path.
- A **trailing** `*` (the only wildcard) is a recursive **prefix** match that crosses `/`. For example, `/admin/*` matches `/admin/users` and `/admin/users/42`.
- A **leading** `*` (the only wildcard) is a **suffix** match that crosses `/`. For example, `*.json` matches `/api/data.json`.
- A **leading and trailing** `*` (exactly two wildcards) is a **substring** match. For example, `*/admin/*` matches any path containing `/admin/`.
- Any other `*` (mid-pattern or multiple) matches a **single path segment** and does **not** cross `/` (via Go's `path.Match`). For example, `/gates/*/requests/*/details` matches `/gates/abc/requests/def/details` but not `/gates/a/b/requests/c/details`.

Matching is **case-insensitive**, and doubled slashes in the request path are merged before matching (unless the pattern itself contains `//`).

Rules are evaluated in definition order; the **first match wins**. Requests that match no rule are shadowed.

A set of **base rules** is **always** appended as the final, catch-all entries: deny `POST`, `PUT`, `DELETE`, and `PATCH` (so only `GET`, `HEAD`, and `OPTIONS` are shadowed by default). Your `MROKI_APP_SHADOW_RULES` are evaluated **before** the base rules — so you can **override** them per pattern (e.g. `allow POST:/api/v1/search`), but you cannot accidentally drop the write-protection by configuring custom rules. To shadow all writes, add explicit `allow` rules for those methods.

```bash
# Deny everything under /health and /admin, allow one search endpoint;
# base rules still deny all other writes
MROKI_APP_SHADOW_RULES="deny *:/health/*,deny *:/admin/*,allow POST:/api/v1/search"
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_SHADOW_RULES` | No | _(none — base rules deny POST/PUT/DELETE/PATCH)_ | Comma-separated `ACTION METHOD:path` rules. First match wins; unmatched requests are shadowed. Evaluated before the always-present base rules. |

> **Note:** Not needed for the Caddy module — Caddy's native route matchers already handle selective shadowing.

### Diff Options

Configure how responses are compared. These options only apply in **Standalone mode** — in API mode, diff computation is handled server-side by mroki-api. All field paths use [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_DIFF_IGNORED_FIELDS` | No | _(none)_ | Comma-separated field paths to ignore during comparison |
| `MROKI_APP_DIFF_INCLUDED_FIELDS` | No | _(none)_ | Comma-separated field paths to include (whitelist mode). When set, only these fields are compared, then ignored fields are applied. |
| `MROKI_APP_DIFF_FLOAT_TOLERANCE` | No | `0` | Tolerance for floating-point comparisons (`0` = exact) |

---

## mroki-hub

The hub is a static SPA. It doesn't read environment variables at runtime — configuration is injected differently depending on the environment:

**Production (Docker):** The container entrypoint script reads `MROKI_APP_*` env vars and generates a `config.js` file that injects them into `window.__MROKI__` before the app loads.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_API_BASE_URL` | Yes | — | mroki-api base URL |
| `MROKI_APP_API_KEY` | Yes | — | API key for authentication |

**Development (Vite dev server):** Vite compiles `VITE_*` env vars into the bundle at build time. Set these in a `.env` file inside `web/mroki-hub/`:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VITE_API_BASE_URL` | Yes | — | mroki-api base URL |
| `VITE_API_KEY` | Yes | — | API key for authentication |

> **Note:** CORS must be configured on mroki-api (`MROKI_APP_CORS_ORIGINS`) to allow requests from the hub.

---

## caddy-mroki

The Caddy module uses `mroki_gate` directive blocks inside a Caddyfile. It operates in standalone mode only.

### Syntax

```caddyfile
mroki_gate {
    live <live_url>
    shadow <shadow_url>
    [sampling_rate <rate>]
    [live_timeout <duration>]
    [shadow_timeout <duration>]
    [max_body_size <bytes>]
    [diff_ignored_fields <comma-separated>]
    [diff_included_fields <comma-separated>]
    [diff_float_tolerance <float>]
}
```

### Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `live` | Yes | — | URL of live/production service |
| `shadow` | Yes | — | URL of shadow/experimental service |
| `sampling_rate` | No | `1.0` | Sample rate (`0.0`–`1.0`, `1.0` = 100%) |
| `live_timeout` | No | `5s` | Live request timeout |
| `shadow_timeout` | No | `10s` | Shadow request timeout |
| `max_body_size` | No | _(unlimited)_ | Skip shadow for requests above this size in bytes (`0` = unlimited) |
| `diff_ignored_fields` | No | _(none)_ | Comma-separated fields to ignore in diff (gjson syntax) |
| `diff_included_fields` | No | _(none)_ | Comma-separated fields to include in diff (whitelist) |
| `diff_float_tolerance` | No | _(exact)_ | Float comparison tolerance |

---

## Examples

### Development (Standalone Mode)

```bash
# mroki-proxy
MROKI_APP_PORT=8080
MROKI_APP_LIVE_URL=http://localhost:3000
MROKI_APP_SHADOW_URL=http://localhost:3001
MROKI_APP_LIVE_TIMEOUT=5s
MROKI_APP_SHADOW_TIMEOUT=10s

# mroki-api
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/mroki
MROKI_APP_API_KEY=dev-test-key-min-16-chars
MROKI_APP_CORS_ORIGINS=http://localhost:5173

# mroki-hub (dev uses VITE_ prefix, see note above)
VITE_API_BASE_URL=http://localhost:8090
VITE_API_KEY=dev-test-key-min-16-chars
```

### Production (API Mode)

```bash
# mroki-proxy
MROKI_APP_PORT=8080
MROKI_APP_API_URL=http://mroki-api:8090
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=your-production-api-key
MROKI_APP_API_TIMEOUT=30s
MROKI_APP_MAX_RETRIES=3
MROKI_APP_RETRY_DELAY=1s
MROKI_APP_LIVE_TIMEOUT=3s
MROKI_APP_SHADOW_TIMEOUT=15s
MROKI_APP_MAX_BODY_SIZE=10485760
MROKI_APP_SAMPLING_RATE=1.0
MROKI_APP_REDACTED_FIELDS=headers.X-Internal-Token,body.user.password

# mroki-api
MROKI_APP_DATABASE_URL=postgres://user:pass@db-host:5432/mroki
MROKI_APP_API_KEY=your-production-api-key
MROKI_APP_RATE_LIMIT=1000
MROKI_APP_RETENTION=720h
MROKI_APP_CLEANUP_INTERVAL=1h
MROKI_APP_CORS_ORIGINS=https://hub.example.com

# mroki-hub
MROKI_APP_API_BASE_URL=https://api.example.com
MROKI_APP_API_KEY=your-production-api-key
```

### Standalone with Diff Tuning

```bash
# mroki-proxy — standalone with selective diff comparison
MROKI_APP_PORT=8080
MROKI_APP_LIVE_URL=https://api.production.example.com
MROKI_APP_SHADOW_URL=https://api.shadow.example.com
MROKI_APP_SAMPLING_RATE=0.5
MROKI_APP_REDACTED_FIELDS=headers.X-Internal-Token
MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at,updated_at
MROKI_APP_DIFF_INCLUDED_FIELDS=user,order
MROKI_APP_DIFF_FLOAT_TOLERANCE=0.001
```
