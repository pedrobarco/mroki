# Getting Started: Full Stack

Run the complete mroki stack with Docker Compose — proxy, API, database, and web UI.

## What You'll Build

A shadow traffic pipeline: the **proxy** forwards requests to live and shadow services, the **API** stores responses and computes diffs, and the **hub** lets you browse results.

```mermaid
graph TD
    Client([Client]) -->|HTTP Request| Proxy[mroki-proxy]
    Proxy -->|Forward| Live[Live Service]
    Proxy -->|Forward| Shadow[Shadow Service]
    Live -->|Live Response| Proxy
    Shadow -->|Shadow Response| Proxy
    Proxy -->|Return live response| Client
    Proxy -.->|Send raw responses| API[mroki-api<br><i>REST API + Diff</i>]
    API -->|Store| DB[(PostgreSQL)]
    Hub[mroki-hub<br><i>Web UI</i>] -->|Query| API
```

## Prerequisites

- **Docker & Docker Compose** — to run the entire stack
- **curl** — for testing (or any HTTP client)

## Step 1: Start the Dev Stack

```bash
docker compose -f build/dev/compose.yaml --profile backend up -d
```

Expected output:

```
✔ Container mroki-db-1    Healthy
✔ Container mroki-api-1   Started
✔ Container mroki-proxy-1 Started
```

This starts **PostgreSQL** (port 5432), **mroki-api** (port 8090), and **mroki-proxy** (port 8080). The proxy is pre-configured with a seed gate ID.

## Step 2: Create a Gate

A **gate** is a pair of live and shadow service URLs. All traffic through the proxy is forwarded to both, and the responses are compared.

```bash
curl -s -X POST http://localhost:8090/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mroki-dev-api-key-16" \
  -d '{
    "name": "httpbin-test",
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }' | jq .
```

Expected response:

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "httpbin-test",
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow",
    "created_at": "2026-05-23T09:00:00Z"
  }
}
```

**Copy the `id` value** — you'll need it in the next step.

## Step 3: Configure the Proxy

Update the proxy to use your new gate, then restart it:

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"  # Replace with your gate ID

docker compose -f build/dev/compose.yaml --profile backend down mroki-proxy
MROKI_APP_GATE_ID=$GATE_ID docker compose -f build/dev/compose.yaml --profile backend up -d mroki-proxy
```

## Step 4: Send Test Traffic

```bash
curl -s -X POST http://localhost:8080/test \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30}' | jq .
```

The proxy forwards the request to **both** live and shadow services simultaneously. The live response is returned to you immediately. In the background, the proxy sends both raw responses to mroki-api, which computes the diff and stores everything in PostgreSQL.

## Step 5: View Results

**Option A — Web UI:** Start the hub and open it in your browser:

```bash
docker compose -f build/dev/compose.yaml --profile frontend up -d
```

Open [http://localhost:5173](http://localhost:5173) to browse gates, requests, and diffs.

**Option B — API:**

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"  # Your gate ID

# List captured requests
curl -s -H "Authorization: Bearer mroki-dev-api-key-16" \
  http://localhost:8090/gates/$GATE_ID/requests | jq .
```

```json
{
  "data": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "method": "POST",
      "path": "/test",
      "created_at": "2026-05-23T09:01:00Z"
    }
  ]
}
```

```bash
# Get request detail with diff (RFC 6902 JSON Patch format)
REQUEST_ID="7c9e6679-7425-40de-944b-e07fc1f90ae7"  # From above

curl -s -H "Authorization: Bearer mroki-dev-api-key-16" \
  http://localhost:8090/gates/$GATE_ID/requests/$REQUEST_ID | jq .
```

## Clean Up

```bash
docker compose -f build/dev/compose.yaml --profile backend --profile frontend down
```

For troubleshooting, see [Troubleshooting](../TROUBLESHOOTING.md).

## What's Next

- [Standalone Proxy](STANDALONE_PROXY.md) — lightweight mode without the API
- [Production Deployment](../production/DOCKER_COMPOSE.md) — deploy for real workloads
- [API Walkthrough](../api/WALKTHROUGH.md) — programmatic access to gates and diffs
- [Configuration](../production/CONFIGURATION.md) — all available options
