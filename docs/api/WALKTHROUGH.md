# API Walkthrough

Step-by-step guide to using the mroki API — from creating your first gate to querying diffs.

## Prerequisites

- **mroki-api running** — see [Full Stack](../getting-started/FULL_STACK.md) or [Docker Compose](../production/DOCKER_COMPOSE.md) setup guides
- **API key** — the value you configured as `MROKI_APP_API_KEY` (minimum 16 characters)
- **curl** (or any HTTP client)

All examples below use `http://localhost:8090` as the API base URL and `your-api-key` as the bearer token. Replace these with your actual values.

## Step 1: Create a Gate

A **gate** represents a live/shadow service pair. The proxy sends traffic to both URLs; the API stores responses and computes diffs.

```bash
curl -X POST http://localhost:8090/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "name": "checkout-api",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.staging.example.com"
  }'
```

**Response:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "checkout-api",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.staging.example.com",
    "diff_config": {
      "ignored_fields": [],
      "included_fields": [],
      "float_tolerance": 0
    },
    "redacted_fields": [],
    "created_at": "2026-03-29T09:00:00Z",
    "stats": {
      "request_count_24h": 0,
      "diff_count_24h": 0,
      "diff_rate": 0,
      "last_active": null
    }
  }
}
```

> **Save the `id` value** — you'll need it for every subsequent step. The examples below use `GATE_ID` as a placeholder; replace it with your actual gate ID.

## Step 2: Configure a Proxy

Point mroki-proxy at the gate you just created. Set these environment variables (or add them to `cmd/mroki-proxy/.env`):

```bash
MROKI_APP_API_URL=http://localhost:8090
MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000
MROKI_APP_API_KEY=your-api-key
MROKI_APP_PORT=8080
```

Then start the proxy:

```bash
cd cmd/mroki-proxy && go run .
```

See [Configuration](../production/CONFIGURATION.md) for the full list of proxy settings.

## Step 3: Capture Traffic

Send requests through the proxy (port 8080 by default). The proxy forwards each request to **both** the live and shadow URLs, returns the live response to the caller, and sends the raw responses to mroki-api in the background. The API computes the diff server-side and stores everything in PostgreSQL.

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "age": 30}'
```

You'll receive the live response immediately. Check the proxy logs for confirmation:

```
DEBUG successfully sent request to API method=POST path=/api/users live_status=200 shadow_status=200
```

## Step 4: List Captured Requests

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"

curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates/$GATE_ID/requests" | jq .
```

**Response:**

```json
{
  "data": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "method": "POST",
      "path": "/api/users",
      "created_at": "2026-01-31T20:00:00Z",
      "live_response": { "status_code": 200, "latency_ms": 142 },
      "shadow_response": { "status_code": 200, "latency_ms": 187 },
      "has_diff": true
    }
  ],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 1,
    "has_more": false
  }
}
```

The `has_diff` field tells you at a glance whether the live and shadow responses differed.

## Step 5: View Request Detail

```bash
GATE_ID="550e8400-e29b-41d4-a716-446655440000"
REQUEST_ID="7c9e6679-7425-40de-944b-e07fc1f90ae7"

curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates/$GATE_ID/requests/$REQUEST_ID" | jq .
```

**Response:**

```json
{
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "method": "POST",
    "path": "/api/users",
    "headers": {
      "Content-Type": ["application/json"],
      "User-Agent": ["curl/7.68.0"]
    },
    "body": "{\"name\":\"Alice\",\"age\":30}",
    "created_at": "2026-01-31T20:00:00Z",
    "live_response": {
      "id": "8d0e7780-8536-51ef-a55c-f18fd2f91bf8",
      "status_code": 200,
      "headers": { "Content-Type": ["application/json"] },
      "body": "{\"id\":123,\"name\":\"Alice\",\"age\":30}",
      "latency_ms": 142,
      "created_at": "2026-01-31T20:00:01Z"
    },
    "shadow_response": {
      "id": "9e1f8891-9647-62f0-b66d-027fe3f02cf9",
      "status_code": 200,
      "headers": { "Content-Type": ["application/json"] },
      "body": "{\"id\":456,\"name\":\"Alice\",\"age\":30}",
      "latency_ms": 187,
      "created_at": "2026-01-31T20:00:01Z"
    },
    "diff": {
      "content": [
        {
          "op": "replace",
          "path": "/id",
          "value": 456
        }
      ]
    }
  }
}
```

The `diff.content` array uses [RFC 6902 JSON Patch](https://datatracker.ietf.org/doc/html/rfc6902) format. Each operation has three fields:

- **`op`** — `add` (field only in shadow), `remove` (field only in live), or `replace` (different values)
- **`path`** — JSON Pointer ([RFC 6901](https://datatracker.ietf.org/doc/html/rfc6901)) to the affected field
- **`value`** — the shadow-side value (omitted for `remove` operations)

## Step 6: Filter and Sort

The request list endpoint supports query parameters for filtering and sorting:

| Parameter | Example | Description |
|-----------|---------|-------------|
| `method` | `GET,POST` | Filter by HTTP method (comma-separated) |
| `path` | `/api/users/*` | Filter by path pattern (supports wildcards) |
| `has_diff` | `true` | Only requests with (`true`) or without (`false`) diffs |
| `from` / `to` | `2026-03-29T00:00:00Z` | Filter by creation timestamp (RFC 3339) |
| `sort` | `created_at` | Sort field: `created_at`, `method`, or `path` |
| `order` | `desc` | Sort direction: `asc` or `desc` |
| `limit` | `10` | Results per page (max 100, default 50) |
| `offset` | `0` | Number of results to skip |

**Example — recent requests with diffs only:**

```bash
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates/$GATE_ID/requests?has_diff=true&sort=created_at&order=desc&limit=10" | jq .
```

## Step 7: Configure Diff Options

Fine-tune how diffs are computed by updating the gate's `diff_config`:

```bash
curl -X PATCH "http://localhost:8090/gates/$GATE_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "diff_config": {
      "ignored_fields": ["timestamp", "request_id"],
      "included_fields": [],
      "float_tolerance": 0.001
    }
  }'
```

**Response:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "checkout-api",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.staging.example.com",
    "diff_config": {
      "ignored_fields": ["timestamp", "request_id"],
      "included_fields": [],
      "float_tolerance": 0.001
    },
    "redacted_fields": [],
    "created_at": "2026-03-29T09:00:00Z",
    "stats": {
      "request_count_24h": 1,
      "diff_count_24h": 1,
      "diff_rate": 100,
      "last_active": "2026-01-31T20:00:00Z"
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `ignored_fields` | JSON field paths to exclude from diff computation (supports wildcards) |
| `included_fields` | JSON field paths to include exclusively in diff computation (supports wildcards) |
| `float_tolerance` | Absolute tolerance for floating-point comparisons (`0` = exact match) |

## Step 8: Configure Field Redaction

Sensitive fields are redacted from stored requests and responses. The defaults (`headers.Authorization`, `headers.Cookie`, `headers.Set-Cookie`, `headers.X-Api-Key`) are always applied. Use `redacted_fields` to add per-gate fields on top:

```bash
curl -X PATCH "http://localhost:8090/gates/$GATE_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "redacted_fields": ["headers.X-Internal-Token", "body.user.password"]
  }'
```

**Response:**

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "checkout-api",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.staging.example.com",
    "diff_config": {
      "ignored_fields": ["timestamp", "request_id"],
      "included_fields": [],
      "float_tolerance": 0.001
    },
    "redacted_fields": ["headers.X-Internal-Token", "body.user.password"],
    "created_at": "2026-03-29T09:00:00Z",
    "stats": {
      "request_count_24h": 1,
      "diff_count_24h": 1,
      "diff_rate": 100,
      "last_active": "2026-01-31T20:00:00Z"
    }
  }
}
```

Redacted field paths use a `headers.` or `body.` prefix. For example, `body.user.password` redacts the `password` field inside `user` in the JSON body.

## What's Next

- [API Reference](REFERENCE.md) — full endpoint specification, error codes, and pagination details
- [Configuration](../production/CONFIGURATION.md) — all environment variables for mroki-api and mroki-proxy
- [Security](../production/SECURITY.md) — API key management, TLS, and network hardening
