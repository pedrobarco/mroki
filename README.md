# mroki

Safe shadow traffic testing for production systems.

mroki mirrors live HTTP traffic to a shadow service, diffs the JSON responses, and surfaces the differences — so you can validate changes against real production behavior before rolling out.

![mroki-hub request detail](docs/assets/hub-request-detail-unified.png)

> See the full [screenshot gallery](docs/SCREENSHOTS.md) for more views.

## Quick Start

```bash
# Start the dev stack (PostgreSQL + API + Agent)
docker compose -f build/dev/compose.yaml up -d

# Create a gate (live/shadow service pair)
curl -s -X POST http://localhost:8090/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mroki-dev-api-key-16" \
  -d '{"live_url": "https://httpbin.org/anything?env=live", "shadow_url": "https://httpbin.org/anything?env=shadow"}'

# Send traffic through the agent proxy
curl http://localhost:8080/get
```

Responses from both services are compared automatically. Open [mroki-hub](http://localhost:5173) to browse gates, requests, and diffs.

See the [Quick Start Guide](docs/guides/QUICK_START.md) for the full walkthrough.

## How It Works

A **gate** is a pair of services: a live (production) URL and a shadow (experimental) URL.

An **agent** is an HTTP proxy that forwards each request to both services, computes a JSON diff of the responses, and reports it to the API — without affecting the live response.

The **hub** is a web UI for managing gates, browsing captured requests, and visualizing response diffs side-by-side.

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP Request
       ↓
┌─────────────────┐
│  mroki-agent    │  (Proxy)
└────┬──────┬─────┘
     │      │
     │      └──────────┐
     ↓                 ↓
┌──────────┐    ┌─────────────┐
│   Live   │    │   Shadow    │
│ Service  │    │   Service   │
└──────────┘    └─────────────┘
     │                 │
     └────────┬────────┘
              ↓
       Compute Diff
              ↓
    ┌─────────────────┐
    │   mroki-api     │  (REST API)
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │   PostgreSQL    │
    └─────────────────┘
             ↑
    ┌─────────────────┐
    │   mroki-hub     │  (Web UI)
    └─────────────────┘
```

## Components

| Component | Description | Docs |
|---|---|---|
| [mroki-agent](docs/components/MROKI_AGENT.md) | HTTP proxy — forwards traffic to live and shadow, computes diffs | [docs](docs/components/MROKI_AGENT.md) |
| [mroki-api](docs/components/MROKI_API.md) | REST API — gate management, request/diff storage | [docs](docs/components/MROKI_API.md) |
| [mroki-hub](docs/components/MROKI_HUB.md) | Web UI — gate dashboard, request browser, diff viewer | [docs](docs/components/MROKI_HUB.md) |
| [caddy-mroki](docs/components/CADDY_MROKI.md) | Caddy module — integrates mroki proxy into Caddy server | [docs](docs/components/CADDY_MROKI.md) |

## Use Cases

**API refactoring** — Test refactored endpoints against real production traffic to catch behavioral regressions before they ship.

**Database migrations** — Run your new schema in shadow mode and verify it returns identical results to the current one.

**Framework upgrades** — Upgrade your framework on the shadow service and validate with real request patterns, not synthetic tests.

## Documentation

- [Quick Start](docs/guides/QUICK_START.md) — Get running in 5 minutes
- [Development](docs/guides/DEVELOPMENT.md) — Local setup, testing, code style
- [Deployment](docs/guides/DEPLOYMENT.md) — Production deployment with Docker Compose and Kubernetes
- [System Overview](docs/architecture/OVERVIEW.md) — Architecture and data flow
- [API Contracts](docs/architecture/API_CONTRACTS.md) — Endpoint specifications
- [Roadmap](docs/TODO.md)

## Development

```bash
# Start dev stack
make dev-up

# Run all tests
make test

# Build all binaries
make build

# Lint
make lint
```

See the [Development Guide](docs/guides/DEVELOPMENT.md) for the full workflow.

## Contributing

Contributions welcome. Please read the [Contributing Guide](docs/CONTRIBUTING.md) before submitting PRs.

This project follows the [Contributor Covenant Code of Conduct](docs/CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE)
