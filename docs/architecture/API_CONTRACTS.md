# API Contracts

This document specifies the REST API provided by `mroki-api`, including endpoints, request/response formats, and error handling.

## Base URL

```
http://localhost:8090
```

## Response Format

All successful API responses follow this structure:

```json
{
  "data": <response_data>
}
```

All error responses follow RFC 7807 (Problem Details for HTTP APIs):

```json
{
  "type": "/errors/invalid-request-body",
  "title": "Invalid Request Body",
  "status": 400,
  "detail": "live_url is required",
  "instance": "/gates"
}
```

**RFC 7807 Error Fields:**
- `type` - URI identifying the error type (relative path)
- `title` - Short, human-readable summary of the error type
- `status` - HTTP status code (matches response status)
- `detail` - Human-readable explanation specific to this occurrence
- `instance` - URI reference identifying the specific request (populated for 4xx errors only)

---

## Authentication

All API endpoints (except health checks) require bearer token authentication.

### Authorization Header Format

```
Authorization: Bearer <your-api-key>
```

### Authentication Errors

**Missing Authorization Header:**
```json
{
  "type": "/errors/unauthorized",
  "title": "Missing Authorization Header",
  "status": 401,
  "detail": "Authorization header is required",
  "instance": "/gates"
}
```

**Invalid Authorization Format:**
```json
{
  "type": "/errors/unauthorized",
  "title": "Invalid Authorization Format",
  "status": 401,
  "detail": "Authorization header must use format: Bearer <token>",
  "instance": "/gates"
}
```

**Invalid API Key:**
```json
{
  "type": "/errors/unauthorized",
  "title": "Invalid API Key",
  "status": 401,
  "detail": "The provided API key is not valid",
  "instance": "/gates"
}
```

### Excluded Endpoints

The following endpoints do not require authentication:
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe

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
curl http://localhost:8090/health/live
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
curl http://localhost:8090/health/ready
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
  "type": "/errors/invalid-request-body",
  "title": "Invalid Gate URL",
  "status": 400,
  "detail": "live_url and shadow_url must use http or https scheme",
  "instance": "/gates"
}
```

**Example:**
```bash
curl -X POST http://localhost:8090/gates \
  -H "Authorization: Bearer your-api-key" \
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
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/550e8400-e29b-41d4-a716-446655440000
```

---

#### GET /gates

**Purpose:** List all gates with optional filtering and sorting

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 50 | Results per page (max: 100) |
| `offset` | integer | 0 | Number of results to skip |
| `live_url` | string | — | Filter by live URL substring (case-insensitive) |
| `shadow_url` | string | — | Filter by shadow URL substring (case-insensitive) |
| `sort` | string | `id` | Sort field: `id`, `live_url`, or `shadow_url` |
| `order` | string | `desc` | Sort direction: `asc` or `desc` |

**Response:**
- `200 OK` on success
- `400 Bad Request` if query parameters are invalid
- Returns empty array if no gates match

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
  ],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 2,
    "has_more": false
  }
}
```

**Examples:**
```bash
# List all gates (default: 50 per page, sorted by id desc)
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates

# Filter by live URL containing "production"
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?live_url=production"

# Filter by shadow URL and sort by live_url ascending
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?shadow_url=staging&sort=live_url&order=asc"

# Paginate with limit and offset
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates?limit=10&offset=20&sort=shadow_url&order=desc"
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
    "content": [
      {
        "op": "replace",
        "path": "/id",
        "value": 456
      }
    ]
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
- `diff` (required) - Computed difference (value object, no ID)
  - `content` (required) - Array of RFC 6902 JSON Patch operations (empty array `[]` when no differences)

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
curl -X POST http://localhost:8090/gates/550e8400-e29b-41d4-a716-446655440000/requests \
  -H "Authorization: Bearer your-api-key" \
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

**Example:**
```bash
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/550e8400-e29b-41d4-a716-446655440000/requests/7c9e6679-7425-40de-944b-e07fc1f90ae7
```

---

#### GET /gates/:gate_id/requests

**Purpose:** List captured requests for a gate with optional filtering and sorting

**Path Parameters:**
- `gate_id` (UUID) - Parent gate identifier

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 50 | Results per page (max: 100) |
| `offset` | integer | 0 | Number of results to skip |
| `method` | string | — | Filter by HTTP method(s), comma-separated (e.g., `GET,POST`) |
| `path` | string | — | Filter by path pattern, supports wildcards (e.g., `/api/users/*`) |
| `from` | RFC3339 | — | Filter requests created on or after this timestamp |
| `to` | RFC3339 | — | Filter requests created on or before this timestamp |
| `agent_id` | string | — | Filter by agent ID (exact match) |
| `has_diff` | boolean | — | Filter by diff existence (`true` = only with diffs, `false` = only without) |
| `sort` | string | `created_at` | Sort field: `created_at`, `method`, or `path` |
| `order` | string | `desc` | Sort direction: `asc` or `desc` |

**Response:**
- `200 OK` on success
- `400 Bad Request` if gate_id is invalid UUID or query parameters are invalid
- Returns empty array if no requests match

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
  ],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 2,
    "has_more": false
  }
}
```

**Examples:**
```bash
# List all requests (default: newest first, 50 per page)
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/$GATE_ID/requests

# Filter by method and path pattern
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates/$GATE_ID/requests?method=GET,POST&path=/api/users/*"

# Filter by date range and only requests with diffs
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates/$GATE_ID/requests?from=2026-01-31T00:00:00Z&to=2026-02-01T00:00:00Z&has_diff=true"

# Sort by path ascending, second page
curl -H "Authorization: Bearer your-api-key" \
  "http://localhost:8090/gates/$GATE_ID/requests?sort=path&order=asc&limit=20&offset=20"
```

---

## Error Handling

### Error Response Format

All errors follow RFC 7807 (Problem Details for HTTP APIs):

```json
{
  "type": "/errors/not-found",
  "title": "Gate Not Found",
  "status": 404,
  "detail": "gate with id \"550e8400-e29b-41d4-a716-446655440000\" does not exist",
  "instance": "/gates/550e8400-e29b-41d4-a716-446655440000"
}
```

**Field Descriptions:**
- `type` - URI reference identifying the problem type (e.g., `/errors/invalid-request-body`)
- `title` - Short, human-readable summary that remains consistent for this error type
- `status` - HTTP status code (always matches the response status header)
- `detail` - Human-readable explanation specific to this occurrence of the error
- `instance` - URI reference to the specific resource/request (auto-populated for 4xx errors, empty for 5xx)

### Error Types

The API uses the following generic error types:

| Type | Title | Status | Usage |
|------|-------|--------|-------|
| `/errors/invalid-request-body` | Invalid Request Body | 400 | Malformed JSON, validation failures, invalid IDs/URLs |
| `/errors/missing-body-field` | Missing Required Field | 400 | Required body field is missing |
| `/errors/missing-path-param` | Missing Path Parameter | 400 | Required path parameter is missing |
| `/errors/missing-query-param` | Missing Query Parameter | 400 | Required query parameter is missing |
| `/errors/invalid-query-param` | Invalid Query Parameter | 400 | Invalid pagination, filters, or sort parameters |
| `/errors/missing-header` | Missing Required Header | 400 | Required header is missing |
| `/errors/unauthorized` | Unauthorized | 401 | Missing or invalid API key |
| `/errors/not-found` | Resource Not Found | 404 | Gate or Request doesn't exist |
| `/errors/rate-limit-exceeded` | Rate Limit Exceeded | 429 | Too many requests from this IP |
| `/errors/internal-error` | Internal Server Error | 500 | Unexpected server errors |

### HTTP Status Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data (validation failed)
- `401 Unauthorized` - Missing or invalid API key
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service not ready (health check failed)

### Common Error Scenarios

#### Invalid UUID Format
```json
{
  "type": "/errors/invalid-request-body",
  "title": "Invalid Gate ID",
  "status": 400,
  "detail": "gate_id must be a valid UUID, got \"not-a-uuid\"",
  "instance": "/gates/not-a-uuid"
}
```

#### Missing Required Field
```json
{
  "type": "/errors/missing-body-field",
  "title": "Missing Required Field",
  "status": 400,
  "detail": "live_url is required",
  "instance": "/gates"
}
```

#### Resource Not Found
```json
{
  "type": "/errors/not-found",
  "title": "Gate Not Found",
  "status": 404,
  "detail": "gate with id \"550e8400-e29b-41d4-a716-446655440000\" does not exist",
  "instance": "/gates/550e8400-e29b-41d4-a716-446655440000"
}
```

#### Invalid URL
```json
{
  "type": "/errors/invalid-request-body",
  "title": "Invalid Gate URL",
  "status": 400,
  "detail": "live_url and shadow_url must use http or https scheme: invalid gate URL: scheme must be http or https, got \"ftp\"",
  "instance": "/gates"
}
```

#### Invalid Pagination
```json
{
  "type": "/errors/invalid-query-param",
  "title": "Invalid Query Parameter",
  "status": 400,
  "detail": "limit and offset must be non-negative integers: invalid pagination parameters: limit must be non-negative, got -10",
  "instance": "/gates"
}
```

#### Internal Server Error
```json
{
  "type": "/errors/internal-error",
  "title": "Internal Server Error",
  "status": 500,
  "detail": "An unknown error occurred. Please try again later."
}
```
**Note:** 5xx errors do NOT include the `instance` field for security reasons.

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

Diffs use RFC 6902 JSON Patch format. The `content` field is an array of patch operations describing the differences between the live and shadow responses.

Each operation has the following structure:

```json
{
  "op": "replace",
  "path": "/id",
  "value": 456
}
```

**Operation types:**
- `add` — A field exists in the shadow response but not in the live response
- `remove` — A field exists in the live response but not in the shadow response
- `replace` — A field exists in both but has a different value

**Fields:**
- `op` (string, required) — The operation type
- `path` (string, required) — JSON Pointer (RFC 6901) to the affected field
- `value` (any, optional) — The shadow response value (omitted for `remove` operations)

**Example (no differences):**
```json
[]
```

**Example (with differences):**
```json
[
  {
    "op": "replace",
    "path": "/id",
    "value": 456
  },
  {
    "op": "add",
    "path": "/new_field",
    "value": "some_value"
  },
  {
    "op": "remove",
    "path": "/old_field"
  }
]
```

---

<!-- Authentication details are documented in the Authentication section above -->

---

## Rate Limiting

**Status:** ✅ Implemented

Per-IP rate limiting using token bucket algorithm.

- Default: 1000 requests per minute per IP
- Configurable via `MROKI_API_RATE_LIMIT` (or `MROKI_APP_RATE_LIMIT`)
- Supports `X-Forwarded-For` header for proxy-aware IP extraction
- Returns `429 Too Many Requests` with `Retry-After` header when exceeded

**Rate limit error response:**
```json
{
  "type": "/errors/rate-limit-exceeded",
  "title": "Rate Limit Exceeded",
  "status": 429,
  "detail": "Rate limit exceeded. Try again in 3 seconds.",
  "instance": "/gates"
}
```

---

## Pagination

**Status:** ✅ Implemented

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
    "total": 250,
    "has_more": true
  }
}
```

**Paginated endpoints:**
- `GET /gates`
- `GET /gates/:gate_id/requests`

---

## CORS

**Status:** ✅ Implemented

CORS is configurable via the `MROKI_APP_CORS_ORIGINS` environment variable. When configured, the following headers are set using the `rs/cors` library:

```
Access-Control-Allow-Origin: <configured origin>
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 86400
```

**Configuration:**
```bash
# Comma-separated list of allowed origins (empty = CORS disabled)
MROKI_APP_CORS_ORIGINS=http://localhost:5173,https://hub.example.com
```

---

## Example Workflows

### Create Gate and Capture Traffic

```bash
# 1. Create a gate
GATE_RESPONSE=$(curl -s -X POST http://localhost:8090/gates \
  -H "Authorization: Bearer your-api-key" \
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
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/$GATE_ID/requests | jq .

# 5. Get specific request details
REQUEST_ID="7c9e6679-7425-40de-944b-e07fc1f90ae7"
curl -H "Authorization: Bearer your-api-key" \
  http://localhost:8090/gates/$GATE_ID/requests/$REQUEST_ID | jq .
```

---

## Related Documentation

- [Architecture Overview](OVERVIEW.md)
- [mroki-api Component](../components/MROKI_API.md)
- [mroki-agent Component](../components/MROKI_AGENT.md)
