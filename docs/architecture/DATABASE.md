# Database Schema

This document describes the PostgreSQL database schema used by mroki-api.

## Overview

The database stores four main entities:
- **Gates** - Live/shadow service pairs
- **Requests** - Captured HTTP requests
- **Responses** - HTTP responses (live and shadow)
- **Diffs** - Computed differences between responses

## Entity Relationship Diagram

```
┌──────────┐
│  gates   │
└────┬─────┘
     │ 1:N
     ↓
┌──────────┐
│ requests │
└────┬─────┘
     │ 1:N
     ↓
┌───────────┐     ┌────────┐
│ responses │────▶│ diffs  │
└───────────┘ N:1 └────────┘
```

## Tables

### gates

Stores live/shadow service pairs.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | UUID | NOT NULL | Primary key, generated UUID v4, immutable |
| `name` | TEXT | NOT NULL | Human-readable gate name, unique, mutable |
| `live_url` | TEXT | NOT NULL | URL of production service, immutable |
| `shadow_url` | TEXT | NOT NULL | URL of shadow/experimental service, immutable |
| `created_at` | TIMESTAMPTZ | NOT NULL | Creation timestamp, immutable, default `NOW()` |

**Indexes:**
- PRIMARY KEY on `id`
- UNIQUE on `name`
- UNIQUE on `(live_url, shadow_url)` (composite — the pair must be unique, individual values may repeat)

**Constraints:**
- `name` is unique and mutable (can be renamed)
- `live_url` and `shadow_url` are immutable after creation
- `created_at` is immutable, set at insert time

**Example:**
```sql
INSERT INTO gates (id, name, live_url, shadow_url, created_at) VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'checkout-api',
    'https://api.production.example.com',
    'https://api.shadow.example.com',
    NOW()
);
```

---

### requests

Stores captured HTTP requests.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | UUID | NOT NULL | Primary key, generated UUID v4 |
| `gate_id` | UUID | NOT NULL | Foreign key to gates.id |
| `method` | TEXT | YES | HTTP method (GET, POST, PUT, etc.) |
| `path` | TEXT | YES | Request path (e.g., "/api/users") |
| `headers` | JSONB | YES | HTTP headers as JSON object |
| `body` | BYTEA | YES | Request body (binary) |
| `created_at` | TIMESTAMPTZ | YES | Timestamp when request was captured |

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `gate_id` (for fast gate-based queries)

**Foreign Keys:**
- `gate_id` REFERENCES `gates(id)` ON DELETE CASCADE

**Example:**
```sql
INSERT INTO requests (
    id, gate_id, method, path,
    headers, body, created_at
) VALUES (
    '7c9e6679-7425-40de-944b-e07fc1f90ae7',
    '550e8400-e29b-41d4-a716-446655440000',
    'MacBook-Pro-a1b2c3d4',
    'POST',
    '/api/users',
    '{"Content-Type": ["application/json"]}'::jsonb,
    '{"name":"Alice","age":30}',
    NOW()
);
```

---

### responses

Stores HTTP responses from live and shadow services.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | UUID | NOT NULL | Primary key, generated UUID v4 |
| `request_id` | UUID | NOT NULL | Foreign key to requests.id |
| `type` | TEXT | YES | Response type: "live" or "shadow" |
| `status_code` | INTEGER | YES | HTTP status code (200, 404, etc.) |
| `headers` | JSONB | YES | HTTP headers as JSON object |
| `body` | BYTEA | YES | Response body (binary) |
| `created_at` | TIMESTAMPTZ | YES | Timestamp when response was received |

**Indexes:**
- PRIMARY KEY on `id`
- INDEX on `request_id` (for fast request-based queries)

**Foreign Keys:**
- `request_id` REFERENCES `requests(id)` ON DELETE CASCADE

**Example:**
```sql
-- Live response
INSERT INTO responses (
    id, request_id, type, status_code,
    headers, body, created_at
) VALUES (
    '8d0e7780-8536-51ef-a55c-f18fd2f91bf8',
    '7c9e6679-7425-40de-944b-e07fc1f90ae7',
    'live',
    200,
    '{"Content-Type": ["application/json"]}'::jsonb,
    '{"id":123,"name":"Alice","age":30}',
    NOW()
);

-- Shadow response
INSERT INTO responses (
    id, request_id, type, status_code,
    headers, body, created_at
) VALUES (
    '9e1f8891-9647-62f0-b66d-027fe3f02cf9',
    '7c9e6679-7425-40de-944b-e07fc1f90ae7',
    'shadow',
    200,
    '{"Content-Type": ["application/json"]}'::jsonb,
    '{"id":456,"name":"Alice","age":30}',
    NOW()
);
```

---

### diffs

Stores computed differences between live and shadow responses.

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `request_id` | UUID | NOT NULL | Primary key & foreign key to requests.id (1:1 relationship) |
| `from_response_id` | UUID | NOT NULL | Foreign key to responses.id (live response) |
| `to_response_id` | UUID | NOT NULL | Foreign key to responses.id (shadow response) |
| `content` | TEXT | NOT NULL | Diff content (JSON Patch format, always saved even if empty) |

**Indexes:**
- PRIMARY KEY on `request_id` (enforces 1:1 relationship with requests)

**Foreign Keys:**
- `request_id` REFERENCES `requests(id)` ON DELETE CASCADE
- `from_response_id` REFERENCES `responses(id)` ON DELETE CASCADE
- `to_response_id` REFERENCES `responses(id)` ON DELETE CASCADE

**Example:**
```sql
INSERT INTO diffs (
    request_id, from_response_id, to_response_id, content
) VALUES (
    '7c9e6679-7425-40de-944b-e07fc1f90ae7',
    '8d0e7780-8536-51ef-a55c-f18fd2f91bf8',
    '9e1f8891-9647-62f0-b66d-027fe3f02cf9',
    '[{"op":"replace","path":"/id","value":456,"oldValue":123}]'
);
```

---

## Data Types

### UUID

All IDs use PostgreSQL UUID type (128-bit, stored as 16 bytes).

**Format:** `550e8400-e29b-41d4-a716-446655440000`

**Generation:** UUID v4 (random) via Go's `google/uuid` library

### TIMESTAMPTZ

All timestamps use `TIMESTAMPTZ` (timestamp with timezone).

**Format:** ISO 8601 with timezone (e.g., `2026-01-31T20:00:00Z`)

**Storage:** Internally stored as UTC, displayed in local timezone

### JSONB

Headers are stored as JSONB for efficient querying.

**Format:**
```json
{
  "Content-Type": ["application/json"],
  "User-Agent": ["mroki-agent/1.0"]
}
```

**Benefits:**
- Binary storage (more efficient than JSON text)
- Supports indexing and querying
- Allows header arrays (multiple values per key)

### BYTEA and TEXT

Request/response bodies are stored as BYTEA (binary), while diff content is stored as TEXT.

**Request/Response Bodies (BYTEA):**
- Handles any content type (JSON, XML, binary, etc.)
- Preserves exact byte representation
- No character encoding issues
- Go: `[]byte` ↔ PostgreSQL BYTEA
- API: Base64 encoded in JSON responses

**Diff Content (TEXT):**
- Stores JSON Patch format (always valid UTF-8)
- Human-readable in database queries
- Better debuggability and inspection
- Go: `string` ↔ PostgreSQL TEXT

---

## Relationships

### Cascade Deletes

All relationships use `ON DELETE CASCADE`:

```
Delete Gate
  ↓
  Delete all Requests for that Gate
    ↓
    Delete all Responses for those Requests
      ↓
      Delete all Diffs for those Responses
```

**Example:**
```sql
-- Delete a gate (cascades to all child records)
DELETE FROM gates WHERE id = '550e8400-e29b-41d4-a716-446655440000';
```

### One-to-Many Relationships

- **Gate → Requests**: One gate has many requests (1:N)
- **Request → Responses**: One request has 2 responses (live + shadow)
- **Request → Diffs**: One request has one diff (1:1)

---

## Indexes

### Performance Indexes

```sql
-- Gate unique constraints
CREATE UNIQUE INDEX gate_name ON gates(name);
CREATE UNIQUE INDEX gate_live_url_shadow_url ON gates(live_url, shadow_url);

-- Find all requests for a gate (frequent query)
CREATE INDEX idx_requests_gate_id ON requests(gate_id);

-- Find all responses for a request (frequent query)
CREATE INDEX idx_responses_request_id ON responses(request_id);

```

### Index Usage

**Query 1: List requests for a gate**
```sql
SELECT * FROM requests 
WHERE gate_id = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY created_at DESC
LIMIT 50;
-- Uses: idx_requests_gate_id
```

**Query 2: Get request with responses**
```sql
SELECT r.*, resp.* 
FROM requests r
JOIN responses resp ON resp.request_id = r.id
WHERE r.id = '7c9e6679-7425-40de-944b-e07fc1f90ae7';
-- Uses: idx_responses_request_id
```

---

## Schema Migrations

### Current State

Schema is managed by ent with versioned migrations via Atlas.

**Schema Directory:** `ent/schema/`
**Generated Code:** `ent/` (via `go generate ./ent/...`)
**Migrations Directory:** `ent/migrate/migrations/`

To generate a new migration after schema changes:
```bash
make api-migrate name=<migration_name>
```

---

## Querying Examples

### Get all gates

```sql
SELECT id, name, live_url, shadow_url, created_at
FROM gates
ORDER BY created_at DESC;
```

### Get requests with diff summary

```sql
SELECT 
    r.id,
    r.method,
    r.path,
    r.created_at,
    d.content IS NOT NULL as has_diff
FROM requests r
LEFT JOIN diffs d ON d.request_id = r.id
WHERE r.gate_id = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY r.created_at DESC
LIMIT 50;
```

### Get request with full details

```sql
SELECT 
    r.*,
    json_agg(
        json_build_object(
            'id', resp.id,
            'type', resp.type,
            'status_code', resp.status_code,
            'body', resp.body
        )
    ) as responses,
    d.content as diff
FROM requests r
LEFT JOIN responses resp ON resp.request_id = r.id
LEFT JOIN diffs d ON d.request_id = r.id
WHERE r.id = '7c9e6679-7425-40de-944b-e07fc1f90ae7'
GROUP BY r.id, d.content;
```

### Count requests per gate

```sql
SELECT
    g.id,
    g.name,
    g.live_url,
    COUNT(r.id) as request_count
FROM gates g
LEFT JOIN requests r ON r.gate_id = g.id
GROUP BY g.id
ORDER BY request_count DESC;
```

### Find requests with diffs

```sql
SELECT r.id, r.method, r.path, r.created_at
FROM requests r
INNER JOIN diffs d ON d.request_id = r.id
WHERE r.gate_id = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY r.created_at DESC;
```

---

## Storage Considerations

### Size Estimates

**Per Request:**
- Request metadata: ~200 bytes
- Request body: Variable (avg ~1KB)
- 2× Response bodies: Variable (avg ~2KB each)
- Diff content: Variable (avg ~500 bytes)
- **Total: ~5-10KB per request**

**Volume Projections:**
- 1,000 requests = ~5-10 MB
- 10,000 requests = ~50-100 MB
- 100,000 requests = ~500 MB - 1 GB
- 1,000,000 requests = ~5-10 GB

### Cleanup Strategy

**Option 1: Time-based retention**
```sql
-- Delete requests older than 30 days
DELETE FROM requests 
WHERE created_at < NOW() - INTERVAL '30 days';
```

**Option 2: Count-based retention**
```sql
-- Keep only last 10,000 requests per gate
DELETE FROM requests
WHERE id IN (
    SELECT id FROM requests
    WHERE gate_id = '550e8400-e29b-41d4-a716-446655440000'
    ORDER BY created_at DESC
    OFFSET 10000
);
```

**Option 3: Partitioning (future)**

Partition `requests` table by `created_at` for efficient archival.

---

## Schema Source

The definitive schema is maintained in:

**Schema Directory:** `ent/schema/`
**Generated Code:** `ent/` (via `go generate ./ent/...`)

**Applied by:** mroki-api on startup via ent auto-migration

---

## Related Documentation

- [Architecture Overview](OVERVIEW.md)
- [API Contracts](API_CONTRACTS.md)
- [mroki-api Component](../components/MROKI_API.md)
