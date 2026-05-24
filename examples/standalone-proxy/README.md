# Standalone Proxy Example

Run mroki-proxy as a single binary — no database, no API server. Diffs are printed to stdout.

## When to Use This

- Quick local testing of a shadow service against production
- CI/CD pipelines asserting zero-diff between service versions
- One-off comparisons without infrastructure setup

## Setup

### 1. Copy and Edit the Environment File

```bash
cp .env.example .env
# Edit .env with your actual live and shadow URLs
```

### 2. Build the Proxy

```bash
# From the repository root
go build -o mroki-proxy ./cmd/mroki-proxy
```

### 3. Run

```bash
./mroki-proxy
```

Or with Docker:

```bash
docker run --rm -p 8080:8080 --env-file .env ghcr.io/pedrobarco/mroki-proxy
```

## Test It

```bash
curl http://localhost:8080/any-path
```

When responses differ you'll see output like:

```
INFO response diff detected method=GET path=/any-path live_status=200 shadow_status=200 changes=2
```

## Full Guide

See the [Standalone Proxy Getting Started](../../docs/getting-started/STANDALONE_PROXY.md) guide for:

- Docker usage and options
- Detailed configuration reference
- Troubleshooting tips

For all available environment variables, see [Configuration](../../docs/production/CONFIGURATION.md).
