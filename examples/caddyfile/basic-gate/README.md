# Basic Caddy Gate Configuration

A simple Caddyfile that demonstrates shadow traffic testing with mroki.

## What This Does

- Listens on port 8080
- Routes matching `/api/*` are shadowed to both live and shadow backends
- All other routes are proxied directly to production

## Prerequisites

- Caddy with mroki module built and installed

## Setup

### 1. Build Caddy with mroki Module

```bash
# From the repository root
cd build/package/caddy-mroki
xcaddy build --with github.com/your-username/mroki/caddy-mroki=../../caddy-mroki
```

### 2. Update Caddyfile

Edit the `Caddyfile` and replace:

```caddyfile
live https://api.production.example.com
shadow https://api.shadow.example.com
```

With your actual URLs.

### 3. Run Caddy

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
# List all gates
curl http://localhost:8081/gates

# List all requests for a gate
curl http://localhost:8081/gates/{gate_id}/requests
```

## Configuration Options

### mroki_gate Directive

```caddyfile
mroki_gate {
    live <url>      # Production backend URL
    shadow <url>    # Shadow backend URL (canary/staging)
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
        mroki_gate {
            live https://api.production.example.com
            shadow https://api-v2-canary.example.com
        }
    }

    # Shadow /orders routes
    route /orders/* {
        mroki_gate {
            live https://orders.production.example.com
            shadow https://orders.staging.example.com
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
        mroki_gate { ... }
    }
}
```

### Performance

- The `mroki_gate` module sends requests in parallel
- Shadow requests don't block the live response
- Diffs are computed and printed locally

### Monitoring

Check Caddy logs:
```bash
./caddy logs
```

## Troubleshooting

**Shadowing not working:**
- Check Caddy logs for errors
- Verify live and shadow URLs are reachable

**Requests timing out:**
- Check live and shadow URLs are accessible
- Increase timeouts if needed

**No diffs appearing:**
- Check Caddy logs for diff output
- Verify shadow service is responding

## Next Steps

1. Configure HTTPS for production
2. Set up multiple gates for different routes
3. Monitor diff results via mroki-hub (web UI)
4. Tune comparison logic based on your needs
