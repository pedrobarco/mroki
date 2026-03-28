# Basic Caddy Gate Configuration

A simple Caddyfile that demonstrates shadow traffic testing with mroki.

## What This Does

- Listens on port 8080
- Routes matching `/api/*` are shadowed to both live and shadow backends
- All other routes are proxied directly to production

## Prerequisites

- Caddy with mroki module built and installed
- mroki-api running on `http://localhost:8081`
- Gate created in mroki-api

## Setup

### 1. Build Caddy with mroki Module

```bash
# From the repository root
cd build/package/caddy-mroki
xcaddy build --with github.com/your-username/mroki/caddy-mroki=../../caddy-mroki
```

### 2. Create a Gate

```bash
curl -X POST http://localhost:8081/api/v1/gates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-shadow-gate",
    "description": "Shadow traffic for /api routes"
  }'
```

Copy the returned gate ID (e.g., `550e8400-e29b-41d4-a716-446655440000`).

### 3. Update Caddyfile

Edit the `Caddyfile` and replace:

```caddyfile
live_url https://api.production.example.com
shadow_url https://api.shadow.example.com
gate_id 550e8400-e29b-41d4-a716-446655440000
```

With your actual URLs and gate ID.

### 4. Run Caddy

```bash
./caddy run --config Caddyfile
```

Or in the background:
```bash
./caddy start --config Caddyfile
```

## Test It

Send requests through Caddy:

```bash
# This request will be shadowed
curl http://localhost:8080/api/users

# This request is just proxied (no shadowing)
curl http://localhost:8080/health
```

## Check for Diffs

```bash
# List all diffs for the gate
curl http://localhost:8081/api/v1/gates/550e8400-e29b-41d4-a716-446655440000/diffs

# Get specific diff details
curl http://localhost:8081/api/v1/diffs/{diff-id}
```

## Configuration Options

### mroki Directive

```caddyfile
mroki {
    live_url <url>      # Production backend URL
    shadow_url <url>    # Shadow backend URL (canary/staging)
    api_url <url>       # mroki API server URL
    gate_id <uuid>      # Gate ID from mroki API
}
```

### Route Patterns

Match specific paths:
```caddyfile
route /api/v1/* { ... }         # Only /api/v1 routes
route /users/* { ... }          # Only /users routes
route /orders/* { ... }         # Only /orders routes
```

## Advanced Example

Multiple gates for different routes:

```caddyfile
:8080 {
    # Shadow /api/v1 routes
    route /api/v1/* {
        mroki {
            live_url https://api.production.example.com
            shadow_url https://api-v2-canary.example.com
            api_url http://localhost:8081
            gate_id 11111111-1111-1111-1111-111111111111
        }
    }

    # Shadow /orders routes
    route /orders/* {
        mroki {
            live_url https://orders.production.example.com
            shadow_url https://orders.staging.example.com
            api_url http://localhost:8081
            gate_id 22222222-2222-2222-2222-222222222222
        }
    }

    # Everything else - no shadowing
    route /* {
        reverse_proxy https://api.production.example.com
    }
}
```

## Production Considerations

### HTTPS

Enable HTTPS in production:

```caddyfile
{
	# Remove this for production
	# auto_https off
}

api.example.com {
    route /api/* {
        mroki { ... }
    }
}
```

### Performance

- The `mroki` module sends requests in parallel
- Shadow requests don't block the live response
- Diffs are sent asynchronously to mroki-api

### Monitoring

Check Caddy logs:
```bash
./caddy logs
```

## Troubleshooting

**Shadowing not working:**
- Verify gate_id exists in mroki-api
- Check Caddy logs for errors
- Ensure mroki-api is reachable

**Requests timing out:**
- Check live_url and shadow_url are accessible
- Increase timeouts if needed

**No diffs appearing:**
- Verify api_url points to running mroki-api
- Check mroki-api logs
- Ensure database connection is working

## Next Steps

1. Configure HTTPS for production
2. Set up multiple gates for different routes
3. Monitor diff results via mroki-hub (web UI)
4. Tune comparison logic based on your needs
