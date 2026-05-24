# Getting Started: Caddy Module

Embed shadow traffic diffing directly in your Caddy server — no extra services required.

## When to Use This

- You're already using **Caddy** as your web server or reverse proxy
- You want lightweight shadow diffing without running a separate proxy, API, or database
- **Standalone mode only** — diffs are printed to Caddy logs, not stored

## Prerequisites

- **Go 1.26+** — [install](https://go.dev/dl/)
- **xcaddy** — Caddy's build tool for custom modules:
  ```bash
  go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
  ```

## Step 1: Build Custom Caddy

Caddy modules must be compiled into the binary. Use `xcaddy` to build a Caddy binary with the mroki module:

```bash
# From a cloned copy of the repo:
xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule=./pkg/caddymodule

# Or, to fetch the module directly (no clone needed):
xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule
```

Verify the module is included:

```bash
./caddy list-modules | grep mroki
```

You should see `http.handlers.mroki_gate` in the output.

## Step 2: Configure Your Caddyfile

### Directive Syntax

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
    [redacted_fields <comma-separated>]
}
```

### Parameter Reference

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `live` | ✅ | — | URL of the live/production service |
| `shadow` | ✅ | — | URL of the shadow/experimental service |
| `sampling_rate` | | `1.0` | Sample rate (0.0–1.0, where 1.0 = 100%) |
| `live_timeout` | | `5s` | Timeout for the live request |
| `shadow_timeout` | | `10s` | Timeout for the shadow request |
| `max_body_size` | | `0` | Skip shadow for requests above this size in bytes (0 = unlimited) |
| `diff_ignored_fields` | | — | Comma-separated fields to ignore in diff (gjson syntax) |
| `diff_included_fields` | | — | Comma-separated fields to include in diff (whitelist mode) |
| `diff_float_tolerance` | | — | Float comparison tolerance |
| `redacted_fields` | | — | Comma-separated fields to redact from diff output |

### Basic Example

```caddyfile
:8080 {
    log {
        output stdout
        format console
    }

    mroki_gate {
        live https://api.production.example.com
        shadow https://api.shadow.example.com
    }
}
```

## Step 3: Run Caddy

```bash
./caddy run --config Caddyfile
```

Expected output:

```
{"level":"info","msg":"using config from file","file":"Caddyfile"}
{"level":"info","msg":"serving initial configuration"}
```

## Step 4: Send Traffic

Send a request through Caddy:

```bash
curl http://localhost:8080/test -d '{"test":true}'
```

When a diff is detected, you'll see it in Caddy's logs:

```
INFO response diff detected method=GET path=/api/test live_status=200 shadow_status=200 changes=2
Diff:
 ~ /body/user: "alice" → "bob"
```

When responses match:

```
DEBUG responses match method=GET path=/api/test live_status=200 shadow_status=200
```

## Examples

### With Sampling

Sample only a percentage of traffic to reduce load on the shadow service:

```caddyfile
:8080 {
    mroki_gate {
        live https://api.production.example.com
        shadow https://api.shadow.example.com
        sampling_rate 0.1  # Only 10% of requests
    }
}
```

### With Diff Options

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

### Production with TLS

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

## Differences from mroki-proxy

| Feature | mroki-proxy | caddy-mroki |
|---------|-------------|-------------|
| Deployment | Standalone binary | Compiled into Caddy |
| Configuration | Environment variables | Caddyfile |
| API Mode | ✅ Sends raw responses to mroki-api | ❌ Standalone only |
| Retry Logic | ✅ Exponential backoff | N/A |

caddy-mroki operates in **standalone mode only** — it computes and prints diffs locally. For API integration (server-side diffing, storage, hub), use mroki-proxy instead.

## What's Next

- **[Full Stack](FULL_STACK.md)** — Add persistence and the web UI for stored diffs and visualization
- **[Configuration](../production/CONFIGURATION.md)** — Full reference for all configuration options
- **[Troubleshooting](../TROUBLESHOOTING.md)** — Common issues and solutions
