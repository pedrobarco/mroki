# Monitoring & Observability

Mroki uses structured logging and request ID correlation to provide observability across the proxy and API components. This guide covers logging configuration, health checks, and log viewing.

## Structured Logging

Both **mroki-proxy** and **mroki-api** use Go's `slog` package with JSON output at `info` level.

**Example JSON log line (proxy):**

```json
{"time":"2026-01-31T20:00:15Z","level":"INFO","msg":"response diff detected","request.id":"7c9e6679","request.method":"POST","request.path":"/api/users","live_status":200,"shadow_status":200}
```

**Key fields:**

- `request.id` — Unique request identifier (correlates across components)
- `request.method` — HTTP method
- `request.path` — Request path
- `live_status` / `shadow_status` — Response status codes (proxy logs)
- `response.status` / `response.latency` — Response details (API logs)

## Request ID Correlation

All components propagate an `X-Request-ID` header (UUID v4) through the entire request lifecycle:

1. **Proxy** generates the ID (or reuses an incoming header), forwards it to live/shadow services and mroki-api
2. **API** middleware extracts or generates the ID, stores it in context, and returns it in the response header
3. The propagated ID becomes the stored `Request.ID`, enabling direct correlation between proxy logs, API logs, and stored entities

To trace a request across components, filter logs by `request.id`:

```bash
# Find all log entries for a specific request
docker compose logs | grep '"request.id":"7c9e6679"'
```

## Health Checks

The API exposes two health check endpoints:

| Endpoint            | Success | Failure | Description              |
|---------------------|---------|---------|--------------------------|
| `GET /health/live`  | 200 OK  | —       | Service is running       |
| `GET /health/ready` | 200 OK  | 503     | Database is connected    |

**Kubernetes probe example:**

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8090
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8090
  initialDelaySeconds: 5
  periodSeconds: 10
```

**Docker healthcheck example:**

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD curl -f http://localhost:8090/health/live || exit 1
```

## Viewing Logs

**Docker:**

```bash
docker logs <container>
docker logs -f mroki-api      # follow mode
```

**Docker Compose:**

```bash
docker compose logs -f mroki-api
docker compose logs -f mroki-proxy
docker compose logs -f            # all services
```

**Kubernetes:**

```bash
kubectl logs -n mroki -l app=mroki-api -f
kubectl logs -n mroki -l app=mroki-proxy -f
```

**systemd:**

```bash
journalctl -u mroki-api -f
journalctl -u mroki-proxy -f
```

## Metrics

Both components expose a Prometheus-scrapeable `/metrics` endpoint. Endpoints are enabled by
default and can be turned off with `MROKI_APP_METRICS_ENABLED=false` (see
[Configuration](CONFIGURATION.md)).

| Component | Endpoint | Port | Auth |
|-----------|----------|------|------|
| **mroki-api** | `GET /metrics` | API port (`MROKI_APP_PORT`, default `8090`) | None — unauthenticated, like the health endpoints |
| **mroki-proxy** | `GET /metrics` | Admin port (`MROKI_APP_ADMIN_PORT`, default `8081`) | None — isolated from proxied traffic |

When enabled, each endpoint exports the standard Go **runtime** (`go_*`, including `go_build_info`
and the richer `go_sched_*` / `go_gc_*` runtime series) and **process** (`process_*`) collectors,
plus the catalog below.

Naming follows a deliberate split. Generic HTTP telemetry follows the OpenTelemetry **semantic
conventions** — unprefixed, unit-suffixed duration histograms
(`http_server_request_duration_seconds`, `http_client_request_duration_seconds`) shared across both
components for compatibility with off-the-shelf dashboards — while mroki-specific domain signals
carry the `mroki_` namespace. There are deliberately **no `*_requests_total` counters**: request
rate is derived from a histogram's `_count` (e.g. `rate(http_server_request_duration_seconds_count[5m])`),
as the OTel HTTP semconv prescribes. The application never emits a `job` / `instance` label —
Prometheus attaches those, plus Kubernetes service-discovery labels (`namespace`, `pod`, `service`,
…), at scrape time, so the unprefixed names do not collide across services.

Under the hood all HTTP metrics flow through `otelhttp`, all `mroki_*` domain metrics through the
OTel Meter API, and the Go runtime/process and database-pool series stay on the client_golang
collectors — everything is exported through the Prometheus bridge onto one registry, so a single
`/metrics` endpoint exposes the whole set.

The two domain metrics are **shared and identical across components**: they are recorded by whichever
component computes the diff — the API in API mode, the standalone proxy or the caddy module in
standalone mode. A standalone proxy therefore exposes the same `mroki_*` series the API exposes in
API mode (with an empty `gate`, since a standalone proxy is not bound to a gate).

### mroki-proxy metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_server_request_duration_seconds` | histogram | `http_request_method`, `http_response_status_code` (+ standard semconv attrs) | Inbound proxy request duration. **No `http_route`** — the proxy is a transparent mirror, so paths are unbounded. Request rate = `rate(..._count[5m])`. |
| `http_client_request_duration_seconds` | histogram | `http_request_method`, `http_response_status_code`, `server_address`, `server_port`, `mroki_target` | Outbound request duration for the live/shadow and API clients. `mroki_target` (`live`/`shadow`/`api`) is a zero-cardinality role alias on top of `server_address`. |
| `mroki_responses_compared_total` | counter | `gate`, `result` | Live/shadow comparisons by outcome. **Standalone mode only**, where the proxy computes the diff itself; `gate` is empty. In API mode the API records this instead. |
| `mroki_diff_operations` | histogram | `gate` | JSON-Patch operation counts per differing comparison. Observed only when `result="diff"`, so `_count` equals the number of diffs. Standalone mode only. |

### mroki-api metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `http_server_request_duration_seconds` | histogram | `http_request_method`, `http_response_status_code`, `http_route` (+ standard semconv attrs) | Inbound API request duration. `http_route` is the matched Go 1.22 ServeMux template (e.g. `/gates/{gate_id}`), never the raw path, keeping cardinality bounded. |
| `mroki_responses_compared_total` | counter | `gate`, `result` | Live/shadow comparisons by outcome (`match`/`diff`), recorded when the API computes the diff (API mode). |
| `mroki_diff_operations` | histogram | `gate` | JSON-Patch operation counts per differing comparison. Observed only when `result="diff"`. |

The API has no outbound HTTP client, so it emits no `http_client_*` series. It also exports the
standard database/sql pool collector (`go_sql_*`, label `db_name="mroki"`) covering open / in-use /
idle connections, wait count and duration, and idle/lifetime closures.

When deployed as the **caddy module**, the same `mroki_responses_compared_total` /
`mroki_diff_operations` series are recorded on Caddy's own metrics endpoint (HTTP telemetry is
provided by Caddy natively).

### Label dictionary

| Label | Values | Notes |
|-------|--------|-------|
| `http_request_method` | HTTP method (`GET`, `POST`, …) | semconv; bounded to known methods. |
| `http_response_status_code` | numeric status (e.g. `200`, `502`) | semconv; present once a response is returned. |
| `http_route` | templated path (e.g. `/gates/{gate_id}`) | **API only**; semconv matched-route template, never the raw path, to keep cardinality bounded. |
| `server_address` / `server_port` | outbound host / port | semconv; outbound client metric only. |
| `mroki_target` | `live`, `shadow`, `api` (`unknown` fallback) | Derived role alias on the outbound client metric, 1:1 with `server_address` (zero added cardinality). |
| `gate` | gate UUID, or empty | One series per gate; empty for the standalone proxy / caddy module. |
| `result` | `match`, `diff`, `error` | Comparison outcome. |

> **Not yet exposed.** Circuit-breaker / transport-failure metrics are deferred to a later phase —
> `http_client_request_duration_seconds_count{mroki_target="api"}` only counts requests that received
> a response, so breaker-open and connection failures are not yet visible as a metric (they remain in
> the logs). Track progress on [Prometheus metrics](https://github.com/pedrobarco/mroki/issues/18).

**Example Prometheus scrape config:**

```yaml
scrape_configs:
  - job_name: mroki-api
    metrics_path: /metrics
    static_configs:
      - targets: ['mroki-api:8090']
  - job_name: mroki-proxy
    metrics_path: /metrics
    static_configs:
      - targets: ['mroki-proxy:8081'] # admin port, not the proxy port
```

The proxy's `/metrics` lives on the admin port so scrape traffic never reaches the upstream service.
The API's `/metrics` is outside the authenticated middleware chain, so no API key is required.
