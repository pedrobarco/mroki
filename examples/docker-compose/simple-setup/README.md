# Simple Docker Compose Setup

A minimal working example to get started with mroki locally.

## What This Does

This example starts:
- PostgreSQL database on port 5432
- mroki-api on port 8081
- mroki-agent on port 8080

## Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- mroki images built locally (or use registry images)

## Build Images First

If you haven't built the images yet:

```bash
# From the repository root
cd /path/to/mroki

# Build API
docker build -f build/mroki-api/Dockerfile -t mroki-api:latest .

# Build Agent
docker build -f build/mroki-agent/Dockerfile -t mroki-agent:latest .
```

## Usage

### 1. Start Services

```bash
docker-compose up -d
```

Wait for services to be healthy:
```bash
docker-compose ps
```

### 2. Create a Gate

```bash
curl -X POST http://localhost:8081/api/v1/gates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-test-gate",
    "description": "Testing shadow traffic"
  }'
```

This returns a gate ID like `550e8400-e29b-41d4-a716-446655440000`.

### 3. Update Agent Configuration

Edit `docker-compose.yaml` and update these environment variables:

```yaml
MROKI_APP_LIVE_URL: https://your-production-api.com
MROKI_APP_SHADOW_URL: https://your-shadow-api.com
MROKI_APP_GATE_ID: 550e8400-e29b-41d4-a716-446655440000  # Use your gate ID
```

Restart the agent:
```bash
docker-compose restart mroki-agent
```

### 4. Send Test Requests

Send requests through the agent:
```bash
curl http://localhost:8080/posts/1
```

The agent will:
1. Forward to both live and shadow URLs
2. Compare responses
3. Store diffs in the database

### 5. Check for Diffs

```bash
curl http://localhost:8081/api/v1/gates/550e8400-e29b-41d4-a716-446655440000/diffs
```

## View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f mroki-agent
docker-compose logs -f mroki-api
```

## Stop Services

```bash
docker-compose down
```

To also remove volumes (deletes database):
```bash
docker-compose down -v
```

## Default Configuration

- **Database**: postgres/mroki_dev_password@localhost:5432/mroki
- **API**: http://localhost:8081
- **Agent**: http://localhost:8080
- **Test URLs**: jsonplaceholder.typicode.com (harmless public API)

## Next Steps

1. Replace live/shadow URLs with your actual services
2. Create a proper gate with your route configuration
3. Update GATE_ID in the docker-compose.yaml
4. Send production-like requests through port 8080
5. Check for diffs via the API

## Troubleshooting

**Services won't start:**
```bash
docker-compose logs
```

**Can't connect to API:**
```bash
curl http://localhost:8081/health/live
```

**Agent not capturing traffic:**
- Verify GATE_ID is correct
- Check agent logs: `docker-compose logs mroki-agent`
- Ensure live/shadow URLs are reachable

**Database connection errors:**
- Wait for postgres to be healthy
- Check postgres logs: `docker-compose logs postgres`
