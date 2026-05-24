# Multi-Gate Caddy Configuration

Shadow different backends per route using multiple `mroki_gate` blocks.

## What This Does

- Listens on port 8080
- `/api/users/*` — shadows a v2 staging backend
- `/api/orders/*` — shadows an orders staging backend at 50 % sampling with diff tuning
- All other routes are proxied directly to production (no shadowing)

## Prerequisites

- Caddy with the mroki module built and installed

## Setup

### 1. Build Caddy with mroki Module

```bash
# From the repository root
cd build/package/caddy-mroki
xcaddy build --with github.com/pedrobarco/mroki/pkg/caddymodule
```

### 2. Update Caddyfile

Edit the `Caddyfile` and replace the example URLs with your actual service endpoints.

### 3. Run Caddy

```bash
./caddy run --config Caddyfile
```

## Test It

```bash
# Shadowed — users gate
curl http://localhost:8080/api/users/123

# Shadowed — orders gate (50 % sampled)
curl http://localhost:8080/api/orders/456

# Not shadowed — proxied directly
curl http://localhost:8080/health
```

## Full Guide

See the [Caddy Module Getting Started](../../../docs/getting-started/CADDY_MODULE.md) guide for the full directive reference, TLS setup, and production tips.
