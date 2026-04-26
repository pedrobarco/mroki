# mroki-api

**REST API for managing gates, computing diffs, and storing traffic data**

mroki-api is a stateless REST API that manages gates (live/shadow service pairs), receives raw captured traffic from mroki-proxy, computes JSON diffs server-side, and stores results in PostgreSQL for later analysis.

## Features

- **Gate Management**: Create and manage live/shadow service pairs with filtering and sorting
- **Server-Side Diff Computation**: Computes JSON diffs from raw live/shadow responses on ingest
- **Traffic Storage**: Persist captured requests, responses, and computed diffs
- **Backward Compatibility**: Accepts pre-computed diffs from proxies (if provided)
- **Query API**: Retrieve captured traffic for analysis with filtering and sorting
- **Request ID Propagation**: `X-Request-ID` header generation and correlation across logs
- **Health Checks**: Kubernetes-ready liveness/readiness probes
- **Connection Pooling**: Efficient PostgreSQL connection management
- **Type-Safe Queries**: Generated queries via sqlc
- **Stateless Design**: Horizontally scalable

## Architecture

```mermaid
graph TD
    subgraph mroki-api
        Handlers["HTTP Handlers<br><i>Gates, Requests, Health</i>"] --> Domain["Domain Services<br><i>Validation, Business Logic</i>"]
        Domain --> DiffEngine["Diff Computation<br><i>pkg/diff · RFC 6902</i>"]
        DiffEngine --> Repo["Repository Layer<br><i>(sqlc)</i>"]
        Repo --> Pool["Connection Pool<br><i>(pgxpool)</i>"]
    end

    Pool -->|PostgreSQL| DB[("PostgreSQL<br><i>Storage</i>")]
```

## Installation

### From Source

```bash
# Clone repository
git clone https://github.com/pedrobarco/mroki.git
cd mroki

# Build
go build -o mroki-api ./cmd/mroki-api

# Run
./mroki-api
```

### Using Go Install

```bash
go install github.com/pedrobarco/mroki/cmd/mroki-api@latest
```

## Configuration

Configuration is via environment variables with the `MROKI_APP_` prefix.

### Required Configuration

```bash
# PostgreSQL connection string
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/mroki

# API key for authentication (required)
MROKI_APP_API_KEY=your-secret-api-key
```

### Optional Configuration

```bash
# Server port (default: 8090)
MROKI_APP_PORT=8090

# Rate limiting (default: 1000 requests per minute per IP)
MROKI_APP_RATE_LIMIT=1000

# Request body size limit in bytes (default: 10485760 = 10MB)
MROKI_APP_MAX_BODY_SIZE=10485760

# CORS allowed origins (comma-separated, empty = disabled)
MROKI_APP_CORS_ORIGINS=http://localhost:5173

# Request retention duration (Go duration format, default: 168h = 7 days)
MROKI_APP_RETENTION=168h

# Cleanup job interval (Go duration format, default: 1h)
MROKI_APP_CLEANUP_INTERVAL=1h

# Connection pool settings
MROKI_APP_DATABASE_MAX_CONNS=25           # default: 25
MROKI_APP_DATABASE_MIN_CONNS=5            # default: 5
MROKI_APP_DATABASE_MAX_CONN_IDLE=5m       # default: 5m
MROKI_APP_DATABASE_MAX_CONN_LIFE=1h       # default: 1h
```

## Running the API

### Prerequisites

**PostgreSQL 15+** must be running. Use Docker Compose for local development:

```bash
cd build/dev
docker compose up -d
```

This starts PostgreSQL on port 5432 with:
- Database: `postgres`
- User: `postgres`
- Password: `postgres`

### Start the API

```bash
cd cmd/mroki-api

# Create .env file
cat > .env << 'EOF'
MROKI_APP_PORT=8090
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
MROKI_APP_API_KEY=my-secret-key
EOF

# Run
go run .
```

**Output:**
```
INFO Started server address=:8090
```

### Verify Health

```bash
# Liveness check
curl http://localhost:8090/health/live
# Output: OK

# Readiness check (verifies DB connection)
curl http://localhost:8090/health/ready
# Output: OK
```

## API Endpoints

See [API Contracts](../architecture/API_CONTRACTS.md) for full endpoint documentation.

### Quick Reference

**Health:**
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe (checks DB)

**Gates:**
- `POST /gates` - Create gate
- `GET /gates/:gate_id` - Get gate by ID
- `GET /gates` - List all gates (with filtering and sorting)

**Requests:**
- `POST /gates/:gate_id/requests` - Create captured request (proxy-to-API)
- `GET /gates/:gate_id/requests/:request_id` - Get request with full details
- `GET /gates/:gate_id/requests` - List requests for gate

## Database Setup

### Schema

The API automatically creates tables on startup using ent's auto-migration. Schema is defined in `ent/schema/`.

**Core Tables:**
- `gates` - Live/shadow service pairs
- `requests` - Captured HTTP requests
- `responses` - Live and shadow responses
- `diffs` - Computed differences (computed server-side from response bodies)

### Manual Schema Management

```bash
# Connect to database
psql -U postgres -d postgres

# View schema
\dt

# Query gates
SELECT id, name, live_url, shadow_url, created_at FROM gates;

# Query requests
SELECT id, method, path, created_at FROM requests;
```

### Migrations

Currently, schema is applied automatically on startup. Future versions will use migration tools (e.g., golang-migrate).

## Configuration Examples

### Local Development

```bash
MROKI_APP_PORT=8090
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
MROKI_APP_API_KEY=dev-api-key
MROKI_APP_DATABASE_MAX_CONNS=10
MROKI_APP_CORS_ORIGINS=http://localhost:5173
```

### Production

```bash
MROKI_APP_PORT=8090
MROKI_APP_DATABASE_URL=postgres://apiuser:secure_password@postgres.internal:5432/mroki?sslmode=require
MROKI_APP_API_KEY=secure-production-key
MROKI_APP_DATABASE_MAX_CONNS=50
MROKI_APP_DATABASE_MIN_CONNS=10
MROKI_APP_DATABASE_MAX_CONN_IDLE=10m
MROKI_APP_DATABASE_MAX_CONN_LIFE=2h
MROKI_APP_RATE_LIMIT=1000
MROKI_APP_RETENTION=720h
MROKI_APP_CLEANUP_INTERVAL=1h
```

### Using Connection String Components

You can also set individual database components:

```bash
MROKI_APP_DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}
```

## Deployment

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mroki-api ./cmd/mroki-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mroki-api .
CMD ["./mroki-api"]
```

**Run:**
```bash
docker build -t mroki-api .
docker run -p 8090:8090 \
  -e MROKI_APP_DATABASE_URL=postgres://postgres:postgres@postgres:5432/postgres \
  -e MROKI_APP_API_KEY=my-secret-key \
  mroki-api
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mroki-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mroki-api
  template:
    metadata:
      labels:
        app: mroki-api
    spec:
      containers:
      - name: mroki-api
        image: mroki-api:latest
        ports:
        - containerPort: 8090
        env:
        - name: MROKI_APP_PORT
          value: "8090"
        - name: MROKI_APP_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: database-url
        - name: MROKI_APP_API_KEY
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: api-key
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8090
          periodSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8090
          periodSeconds: 5
          failureThreshold: 2
        startupProbe:
          httpGet:
            path: /health/ready
            port: 8090
          periodSeconds: 5
          failureThreshold: 12
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: mroki-api
spec:
  selector:
    app: mroki-api
  ports:
  - port: 80
    targetPort: 8090
  type: ClusterIP
---
apiVersion: v1
kind: Secret
metadata:
  name: mroki-secrets
type: Opaque
stringData:
  database-url: "postgres://apiuser:secure_password@postgres:5432/mroki?sslmode=require"
```

### Docker Compose (Full Stack)

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: mroki
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  mroki-api:
    build: .
    ports:
      - "8090:8090"
    environment:
      MROKI_APP_PORT: 8090
      MROKI_APP_DATABASE_URL: postgres://postgres:postgres@postgres:5432/mroki
      MROKI_APP_API_KEY: my-secret-key
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
```

## Usage Examples

### Create a Gate

```bash
curl -X POST http://localhost:8090/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "name": "checkout-api",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.shadow.example.com"
  }'

# Response:
# {
#   "data": {
#     "id": "550e8400-e29b-41d4-a716-446655440000",
#     "name": "checkout-api",
#     "live_url": "https://api.production.example.com",
#     "shadow_url": "https://api.shadow.example.com",
#     "created_at": "2026-03-29T09:00:00Z"
#   }
# }
```

### List All Gates

```bash
# List all gates (default: sorted by created_at desc)
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates | jq .

# Filter by name containing "checkout"
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?name=checkout" | jq .

# Filter by live URL containing "production"
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?live_url=production" | jq .

# Sort by name ascending
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?sort=name&order=asc" | jq .

# Combine filtering, sorting, and pagination
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?name=api&sort=created_at&order=desc&limit=10&offset=0" | jq .

# Response:
# {
#   "data": [
#     {
#       "id": "550e8400-e29b-41d4-a716-446655440000",
#       "name": "checkout-api",
#       "live_url": "https://api.production.example.com",
#       "shadow_url": "https://api.shadow.example.com",
#       "created_at": "2026-03-29T09:00:00Z"
#     }
#   ],
#   "pagination": {
#     "limit": 50,
#     "offset": 0,
#     "total": 1,
#     "has_more": false
#   }
# }
```

### Get Gate by ID

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/$GATE_ID | jq .
```

### List Captured Requests

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/$GATE_ID/requests | jq .

# Response:
# {
#   "data": [
#     {
#       "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
#       "method": "POST",
#       "path": "/api/users",
#       "created_at": "2026-01-31T20:00:00Z"
#     }
#   ]
# }
```

### Get Request Details

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"
REQUEST_ID="7c9e6679-7425-40de-944b-e07fc1f90ae7"
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/$GATE_ID/requests/$REQUEST_ID | jq .

# Response includes full request, responses, and diff
```

## Logging

All logs use structured logging (slog) with JSON output.

**Log Levels:**
- `DEBUG` - Database queries, detailed operations
- `INFO` - Normal operations (requests created, server started)
- `WARN` - Recoverable errors
- `ERROR` - Unrecoverable errors

**Example Log Output:**
```json
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"Started server","address":":8090"}
{"time":"2026-01-31T20:00:15Z","level":"INFO","msg":"200: OK","request.id":"7c9e6679-7425-40de-944b-e07fc1f90ae7","request.method":"GET","request.path":"/gates","response.status":200,"response.latency":"1.234ms"}
{"time":"2026-01-31T20:00:30Z","level":"ERROR","msg":"API error","request.id":"8d0e7780-8536-51ef-a55c-f18fd2f91bf8","error.type":"/errors/not-found","error.title":"Gate Not Found","error.status":404}
```

> **Note:** Every request is assigned an `X-Request-ID` (generated if not provided by the client). This ID appears in all log entries as `request.id` and is returned in the `X-Request-ID` response header, enabling end-to-end correlation across proxy and API logs.

## Performance

**Throughput:** ~500 req/s per instance (database-bound)

**Memory:** ~100MB baseline + connection pool overhead

**Database Connections:**
- Min: 5 (always maintained)
- Max: 25 (default, configurable)
- Idle timeout: 5m (connections closed if unused)
- Max lifetime: 1h (connections recreated periodically)

**Bottlenecks:**
- PostgreSQL write throughput
- Request body size (stored as JSONB)
- Server-side diff computation (synchronous during ingest — large JSON responses increase write latency)

**Optimization:**
- Use read replicas for query endpoints
- Increase `MAX_CONNS` for high traffic
- Consider partitioning `requests` table by `created_at`

## Troubleshooting

### API won't start

**Problem:** `panic: configuration validation failed: database.url is required`

**Solution:** Set `MROKI_APP_DATABASE_URL` environment variable.

---

**Problem:** `panic: failed to create connection pool: connection refused`

**Solution:** Ensure PostgreSQL is running and accessible:

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
psql -U postgres -h localhost -p 5432 -d postgres
```

---

**Problem:** `panic: configuration validation failed: port must be between 1 and 65535`

**Solution:** Set valid `MROKI_APP_PORT` (1-65535).

### Health check fails

**Problem:** `GET /health/ready` returns 503

**Solution:** Database is unreachable. Check:

```bash
# Test database connectivity
psql -U postgres -h localhost -p 5432 -d postgres

# Check connection string format
echo $MROKI_APP_DATABASE_URL
# Should be: postgres://user:pass@host:port/database
```

### Cannot create gate

**Problem:** `POST /gates` returns 500

**Solution:** Check logs for database errors. Common causes:
- Database schema not created
- Connection pool exhausted
- Invalid URL format in request

### Cannot query requests

**Problem:** `GET /gates/:gate_id/requests` returns empty array

**Possible causes:**
1. No traffic has been captured yet
2. Proxy not configured correctly
3. Proxy API failures (check proxy logs)

**Debug:**
```bash
# Check database directly
psql -U postgres -d postgres
SELECT COUNT(*) FROM requests WHERE gate_id = '550e8400-e29b-41d4-a716-446655440000';
```

## Security

**Implemented:**
- API key authentication (`Authorization: Bearer <key>`, configured via `MROKI_APP_API_KEY`)
- Rate limiting (token bucket, configurable via `MROKI_APP_RATE_LIMIT`, default 1000 req/min/IP)
- Request body size limits (configurable via `MROKI_APP_MAX_BODY_SIZE`, default 10MB)
- Input validation via domain value objects
- SQL injection prevention via parameterized queries (sqlc)
- CORS with configurable allowed origins (`MROKI_APP_CORS_ORIGINS`)
- HTTP timeouts and graceful shutdown
- RFC 7807 error responses

**Not yet implemented:**
- TLS/HTTPS — use a reverse proxy (nginx, Caddy, cloud LB) to terminate TLS
- RBAC / multi-tenant authorization
- PII redaction in captured requests
- Proxy-to-API mutual TLS

**Production recommendations:**
- Use a strong, randomly generated API key
- Terminate TLS at load balancer or reverse proxy
- Restrict database user permissions
- Store secrets securely (Kubernetes secrets, AWS Secrets Manager, etc.)

## Testing

```bash
# Run all tests
go test ./internal/...

# Run API handler tests
go test ./internal/interfaces/http/handlers/...

# Run domain tests
go test ./internal/domain/...

# Run with race detection
go test -race ./...

# Get coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Current test coverage:** 62.8% overall
- Domain layer: 98.6%
- Application layer: 85%+
- Infrastructure layer: 68%
- Interface layer: 80%+

---

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [API Contracts](../architecture/API_CONTRACTS.md)
- [Quick Start Guide](../guides/QUICK_START.md)
- [Development Guide](../guides/DEVELOPMENT.md)
- [mroki-proxy Component](MROKI_PROXY.md)
- [mroki-hub Component](MROKI_HUB.md)
