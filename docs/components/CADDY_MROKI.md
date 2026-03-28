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
}
```

**Parameters:**
- `live` (required) - URL of live/production service
- `shadow` (required) - URL of shadow/experimental service
- `sampling_rate` (optional) - Sample rate (0.0-1.0, default: 1.0 = 100%)
- `live_timeout` (optional) - Live request timeout (default: 5s)
- `shadow_timeout` (optional) - Shadow request timeout (default: 10s)

### Complete Caddyfile Example

```caddyfile
# Listen on port 8080
:8080 {
    # Apply mroki gate to all requests
    mroki_gate {
        live https://api.production.example.com
        shadow https://api.shadow.example.com
        sampling_rate 1.0
        live_timeout 3s
        shadow_timeout 15s
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
  Return Live      Discard Shadow
  Response         (metadata logged)
```

**Key Behavior:**
1. Caddy receives request
2. `mroki_gate` handler forwards to both live and shadow
3. Live response returned to client immediately
4. Shadow response metadata logged in background (no diff computation)
5. No API integration currently — responses are not stored or diffed

## Differences from mroki-agent

### Similarities
- Same proxy logic (`pkg/proxy`)
- Same timeout behavior
- Same sampling support

### Differences

| Feature | mroki-agent | caddy-mroki |
|---------|-------------|-------------|
| Deployment | Standalone binary | Compiled into Caddy |
| Configuration | Environment variables | Caddyfile |
| API Integration | ✅ Yes (sends raw responses to API) | ❌ Not yet implemented |
| Diff Computation | ✅ Server-side (API mode) or local (standalone) | ❌ None — only logs metadata |
| Agent ID | ✅ Persists to disk | ❌ Not applicable |
| Retry Logic | ✅ Exponential backoff | ❌ Not applicable (no API) |
| Max Body Size Check | ✅ Yes | ❌ Not yet implemented |
| Use Case | Production deployments | Caddy-based infrastructures |

**Note:** caddy-mroki currently lacks API integration and diff computation. Shadow responses are captured but not stored or compared. This is planned for a future release to reach feature parity with mroki-agent.

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

Shadow response captures are logged via Caddy's logging system:

```json
{"level":"info","msg":"shadow response captured","method":"GET","path":"/api/test","live_status":200,"shadow_status":200}
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

1. **No API Integration:** Responses are not sent to mroki-api (coming in future release)
2. **No Diff Computation:** Shadow responses are logged but not compared with live responses
3. **No Persistent State:** No agent ID persistence
4. **No Max Body Size Check:** All requests forwarded to shadow regardless of body size
5. **Limited Observability:** Only Caddy logs, no metrics yet
6. **Caddyfile Only:** No JSON config support yet

## Future Enhancements

- [ ] API integration (send raw responses to mroki-api for server-side diffing)
- [ ] Standalone diff mode (compute and log diffs locally, matching mroki-agent standalone)
- [ ] Max body size check (skip shadow for large payloads)
- [ ] Metrics export (Prometheus)
- [ ] JSON configuration support
- [ ] Per-route sampling configuration

## Use Cases

### When to Use caddy-mroki

- Already using Caddy as reverse proxy
- Want zero-deployment shadow testing
- Simple use cases (logging shadow captures, no storage)
- Quick experimentation

### When to Use mroki-agent Instead

- Need API integration for diff storage and analysis
- Need diff computation (server-side via API or local in standalone mode)
- Want persistent agent identity
- Need retry logic for reliability
- Multi-service deployments
- Production-critical scenarios

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [mroki-agent Component](MROKI_AGENT.md) - Standalone agent alternative
- [Quick Start Guide](../guides/QUICK_START.md)
