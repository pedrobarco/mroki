# caddy-mroki

**Caddy server module for embedded mroki proxy functionality**

caddy-mroki is a Caddy v2 module that integrates mroki's shadow traffic testing capabilities directly into the Caddy web server. This allows you to use mroki without deploying a standalone agent.

## Features

- **Native Caddy Integration**: Use mroki via Caddyfile directive
- **Zero External Dependencies**: No separate agent process needed
- **Caddy Module System**: Leverages Caddy's plugin architecture
- **Simple Configuration**: Clean Caddyfile syntax
- **Same Proxy Logic**: Uses the same battle-tested `pkg/proxy` package as mroki-agent

## Installation

### Build Custom Caddy Binary

Caddy modules must be compiled into the binary. Use `xcaddy` to build:

```bash
# Install xcaddy
go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

# Build Caddy with mroki module
xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule

# This creates a `caddy` binary in the current directory
```

### Using the Included Main

Alternatively, use the included main package:

```bash
cd cmd/caddy-mroki
go build -o caddy-mroki .

# Or run directly
go run .
```

## Configuration

### Caddyfile Syntax

```caddyfile
mroki_gate {
    live <live_url>
    shadow <shadow_url>
    [sampling_rate <rate>]
    [live_timeout <duration>]
    [shadow_timeout <duration>]
    [max_body_size <bytes>]
    [diff_ignored_fields <comma-separated>]
    [diff_included_fields <comma-separated>]
    [diff_float_tolerance <float>]
}
```

**Parameters:**
- `live` (required) - URL of live/production service
- `shadow` (required) - URL of shadow/experimental service
- `sampling_rate` (optional) - Sample rate (0.0-1.0, default: 1.0 = 100%)
- `live_timeout` (optional) - Live request timeout (default: 5s)
- `shadow_timeout` (optional) - Shadow request timeout (default: 10s)
- `max_body_size` (optional) - Skip shadow for requests above this size in bytes (0=unlimited)
- `diff_ignored_fields` (optional) - Comma-separated fields to ignore in diff (gjson syntax)
- `diff_included_fields` (optional) - Comma-separated fields to include in diff (whitelist)
- `diff_float_tolerance` (optional) - Float comparison tolerance

### Complete Caddyfile Example

```caddyfile
:8080 {
    mroki_gate {
        live https://api.production.example.com
        shadow https://api.shadow.example.com
        sampling_rate 1.0
        live_timeout 3s
        shadow_timeout 15s
        max_body_size 10485760
        diff_ignored_fields timestamp,created_at
    }
}
```

### Multiple Routes

```caddyfile
:8080 {
    # Gate for /api/* routes
    route /api/* {
        mroki_gate {
            live https://api.production.example.com
            shadow https://api.shadow.example.com
        }
    }

    # Gate for /checkout/* routes
    route /checkout/* {
        mroki_gate {
            live https://checkout.production.example.com
            shadow https://checkout.shadow.example.com
            sampling_rate 0.5  # Only 50% of traffic
        }
    }

    # Regular reverse proxy for other routes
    reverse_proxy /admin/* admin-backend:8081
}
```

### Sampling

Sample only a percentage of traffic to reduce load:

```caddyfile
:8080 {
    mroki_gate {
        live https://api.production.example.com
        shadow https://api.shadow.example.com
        sampling_rate 0.1  # Only 10% of requests
    }
}
```

## Running

```bash
# Using custom-built binary
./caddy run --config Caddyfile

# Or with the included main
cd cmd/caddy-mroki
go run . run --config Caddyfile
```

## How It Works

The module integrates into Caddy's HTTP handler chain:

```
Client Request
     │
     ↓
Caddy HTTP Server
     │
     ↓
mroki_gate Handler
     │
     ├─────────────────┐
     ↓                 ↓
  Live URL         Shadow URL
     │                 │
     ↓                 ↓
  Return Live      Shadow Response
  Response         Captured
```

**Key Behavior:**
1. Caddy receives request
2. `mroki_gate` handler forwards to both live and shadow
3. Live response returned to client immediately
4. Diff computed and printed locally in background

## Differences from mroki-agent

### Similarities
- Same proxy logic (`pkg/proxy`)
- Same timeout behavior
- Same sampling support
- Same max body size check
- Same local diff computation and output
- Same diff options (ignored fields, included fields, float tolerance)

### Differences

| Feature | mroki-agent | caddy-mroki |
|---------|-------------|-------------|
| Deployment | Standalone binary | Compiled into Caddy |
| Configuration | Environment variables | Caddyfile |
| API Mode | ✅ Sends raw responses to mroki-api | ❌ Standalone only |
| Agent ID | Persisted to disk | N/A |
| Retry Logic | ✅ Exponential backoff | N/A |

caddy-mroki operates in standalone mode only — it computes and prints diffs locally. For API integration (server-side diffing, storage, hub), use mroki-agent.

## Example Deployment

### Local Testing

**Caddyfile:**
```caddyfile
:8080 {
    log {
        output stdout
        format console
    }

    mroki_gate {
        live https://httpbin.org/anything?service=live
        shadow https://httpbin.org/anything?service=shadow
    }
}
```

**Run:**
```bash
./caddy run --config Caddyfile

# Send test request
curl http://localhost:8080/test -d '{"test":true}'
```

### Production with TLS

**Caddyfile:**
```caddyfile
api.example.com {
    # Automatic HTTPS via Let's Encrypt
    
    log {
        output file /var/log/caddy/access.log
    }

    mroki_gate {
        live https://api-prod.internal:8081
        shadow https://api-shadow.internal:8081
        sampling_rate 0.2  # 20% of traffic
        live_timeout 2s
        shadow_timeout 10s
    }
}
```

### Docker

**Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder

# Install xcaddy
RUN go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest

# Build Caddy with mroki module
RUN xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/caddy .
COPY Caddyfile .
CMD ["./caddy", "run", "--config", "Caddyfile"]
```

**Run:**
```bash
docker build -t caddy-mroki .
docker run -p 8080:8080 -v $(pwd)/Caddyfile:/root/Caddyfile caddy-mroki
```

## Logging

Diff results are logged locally:

```
INFO response diff detected method=GET path=/api/test live_status=200 shadow_status=200 changes=2
Diff:
 ~ /body/user: "alice" → "bob"
```

When responses match:
```
DEBUG responses match method=GET path=/api/test live_status=200 shadow_status=200
```

**Configure logging in Caddyfile:**
```caddyfile
:8080 {
    log {
        output file /var/log/caddy/access.log {
            roll_size 100mb
            roll_keep 10
        }
        format json
        level INFO
    }

    mroki_gate {
        live https://api.production.example.com
        shadow https://api.shadow.example.com
    }
}
```

## Troubleshooting

### Module not found

**Problem:** `Error: module 'http.handlers.mroki_gate' not found`

**Solution:** Ensure the module is compiled into Caddy:

```bash
# Rebuild with xcaddy
xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule

# Verify module is included
./caddy list-modules | grep mroki
```

---

### Invalid configuration

**Problem:** `Error: live URL is required`

**Solution:** Ensure both `live` and `shadow` are specified in Caddyfile:

```caddyfile
mroki_gate {
    live https://api.production.example.com
    shadow https://api.shadow.example.com
}
```

---

### High latency

**Problem:** Requests are slow through Caddy

**Solution:** Reduce `live_timeout` (this blocks client response):

```caddyfile
mroki_gate {
    live https://api.production.example.com
    shadow https://api.shadow.example.com
    live_timeout 2s  # Reduced from default 5s
}
```

## Limitations

1. **Standalone Only:** No API integration — diffs are printed locally, not stored
2. **Limited Observability:** Only Caddy logs, no metrics yet
3. **Caddyfile Only:** No JSON config support yet

## Future Enhancements

- [ ] Metrics export (Prometheus)
- [ ] JSON configuration support
- [ ] Per-route sampling configuration

## Use Cases

### When to Use caddy-mroki

- Already using Caddy as reverse proxy
- Want quick local diff output without a separate binary
- Don't need API storage or hub visualization

### When to Use mroki-agent Instead

- Need API integration (server-side diffing, storage, hub visualization)
- Not using Caddy
- Need persistent agent identity and retry logic

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [mroki-agent Component](MROKI_AGENT.md) - Standalone agent alternative
- [Quick Start Guide](../guides/QUICK_START.md)
