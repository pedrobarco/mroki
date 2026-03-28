# mroki Examples

Working example configurations to help you get started with mroki.

For a full local dev stack (PostgreSQL + API + Agent), see [`build/dev/`](../build/dev/) or the [Quick Start Guide](../docs/guides/QUICK_START.md).

## Available Examples

### Caddyfile

#### [basic-gate](caddyfile/basic-gate/)
Basic Caddy configuration showing how to shadow specific routes.
- Single gate configuration
- Route matching examples
- Production-ready template

**Use when:** You want to integrate mroki with Caddy server for shadow traffic testing.

## Quick Start

```bash
cd examples/caddyfile/basic-gate
# Build Caddy with mroki module first (see README)
./caddy run --config Caddyfile
```

See the [README](caddyfile/basic-gate/README.md) for detailed instructions.

## Example Workflow

1. **Start services** using the [dev stack](../build/dev/) or [deployment configs](../deployments/)
2. **Create a gate** via mroki-api
3. **Configure Caddy** with the gate ID
4. **Send test requests** through the proxy
5. **View diffs** via the API or mroki-hub

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
