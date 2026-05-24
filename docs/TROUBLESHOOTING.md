# Troubleshooting

Common issues across mroki components — organized by category with **Symptom → Cause → Fix**.

---

## Authentication Errors

**Symptom:** `401 Unauthorized` or `{"type":"about:blank","title":"Unauthorized","status":401,...}`
**Cause:** Missing or mismatched API key. The `Authorization: Bearer <key>` header is absent, or the key doesn't match across services.
**Fix:**
```bash
# Verify keys match between API and proxy
grep API_KEY cmd/mroki-api/.env
grep API_KEY cmd/mroki-proxy/.env

# Test with explicit Bearer token
curl -H "Authorization: Bearer dev-test-key-min-16-chars" http://localhost:8090/gates
```

---

**Symptom:** `{"type":"about:blank","title":"Invalid API Key","status":401,...}`
**Cause:** API key is shorter than 16 characters.
**Fix:** Update both `.env` files with a key ≥ 16 characters:
```bash
echo 'MROKI_APP_API_KEY=your-new-key-min-16-chars' >> cmd/mroki-api/.env
echo 'MROKI_APP_API_KEY=your-new-key-min-16-chars' >> cmd/mroki-proxy/.env
# Restart both services
```

---

**Symptom:** `429 Too Many Requests`
**Cause:** Rate limit exceeded (default: 1000 requests/min/IP).
**Fix:** Wait 60 seconds, or increase the limit:
```bash
# In cmd/mroki-api/.env
MROKI_APP_RATE_LIMIT=5000
# Restart API
```

---

## Proxy Issues

**Symptom:** `Configuration validation failed: must configure either API mode or standalone mode`
**Cause:** Neither API mode nor standalone mode is fully configured.
**Fix:** Set one complete mode — not a mix of both:
```bash
# API mode (all three required)
MROKI_APP_API_URL=http://localhost:8090
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=dev-test-key-min-16-chars

# OR standalone mode (both required)
MROKI_APP_LIVE_URL=https://api.production.example.com
MROKI_APP_SHADOW_URL=https://api.shadow.example.com
```

---

**Symptom:** `Configuration validation failed: gate_id must be a valid UUID`
**Cause:** `GATE_ID` is not a valid UUID.
**Fix:** Create a gate via mroki-api first, then use the returned UUID.

---

**Symptom:** `Configuration validation failed: read_timeout must be less than write_timeout`
**Cause:** Server timeouts violate the required ordering.
**Fix:** Ensure `READ_TIMEOUT` < `WRITE_TIMEOUT` < `IDLE_TIMEOUT`.

---

**Symptom:** `connection refused` when proxy tries to reach live or shadow service.
**Cause:** Target service is down or URL is wrong.
**Fix:** Verify the live/shadow URLs are reachable:
```bash
curl https://api.production.example.com/health
curl https://api.shadow.example.com/health
```

---

**Symptom:** Requests go through but no diffs appear.
**Cause:** (1) Responses are not JSON (`Content-Type: application/json` required), (2) responses are identical, (3) API integration not configured (logs show "Running in standalone mode"), or (4) API is unreachable.
**Fix:**
```bash
# Check proxy logs for errors
grep ERROR proxy.log

# Verify API is reachable (API mode)
curl http://localhost:8090/health/live

# Test with httpbin (guaranteed to produce diffs)
MROKI_APP_LIVE_URL=https://httpbin.org/anything?service=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?service=shadow
```

---

**Symptom:** Requests through the proxy are slow (high latency).
**Cause:** `LIVE_TIMEOUT` is too high, or the live service itself is slow.
**Fix:**
```bash
# Reduce live timeout (default: 5s) — this blocks the client response
MROKI_APP_LIVE_TIMEOUT=2s

# Shadow timeout doesn't affect client response time
MROKI_APP_SHADOW_TIMEOUT=30s
```

---

## API Issues

**Symptom:** `Configuration validation failed: database.url is required`
**Cause:** `MROKI_APP_DATABASE_URL` is not set.
**Fix:** Set the environment variable:
```bash
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
```

---

**Symptom:** `Configuration validation failed: port must be between 1 and 65535`
**Cause:** Invalid `MROKI_APP_PORT` value.
**Fix:** Set a valid port (1–65535).

---

**Symptom:** `GET /health/ready` returns 503.
**Cause:** Database is unreachable.
**Fix:**
```bash
# Test database connectivity
psql -U postgres -h localhost -p 5432 -d postgres

# Verify connection string format
echo $MROKI_APP_DATABASE_URL
# Expected: postgres://user:pass@host:port/database
```

---

**Symptom:** `POST /gates` returns 500.
**Cause:** Database schema not created, connection pool exhausted, or invalid URL in the request body.
**Fix:** Check API logs for the specific database error. Ensure the API started successfully and ran auto-migration.

---

## Database Issues

**Symptom:** `failed to create connection pool: connection refused`
**Cause:** PostgreSQL is not running or not accessible at the configured host/port.
**Fix:**
```bash
# Check Docker is running
docker ps | grep postgres

# Restart PostgreSQL
docker compose -f build/dev/compose.yaml restart

# Test connection directly
psql -U postgres -h localhost -p 5432 -d postgres

# Check firewall (production)
sudo ufw status
```

---

**Symptom:** Connection pool exhausted — queries hang or time out.
**Cause:** Too many concurrent connections, or connections are leaking.
**Fix:** Increase pool size or investigate slow queries:
```bash
# Increase max connections (default: 25)
MROKI_APP_DATABASE_MAX_CONNS=50

# Check active connections
psql -U postgres -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'postgres';"
```

---

**Symptom:** Schema or migration errors on startup.
**Cause:** The API auto-migrates on startup using ent. Failures indicate a database permission issue or incompatible schema state.
**Fix:** Connect directly and inspect:
```bash
psql -U postgres -d postgres
\dt  -- list tables
```

---

## Caddy Module Issues

**Symptom:** `Error: module 'http.handlers.mroki_gate' not found`
**Cause:** The mroki module is not compiled into the Caddy binary.
**Fix:**
```bash
# Rebuild with xcaddy
xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule

# Verify module is included
./caddy list-modules | grep mroki
```

---

**Symptom:** `Error: live URL is required`
**Cause:** Missing required `live` or `shadow` directive in the Caddyfile.
**Fix:**
```caddyfile
mroki_gate {
    live https://api.production.example.com
    shadow https://api.shadow.example.com
}
```

---

**Symptom:** Requests are slow through Caddy (high latency).
**Cause:** Default `live_timeout` (5s) is too high for your use case.
**Fix:**
```caddyfile
mroki_gate {
    live https://api.production.example.com
    shadow https://api.shadow.example.com
    live_timeout 2s
}
```

---

## Hub Issues

**Symptom:** Hub can't connect to the API — network errors in the browser console.
**Cause:** `VITE_API_BASE_URL` (dev) or `MROKI_APP_API_BASE_URL` (production) is not set, or the API is not running.
**Fix:**
```bash
# Dev: create .env in web/mroki-hub
VITE_API_BASE_URL=http://localhost:8090
VITE_API_KEY=your-api-key

# Verify the API is reachable
curl http://localhost:8090/health/live
```

---

**Symptom:** CORS errors in the browser console (e.g., `Access-Control-Allow-Origin` missing).
**Cause:** mroki-api does not have the hub's origin in its CORS allowlist.
**Fix:**
```bash
# In cmd/mroki-api/.env — add the hub origin
MROKI_APP_CORS_ORIGINS=http://localhost:5173

# For production, include all hub origins (comma-separated)
MROKI_APP_CORS_ORIGINS=http://localhost:5173,https://hub.example.com

# Restart mroki-api
```

---

## Debugging Tips

### Check Structured Logs

All components use structured logging (slog) with JSON output. Key fields:

- `request.id` — correlates a request across proxy, API, and stored entities
- `request.method`, `request.path` — the original request
- `response.status`, `response.latency` — response metadata
- `error.type`, `error.title`, `error.status` — RFC 7807 error details

Example:
```json
{"time":"2026-01-31T20:00:15Z","level":"INFO","msg":"200: OK","request.id":"7c9e6679-7425-40de-944b-e07fc1f90ae7","request.method":"GET","request.path":"/gates","response.status":200,"response.latency":"1.234ms"}
```

### X-Request-ID Correlation

Every request is assigned an `X-Request-ID` (UUID v4). If the client provides the header, it is reused; otherwise one is generated. This ID:

- Appears in all log entries as `request.id`
- Is returned in the `X-Request-ID` response header
- Is propagated from proxy → live/shadow services → mroki-api
- Becomes the stored `Request.ID` in the database

Use it to trace a single request end-to-end across all components.

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the proxy
cd cmd/mroki-proxy
dlv debug

# Set breakpoints
(dlv) break main.main
(dlv) continue
```

### Check Service Status

```bash
# Docker Compose
docker compose ps
docker compose logs mroki-api
docker compose logs mroki-proxy

# Kubernetes
kubectl get pods -n mroki
kubectl describe pod mroki-api-xxx -n mroki
kubectl logs mroki-api-xxx -n mroki

# Systemd
systemctl status mroki-api
systemctl status mroki-proxy
```
