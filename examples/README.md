# mroki Examples

Working example configurations to help you get started with mroki.

## Available Examples

### Docker Compose

#### [simple-setup](docker-compose/simple-setup/)
A minimal working setup for local development and testing.
- PostgreSQL + mroki-api + mroki-agent
- Ready to use with test configuration
- Includes health checks and proper service dependencies

**Use when:** You want to quickly test mroki locally or learn how the components work together.

### Caddyfile

#### [basic-gate](caddyfile/basic-gate/)
Basic Caddy configuration showing how to shadow specific routes.
- Single gate configuration
- Route matching examples
- Production-ready template

**Use when:** You want to integrate mroki with Caddy server for shadow traffic testing.

## Quick Start

### Option 1: Docker Compose (Easiest)

```bash
cd examples/docker-compose/simple-setup
docker-compose up -d
```

See the [README](docker-compose/simple-setup/README.md) for detailed instructions.

### Option 2: Caddy Integration

```bash
cd examples/caddyfile/basic-gate
# Build Caddy with mroki module first (see README)
./caddy run --config Caddyfile
```

See the [README](caddyfile/basic-gate/README.md) for detailed instructions.

## Example Workflow

1. **Start services** using Docker Compose example
2. **Create a gate** via mroki-api
3. **Configure agent or Caddy** with the gate ID
4. **Send test requests** through the proxy
5. **Check for diffs** via the API or future web UI

## Need Help?

- [Quick Start Guide](../docs/guides/QUICK_START.md)
- [Development Guide](../docs/guides/DEVELOPMENT.md)
- [Deployment Guide](../docs/guides/DEPLOYMENT.md)

## Contributing Examples

Have a useful configuration? Please contribute!

1. Create a new directory under the appropriate category
2. Include a working configuration file
3. Add a detailed README.md explaining the use case
4. Submit a pull request
