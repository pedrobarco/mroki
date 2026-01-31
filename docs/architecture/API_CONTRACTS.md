# API Contracts

This document specifies the REST API provided by `mroki-api`, including endpoints, request/response formats, and error handling.

## Base URL

```
http://localhost:8081
```

## Response Format

All successful API responses follow this structure:

```json
{
  "data": <response_data>
}
```

All error responses follow this structure:

```json
{
  "error": {
    "message": "human-readable error message",
    "details": "additional error context (optional)"
  }
}
```

---

## Endpoints

### Health Checks

#### GET /health/live

**Purpose:** Kubernetes liveness probe - checks if process is running

**Response:**
- `200 OK` - Process is alive
- Body: `OK` (plain text)

**Example:**
```bash
curl http://localhost:8081/health/live
```

---

#### GET /health/ready

**Purpose:** Kubernetes readiness probe - checks if service can handle traffic

**Response:**
- `200 OK` - Service is ready (database connected)
- `503 Service Unavailable` - Service not ready (database unreachable)
- Body: `OK` or error message (plain text)

**Example:**
```bash
curl http://localhost:8081/health/ready
```

---

### Gates

#### POST /gates

**Purpose:** Create a new gate (live/shadow service pair)

**Request Body:**
```json
{
  "live_url": "https://api.production.example.com",
  "shadow_url": "https://api.shadow.example.com"
}
```

**Validation:**
- `live_url` is required, must be valid HTTP/HTTPS URL
- `shadow_url` is required, must be valid HTTP/HTTPS URL

**Response:**
- `201 Created` on success
- `400 Bad Request` if validation fails

**Success Response Body:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.shadow.example.com"
  }
}
```

**Error Response Examples:**
```json
{
  "error": {
    "message": "invalid URL",
    "details": "live_url must use http or https scheme"
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -d '{
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }'
```

---

#### GET /gates/:gate_id

**Purpose:** Retrieve a specific gate by ID

**Path Parameters:**
- `gate_id` (UUID) - Gate identifier

**Response:**
- `200 OK` on success
- `400 Bad Request` if gate_id is invalid UUID
- `404 Not Found` if gate doesn't exist

**Success Response Body:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "live_url": "https://api.production.example.com",
    "shadow_url": "https://api.shadow.example.com"
  }
}
```

**Example:**
```bash
curl http://localhost:8081/gates/550e8400-e29b-41d4-a716-446655440000
```

---

#### GET /gates

**Purpose:** List all gates

**Response:**
- `200 OK` on success
- Returns empty array if no gates exist

**Success Response Body:**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "live_url": "https://api.production.example.com",
      "shadow_url": "https://api.shadow.example.com"
    },
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "live_url": "https://api2.production.example.com",
      "shadow_url": "https://api2.shadow.example.com"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8081/gates
```

---

### Requests

#### POST /gates/:gate_id/requests

**Purpose:** Create a captured request (called by mroki-agent)

**Path Parameters:**
- `gate_id` (UUID) - Parent gate identifier

**Request Body:**
```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "agent_id": "MacBook-Pro-a1b2c3d4-5678-90ab-cdef-1234567890ab",
  "method": "POST",
  "path": "/api/users",
  "headers": {
    "Content-Type": ["application/json"],
    "User-Agent": ["curl/7.68.0"]
  },
  "body": "{\"name\":\"Alice\",\"age\":30}",
  "created_at": "2026-01-31T20:00:00Z",
  "responses": [
    {
      "id": "8d0e7780-8536-51ef-a55c-f18fd2f91bf8",
      "type": "live",
      "status_code": 200,
      "headers": {
        "Content-Type": ["application/json"]
      },
      "body": "{\"id\":123,\"name\":\"Alice\",\"age\":30}",
      "created_at": "2026-01-31T20:00:01Z"
    },
    {
      "id": "9e1f8891-9647-62f0-b66d-027fe3f02cf9",
      "type": "shadow",
      "status_code": 200,
      "headers": {
        "Content-Type": ["application/json"]
      },
      "body": "{\"id\":456,\"name\":\"Alice\",\"age\":30}",
      "created_at": "2026-01-31T20:00:01Z"
    }
  ],
  "diff": {
    "content": "[{\"op\":\"replace\",\"path\":\"/id\",\"value\":456,\"oldValue\":123}]"
  }
}
```

**Field Descriptions:**
- `id` (optional) - Request UUID, generated if omitted
- `agent_id` (required) - Capturing agent identifier
- `method` (required) - HTTP method (GET, POST, etc.)
- `path` (required) - Request path
- `headers` (required) - Request headers
- `body` (required) - Request body (string)
- `created_at` (required) - Request timestamp
- `responses` (required) - Array of 2 responses (live + shadow)
  - `id` (optional) - Response UUID, generated if omitted
  - `type` (required) - "live" or "shadow"
  - `status_code` (required) - HTTP status code
  - `headers` (required) - Response headers
  - `body` (required) - Response body (string)
  - `created_at` (required) - Response timestamp
- `diff` (required) - Computed difference
  - `content` (required) - JSON diff in RFC 6902 format

**Response:**
- `201 Created` on success
- `400 Bad Request` if validation fails

**Success Response Body:**
```json
{
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "method": "POST",
    "path": "/api/users",
    "created_at": "2026-01-31T20:00:00Z"
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8081/gates/550e8400-e29b-41d4-a716-446655440000/requests \
  -H "Content-Type: application/json" \
  -d @captured_request.json
```

---

#### GET /gates/:gate_id/requests/:request_id

**Purpose:** Retrieve a specific captured request with full details

**Path Parameters:**
- `gate_id` (UUID) - Parent gate identifier
- `request_id` (UUID) - Request identifier

**Response:**
- `200 OK` on success
- `400 Bad Request` if IDs are invalid UUIDs
- `404 Not Found` if request or gate doesn't exist

**Success Response Body:**
```json
{
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "method": "POST",
    "path": "/api/users",
    "created_at": "2026-01-31T20:00:00Z",
    "responses": [
      {
        "id": "8d0e7780-8536-51ef-a55c-f18fd2f91bf8",
        "type": "live",
        "status_code": 200,
        "headers": {
          "Content-Type": ["application/json"]
        },
        "body": "{\"id\":123,\"name\":\"Alice\",\"age\":30}",
        "created_at": "2026-01-31T20:00:01Z"
      },
      {
        "id": "9e1f8891-9647-62f0-b66d-027fe3f02cf9",
        "type": "shadow",
        "status_code": 200,
        "headers": {
          "Content-Type": ["application/json"]
        },
        "body": "{\"id\":456,\"name\":\"Alice\",\"age\":30}",
        "created_at": "2026-01-31T20:00:01Z"
      }
    ],
    "diff": {
      "id": "a02g9902-a758-73g1-c77e-138gf4g13dga0",
      "content": "[{\"op\":\"replace\",\"path\":\"/id\",\"value\":456,\"oldValue\":123}]"
    }
  }
}
```

**Example:**
```bash
curl http://localhost:8081/gates/550e8400-e29b-41d4-a716-446655440000/requests/7c9e6679-7425-40de-944b-e07fc1f90ae7
```

---

#### GET /gates/:gate_id/requests

**Purpose:** List all captured requests for a gate

**Path Parameters:**
- `gate_id` (UUID) - Parent gate identifier

**Response:**
- `200 OK` on success
- `400 Bad Request` if gate_id is invalid UUID
- Returns empty array if no requests exist

**Success Response Body:**
```json
{
  "data": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "method": "POST",
      "path": "/api/users",
      "created_at": "2026-01-31T20:00:00Z"
    },
    {
      "id": "8d9f7890-8536-51ef-a55c-f18fd2f91bf9",
      "method": "GET",
      "path": "/api/users/123",
      "created_at": "2026-01-31T20:01:00Z"
    }
  ]
}
```

**Example:**
```bash
curl http://localhost:8081/gates/550e8400-e29b-41d4-a716-446655440000/requests
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "message": "brief error description",
    "details": "detailed error context (optional)"
  }
}
```

### HTTP Status Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data (validation failed)
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service not ready (health check failed)

### Common Error Scenarios

#### Invalid UUID Format
```json
{
  "error": {
    "message": "invalid gate ID",
    "details": "gate_id must be a valid UUID"
  }
}
```

#### Missing Required Field
```json
{
  "error": {
    "message": "missing required property",
    "details": "live_url is required"
  }
}
```

#### Resource Not Found
```json
{
  "error": {
    "message": "gate not found",
    "details": "no gate exists with id 550e8400-e29b-41d4-a716-446655440000"
  }
}
```

#### Invalid URL
```json
{
  "error": {
    "message": "invalid URL",
    "details": "live_url must use http or https scheme"
  }
}
```

---

## Data Types

### UUID Format

All IDs use UUID v4 format:

```
550e8400-e29b-41d4-a716-446655440000
```

### Timestamp Format

All timestamps use RFC3339 format with timezone:

```
2026-01-31T20:00:00Z
```

### Headers Format

Headers are represented as maps with string arrays (to support multiple values):

```json
{
  "Content-Type": ["application/json"],
  "X-Custom-Header": ["value1", "value2"]
}
```

### Diff Format

Diffs use JSON Patch (RFC 6902) format:

```json
[
  {
    "op": "replace",
    "path": "/id",
    "value": 456,
    "oldValue": 123
  },
  {
    "op": "add",
    "path": "/new_field",
    "value": "new_value"
  },
  {
    "op": "remove",
    "path": "/old_field",
    "oldValue": "old_value"
  }
]
```

**Operations:**
- `replace` - Field value changed
- `add` - Field added in shadow response
- `remove` - Field removed in shadow response

---

## Authentication

**Current:** No authentication required (v1)

**Planned (v2):** API key authentication via header:

```
Authorization: Bearer <api_key>
```

---

## Rate Limiting

**Current:** No rate limiting (v1)

**Planned (v2):**
- 1000 requests/minute per API key
- 429 Too Many Requests response when exceeded

---

## Pagination

**Current:** No pagination (v1) - all results returned

**Planned (v2):**

Query parameters:
- `limit` - Results per page (default: 50, max: 100)
- `offset` - Skip N results (default: 0)

Response includes pagination metadata:

```json
{
  "data": [...],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 250
  }
}
```

---

## CORS

**Current:** No CORS headers (v1)

**Planned (v2):** Configurable CORS for hub integration

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

---

## Example Workflows

### Create Gate and Capture Traffic

```bash
# 1. Create a gate
GATE_RESPONSE=$(curl -s -X POST http://localhost:8081/gates \
  -H "Content-Type: application/json" \
  -d '{
    "live_url": "https://httpbin.org/anything?service=live",
    "shadow_url": "https://httpbin.org/anything?service=shadow"
  }')

GATE_ID=$(echo $GATE_RESPONSE | jq -r '.data.id')
echo "Gate ID: $GATE_ID"

# 2. Configure agent with gate ID (see mroki-agent docs)

# 3. Send traffic through agent (agent will POST to API)

# 4. List captured requests
curl http://localhost:8081/gates/$GATE_ID/requests | jq .

# 5. Get specific request details
REQUEST_ID="7c9e6679-7425-40de-944b-e07fc1f90ae7"
curl http://localhost:8081/gates/$GATE_ID/requests/$REQUEST_ID | jq .
```

---

## Related Documentation

- [Architecture Overview](OVERVIEW.md)
- [mroki-api Component](../components/MROKI_API.md)
- [mroki-agent Component](../components/MROKI_AGENT.md)
