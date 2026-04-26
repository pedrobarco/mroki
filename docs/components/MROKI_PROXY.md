# mroki-proxy

**Transparent HTTP proxy for shadow traffic testing**

mroki-proxy is a lightweight proxy that intercepts HTTP traffic, forwards it to both live (production) and shadow (experimental) services, and sends the raw responses to mroki-api for server-side diff computation and storage. In standalone mode (no API), the proxy computes and prints diffs locally.

## Features

- **Transparent Proxying**: Clients see no difference — live responses returned immediately
- **Dual Operating Modes**: API mode (sends raw responses to mroki-api) or Standalone mode (computes and prints diffs locally)
- **Sampling Rate**: Configurable percentage of traffic forwarded to shadow (0.0–1.0)
- **Max Body Size**: Skip shadow proxying for requests exceeding a configurable body size
- **Configurable Diff Options** (standalone mode only): Field filtering, array sorting, float tolerance
- **Parallel Forwarding**: Live and shadow requests execute concurrently
- **Server-Side Diffing** (API mode): Sends raw responses to mroki-api — diff computation happens server-side
- **Local Diffing** (standalone mode): Computes JSON diffs and prints them to stdout
- **Resilient HTTP Transport**: Composable RoundTripper stack with retry (exponential backoff via failsafe-go), circuit breaker, auth, and logging
- **Circuit Breaker**: Stops retrying when API is down; opens after configurable failure threshold, auto-recovers after delay
- **Request ID Propagation**: Generates `X-Request-ID` (UUID v4) per request or reuses incoming header; propagated to live/shadow services and mroki-api
- **Best-Effort Delivery**: API failures never affect live traffic
- **Structured Logging**: All events logged with context

## Architecture

```mermaid
graph TD
    Client([Client Request]) --> Handler

    subgraph mroki-proxy
        Handler[HTTP Proxy Handler] --> LiveFwd[Live Fwd]
        Handler --> ShadowFwd[Shadow Fwd]
        LiveFwd & ShadowFwd --> Callback{Callback}
        Callback -->|API mode| APIClient[API Client<br><i>Retry + Circuit Breaker</i>]
        Callback -->|Standalone mode| Differ[pkg/diff<br><i>Compute diff locally</i>]
    end

    APIClient -->|Send raw responses| API["mroki-api<br><i>(computes diff server-side)</i>"]
    Differ --> Stdout([stdout])
```

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/pedrobarco/mroki.git
cd mroki

# Build
go build -o mroki-proxy ./cmd/mroki-proxy

# Run
./mroki-proxy
```

### Using Go Install

```bash
go install github.com/pedrobarco/mroki/cmd/mroki-proxy@latest
```

## Configuration

Configuration is via environment variables with the `MROKI_APP_` prefix.

### Operating Modes

The proxy supports two operating modes. You must configure ONE mode:

#### API Mode (Recommended)

In API mode, the proxy fetches gate configuration (live/shadow URLs) from mroki-api on startup.

**Required:**
```bash
# mroki-api server URL
MROKI_APP_API_URL="http://localhost:8081"

# Gate ID from mroki-api (must be valid UUID)
MROKI_APP_GATE_ID="550e8400-e29b-41d4-a716-446655440000"

# API key for authentication
MROKI_APP_API_KEY="dev-test-key-min-16-chars"
```

**Optional:**
```bash
# Maximum retry attempts for API requests (default: 3)
MROKI_APP_MAX_RETRIES=3

# Initial delay between retries, doubles each attempt (default: 1s)
MROKI_APP_RETRY_DELAY=1s

# Overall deadline for API calls including all retries (default: 30s)
MROKI_APP_API_TIMEOUT=30s

# Circuit breaker: consecutive failures before opening (default: 5)
MROKI_APP_CB_FAILURE_THRESHOLD=5

# Circuit breaker: delay before transitioning from open to half-open (default: 1m)
MROKI_APP_CB_DELAY=1m

# Circuit breaker: successes in half-open state before closing (default: 2)
MROKI_APP_CB_SUCCESS_THRESHOLD=2
```

#### Standalone Mode

In standalone mode, the proxy uses hardcoded URLs from environment variables. No API communication.

**Required:**
```bash
# Live service (production)
MROKI_APP_LIVE_URL="https://api.production.example.com"

# Shadow service (experimental)
MROKI_APP_SHADOW_URL="https://api.shadow.example.com"
```

### Server Configuration

Works in both modes:

```bash
# Proxy server port (default: 8080)
MROKI_APP_PORT=8080

# Maximum request body size for shadow proxying (default: 10MB, 0=unlimited)
MROKI_APP_MAX_BODY_SIZE=10485760

# Sampling rate for shadow traffic (0.0-1.0, default: 1.0 = 100%)
# MROKI_APP_SAMPLING_RATE=0.5

# Live request timeout (default: 5s)
# Blocks client response - keep tight!
MROKI_APP_LIVE_TIMEOUT=5s

# Shadow request timeout (default: 10s)
# Doesn't block client - can be longer
MROKI_APP_SHADOW_TIMEOUT=10s
```

### Diff Configuration (Optional)

Configure how responses are compared. These options only apply in **Standalone mode** — in API mode, diff computation is handled server-side by mroki-api.

#### `MROKI_APP_DIFF_IGNORED_FIELDS`

Comma-separated list of JSON field paths to ignore during comparison.

**Syntax:** gjson path syntax (supports nested fields and array wildcards)

**Examples:**
```bash
# Simple fields
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,id"

# Nested fields
MROKI_APP_DIFF_IGNORED_FIELDS="metadata.timestamp,user.created_at"

# Array wildcards (# matches any array element)
MROKI_APP_DIFF_IGNORED_FIELDS="users.#.id,orders.#.created_at"

# Multiple patterns
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,metadata.created_at,users.#.updated_at"
```

#### `MROKI_APP_DIFF_INCLUDED_FIELDS`

Comma-separated list of JSON field paths to include (whitelist mode). When set, ONLY these fields are compared, then `DIFF_IGNORED_FIELDS` is applied.

**Examples:**
```bash
# Only compare user and order fields
MROKI_APP_DIFF_INCLUDED_FIELDS="user,order"

# Combine with ignored fields
MROKI_APP_DIFF_INCLUDED_FIELDS="user,order"
MROKI_APP_DIFF_IGNORED_FIELDS="user.created_at,order.created_at"
```

#### `MROKI_APP_DIFF_FLOAT_TOLERANCE`

Tolerance for floating point comparisons. Allows small differences that might occur due to rounding.

**Default:** `0` (exact comparison)

**Example:**
```bash
# Allow 0.1% difference (useful for prices, percentages)
MROKI_APP_DIFF_FLOAT_TOLERANCE=0.001
```

**Field Pattern Syntax:**

The diff configuration uses [gjson path syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

| Pattern | Description | Example JSON | Matches |
|---------|-------------|--------------|---------|
| `field` | Top-level field | `{"field": 1}` | `field` |
| `a.b` | Nested field | `{"a": {"b": 1}}` | `a.b` |
| `a.#.b` | Array wildcard | `{"a": [{"b": 1}, {"b": 2}]}` | `a[0].b`, `a[1].b` |
| `a.b\\.c` | Escaped dot | `{"a": {"b.c": 1}}` | `a["b.c"]` |

**Common patterns:**
```bash
# Ignore all timestamps
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,created_at,updated_at,deleted_at"

# Ignore IDs in nested arrays
MROKI_APP_DIFF_IGNORED_FIELDS="users.#.id,users.#.posts.#.id"

# Ignore metadata object
MROKI_APP_DIFF_IGNORED_FIELDS="metadata"

# Ignore specific nested field
MROKI_APP_DIFF_IGNORED_FIELDS="response.data.internal.debug_info"
```

## Running the Proxy

## Running the Proxy

### API Mode

```bash
cd cmd/mroki-proxy

# 1. Create gate in mroki-api first
GATE_RESPONSE=$(curl -s -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-test-key-min-16-chars" \
  -d '{
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }')

GATE_ID=$(echo $GATE_RESPONSE | jq -r '.data.id')

# 2. Configure proxy
cat > .env << EOF
MROKI_APP_PORT=8080
MROKI_APP_API_URL=http://localhost:8081
MROKI_APP_GATE_ID=$GATE_ID
MROKI_APP_API_KEY=dev-test-key-min-16-chars

# Optional: Diff configuration
MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at,metadata.request_id
EOF

# 3. Run
go run .
```

**Output:**
```
INFO Starting in API mode api_url=http://localhost:8081 gate_id=550e8400-...
INFO Gate configuration loaded gate_id=550e8400-... live_url=https://httpbin.org/... shadow_url=https://httpbin.org/...
DEBUG Diff options configured ignored_fields=[timestamp created_at metadata.request_id]
INFO Started server address=:8080
```

### Standalone Mode (No API)

```bash
cd cmd/mroki-proxy

# Create .env file
cat > .env << 'EOF'
MROKI_APP_LIVE_URL=https://httpbin.org/anything?service=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?service=shadow
MROKI_APP_PORT=8080

# Optional: Diff configuration
MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,id
EOF

# Run
go run .
```

**Output:**
```
INFO Starting in standalone mode live_url=https://httpbin.org/... shadow_url=https://httpbin.org/...
DEBUG Diff options configured ignored_fields=[timestamp id]
INFO Started server address=:8080
```

### Sending Test Traffic

```bash
# Send request through proxy
curl -X POST http://localhost:8080/test \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30}'

# Proxy logs (API mode):
# DEBUG successfully sent request to API method=POST path=/test live_status=200 shadow_status=200

# Proxy logs (standalone mode):
# INFO response diff detected method=POST path=/test live_status=200 shadow_status=200 changes=2
# Diff:
#  ~ /body/user: "alice" → "bob"
```

## Behavior

### Request Flow

1. **Client sends request** to proxy (e.g., `POST http://localhost:8080/api/users`)
2. **Proxy generates `X-Request-ID`** (UUID v4) if not present in the incoming request, or reuses the existing header value. The ID is set on the request header and returned in the response header.
3. **Agent forwards** to both live and shadow services in parallel (same `X-Request-ID` propagated)
4. **Live response returned** to client immediately (shadow still processing)
5. **Background processing:**
   - Wait for shadow response
   - **API mode:** Send raw responses to mroki-api with `X-Request-ID` header (used as the domain Request ID, diff computed server-side)
   - **Standalone mode:** Compute JSON diff locally and print to stdout
6. **Failures logged** but never propagate to client

### Response Selection

- Client always receives the **live service response**
- Shadow response is for comparison only
- Shadow failures don't affect client

### Diff Computation

**When diffs are computed:**
- Both responses have `Content-Type: application/json`
- Both responses are valid JSON
- Responses have different content

**When diffs are NOT computed:**
- Non-JSON responses (HTML, images, etc.)
- Malformed JSON
- Identical responses (no diff to capture)

**Diff Format:** JSON Patch (RFC 6902)

```json
[
  {
    "op": "replace",
    "path": "/id",
    "value": 456,
    "oldValue": 123
  }
]
```

### Resilient HTTP Client

API requests are handled by a composable `http.RoundTripper` stack powered by [failsafe-go](https://failsafe-go.dev/):

```
http.Client
  └─ failsafehttp.RoundTripper  ← retry (exponential backoff) + circuit breaker
       └─ loggingRoundTripper    ← log method/URL/status/latency per attempt
            └─ authRoundTripper  ← Bearer token, Content-Type
                 └─ http.DefaultTransport
```

**Retry Policy:**
- Exponential backoff (default: 1s → 8s)
- Retries on 5xx responses, 429 (Too Many Requests), and network errors
- Respects `Retry-After` headers
- Configurable via `MAX_RETRIES` and `RETRY_DELAY`

**Circuit Breaker:**
- **Closed** (normal): all requests pass through
- **Open** (after N consecutive failures): requests fail immediately with `ErrOpen`, logged as warning and skipped
- **Half-Open** (after delay): allows test requests; closes on success, re-opens on failure
- Configurable via `CB_FAILURE_THRESHOLD`, `CB_DELAY`, `CB_SUCCESS_THRESHOLD`

**Note:** Client request completes successfully regardless of API failures — the circuit breaker only affects background API requests.

## Configuration Examples

### API Mode (Recommended)

```bash
# Proxy fetches gate URLs from API, sends raw responses (diff computed server-side)
MROKI_APP_API_URL=http://localhost:8081
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=dev-test-key-min-16-chars
MROKI_APP_PORT=8080

# Note: DIFF_* options are NOT used in API mode — diff is computed by mroki-api
```

### Standalone Mode with Field Filtering

```bash
# Hardcoded URLs
MROKI_APP_LIVE_URL=http://localhost:3000
MROKI_APP_SHADOW_URL=http://localhost:3001
MROKI_APP_PORT=8080

# Only compare specific fields
MROKI_APP_DIFF_INCLUDED_FIELDS=user.email,user.name,order.total
MROKI_APP_DIFF_IGNORED_FIELDS=user.last_login
```

### Production with External Services

```bash
# API Mode — proxy is a thin forwarder, diff computed server-side
MROKI_APP_API_URL=https://mroki-api.internal.example.com
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=production-key-min-16-chars
MROKI_APP_PORT=80
MROKI_APP_LIVE_TIMEOUT=3s
MROKI_APP_SHADOW_TIMEOUT=15s
```

### Testing with httpbin

```bash
# Different query params to differentiate responses
MROKI_APP_LIVE_URL=https://httpbin.org/anything?service=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?service=shadow
MROKI_APP_PORT=8080

# Ignore httpbin-specific fields
MROKI_APP_DIFF_IGNORED_FIELDS=origin,url,headers.Host
```

## Deployment

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mroki-proxy ./cmd/mroki-proxy

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mroki-proxy .
CMD ["./mroki-proxy"]
```

**Run:**
```bash
docker build -t mroki-proxy .
docker run -p 8080:8080 \
  -e MROKI_APP_API_URL=http://mroki-api:8081 \
  -e MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000 \
  -e MROKI_APP_API_KEY=dev-test-key-min-16-chars \
  -e MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at \
  mroki-proxy
```

### Kubernetes Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
      # Main application
      - name: app
        image: my-app:latest
        ports:
        - containerPort: 3000
      
      # mroki-proxy sidecar
      - name: mroki-proxy
        image: mroki-proxy:latest
        ports:
        - containerPort: 8080
        env:
        - name: MROKI_APP_API_URL
          value: "http://mroki-api:8081"
        - name: MROKI_APP_GATE_ID
          valueFrom:
            configMapKeyRef:
              name: mroki-config
              key: gate-id
        - name: MROKI_APP_API_KEY
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: api-key
        - name: MROKI_APP_DIFF_IGNORED_FIELDS
          value: "timestamp,created_at,metadata.request_id"
---
apiVersion: v1
kind: Service
metadata:
  name: my-app
spec:
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080  # Route to proxy, not app
```

### Standalone Service

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mroki-proxy
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mroki-proxy
  template:
    metadata:
      labels:
        app: mroki-proxy
    spec:
      containers:
      - name: mroki-proxy
        image: mroki-proxy:latest
        ports:
        - containerPort: 8080
        env:
        - name: MROKI_APP_API_URL
          value: "http://mroki-api:8081"
        - name: MROKI_APP_GATE_ID
          value: "550e8400-e29b-41d4-a716-446655440000"
        - name: MROKI_APP_API_KEY
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: api-key
        - name: MROKI_APP_DIFF_IGNORED_FIELDS
          value: "timestamp,created_at"
---
apiVersion: v1
kind: Service
metadata:
  name: mroki-proxy
spec:
  selector:
    app: mroki-proxy
  ports:
  - port: 80
    targetPort: 8080
```

## Logging

All logs use structured logging (slog) with JSON output.

**Log Levels:**
- `DEBUG` - Detailed operation info (API sends, diff details)
- `INFO` - Normal operation (requests processed, proxy started)
- `WARN` - Recoverable errors (API failures, retry attempts)
- `ERROR` - Unrecoverable errors (exhausted retries)

**Example Log Output:**
```json
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"API integration enabled","api_url":"http://localhost:8081","gate_id":"550e8400"}
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"Started server","address":":8080","live":"https://api.example.com"}
{"time":"2026-01-31T20:00:15Z","level":"INFO","msg":"response diff detected","method":"POST","path":"/api/users","live_status":200,"shadow_status":200}
{"time":"2026-01-31T20:00:15Z","level":"DEBUG","msg":"successfully sent request to API","method":"POST","path":"/api/users","has_diff":true}
```

## Troubleshooting

### Proxy won't start

**Problem:** `panic: configuration validation failed: must configure either API mode or standalone mode`

**Solution:** Set either API mode (API_URL + GATE_ID + API_KEY) or standalone mode (LIVE_URL + SHADOW_URL), not both.

---

**Problem:** `panic: configuration validation failed: gate_id must be a valid UUID`

**Solution:** Ensure `GATE_ID` is a valid UUID format (create gate via mroki-api first).

---

**Problem:** `panic: configuration validation failed: api_url and gate_id must both be set or both be empty`

**Solution:** Either set both `API_URL` and `GATE_ID`, or remove both for standalone mode.

---

**Problem:** `panic: configuration validation failed: gate_id must be a valid UUID`

**Solution:** Ensure `GATE_ID` is a valid UUID format (create gate via mroki-api first).

### No diffs captured

**Problem:** Requests go through but no diffs appear in API

**Possible causes:**
1. Responses are not JSON (`Content-Type: application/json` required)
2. Responses are identical (no diff to capture)
3. API integration not configured (check logs for "Running in standalone mode")
4. API is unreachable (check logs for retry messages)

**Debug:**
```bash
# Check proxy logs for API errors
grep ERRORproxy.log

# Verify API is reachable
curl http://localhost:8081/health/live

# Test with httpbin (guaranteed to produce diffs)
MROKI_APP_LIVE_URL=https://httpbin.org/anything?service=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?service=shadow
```

### High latency

**Problem:** Requests through proxy are slow

**Possible causes:**
1. `LIVE_TIMEOUT` is too high (blocks client response)
2. Live service is slow
3. Network issues

**Solution:**
```bash
# Reduce live timeout (default: 5s)
MROKI_APP_LIVE_TIMEOUT=2s

# Shadow timeout doesn't affect client
MROKI_APP_SHADOW_TIMEOUT=30s
```

## Performance

**Throughput:** ~1000 req/s per proxy instance (depends on service latency)

**Memory:** ~50MB baseline + ~1KB per in-flight request

**CPU:** <5% idle, scales with traffic volume

**Bottlenecks:**
- Live service latency (blocks client)
- Shadow service latency (blocks diff computation)
- API write latency (doesn't block client)

## Security Considerations

- **No TLS termination:** Use reverse proxy (nginx, Caddy) for HTTPS
- **No authentication:** Anyone can send traffic through proxy
- **Request logging:** All traffic captured (may contain PII/secrets)
- **Network access:** Proxy needs access to both live and shadow services

**What's secure:**
- ✅ Best-effort delivery: API failures never affect live traffic
- ✅ Retry with exponential backoff + circuit breaker
- ✅ Configurable timeouts
- ✅ No secrets in logs

**Production recommendations:**
- Use TLS for proxy→API communication
- Deploy in isolated network (Kubernetes sidecar pattern)
- Review data retention and PII policies
- Consider sensitive header redaction

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [API Contracts](../architecture/API_CONTRACTS.md)
- [Quick Start Guide](../guides/QUICK_START.md)
- [Development Guide](../guides/DEVELOPMENT.md)
- [mroki-api Component](MROKI_API.md)
