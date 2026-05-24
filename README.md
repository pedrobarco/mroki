<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/mroki-logo-banner-dark.png" />
    <img src="docs/assets/brand/mroki-logo-banner-light.png" alt="mroki" width="600" />
  </picture>
</p>

<p align="center">Safe shadow traffic testing for production systems.</p>

mroki mirrors live HTTP traffic to a shadow service, diffs the JSON responses, and surfaces the differences — so you can validate changes against real production behavior before rolling out.

![mroki-hub request detail](docs/assets/screenshots/hub-request-detail-split.png)

> See the full [screenshot gallery](docs/SCREENSHOTS.md) for more views.

## Quick Start

```bash
# Start the dev stack (PostgreSQL + API + Proxy)
docker compose -f build/dev/compose.yaml up -d

# Create a gate (live/shadow service pair)
curl -s -X POST http://localhost:8090/gates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mroki-dev-api-key-16" \
  -d '{"live_url": "https://httpbin.org/anything?env=live", "shadow_url": "https://httpbin.org/anything?env=shadow"}'

# Send traffic through the proxy
curl http://localhost:8080/get
```

Responses from both services are compared automatically. Open [mroki-hub](http://localhost:5173) to browse gates, requests, and diffs.

See the [Getting Started](docs/getting-started/FULL_STACK.md) guide for the full walkthrough.

## How It Works

A **gate** is a pair of services: a live (production) URL and a shadow (experimental) URL.

A **proxy** forwards each request to both services and sends the raw responses to the API — without affecting the live response. The API computes the JSON diff server-side.

The **hub** is a web UI for managing gates, browsing captured requests, and visualizing response diffs side-by-side.

## Architecture

```mermaid
graph TD
    Client([Client]) -->|HTTP Request| Proxy[mroki-proxy]
    Proxy -->|Forward| Live[Live Service]
    Proxy -->|Forward| Shadow[Shadow Service]
    Live -->|Live Response| Proxy
    Shadow -->|Shadow Response| Proxy
    Proxy -->|Return live response| Client
    Proxy -.->|Send raw responses| API[mroki-api<br><i>REST API + Diff</i>]
    API -->|Store| DB[(PostgreSQL)]
    Hub[mroki-hub<br><i>Web UI</i>] -->|Query| API
```

## Use Cases

**API refactoring** — Test refactored endpoints against real production traffic to catch behavioral regressions before they ship.

**Database migrations** — Run your new schema in shadow mode and verify it returns identical results to the current one.

**Framework upgrades** — Upgrade your framework on the shadow service and validate with real request patterns, not synthetic tests.

## Documentation

| | Guide | Description |
|---|---|---|
| 🚀 | **Getting Started** | |
| | [Full Stack](docs/getting-started/FULL_STACK.md) | All-in-one Docker Compose — proxy, API, hub, database |
| | [Standalone Proxy](docs/getting-started/STANDALONE_PROXY.md) | Single binary, diffs to stdout, no database |
| | [Caddy Module](docs/getting-started/CADDY_MODULE.md) | Embedded in an existing Caddy server |
| 🏗️ | **Production** | |
| | [Docker Compose](docs/production/DOCKER_COMPOSE.md) | Production deployment with Compose |
| | [Kubernetes](docs/production/KUBERNETES.md) | K8s manifests and Helm charts |
| | [Configuration](docs/production/CONFIGURATION.md) | All environment variables and options |
| | [Security](docs/production/SECURITY.md) | Auth, TLS, redaction, hardening |
| | [Monitoring](docs/production/MONITORING.md) | Logging, health checks, observability |
| 🔌 | **API** | |
| | [Walkthrough](docs/api/WALKTHROUGH.md) | Step-by-step: create a gate, capture traffic, query diffs |
| | [Reference](docs/api/REFERENCE.md) | Full endpoint specification |
| 📖 | **Reference** | |
| | [Architecture](docs/architecture/OVERVIEW.md) | System design, data flow, technology choices |
| | [Diff Pipeline](docs/architecture/DIFF_ANALYSIS.md) | How responses are compared and diffed |
| | [Troubleshooting](docs/TROUBLESHOOTING.md) | Common issues and fixes |
| | [Roadmap](docs/ROADMAP.md) | What's planned |

## Development

```bash
make dev-up   # Start dev stack
make test     # Run all tests
make build    # Build all binaries
make lint     # Lint
```

See the [Development Guide](docs/development/DEVELOPMENT.md) for the full workflow.

## Contributing

Contributions welcome. Please read the [Contributing Guide](docs/development/CONTRIBUTING.md) before submitting PRs.

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE)
