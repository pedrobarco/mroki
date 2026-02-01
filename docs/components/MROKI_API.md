# mroki-api

**REST API for managing gates and storing traffic diffs**

mroki-api is a stateless REST API that manages gates (live/shadow service pairs), receives captured traffic from mroki-agent, and stores request/response diffs in PostgreSQL for later analysis.

## Features

- **Gate Management**: Create and manage live/shadow service pairs
- **Traffic Storage**: Persist captured requests, responses, and diffs
- **Query API**: Retrieve captured traffic for analysis
- **Health Checks**: Kubernetes-ready liveness/readiness probes
- **Connection Pooling**: Efficient PostgreSQL connection management
- **Type-Safe Queries**: Generated queries via sqlc
- **Stateless Design**: Horizontally scalable

## Architecture

```
┌──────────────────────────────────────┐
│          mroki-api                   │
│                                      │
│  ┌────────────────────────────────┐  │
│  │  HTTP Handlers                 │  │
│  │  (Gates, Requests, Health)     │  │
│  └────────────┬───────────────────┘  │
│               │                      │
│  ┌────────────▼───────────────────┐  │
│  │  Domain Services               │  │
│  │  (Validation, Business Logic)  │  │
│  └────────────┬───────────────────┘  │
│               │                      │
│  ┌────────────▼───────────────────┐  │
│  │  Repository Layer (sqlc)       │  │
│  └────────────┬───────────────────┘  │
│               │                      │
│  ┌────────────▼───────────────────┐  │
│  │  Connection Pool (pgxpool)     │  │
│  └────────────────────────────────┘  │
└────────────────┬─────────────────────┘
                 │ PostgreSQL
                 ↓
        ┌─────────────────┐
        │   PostgreSQL    │
        │   (Storage)     │
        └─────────────────┘
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
```

### Optional Configuration

```bash
# Server port (default: 8090)
MROKI_APP_PORT=8090

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
cd build/mroki-api
docker-compose up -d
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
MROKI_APP_PORT=8081
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
EOF

# Run
go run .
```

**Output:**
```
INFO Started server address=:8081
```

### Verify Health

```bash
# Liveness check
curl http://localhost:8081/health/live
# Output: OK

# Readiness check (verifies DB connection)
curl http://localhost:8081/health/ready
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
- `GET /gates` - List all gates

**Requests:**
- `POST /gates/:gate_id/requests` - Create captured request (agent-to-API)
- `GET /gates/:gate_id/requests/:request_id` - Get request with full details
- `GET /gates/:gate_id/requests` - List requests for gate

## Database Setup

### Schema

The API automatically creates tables on startup. Schema is defined in `internal/infrastructure/persistence/postgres/schema.sql`.

**Core Tables:**
- `gates` - Live/shadow service pairs
- `requests` - Captured HTTP requests
- `responses` - Live and shadow responses
- `diffs` - Computed differences

### Manual Schema Management

```bash
# Connect to database
psql -U postgres -d postgres

# View schema
\dt

# Query gates
SELECT id, live_url, shadow_url FROM gates;

# Query requests
SELECT id, method, path, created_at FROM requests;
```

### Migrations

Currently, schema is applied automatically on startup. Future versions will use migration tools (e.g., golang-migrate).

## Configuration Examples

### Local Development

```bash
MROKI_APP_PORT=8081
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
MROKI_APP_DATABASE_MAX_CONNS=10
```

### Production

```bash
MROKI_APP_PORT=8081
MROKI_APP_DATABASE_URL=postgres://apiuser:secure_password@postgres.internal:5432/mroki?sslmode=require
MROKI_APP_DATABASE_MAX_CONNS=50
MROKI_APP_DATABASE_MIN_CONNS=10
MROKI_APP_DATABASE_MAX_CONN_IDLE=10m
MROKI_APP_DATABASE_MAX_CONN_LIFE=2h
```

### Using Connection String Components

You can also set individual database components:

```bash
MROKI_APP_DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}
```

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
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
docker run -p 8081:8081 \
  -e MROKI_APP_DATABASE_URL=postgres://postgres:postgres@postgres:5432/postgres \
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
        - containerPort: 8081
        env:
        - name: MROKI_APP_PORT
          value: "8081"
        - name: MROKI_APP_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: database-url
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8081
          periodSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8081
          periodSeconds: 5
          failureThreshold: 2
        startupProbe:
          httpGet:
            path: /health/ready
            port: 8081
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
    targetPort: 8081
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
      - "8081:8081"
    environment:
      MROKI_APP_PORT: 8081
      MROKI_APP_DATABASE_URL: postgres://postgres:postgres@postgres:5432/mroki
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
```

## Usage Examples

### Create a Gate

```bash
curl -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -d '{
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.shadow.example.com"
  }'

# Response:
# {
#   "data": {
#     "id": "550e8400-e29b-41d4-a716-446655440000",
#     "live_url": "https://api.production.example.com",
#     "shadow_url": "https://api.shadow.example.com"
#   }
# }
```

### List All Gates

```bash
curl http://localhost:8081/gates | jq .

# Response:
# {
#   "data": [
#     {
#       "id": "550e8400-e29b-41d4-a716-446655440000",
#       "live_url": "https://api.production.example.com",
#       "shadow_url": "https://api.shadow.example.com"
#     }
#   ]
# }
```

### Get Gate by ID

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"
curl http://localhost:8081/gates/$GATE_ID | jq .
```

### List Captured Requests

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"
curl http://localhost:8081/gates/$GATE_ID/requests | jq .

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
curl http://localhost:8081/gates/$GATE_ID/requests/$REQUEST_ID | jq .

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
{"time":"2026-01-31T20:00:00Z","level":"INFO","msg":"Started server","address":":8081"}
{"time":"2026-01-31T20:00:15Z","level":"INFO","msg":"gate created","gate_id":"550e8400-e29b-41d4-a716-446655440000"}
{"time":"2026-01-31T20:00:30Z","level":"INFO","msg":"request created","gate_id":"550e8400","request_id":"7c9e6679","agent_id":"MacBook-Pro-a1b2c3d4"}
```

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
- Diff computation size

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
2. Agent not configured correctly
3. Agent API failures (check agent logs)

**Debug:**
```bash
# Check database directly
psql -U postgres -d postgres
SELECT COUNT(*) FROM requests WHERE gate_id = '550e8400-e29b-41d4-a716-446655440000';
```

## Security Considerations

**Current Status (v1):**
- **No authentication:** Anyone can create/query gates
- **No TLS:** HTTP only - use reverse proxy for HTTPS
- **No rate limiting:** Can be overwhelmed by traffic

**What's secure:**
- ✅ SQL injection prevented by parameterized queries (sqlc)
- ✅ Input validation in domain layer
- ✅ Connection pooling configured
- ✅ Health checks available

**Production recommendations:**
- Add API key authentication
- Use TLS/HTTPS (terminate at load balancer)
- Implement rate limiting (nginx, API gateway, or application-level)
- Restrict database user permissions
- Store database credentials securely (Kubernetes secrets, AWS Secrets Manager, etc.)

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

## Performance

**Expected baseline:**
- Throughput: ~500 req/s per instance (database-bound)
- Memory: ~100MB baseline + connection pool overhead

**Database Connection Pool:**
- Min: 5 (always maintained)
- Max: 25 (default, configurable)
- Idle timeout: 5m
- Max lifetime: 1h

**Tuning:**
```bash
# High traffic configuration
MROKI_APP_DATABASE_MAX_CONNS=100
MROKI_APP_DATABASE_MIN_CONNS=20
MROKI_APP_DATABASE_MAX_CONN_IDLE=10m
MROKI_APP_DATABASE_MAX_CONN_LIFE=2h
```

**Bottlenecks:**
- PostgreSQL write throughput
- Request body size (stored as JSONB)
- Diff computation size

**Optimization tips:**
- Use read replicas for GET endpoints
- Increase `MAX_CONNS` for high traffic
- Consider partitioning `requests` table by `created_at`
- Scale horizontally with load balancer

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [API Contracts](../architecture/API_CONTRACTS.md)
- [Quick Start Guide](../guides/QUICK_START.md)
- [Development Guide](../guides/DEVELOPMENT.md)
- [mroki-agent Component](MROKI_AGENT.md)
- [mroki-hub Component](MROKI_HUB.md)
