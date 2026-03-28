# mroki-agent

**Transparent HTTP proxy for shadow traffic testing**

mroki-agent is a lightweight proxy that intercepts HTTP traffic, forwards it to both live (production) and shadow (experimental) services, computes response differences, and optionally sends captured data to mroki-api for storage and analysis.

## Features

- **Transparent Proxying**: Clients see no difference - live responses returned immediately
- **Dual Operating Modes**: API mode (fetches gate config from API) or Standalone mode (hardcoded URLs)
- **Configurable Diff Options**: Field filtering, array sorting, float tolerance
- **Parallel Forwarding**: Live and shadow requests execute concurrently
- **JSON Diff Computing**: Automatically compares JSON responses with customizable filtering
- **Retry Logic**: Exponential backoff for API failures (1s, 2s, 4s)
- **Best-Effort Delivery**: API failures never affect live traffic
- **Agent ID Persistence**: Identity survives restarts
- **Structured Logging**: All events logged with context

## Architecture

```
Client Request
     │
     ↓
┌────────────────────────────────────┐
│         mroki-agent                │
│  ┌──────────────────────────────┐  │
│  │  HTTP Proxy Handler          │  │
│  └────────┬──────────┬──────────┘  │
│           │          │              │
│    ┌──────▼──┐  ┌───▼──────┐       │
│    │  Live   │  │  Shadow  │       │
│    │ Fwd     │  │  Fwd     │       │
│    └──────┬──┘  └───┬──────┘       │
│           │          │              │
│  ┌────────▼──────────▼──────────┐  │
│  │   Diff Computer (Background) │  │
│  └──────────────┬───────────────┘  │
│                 │                   │
│  ┌──────────────▼───────────────┐  │
│  │  API Client (Retry Logic)    │  │
│  └──────────────────────────────┘  │
└─────────────────┬──────────────────┘
                  │ HTTP
                  ↓
            mroki-api
```

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/pedrobarco/mroki.git
cd mroki

# Build
go build -o mroki-agent ./cmd/mroki-agent

# Run
./mroki-agent
```

### Using Go Install

```bash
go install github.com/pedrobarco/mroki/cmd/mroki-agent@latest
```

## Configuration

Configuration is via environment variables with the `MROKI_APP_` prefix.

### Operating Modes

The agent supports two operating modes. You must configure ONE mode:

#### API Mode (Recommended)

In API mode, the agent fetches gate configuration (live/shadow URLs) from mroki-api on startup.

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

# Timeout for each API request attempt (default: 30s)
MROKI_APP_API_TIMEOUT=30s
```

#### Standalone Mode

In standalone mode, the agent uses hardcoded URLs from environment variables. No API communication.

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

# Maximum request body size for diffing (default: 10MB, 0=unlimited)
MROKI_APP_MAX_BODY_SIZE=10485760

# Live request timeout (default: 5s)
# Blocks client response - keep tight!
MROKI_APP_LIVE_TIMEOUT=5s

# Shadow request timeout (default: 10s)
# Doesn't block client - can be longer
MROKI_APP_SHADOW_TIMEOUT=10s
```

### Diff Configuration (Optional)

Configure how responses are compared. Works in both API and Standalone modes.

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

## Running the Agent

## Running the Agent

### API Mode

```bash
cd cmd/mroki-agent

# 1. Create gate in mroki-api first
GATE_RESPONSE=$(curl -s -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-test-key-min-16-chars" \
  -d '{
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }')

GATE_ID=$(echo $GATE_RESPONSE | jq -r '.data.id')

# 2. Configure agent
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
INFO Agent ID loaded agent_id=MacBook-Pro-a1b2c3d4
INFO Starting in API mode api_url=http://localhost:8081 gate_id=550e8400-...
INFO Gate configuration loaded gate_id=550e8400-... live_url=https://httpbin.org/... shadow_url=https://httpbin.org/...
DEBUG Diff options configured ignored_fields=[timestamp created_at metadata.request_id]
INFO Started server address=:8080
```

### Standalone Mode (No API)

```bash
cd cmd/mroki-agent

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
INFO Agent ID loaded agent_id=MacBook-Pro-a1b2c3d4
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

# Agent logs (API mode):
# INFO response diff detected method=POST path=/test live_status=200 shadow_status=200
# DEBUG successfully sent request to API method=POST path=/test has_diff=true
```

## Agent ID

The agent automatically generates a unique ID on first run and persists it to `.agent_id` in the working directory.

**Format:** `{hostname}-{uuid}`

**Examples:**
- `MacBook-Pro-a1b2c3d4-5678-90ab-cdef-1234567890ab`
- `web-server-01-550e8400-e29b-41d4-a716-446655440000`
- `api-prod-f47ac10b-58cc-4372-a567-0e02b2c3d479`

**Manual Override:**

Create a `.agent_id` file with your desired ID:

```bash
echo "my-custom-agent-id-550e8400-e29b-41d4-a716-446655440000" > .agent_id
```

**Purpose:**
- Track which agent captured traffic (debugging)
- Monitor agent health across restarts
- Identify traffic source in multi-agent deployments

## Behavior

### Request Flow

1. **Client sends request** to agent (e.g., `POST http://localhost:8080/api/users`)
2. **Agent forwards** to both live and shadow services in parallel
3. **Live response returned** to client immediately (shadow still processing)
4. **Background processing:**
   - Wait for shadow response
   - Compute JSON diff
   - Send to mroki-api (if configured)
5. **Failures logged** but never propagate to client

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

### Retry Logic

If API requests fail, agent retries with exponential backoff:

- **Attempt 1:** Immediate
- **Attempt 2:** Wait 1s
- **Attempt 3:** Wait 2s
- **Attempt 4:** Wait 4s
- **Give up:** After 4 attempts (~8s total)

**Logged output:**
```
WARN API request failed attempt=1 error="connection refused"
INFO retrying API request attempt=1 delay=1s
WARN API request failed attempt=2 error="connection refused"
INFO retrying API request attempt=2 delay=2s
WARN API request failed attempt=3 error="connection refused"
INFO retrying API request attempt=3 delay=4s
ERROR all retries exhausted attempts=4
```

**Note:** Client request completes successfully regardless of API failures.

## Configuration Examples

### API Mode with Diff Options

```bash
# Agent fetches gate URLs from API
MROKI_APP_API_URL=http://localhost:8081
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=dev-test-key-min-16-chars
MROKI_APP_PORT=8080

# Ignore timestamp fields
MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at,updated_at
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
# API Mode
MROKI_APP_API_URL=https://mroki-api.internal.example.com
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=production-key-min-16-chars
MROKI_APP_PORT=80
MROKI_APP_LIVE_TIMEOUT=3s
MROKI_APP_SHADOW_TIMEOUT=15s

# Diff configuration for production API
MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,request_id,users.#.last_seen
MROKI_APP_DIFF_FLOAT_TOLERANCE=0.0001
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
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mroki-agent ./cmd/mroki-agent

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mroki-agent .
CMD ["./mroki-agent"]
```

**Run:**
```bash
docker build -t mroki-agent .
docker run -p 8080:8080 \
  -e MROKI_APP_API_URL=http://mroki-api:8081 \
  -e MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000 \
  -e MROKI_APP_API_KEY=dev-test-key-min-16-chars \
  -e MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at \
  mroki-agent
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
      
      # mroki-agent sidecar
      - name: mroki-agent
        image: mroki-agent:latest
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
    targetPort: 8080  # Route to agent, not app
```

### Standalone Service

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mroki-agent
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mroki-agent
  template:
    metadata:
      labels:
        app: mroki-agent
    spec:
      containers:
      - name: mroki-agent
        image: mroki-agent:latest
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
  name: mroki-agent
spec:
  selector:
    app: mroki-agent
  ports:
  - port: 80
    targetPort: 8080
```

## Logging

All logs use structured logging (slog) with JSON output.

**Log Levels:**
- `DEBUG` - Detailed operation info (API sends, diff details)
- `INFO` - Normal operation (requests processed, agent started)
- `WARN` - Recoverable errors (API failures, retry attempts)
- `ERROR` - Unrecoverable errors (exhausted retries)

**Example Log Output:**
```json
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"Agent ID loaded","agent_id":"MacBook-Pro-a1b2c3d4"}
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"API integration enabled","api_url":"http://localhost:8081","gate_id":"550e8400"}
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"Started server","address":":8080","live":"https://api.example.com"}
{"time":"2026-01-31T20:00:15Z","level":"INFO","msg":"response diff detected","method":"POST","path":"/api/users","live_status":200,"shadow_status":200}
{"time":"2026-01-31T20:00:15Z","level":"DEBUG","msg":"successfully sent request to API","method":"POST","path":"/api/users","has_diff":true}
```

## Troubleshooting

### Agent won't start

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
# Check agent logs for API errors
grep ERROR agent.log

# Verify API is reachable
curl http://localhost:8081/health/live

# Test with httpbin (guaranteed to produce diffs)
MROKI_APP_LIVE_URL=https://httpbin.org/anything?service=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?service=shadow
```

### High latency

**Problem:** Requests through agent are slow

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

**Throughput:** ~1000 req/s per agent instance (depends on service latency)

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
- **Network access:** Agent needs access to both live and shadow services

**What's secure:**
- ✅ Best-effort delivery: API failures never affect live traffic
- ✅ Retry logic with exponential backoff
- ✅ Configurable timeouts
- ✅ No secrets in logs

**Production recommendations:**
- Use TLS for agent→API communication
- Deploy in isolated network (Kubernetes sidecar pattern)
- Review data retention and PII policies
- Consider sensitive header redaction

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [API Contracts](../architecture/API_CONTRACTS.md)
- [Quick Start Guide](../guides/QUICK_START.md)
- [Development Guide](../guides/DEVELOPMENT.md)
- [mroki-api Component](MROKI_API.md)
