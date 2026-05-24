# mroki Examples

Working example configurations to help you get started with mroki.

For a full local dev stack (PostgreSQL + API + Proxy), see [`build/dev/`](../build/dev/) or the [Getting Started](../docs/getting-started/FULL_STACK.md) guide.

## Available Examples

### Standalone Proxy

#### [standalone-proxy](standalone-proxy/)
Run mroki-proxy as a single binary — diffs printed to stdout, no database required.
- `.env.example` template with all relevant settings
- Quick-start for local testing and CI/CD pipelines

**Use when:** You want the simplest possible setup with no infrastructure dependencies.

### Caddyfile

#### [basic-gate](caddyfile/basic-gate/)
Basic Caddy configuration showing how to shadow specific routes.
- Single gate configuration
- Route matching examples
- Production-ready template

**Use when:** You want to integrate mroki with Caddy server for shadow traffic testing.

#### [multi-gate](caddyfile/multi-gate/)
Multiple route-based gates with per-route configuration.
- Different shadow backends per route
- Per-route sampling rates and diff tuning
- Non-shadowed fallback route

**Use when:** You need to shadow several APIs independently with different settings.

## Quick Start

```bash
# Standalone proxy
cd examples/standalone-proxy
cp .env.example .env   # edit with your URLs
go build -o mroki-proxy ../../cmd/mroki-proxy && ./mroki-proxy

# Caddy — basic gate
cd examples/caddyfile/basic-gate
./caddy run --config Caddyfile

# Caddy — multi gate
cd examples/caddyfile/multi-gate
./caddy run --config Caddyfile
```

See each example's README for detailed instructions.

## Example Workflow

1. **Start services** using the [dev stack](../build/dev/) or [deployment configs](../deployments/)
2. **Create a gate** via mroki-api
3. **Configure Caddy** with the gate ID
4. **Send test requests** through the proxy
5. **View diffs** via the API or mroki-hub

## Need Help?

- [Getting Started](../docs/getting-started/FULL_STACK.md)
- [Development Guide](../docs/development/DEVELOPMENT.md)
- [Production Deployment](../docs/production/DOCKER_COMPOSE.md)

## Contributing Examples

Have a useful configuration? Please contribute!

1. Create a new directory under the appropriate category
2. Include a working configuration file
3. Add a detailed README.md explaining the use case
4. Submit a pull request
