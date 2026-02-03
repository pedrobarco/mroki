# Quick Start Guide

Get mroki up and running in 5 minutes.

## Prerequisites

- **Docker & Docker Compose** - For running PostgreSQL
- **Go 1.21+** - For building/running binaries
- **curl** - For testing (or any HTTP client)

## Step 1: Start PostgreSQL

```bash
cd /Users/barco/repos/pedrobarco/mroki

# Start PostgreSQL via Docker Compose
docker-compose -f build/mroki-api/docker-compose.yaml up -d

# Verify it's running
docker ps | grep postgres
# Should show postgres:15-alpine container running on port 5432
```

## Step 2: Start mroki-api

Open a new terminal:

```bash
cd cmd/mroki-api

# Create configuration file
cat > .env << 'EOF'
MROKI_APP_PORT=8081
MROKI_APP_API_KEY=dev-test-key-min-16-chars
MROKI_APP_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
EOF

# Start the API
go run .
```

**Expected output:**
```
INFO Starting server address=:8081
```

Keep this terminal open.

## Step 3: Create a Gate

Open a new terminal:

```bash
# Create a gate (live/shadow service pair)
curl -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-test-key-min-16-chars" \
  -d '{
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }'
```

**Expected response:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }
}
```

**Copy the `id` value** - you'll need it in the next step.

## Step 4: Start mroki-agent

Open a new terminal:

```bash
cd cmd/mroki-agent

# Create configuration file (replace GATE_ID with your actual gate ID)
cat > .env << 'EOF'
MROKI_APP_PORT=8080
MROKI_APP_API_URL=http://localhost:8081
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=dev-test-key-min-16-chars
EOF

# IMPORTANT: Replace the GATE_ID in .env with your actual gate ID from step 3

# Start the agent
go run .
```

**Expected output:**
```
INFO Agent ID loaded agent_id=MacBook-Pro-a1b2c3d4-...
INFO Starting in API mode api_url=http://localhost:8081 gate_id=550e8400-...
INFO Gate configuration loaded gate_id=550e8400-... live_url=https://httpbin.org/... shadow_url=https://httpbin.org/...
INFO Started server address=:8080
```

Keep this terminal open.

## Step 5: Send Test Traffic

Open a new terminal:

```bash
# Send a test request through the agent
curl -X POST http://localhost:8080/test \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30}'
```

**What happens:**
1. Agent forwards request to **both** live and shadow services
2. Live response returned to you immediately
3. Agent computes diff in background
4. Diff sent to mroki-api and stored in PostgreSQL

**Check agent logs:** You should see:
```
INFO response diff detected method=POST path=/test live_status=200 shadow_status=200
DEBUG successfully sent request to API method=POST path=/test has_diff=true
```

## Step 6: View Captured Requests

```bash
# List all captured requests for your gate
GATE_ID="550e8400-e29b-41d4-a716-446655440000"  # Replace with your gate ID
curl -H "Authorization: Bearer dev-test-key-min-16-chars" \
  http://localhost:8081/gates/$GATE_ID/requests | jq .
```

**Expected response:**
```json
{
  "data": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "method": "POST",
      "path": "/test",
      "created_at": "2026-01-31T20:00:00Z"
    }
  ]
}
```

## Step 7: View Request Details

```bash
# Get full request details including diff
GATE_ID="550e8400-e29b-41d4-a716-446655440000"  # Your gate ID
REQUEST_ID="7c9e6679-7425-40de-944b-e07fc1f90ae7"  # From step 6

curl -H "Authorization: Bearer dev-test-key-min-16-chars" \
  http://localhost:8081/gates/$GATE_ID/requests/$REQUEST_ID | jq .
```

**Response includes:**
- Original request (method, path, headers, body)
- Live response (status, headers, body)
- Shadow response (status, headers, body)
- Computed diff (JSON patch format)

## Congratulations!

You've successfully:
- ✅ Started PostgreSQL
- ✅ Started mroki-api
- ✅ Created a gate
- ✅ Started mroki-agent
- ✅ Sent traffic through the agent
- ✅ Viewed captured diffs via API

## What's Next?

### Test with Your Own Services

Create a new gate with your actual service URLs:

```bash
# Create gate with your services
curl -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-test-key-min-16-chars" \
  -d '{
    "live_url": "http://localhost:3000",
    "shadow_url": "http://localhost:3001"
  }'

# Get the gate ID from response, then update agent config
# Edit cmd/mroki-agent/.env and update MROKI_APP_GATE_ID
```

### Explore Advanced Features

**Try Different Requests:**
```bash
# GET request
curl http://localhost:8080/users/123

# POST with JSON
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@example.com"}'

# PUT request
curl -X PUT http://localhost:8080/users/123 \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob Updated"}'
```

**View All Gates:**
```bash
curl -H "Authorization: Bearer dev-test-key-min-16-chars" \
  http://localhost:8081/gates | jq .
```

**Check API Health:**
```bash
curl http://localhost:8081/health/live
curl http://localhost:8081/health/ready
```

### Test Agent Features

**Agent ID Persistence:**
```bash
# Check the agent ID file
cat cmd/mroki-agent/.agent_id

# Stop agent (Ctrl+C), then restart
cd cmd/mroki-agent && go run .

# Notice the SAME agent_id in logs - it persisted!
```

**Standalone Mode (No API):**
```bash
# Edit cmd/mroki-agent/.env - remove API config, add URLs
cat > cmd/mroki-agent/.env << 'EOF'
MROKI_APP_LIVE_URL=https://httpbin.org/anything?service=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?service=shadow
MROKI_APP_PORT=8080
EOF

# Restart agent
cd cmd/mroki-agent && go run .

# Should see: "Starting in standalone mode"
# Requests still work, but diffs are not stored
```

**Test Diff Configuration:**
```bash
# Add diff options to cmd/mroki-agent/.env
cat >> cmd/mroki-agent/.env << 'EOF'
MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at,url
MROKI_APP_DIFF_SORT_ARRAYS=true
EOF

# Restart agent
cd cmd/mroki-agent && go run .

# Should see: "Diff options configured" in logs
```

**Test Retry Logic:**
```bash
# Stop mroki-api (Ctrl+C in API terminal)

# Send request through agent
curl http://localhost:8080/test -H "Content-Type: application/json" -d '{}'

# Check agent logs - should show retry attempts:
# WARN API request failed attempt=1 error="connection refused"
# INFO retrying API request attempt=1 delay=1s
# ...

# But your request still succeeds! Agent never fails live traffic.
```

## Troubleshooting

### Authentication Errors

**Problem:** `401 Unauthorized` or `{"type":"about:blank","title":"Unauthorized","status":401,...}`

**Solution:** Ensure API_KEY is configured and Bearer token is included:

```bash
# Check API key in .env files match
grep API_KEY cmd/mroki-api/.env
grep API_KEY cmd/mroki-agent/.env

# Test with explicit Bearer token
curl -H "Authorization: Bearer dev-test-key-min-16-chars" \
  http://localhost:8081/gates
```

---

**Problem:** `{"type":"about:blank","title":"Invalid API Key","status":401,...}`

**Solution:** API key is too short or doesn't match. API keys must be at least 16 characters and match between API and agent:

```bash
# Update both .env files with same key
echo 'MROKI_APP_API_KEY=your-new-key-min-16-chars' >> cmd/mroki-api/.env
echo 'MROKI_APP_API_KEY=your-new-key-min-16-chars' >> cmd/mroki-agent/.env

# Restart both services
```

---

**Problem:** `429 Too Many Requests`

**Solution:** Rate limit exceeded (default: 1000 requests per minute per IP). Wait 60 seconds or adjust:

```bash
# Add to cmd/mroki-api/.env
MROKI_APP_RATE_LIMIT=5000  # Increase to 5000 req/min

# Restart API
```

---

### PostgreSQL connection fails

**Problem:** `failed to create connection pool: connection refused`

**Solution:**
```bash
# Check Docker is running
docker ps

# Restart PostgreSQL
docker-compose -f build/mroki-api/docker-compose.yaml restart
```

---

### Agent won't start

**Problem:** `configuration validation failed: must configure either API mode or standalone mode`

**Solution:** Either set API mode (API_URL + GATE_ID + API_KEY) or standalone mode (LIVE_URL + SHADOW_URL), not both:

```bash
# Option 1: API mode (all three required)
MROKI_APP_API_URL=http://localhost:8081
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=dev-test-key-min-16-chars

# Option 2: Standalone mode (both required)
MROKI_APP_LIVE_URL=https://api.production.example.com
MROKI_APP_SHADOW_URL=https://api.shadow.example.com
```

---

### No diffs captured

**Problem:** Requests go through but no diffs appear

**Possible causes:**
1. Responses are not JSON (only JSON diffs are captured)
2. Responses are identical (no diff)
3. API is down (check retry logs)

**Debug:**
```bash
# Verify both responses are JSON
curl http://localhost:8080/test -H "Content-Type: application/json" -d '{"test":true}'

# Check if httpbin returns JSON
curl https://httpbin.org/anything?service=live
```

## Quick Reference

**Start Stack:**
```bash
# Terminal 1: PostgreSQL
docker-compose -f build/mroki-api/docker-compose.yaml up -d

# Terminal 2: API
cd cmd/mroki-api && go run .

# Terminal 3: Agent
cd cmd/mroki-agent && go run .
```

**Stop Stack:**
```bash
# Stop agent: Ctrl+C in terminal 3
# Stop API: Ctrl+C in terminal 2
# Stop PostgreSQL:
docker-compose -f build/mroki-api/docker-compose.yaml down
```

**Key Endpoints:**
- Agent proxy: `http://localhost:8080`
- API: `http://localhost:8081`
- Create gate: `POST http://localhost:8081/gates` (requires Bearer token)
- List gates: `GET http://localhost:8081/gates` (requires Bearer token)
- List requests: `GET http://localhost:8081/gates/:gate_id/requests` (requires Bearer token)
- Get request: `GET http://localhost:8081/gates/:gate_id/requests/:request_id` (requires Bearer token)

**Configuration:**
- API requires Bearer token authentication (minimum 16 characters)
- Rate limit: 1000 requests/minute per IP (configurable via `MROKI_APP_RATE_LIMIT`)
- Max body size: 10MB (configurable via `MROKI_APP_MAX_BODY_SIZE`)
- All API calls need: `Authorization: Bearer <your-api-key>`

## Next Steps

- Read [Architecture Overview](../architecture/OVERVIEW.md)
- Explore [API Contracts](../architecture/API_CONTRACTS.md)
- Learn about [mroki-agent](../components/MROKI_AGENT.md)
- Learn about [mroki-api](../components/MROKI_API.md)
- Check [Development Guide](DEVELOPMENT.md) for contributing
