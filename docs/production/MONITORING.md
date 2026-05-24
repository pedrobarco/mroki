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

Prometheus metrics are planned for a future release. Planned metric names include:

- `mroki_proxy_requests_total` — Total requests proxied
- `mroki_proxy_shadow_skipped` — Shadow requests skipped (sampling / body size)
- `mroki_proxy_api_failures` — API send failures
- `mroki_api_requests_total` — API request count
- `mroki_api_diffs_computed` — Diffs computed server-side

See [Roadmap](../ROADMAP.md) Phase 4 for implementation timeline.
