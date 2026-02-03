# mroki-agent Configuration Guide

## Overview

The mroki-agent is a proxy that intercepts HTTP requests, forwards them to both live and shadow services, compares the responses, and optionally sends the diff results to the mroki-api for analysis.

## Operating Modes

The agent supports two operating modes:

### 1. API Mode (Recommended)

In API mode, the agent:
- Fetches gate configuration (live/shadow URLs) from mroki-api on startup
- Sends captured request/response/diff data to the API
- Retries with exponential backoff if API is temporarily unavailable

**Configuration:**
```bash
MROKI_APP_API_URL="http://localhost:8081"
MROKI_APP_GATE_ID="550e8400-e29b-41d4-a716-446655440000"
MROKI_APP_API_KEY="your-secret-key-minimum-16-chars"
```

### 2. Standalone Mode

In standalone mode, the agent:
- Uses hardcoded live/shadow URLs from environment variables
- Does not communicate with mroki-api
- Logs diffs locally only

**Configuration:**
```bash
MROKI_APP_LIVE_URL="https://api.production.com"
MROKI_APP_SHADOW_URL="https://api.staging.com"
```

## Diff Configuration

Regardless of operating mode, you can configure how responses are compared using diff options. This allows you to:

- **Ignore specific fields** (e.g., timestamps, auto-generated IDs)
- **Whitelist fields** (only compare specific fields)
- **Sort arrays** before comparison (ignore order differences)
- **Set float tolerance** (allow small differences in floating point numbers)

### Environment Variables

#### `MROKI_APP_DIFF_IGNORED_FIELDS`

Comma-separated list of JSON field paths to ignore during comparison.

**Syntax:** gjson path syntax (supports nested fields and array wildcards)

**Examples:**
```bash
# Simple fields
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,id"

# Nested fields
MROKI_APP_DIFF_IGNORED_FIELDS="metadata.timestamp,user.created_at"

# Array wildcards (# matches any array element)
MROKI_APP_DIFF_IGNORED_FIELDS="users.#.id,orders.#.created_at"

# Multiple patterns
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,metadata.created_at,users.#.updated_at"
```

#### `MROKI_APP_DIFF_INCLUDED_FIELDS`

Comma-separated list of JSON field paths to include (whitelist mode). When set, ONLY these fields are compared, then `DIFF_IGNORED_FIELDS` is applied.

**Examples:**
```bash
# Only compare user and order fields
MROKI_APP_DIFF_INCLUDED_FIELDS="user,order"

# Combine with ignored fields
MROKI_APP_DIFF_INCLUDED_FIELDS="user,order"
MROKI_APP_DIFF_IGNORED_FIELDS="user.created_at,order.created_at"
```

#### `MROKI_APP_DIFF_SORT_ARRAYS`

Boolean flag to sort arrays before comparison. Useful when array order doesn't matter.

**Default:** `false`

**Example:**
```bash
MROKI_APP_DIFF_SORT_ARRAYS=true
```

#### `MROKI_APP_DIFF_FLOAT_TOLERANCE`

Tolerance for floating point comparisons. Allows small differences that might occur due to rounding.

**Default:** `0` (exact comparison)

**Example:**
```bash
# Allow 0.1% difference (useful for prices, percentages)
MROKI_APP_DIFF_FLOAT_TOLERANCE=0.001
```

## Configuration Examples

### Example 1: API Mode with Timestamp Filtering

```bash
# API Mode
MROKI_APP_API_URL="http://localhost:8081"
MROKI_APP_GATE_ID="550e8400-e29b-41d4-a716-446655440000"
MROKI_APP_API_KEY="my-secret-key"

# Ignore all timestamp fields
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,created_at,updated_at,metadata.timestamp"

# Sort arrays (order doesn't matter)
MROKI_APP_DIFF_SORT_ARRAYS=true
```

### Example 2: Standalone Mode with Whitelist

```bash
# Standalone Mode
MROKI_APP_LIVE_URL="https://api.production.com"
MROKI_APP_SHADOW_URL="https://api.staging.com"

# Only compare critical business fields
MROKI_APP_DIFF_INCLUDED_FIELDS="user.email,user.name,order.total,order.items"

# Within those fields, ignore timestamps
MROKI_APP_DIFF_IGNORED_FIELDS="user.last_login,order.items.#.added_at"
```

### Example 3: E-commerce API with Nested Arrays

```bash
# API Mode
MROKI_APP_API_URL="http://localhost:8081"
MROKI_APP_GATE_ID="abc-123"
MROKI_APP_API_KEY="secret"

# Ignore auto-generated fields in nested arrays
MROKI_APP_DIFF_IGNORED_FIELDS="order_id,created_at,orders.#.id,orders.#.items.#.item_id"

# Allow small price differences (0.01%)
MROKI_APP_DIFF_FLOAT_TOLERANCE=0.0001

# Sort arrays
MROKI_APP_DIFF_SORT_ARRAYS=true
```

### Example 4: Testing with Specific Fields Only

```bash
# Standalone Mode
MROKI_APP_LIVE_URL="https://api-v1.com"
MROKI_APP_SHADOW_URL="https://api-v2.com"

# Only compare the response data field (ignore metadata)
MROKI_APP_DIFF_INCLUDED_FIELDS="data"

# Within data, ignore internal IDs
MROKI_APP_DIFF_IGNORED_FIELDS="data.internal_id,data.items.#._id"
```

## Startup Flow

### API Mode Startup

1. Agent reads configuration from environment variables
2. Agent calls `GET /gates/{gate_id}` to fetch gate configuration
3. If request fails, agent retries with exponential backoff (default: 3 retries)
4. Agent parses diff options from environment
5. Agent starts proxy server with fetched URLs and diff options

**Retry Logic:**
- Attempt 1: Immediate
- Attempt 2: After 1 second
- Attempt 3: After 2 seconds
- Attempt 4: After 4 seconds

If all retries fail, the agent exits with an error.

### Standalone Mode Startup

1. Agent reads configuration from environment variables
2. Agent validates live/shadow URLs
3. Agent parses diff options from environment
4. Agent starts proxy server

## Runtime Behavior

When a request is proxied:

1. Request is forwarded to **both** live and shadow services (parallel)
2. Live response is returned to client immediately
3. In background:
   - Responses are compared using configured diff options
   - If API mode: diff results sent to mroki-api
   - If standalone mode: diff results logged locally

## Field Pattern Syntax

The diff configuration uses [gjson path syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md):

| Pattern | Description | Example JSON | Matches |
|---------|-------------|--------------|---------|
| `field` | Top-level field | `{"field": 1}` | `field` |
| `a.b` | Nested field | `{"a": {"b": 1}}` | `a.b` |
| `a.#.b` | Array wildcard | `{"a": [{"b": 1}, {"b": 2}]}` | `a[0].b`, `a[1].b` |
| `a.b\\.c` | Escaped dot | `{"a": {"b.c": 1}}` | `a["b.c"]` |

**Common patterns:**
```bash
# Ignore all timestamps
"timestamp,created_at,updated_at,deleted_at"

# Ignore IDs in nested arrays
"users.#.id,users.#.posts.#.id"

# Ignore metadata object
"metadata"

# Ignore specific nested field
"response.data.internal.debug_info"
```

## Troubleshooting

### Agent fails to start: "must configure either API mode or standalone mode"

**Problem:** Neither API configuration nor standalone URLs are provided.

**Solution:** Set either API mode vars (API_URL + GATE_ID + API_KEY) or standalone vars (LIVE_URL + SHADOW_URL).

### Agent fails to start: "failed to fetch gate after X attempts"

**Problem:** Cannot reach mroki-api or gate doesn't exist.

**Solutions:**
- Check API_URL is correct and mroki-api is running
- Verify GATE_ID exists (create gate via `POST /gates`)
- Check API_KEY matches the server configuration
- Check network connectivity

### Diff options not being applied

**Problem:** Diffs still show ignored fields.

**Solutions:**
- Check field patterns match your JSON structure
- Use gjson syntax tester to verify patterns
- Check logs for "Diff options configured" message
- Verify fields are spelled correctly (case-sensitive)

### "Invalid gate ID" error

**Problem:** GATE_ID is not a valid UUID.

**Solution:** Get the correct UUID from mroki-api gate creation response.

## Advanced Configuration

### Custom Timeouts

```bash
# Live request timeout (blocks client)
MROKI_APP_LIVE_TIMEOUT=5s

# Shadow request timeout (doesn't block client)
MROKI_APP_SHADOW_TIMEOUT=10s

# API request timeout
MROKI_APP_API_TIMEOUT=30s
```

### Retry Configuration

```bash
# Maximum retry attempts for API calls
MROKI_APP_MAX_RETRIES=3

# Initial delay between retries (doubles each attempt)
MROKI_APP_RETRY_DELAY=1s
```

### Body Size Limits

```bash
# Maximum request body size for diffing (bytes)
# Larger bodies skip shadow/diff and only proxy to live
MROKI_APP_MAX_BODY_SIZE=10485760  # 10MB

# Set to 0 for unlimited (always diff)
MROKI_APP_MAX_BODY_SIZE=0
```

## Complete Example Configuration

```bash
# ===== API Mode =====
MROKI_APP_API_URL="http://localhost:8081"
MROKI_APP_GATE_ID="550e8400-e29b-41d4-a716-446655440000"
MROKI_APP_API_KEY="your-secret-key-minimum-16-chars"

# ===== Server Configuration =====
MROKI_APP_PORT=8080
MROKI_APP_LIVE_TIMEOUT=5s
MROKI_APP_SHADOW_TIMEOUT=10s
MROKI_APP_MAX_BODY_SIZE=10485760

# ===== Diff Configuration =====
MROKI_APP_DIFF_IGNORED_FIELDS="timestamp,created_at,updated_at,metadata.request_id,users.#.last_seen"
MROKI_APP_DIFF_SORT_ARRAYS=true
MROKI_APP_DIFF_FLOAT_TOLERANCE=0.001

# ===== API Integration =====
MROKI_APP_MAX_RETRIES=3
MROKI_APP_RETRY_DELAY=1s
MROKI_APP_API_TIMEOUT=30s
```

## See Also

- [gjson Path Syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)
- [mroki-api Documentation](../mroki-api/README.md)
