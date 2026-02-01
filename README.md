# mroki

**Safe shadow traffic testing for production systems**

mroki is a platform for testing service changes by comparing production traffic against shadow deployments in real-time. Send live traffic to both your production and shadow services, compute response diffs, and analyze behavior differences without impacting users.

## What is mroki?

mroki enables you to:

- **Mirror production traffic** to shadow services running experimental code
- **Compare responses** between live and shadow deployments automatically
- **Visualize differences** through a web interface
- **Make confident releases** by understanding real-world behavior before rollout

## Core Concepts

### Gates
A **gate** represents a pair of services: a live (production) service and a shadow (experimental) service. Traffic flowing through a gate is forwarded to both services.

### Agents
An **agent** is a proxy deployed in your infrastructure that intercepts HTTP traffic, forwards it to both live and shadow services, and captures response differences.

### Diffs
A **diff** is the computed difference between live and shadow service responses. mroki automatically compares JSON responses and tracks what changed.

### Hub
The **hub** is a web interface for managing gates, viewing captured requests, analyzing diffs, and monitoring agent health.

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

### [mroki-agent](docs/components/MROKI_AGENT.md)
Proxy that intercepts traffic and forwards to live/shadow services. Computes diffs and sends to API.

**Key Features:**
- Transparent HTTP proxy
- Exponential backoff retry logic
- Best-effort delivery (never fails live traffic)
- Agent ID persistence

### [mroki-api](docs/components/MROKI_API.md)
REST API for managing gates and storing captured traffic diffs.

**Key Features:**
- Gate management (CRUD)
- Request/response storage
- Diff persistence
- Health check endpoints

### [mroki-hub](docs/components/MROKI_HUB.md)
Web interface for visualizing diffs and managing the system.

**Key Features:**
- Gate dashboard
- Request browser
- Diff visualization
- Agent monitoring

### [caddy-mroki](docs/components/CADDY_MROKI.md)
Caddy module for integrating mroki proxy into Caddy server.

## Quick Start

Get mroki running in 5 minutes:

1. **Start PostgreSQL** - `docker-compose -f build/mroki-api/docker-compose.yaml up -d`
2. **Start API** - Configure and run mroki-api
3. **Create a Gate** - Define your live/shadow service pair
4. **Start Agent** - Run the proxy to capture traffic
5. **Send Traffic** - Test with sample requests

For detailed step-by-step instructions, see the [Quick Start Guide](docs/guides/QUICK_START.md).

## Documentation

### Architecture
- [System Overview](docs/architecture/OVERVIEW.md) - Architecture, data flow, and design
- [API Contracts](docs/architecture/API_CONTRACTS.md) - API endpoint specifications

### Components
- [mroki-agent](docs/components/MROKI_AGENT.md) - Agent documentation
- [mroki-api](docs/components/MROKI_API.md) - API documentation
- [mroki-hub](docs/components/MROKI_HUB.md) - Web UI documentation
- [caddy-mroki](docs/components/CADDY_MROKI.md) - Caddy module documentation

### Guides
- [Quick Start](docs/guides/QUICK_START.md) - Get started in 5 minutes
- [Development](docs/guides/DEVELOPMENT.md) - Development workflow
- [Deployment](docs/guides/DEPLOYMENT.md) - Production deployment

## Use Cases

### 1. API Refactoring
Test refactored endpoints against production traffic to ensure behavioral compatibility.

### 2. Performance Testing
Compare response times between current and optimized implementations under real load.

### 3. Database Migration
Validate that new database schema produces identical results to the old one.

### 4. Framework Upgrades
Test major framework version upgrades with confidence using real production patterns.

### 5. A/B Testing
Compare different algorithm implementations with actual production data.

## Technology Stack

- **Backend:** Go 1.21+
- **API Framework:** net/http (stdlib)
- **Database:** PostgreSQL 15+
- **Frontend:** Vue 3 + TypeScript
- **Build Tool:** Vite
- **UI Components:** Custom component library

## Development

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run specific component tests
go test ./cmd/mroki-agent/...
go test ./internal/domain/...

# Build binaries
go build -o mroki-agent ./cmd/mroki-agent
go build -o mroki-api ./cmd/mroki-api
```

## Project Status

**Current State:** Phase 1 Complete

- ✅ Agent → API integration
- ✅ Request capture and diff computation
- ✅ Retry logic with exponential backoff
- ✅ Agent ID persistence
- ✅ 237+ tests, 0 failures
- 🚧 Web UI (mroki-hub) - In development

## License

MIT License - see LICENSE file for details

## Contributing

Contributions welcome! Please read our contributing guidelines before submitting PRs.

## Support

For issues and questions:
- Open an issue on GitHub
- Check the [documentation](docs/)
- Review the [Quick Start Guide](docs/guides/QUICK_START.md)
