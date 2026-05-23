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

---

## mroki-proxy

The proxy supports two mutually exclusive operating modes: **API mode** and **Standalone mode**. You must configure exactly one.

### Common

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_PORT` | No | `8080` | Proxy server port |
| `MROKI_APP_LIVE_TIMEOUT` | No | `5s` | Live request timeout — blocks client response, keep tight |
| `MROKI_APP_SHADOW_TIMEOUT` | No | `10s` | Shadow request timeout — does not block client |
| `MROKI_APP_MAX_BODY_SIZE` | No | `10485760` | Skip shadow for requests above this size in bytes (`0` = unlimited) |
| `MROKI_APP_SAMPLING_RATE` | No | `1.0` | Shadow traffic sampling rate (`0.0`–`1.0`, `1.0` = 100%) |
| `MROKI_APP_READ_TIMEOUT` | No | `30s` | Server read timeout |
| `MROKI_APP_WRITE_TIMEOUT` | No | `60s` | Server write timeout (must be ≥ live timeout) |
| `MROKI_APP_IDLE_TIMEOUT` | No | `120s` | Server idle timeout |

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

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_REDACTED_FIELDS` | No | _(none)_ | Comma-separated additional fields to redact (gjson path notation). Redacted fields are also excluded from diff computation. |

### Diff Options

Configure how responses are compared. These options only apply in **Standalone mode** — in API mode, diff computation is handled server-side by mroki-api. All field paths use [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md).

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MROKI_APP_DIFF_IGNORED_FIELDS` | No | _(none)_ | Comma-separated field paths to ignore during comparison |
| `MROKI_APP_DIFF_INCLUDED_FIELDS` | No | _(none)_ | Comma-separated field paths to include (whitelist mode). When set, only these fields are compared, then ignored fields are applied. |
| `MROKI_APP_DIFF_FLOAT_TOLERANCE` | No | `0` | Tolerance for floating-point comparisons (`0` = exact) |

---

## mroki-hub

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

# mroki-hub
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
VITE_API_BASE_URL=https://api.example.com
VITE_API_KEY=your-production-api-key
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
