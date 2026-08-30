# mroki API Reference

This file is generated from the OpenAPI specification under `docs/api/openapi/`. **Do not edit it by hand** — run `make api-docs` to regenerate it.

## Base URL

The local development server is available at `http://localhost:8090`.

## Authentication

Every endpoint except the infrastructure endpoints (`/health/live`, `/health/ready`, `/metrics`) requires a bearer token supplied as `Authorization: Bearer <your-api-key>`.

## Response envelope

Successful responses wrap their payload in a `data` field. List endpoints add a `pagination` object (`limit`, `offset`, `total`, `has_more`).

## Errors

Errors are returned as `application/json` following [RFC 7807](https://www.rfc-editor.org/rfc/rfc7807) with the fields `type`, `title`, `status`, `detail`, and (for 4xx) `instance`. The `type` is a relative URI such as `/errors/not-found`, `/errors/invalid-request-body`, `/errors/invalid-query-param`, `/errors/unauthorized`, `/errors/conflict`, `/errors/rate-limit-exceeded`, or `/errors/internal-error`. See the `Problem` schema below.

## Table of Contents

HTTP Request | Description
-------------|------------
GET [/config](#getconfig) | Get server configuration
GET [/stats](#getstats) | Get global statistics
GET [/gates](#getgates) | List gates
POST [/gates](#postgates) | Create a gate
GET [/gates/{gate_id}](#getgatesgateid) | Get a gate
PATCH [/gates/{gate_id}](#patchgatesgateid) | Update a gate
DELETE [/gates/{gate_id}](#deletegatesgateid) | Delete a gate
GET [/gates/{gate_id}/requests](#getgatesgateidrequests) | List requests for a gate
POST [/gates/{gate_id}/requests](#postgatesgateidrequests) | Create a request
GET [/gates/{gate_id}/requests/{request_id}](#getgatesgateidrequestsrequestid) | Get a request
GET [/health/live](#gethealthlive) | Liveness probe
GET [/health/ready](#gethealthready) | Readiness probe
GET [/metrics](#getmetrics) | Prometheus metrics

## Config

### GET /config

Returns the read-only, server-wide configuration the hub needs.

### Responses

#### 200 Response

The server configuration.

```json
{
   "data": {
      "retention": "720h"
   }
}
```

#### Field Definitions

- `data` *(Config, required)* Read-only, server-wide settings the hub needs to render its UI. It is not
tied to any gate; it reflects the API's own configuration.


**Config**
- `retention` *(string, required)*: The global retention floor as a Go duration string (e.g. `720h`). Every
gate is pruned no sooner than this, and any per-gate override must be at
least this value.


#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

## Gates

### GET /gates

Returns a paginated list of gates with optional filtering and sorting.

#### Query Parameters

- `limit` *(integer)* Maximum number of items to return. Defaults to 50; capped at 100.

- `offset` *(integer)* Number of items to skip before the returned page. Defaults to 0.

- `name` *(string)* Filter gates whose name contains this substring.

- `live_url` *(string)* Filter gates whose live URL contains this substring.

- `shadow_url` *(string)* Filter gates whose shadow URL contains this substring.

- `sort` *(string)* Field to sort gates by. Enums: `id`, `name`, `live_url`, `shadow_url`, `created_at`

- `order` *(string)* Sort direction. Enums: `asc`, `desc`

### Responses

#### 200 Response

A paginated list of gates.

```json
{
   "data": [
      {
         "created_at": "2024-04-20T09:00:00Z",
         "diff_config": {
            "float_tolerance": 0.0001,
            "ignored_fields": [
               "/timestamp",
               "/request_id"
            ],
            "included_fields": [],
            "sort_arrays": false
         },
         "id": "9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
         "live_url": "https://live.example.com",
         "name": "users-service",
         "redacted_fields": [
            "/password",
            "/token"
         ],
         "retention": "168h",
         "shadow_url": "https://shadow.example.com",
         "stats": {
            "diff_count_24h": 6,
            "diff_rate": 0.025,
            "last_active": "2024-05-01T12:34:56Z",
            "request_count_24h": 240
         }
      }
   ],
   "pagination": {
      "has_more": true,
      "limit": 50,
      "offset": 0,
      "total": 128
   }
}
```

#### Field Definitions

- `data` *(array of Gate, required)* The gates in this page.
- `pagination` *(PaginationMeta, required)* Pagination metadata for list responses.

**Gate**
- `id` *(string, required)*: Unique gate identifier.
- `name` *(string, required)*: Human-readable gate name (unique across gates).
- `live_url` *(string, required)*: Base URL of the live (production) service.
- `shadow_url` *(string, required)*: Base URL of the shadow (candidate) service.
- `diff_config` *(DiffConfig, required)*: Per-gate diff computation settings.
- `redacted_fields` *(string array, required)*: JSON paths whose values are redacted before storage.
- `retention` *(string, required)*: Per-gate retention as a Go duration string (e.g. `168h`). An empty string
means the gate uses the global retention floor.

- `created_at` *(string, required)*: When the gate was created (RFC 3339).
- `stats` *(GateStats, required)*: Computed statistics for a single gate.

**DiffConfig**
- `ignored_fields` *(string array, required)*: JSON paths excluded from comparison.
- `included_fields` *(string array, required)*: JSON paths to compare exclusively. When non-empty, only these paths are
compared and everything else is ignored.

- `float_tolerance` *(number, required)*: Absolute tolerance applied when comparing floating-point numbers.
- `sort_arrays` *(boolean, required)*: Whether arrays are sorted before comparison so element order is ignored.

**GateStats**
- `request_count_24h` *(integer, required)*: Number of requests captured for this gate in the last 24 hours.
- `diff_count_24h` *(integer, required)*: Number of requests that produced a diff in the last 24 hours.
- `diff_rate` *(number, required)*: Fraction of requests that produced a diff in the last 24 hours (0.0–1.0).
- `last_active` *(string, required)*: Timestamp of the most recent captured request, or null if none.

**PaginationMeta**
- `limit` *(integer, required)*: Maximum number of items returned in this page.
- `offset` *(integer, required)*: Number of items skipped before this page.
- `total` *(integer, required)*: Total number of items matching the query.
- `has_more` *(boolean, required)*: Whether more items are available beyond this page.

#### 400 Response

The request was malformed (invalid query parameter).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

### POST /gates

Creates a new gate with live and shadow URLs.

### Request

```json
{
   "live_url": "https://live.example.com",
   "name": "users-service",
   "shadow_url": "https://shadow.example.com"
}
```

#### Field Definitions

- `name` *(string, required)* Human-readable gate name (must be unique across gates).
- `live_url` *(string, required)* Base URL of the live (production) service.
- `shadow_url` *(string, required)* Base URL of the shadow (candidate) service.

### Responses

#### 201 Response

The created gate.

```json
{
   "data": {
      "created_at": "2024-04-20T09:00:00Z",
      "diff_config": {
         "float_tolerance": 0.0001,
         "ignored_fields": [
            "/timestamp",
            "/request_id"
         ],
         "included_fields": [],
         "sort_arrays": false
      },
      "id": "9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
      "live_url": "https://live.example.com",
      "name": "users-service",
      "redacted_fields": [
         "/password",
         "/token"
      ],
      "retention": "168h",
      "shadow_url": "https://shadow.example.com",
      "stats": {
         "diff_count_24h": 6,
         "diff_rate": 0.025,
         "last_active": "2024-05-01T12:34:56Z",
         "request_count_24h": 240
      }
   }
}
```

#### Field Definitions

See [GateResponse](#gateresponse)

#### 400 Response

The request was malformed (invalid or missing body field).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 409 Response

A gate with the same name or live/shadow URL pair already exists.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

### GET /gates/{gate_id}

Returns a single gate by its identifier.

#### Path Parameters

- `gate_id` *(string, required)* The unique identifier of the gate.

### Responses

#### 200 Response

The requested gate.

```json
{
   "data": {
      "created_at": "2024-04-20T09:00:00Z",
      "diff_config": {
         "float_tolerance": 0.0001,
         "ignored_fields": [
            "/timestamp",
            "/request_id"
         ],
         "included_fields": [],
         "sort_arrays": false
      },
      "id": "9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
      "live_url": "https://live.example.com",
      "name": "users-service",
      "redacted_fields": [
         "/password",
         "/token"
      ],
      "retention": "168h",
      "shadow_url": "https://shadow.example.com",
      "stats": {
         "diff_count_24h": 6,
         "diff_rate": 0.025,
         "last_active": "2024-05-01T12:34:56Z",
         "request_count_24h": 240
      }
   }
}
```

#### Field Definitions

See [GateResponse](#gateresponse)

#### 400 Response

The request was malformed (invalid gate id).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 404 Response

No gate exists with the given id.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

### PATCH /gates/{gate_id}

Updates a gate. All fields are optional; omitted fields are left unchanged.


#### Path Parameters

- `gate_id` *(string, required)* The unique identifier of the gate.

### Request

```json
{
   "diff_config": {
      "float_tolerance": 0.0001,
      "ignored_fields": [
         "/timestamp",
         "/request_id"
      ],
      "included_fields": [],
      "sort_arrays": false
   },
   "name": "users-service-v2",
   "redacted_fields": [
      "/password"
   ],
   "retention": "168h"
}
```

#### Field Definitions

- `name` *(string)* New gate name.
- `diff_config` *(DiffConfig)* Per-gate diff computation settings.
- `redacted_fields` *(string array)* Replacement list of JSON paths to redact before storage.
- `retention` *(string)* Per-gate retention as a Go duration string. This field is tri-state:
omit it to leave the value unchanged; send `null` or an empty string to
reset to the global retention floor; send a duration (e.g. `168h`) to set
a custom value.


**DiffConfig**
- `ignored_fields` *(string array, required)*: JSON paths excluded from comparison.
- `included_fields` *(string array, required)*: JSON paths to compare exclusively. When non-empty, only these paths are
compared and everything else is ignored.

- `float_tolerance` *(number, required)*: Absolute tolerance applied when comparing floating-point numbers.
- `sort_arrays` *(boolean, required)*: Whether arrays are sorted before comparison so element order is ignored.

### Responses

#### 200 Response

The updated gate.

```json
{
   "data": {
      "created_at": "2024-04-20T09:00:00Z",
      "diff_config": {
         "float_tolerance": 0.0001,
         "ignored_fields": [
            "/timestamp",
            "/request_id"
         ],
         "included_fields": [],
         "sort_arrays": false
      },
      "id": "9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
      "live_url": "https://live.example.com",
      "name": "users-service",
      "redacted_fields": [
         "/password",
         "/token"
      ],
      "retention": "168h",
      "shadow_url": "https://shadow.example.com",
      "stats": {
         "diff_count_24h": 6,
         "diff_rate": 0.025,
         "last_active": "2024-05-01T12:34:56Z",
         "request_count_24h": 240
      }
   }
}
```

#### Field Definitions

See [GateResponse](#gateresponse)

#### 400 Response

The request was malformed (invalid body or gate id).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 404 Response

No gate exists with the given id.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 409 Response

The update conflicts with an existing gate (duplicate name or URL pair).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

### DELETE /gates/{gate_id}

Deletes a gate and all of its captured requests.

#### Path Parameters

- `gate_id` *(string, required)* The unique identifier of the gate.

### Responses

#### 204 Response

The gate was deleted.

#### 400 Response

The request was malformed (invalid gate id).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 404 Response

No gate exists with the given id.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

## Health

### GET /health/live

Always responds with `200 OK` while the process is running. Intended for
Kubernetes liveness probes. Unauthenticated; bypasses the API middleware.


### Responses

#### 200 Response

The process is alive.

### GET /health/ready

Checks database connectivity within a 1-second timeout. Returns `200 OK`
when reachable, or `503` with a diagnostic message otherwise. Intended for
Kubernetes readiness and startup probes. Unauthenticated; bypasses the API
middleware.


### Responses

#### 200 Response

The service is ready to accept traffic.

#### 503 Response

The service is not ready (database unreachable).

## Metrics

### GET /metrics

Exposes metrics in the Prometheus text exposition format. Only mounted when
metrics are enabled. Unauthenticated; bypasses the API middleware.


### Responses

#### 200 Response

Metrics in the Prometheus text exposition format.

## Requests

### GET /gates/{gate_id}/requests

Returns a paginated list of requests captured for a gate, with optional filtering and sorting.

#### Path Parameters

- `gate_id` *(string, required)* The unique identifier of the gate.

#### Query Parameters

- `limit` *(integer)* Maximum number of items to return. Defaults to 50; capped at 100.

- `offset` *(integer)* Number of items to skip before the returned page. Defaults to 0.

- `method` *(string)* Filter by HTTP method. Accepts a comma-separated list to match any of the
listed methods (e.g. `GET,POST`).


- `path` *(string)* Filter requests whose path contains this substring.

- `from` *(string)* Only include requests created at or after this timestamp (RFC 3339).

- `to` *(string)* Only include requests created at or before this timestamp (RFC 3339).

- `has_diff` *(boolean)* Filter by whether a request produced a diff.

- `sort` *(string)* Field to sort requests by. Enums: `created_at`, `method`, `path`

- `order` *(string)* Sort direction. Enums: `asc`, `desc`

### Responses

#### 200 Response

A paginated list of request summaries.

```json
{
   "data": [
      {
         "created_at": "2024-05-01T12:34:56Z",
         "has_diff": false,
         "id": "3f1c2d4e-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
         "live_response": {
            "latency_ms": 42,
            "status_code": 200
         },
         "method": "GET",
         "path": "/api/users/123",
         "raw_query": "page=2\u0026limit=10",
         "shadow_response": {
            "latency_ms": 42,
            "status_code": 200
         }
      }
   ],
   "pagination": {
      "has_more": true,
      "limit": 50,
      "offset": 0,
      "total": 128
   }
}
```

#### Field Definitions

- `data` *(array of Request, required)* The requests in this page.
- `pagination` *(PaginationMeta, required)* Pagination metadata for list responses.

**Request**
- `id` *(string, required)*: Unique request identifier.
- `method` *(string, required)*: HTTP method of the captured request.
- `path` *(string, required)*: Request path.
- `raw_query` *(string)*: Raw query string, omitted when empty.
- `created_at` *(string, required)*: When the request was captured (RFC 3339).
- `live_response` *(ResponseSummary, required)*: Lightweight response summary used in request listings.
- `shadow_response` *(ResponseSummary, required)*: Lightweight response summary used in request listings.
- `has_diff` *(boolean, required)*: Whether a diff was detected between the live and shadow responses.

**ResponseSummary**
- `status_code` *(integer, required)*: HTTP status code of the response.
- `latency_ms` *(integer, required)*: Response time in milliseconds.

**ResponseSummary**
- `status_code` *(integer, required)*: HTTP status code of the response.
- `latency_ms` *(integer, required)*: Response time in milliseconds.

**PaginationMeta**
- `limit` *(integer, required)*: Maximum number of items returned in this page.
- `offset` *(integer, required)*: Number of items skipped before this page.
- `total` *(integer, required)*: Total number of items matching the query.
- `has_more` *(boolean, required)*: Whether more items are available beyond this page.

#### 400 Response

The request was malformed (invalid query or path parameter).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 404 Response

No gate exists with the given id.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

### POST /gates/{gate_id}/requests

Submits a captured request with its live and shadow responses. If a diff is
not supplied, the API computes it server-side.


#### Path Parameters

- `gate_id` *(string, required)* The unique identifier of the gate.

### Request

```json
{
   "body": "",
   "created_at": "2024-05-01T12:34:56Z",
   "diff": {
      "content": [
         {
            "op": "replace",
            "path": "/data/name",
            "value": "new-value"
         }
      ]
   },
   "headers": {
      "Accept": [
         "application/json"
      ]
   },
   "id": "3f1c2d4e-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
   "live_response": {
      "body": "eyJvayI6dHJ1ZX0=",
      "created_at": "2024-05-01T12:34:56Z",
      "headers": {
         "Content-Type": [
            "application/json"
         ]
      },
      "id": "7a8b9c0d-1e2f-4a3b-9c8d-7e6f5a4b3c2d",
      "latency_ms": 42,
      "status_code": 200
   },
   "method": "GET",
   "path": "/api/users/123",
   "raw_query": "page=2\u0026limit=10",
   "shadow_response": {
      "body": "eyJvayI6dHJ1ZX0=",
      "created_at": "2024-05-01T12:34:56Z",
      "headers": {
         "Content-Type": [
            "application/json"
         ]
      },
      "id": "7a8b9c0d-1e2f-4a3b-9c8d-7e6f5a4b3c2d",
      "latency_ms": 42,
      "status_code": 200
   }
}
```

#### Field Definitions

- `id` *(string)* Optional request identifier. When omitted, the API uses the `X-Request-ID`
header if present, otherwise it generates one.

- `method` *(string, required)* HTTP method of the captured request.
- `path` *(string, required)* Request path.
- `raw_query` *(string)* Raw query string, omitted when empty.
- `headers` *(object, required)* Request headers as a map of header name to a list of values.
- `body` *(string, required)* Base64-encoded request body.
- `created_at` *(string, required)* When the request was captured (RFC 3339).
- `live_response` *(ResponsePayload, required)* A single HTTP response submitted by a proxy.
- `shadow_response` *(ResponsePayload, required)* A single HTTP response submitted by a proxy.
- `diff` *(DiffPayload)* The computed difference between two responses.

**ResponsePayload**
- `id` *(string)*: Optional response identifier; generated by the API when omitted.
- `status_code` *(integer, required)*: HTTP status code of the response.
- `headers` *(object, required)*: Response headers as a map of header name to a list of values.
- `body` *(string, required)*: Base64-encoded response body.
- `latency_ms` *(integer, required)*: Response time in milliseconds.
- `created_at` *(string, required)*: When the response was captured (RFC 3339).

**ResponsePayload**
- `id` *(string)*: Optional response identifier; generated by the API when omitted.
- `status_code` *(integer, required)*: HTTP status code of the response.
- `headers` *(object, required)*: Response headers as a map of header name to a list of values.
- `body` *(string, required)*: Base64-encoded response body.
- `latency_ms` *(integer, required)*: Response time in milliseconds.
- `created_at` *(string, required)*: When the response was captured (RFC 3339).

**DiffPayload**
- `content` *(array of PatchOp, required)*: RFC 6902 JSON Patch operations.

**PatchOp**
- `op` *(string, required)*: The JSON Patch operation.. Enums: `add`, `remove`, `replace`
- `path` *(string, required)*: JSON Pointer (RFC 6901) to the location that differs.
- `value`: The value associated with the operation. Present for `add` and `replace`
operations; omitted for `remove`. May be any JSON type.


### Responses

#### 201 Response

The created request summary.

```json
{
   "data": {
      "created_at": "2024-05-01T12:34:56Z",
      "has_diff": false,
      "id": "3f1c2d4e-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
      "live_response": {
         "latency_ms": 42,
         "status_code": 200
      },
      "method": "GET",
      "path": "/api/users/123",
      "raw_query": "page=2\u0026limit=10",
      "shadow_response": {
         "latency_ms": 42,
         "status_code": 200
      }
   }
}
```

#### Field Definitions

- `data` *(Request, required)* Summary of a captured request, used in listings.

**Request**
- `id` *(string, required)*: Unique request identifier.
- `method` *(string, required)*: HTTP method of the captured request.
- `path` *(string, required)*: Request path.
- `raw_query` *(string)*: Raw query string, omitted when empty.
- `created_at` *(string, required)*: When the request was captured (RFC 3339).
- `live_response` *(ResponseSummary, required)*: Lightweight response summary used in request listings.
- `shadow_response` *(ResponseSummary, required)*: Lightweight response summary used in request listings.
- `has_diff` *(boolean, required)*: Whether a diff was detected between the live and shadow responses.

**ResponseSummary**
- `status_code` *(integer, required)*: HTTP status code of the response.
- `latency_ms` *(integer, required)*: Response time in milliseconds.

**ResponseSummary**
- `status_code` *(integer, required)*: HTTP status code of the response.
- `latency_ms` *(integer, required)*: Response time in milliseconds.

#### 400 Response

The request was malformed (invalid or missing body field).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 404 Response

No gate exists with the given id.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

### GET /gates/{gate_id}/requests/{request_id}

Returns a single captured request with full responses and diff.

#### Path Parameters

- `gate_id` *(string, required)* The unique identifier of the gate.

- `request_id` *(string, required)* The unique identifier of the request.

### Responses

#### 200 Response

The requested request detail.

```json
{
   "data": {
      "created_at": "2024-05-01T12:34:56Z",
      "diff": {
         "config": {
            "float_tolerance": 0.0001,
            "ignored_fields": [
               "/timestamp",
               "/request_id"
            ],
            "included_fields": [],
            "sort_arrays": false
         },
         "content": [
            {
               "op": "replace",
               "path": "/data/name",
               "value": "new-value"
            }
         ]
      },
      "headers": {
         "Accept": [
            "application/json"
         ]
      },
      "id": "3f1c2d4e-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
      "live_response": {
         "body": "eyJvayI6dHJ1ZX0=",
         "created_at": "2024-05-01T12:34:56Z",
         "headers": {
            "Content-Type": [
               "application/json"
            ]
         },
         "id": "7a8b9c0d-1e2f-4a3b-9c8d-7e6f5a4b3c2d",
         "latency_ms": 42,
         "status_code": 200
      },
      "method": "GET",
      "path": "/api/users/123",
      "raw_query": "page=2\u0026limit=10",
      "shadow_response": {
         "body": "eyJvayI6dHJ1ZX0=",
         "created_at": "2024-05-01T12:34:56Z",
         "headers": {
            "Content-Type": [
               "application/json"
            ]
         },
         "id": "7a8b9c0d-1e2f-4a3b-9c8d-7e6f5a4b3c2d",
         "latency_ms": 42,
         "status_code": 200
      }
   }
}
```

#### Field Definitions

- `data` *(RequestDetail, required)* A complete request with full responses and diff.

**RequestDetail**
- `id` *(string, required)*: Unique request identifier.
- `method` *(string, required)*: HTTP method of the captured request.
- `path` *(string, required)*: Request path.
- `raw_query` *(string)*: Raw query string, omitted when empty.
- `headers` *(object, required)*: Request headers as a map of header name to a list of values.
- `body` *(string, required)*: Base64-encoded request body, or null when absent.
- `created_at` *(string, required)*: When the request was captured (RFC 3339).
- `live_response` *(ResponseDetail, required)*: A response with full details, used in the request detail view.
- `shadow_response` *(ResponseDetail, required)*: A response with full details, used in the request detail view.
- `diff` *(DiffDetail, required)*: Diff content and the diff config snapshot used to compute it.

**ResponseDetail**
- `id` *(string, required)*: Unique response identifier.
- `status_code` *(integer, required)*: HTTP status code of the response.
- `headers` *(object, required)*: Response headers as a map of header name to a list of values.
- `body` *(string, required)*: Base64-encoded response body, or null when absent.
- `latency_ms` *(integer, required)*: Response time in milliseconds.
- `created_at` *(string, required)*: When the response was captured (RFC 3339).

**ResponseDetail**
- `id` *(string, required)*: Unique response identifier.
- `status_code` *(integer, required)*: HTTP status code of the response.
- `headers` *(object, required)*: Response headers as a map of header name to a list of values.
- `body` *(string, required)*: Base64-encoded response body, or null when absent.
- `latency_ms` *(integer, required)*: Response time in milliseconds.
- `created_at` *(string, required)*: When the response was captured (RFC 3339).

**DiffDetail**
- `content` *(array of PatchOp, required)*: RFC 6902 JSON Patch operations describing the differences.
- `config` *(DiffConfig, required)*: Per-gate diff computation settings.

**PatchOp**
- `op` *(string, required)*: The JSON Patch operation.. Enums: `add`, `remove`, `replace`
- `path` *(string, required)*: JSON Pointer (RFC 6901) to the location that differs.
- `value`: The value associated with the operation. Present for `add` and `replace`
operations; omitted for `remove`. May be any JSON type.


**DiffConfig**
- `ignored_fields` *(string array, required)*: JSON paths excluded from comparison.
- `included_fields` *(string array, required)*: JSON paths to compare exclusively. When non-empty, only these paths are
compared and everything else is ignored.

- `float_tolerance` *(number, required)*: Absolute tolerance applied when comparing floating-point numbers.
- `sort_arrays` *(boolean, required)*: Whether arrays are sorted before comparison so element order is ignored.

#### 400 Response

The request was malformed (invalid gate or request id).

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 404 Response

No request exists with the given id for this gate.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

## Stats

### GET /stats

Returns cross-gate aggregate statistics.

### Responses

#### 200 Response

The global statistics.

```json
{
   "data": {
      "total_diff_rate": 0.042,
      "total_gates": 12,
      "total_requests_24h": 3480
   }
}
```

#### Field Definitions

- `data` *(GlobalStats, required)* Cross-gate aggregate statistics.

**GlobalStats**
- `total_gates` *(integer, required)*: Total number of gates.
- `total_requests_24h` *(integer, required)*: Total number of requests captured across all gates in the last 24 hours.
- `total_diff_rate` *(number, required)*: Fraction of requests in the last 24 hours that produced a diff (0.0–1.0).

#### 401 Response

Authentication failed because the bearer token was missing or invalid.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 429 Response

The client has exceeded the rate limit.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

#### 500 Response

An unexpected error occurred while processing the request.

```json
{
   "detail": "no gate exists with the given id",
   "instance": "/gates/9b2e6f0e-6b7a-4c1d-8f3a-2a1b0c9d8e7f",
   "status": 404,
   "title": "Not Found",
   "type": "/errors/not-found"
}
```

## Shared Schema Definitions

### GateResponse

Used in: GET /gates/{gate_id}, PATCH /gates/{gate_id}, POST /gates

- `data` *(Gate, required)* A traffic-testing gate with live and shadow URLs.

**Gate**
- `id` *(string, required)*: Unique gate identifier.
- `name` *(string, required)*: Human-readable gate name (unique across gates).
- `live_url` *(string, required)*: Base URL of the live (production) service.
- `shadow_url` *(string, required)*: Base URL of the shadow (candidate) service.
- `diff_config` *(DiffConfig, required)*: Per-gate diff computation settings.
- `redacted_fields` *(string array, required)*: JSON paths whose values are redacted before storage.
- `retention` *(string, required)*: Per-gate retention as a Go duration string (e.g. `168h`). An empty string
means the gate uses the global retention floor.

- `created_at` *(string, required)*: When the gate was created (RFC 3339).
- `stats` *(GateStats, required)*: Computed statistics for a single gate.

**DiffConfig**
- `ignored_fields` *(string array, required)*: JSON paths excluded from comparison.
- `included_fields` *(string array, required)*: JSON paths to compare exclusively. When non-empty, only these paths are
compared and everything else is ignored.

- `float_tolerance` *(number, required)*: Absolute tolerance applied when comparing floating-point numbers.
- `sort_arrays` *(boolean, required)*: Whether arrays are sorted before comparison so element order is ignored.

**GateStats**
- `request_count_24h` *(integer, required)*: Number of requests captured for this gate in the last 24 hours.
- `diff_count_24h` *(integer, required)*: Number of requests that produced a diff in the last 24 hours.
- `diff_rate` *(number, required)*: Fraction of requests that produced a diff in the last 24 hours (0.0–1.0).
- `last_active` *(string, required)*: Timestamp of the most recent captured request, or null if none.

### Problem

Used in: DELETE /gates/{gate_id}, GET /config, GET /gates, GET /gates/{gate_id}, GET /gates/{gate_id}/requests, GET /gates/{gate_id}/requests/{request_id}, GET /stats, PATCH /gates/{gate_id}, POST /gates, POST /gates/{gate_id}/requests

- `type` *(string, required)* A relative URI reference identifying the error type.
- `title` *(string, required)* A short, human-readable summary of the error type.
- `status` *(integer, required)* The HTTP status code, matching the response status.
- `detail` *(string)* A human-readable explanation specific to this occurrence.
- `instance` *(string)* A relative URI reference identifying the specific request. Populated for
4xx errors only.


