# Production: Docker Compose

Deploy mroki on a single host using Docker Compose.

## Overview

Docker Compose is the simplest production deployment option. Use it when:

- You're running on a **single host** (VM, bare-metal, or VPS)
- You want **minimal operational overhead** — no orchestrator to manage
- Your scale is **small to medium** (one proxy instance is sufficient)

For larger or multi-node deployments, see [Kubernetes](KUBERNETES.md).

## Prerequisites

- Docker Engine 24+
- Docker Compose v2 (`docker compose` — not the legacy `docker-compose`)

## Deploy

### 1. Create a `.env` file

```bash
cat > .env <<'EOF'
DB_PASSWORD=change-me-strong-password-here
API_KEY=production-api-key-min-16-chars
GATE_ID=550e8400-e29b-41d4-a716-446655440000
EOF
```

### 2. Create `docker-compose.yaml`

```yaml
services:
  postgres:
    image: postgres:15-alpine
    restart: always
    environment:
      POSTGRES_USER: mroki
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: mroki
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mroki"]
      interval: 10s
      timeout: 5s
      retries: 5

  mroki-db-migrator:
    image: ghcr.io/pedrobarco/mroki/mroki-db-migrator:latest
    command: ["--url", "postgres://mroki:${DB_PASSWORD}@postgres:5432/mroki?sslmode=disable"]
    restart: "no"
    depends_on:
      postgres:
        condition: service_healthy

  mroki-api:
    image: ghcr.io/pedrobarco/mroki-api:latest
    restart: always
    ports:
      - "8090:8090"
    environment:
      MROKI_APP_PORT: 8090
      MROKI_APP_DATABASE_URL: postgres://mroki:${DB_PASSWORD}@postgres:5432/mroki
      MROKI_APP_API_KEY: ${API_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      mroki-db-migrator:
        condition: service_completed_successfully

  mroki-proxy:
    image: ghcr.io/pedrobarco/mroki-proxy:latest
    restart: always
    ports:
      - "8080:8080"
    environment:
      MROKI_APP_PORT: 8080
      MROKI_APP_API_URL: http://mroki-api:8090
      MROKI_APP_GATE_ID: ${GATE_ID}
      MROKI_APP_API_KEY: ${API_KEY}
    depends_on:
      - mroki-api

  mroki-hub:
    image: ghcr.io/pedrobarco/mroki-hub:latest
    restart: always
    ports:
      - "3000:80"
    environment:
      MROKI_APP_API_BASE_URL: http://mroki-api:8090
      MROKI_APP_API_KEY: ${API_KEY}
    depends_on:
      - mroki-api

volumes:
  postgres_data:
```

### 3. Start the stack

```bash
docker compose up -d
docker compose ps
```

## Configuration

See [Configuration](CONFIGURATION.md) for all available environment variables, TLS setup, and tuning options.

## Systemd Alternative

If you prefer running binaries directly on a Linux host instead of Docker:

```bash
# Create a dedicated user
sudo useradd -r -s /bin/false mroki
sudo mkdir -p /opt/mroki
sudo cp mroki-api mroki-proxy /opt/mroki/
sudo chown -R mroki:mroki /opt/mroki
```

Create unit files under `/etc/systemd/system/`:

**mroki-api.service:**
```ini
[Unit]
Description=mroki API Server
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=mroki
Group=mroki
WorkingDirectory=/opt/mroki
Environment="MROKI_APP_PORT=8090"
Environment="MROKI_APP_DATABASE_URL=postgres://mroki:password@localhost:5432/mroki"
Environment="MROKI_APP_API_KEY=production-api-key-min-16-chars"
ExecStart=/opt/mroki/mroki-api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mroki-api mroki-proxy
```

## Backup & Recovery

```bash
# Backup (from the Compose host)
docker compose exec postgres pg_dump -U mroki mroki > mroki_backup_$(date +%Y%m%d).sql

# Restore
docker compose exec -T postgres psql -U mroki mroki < mroki_backup_20260131.sql

# Automated daily backup via cron
0 2 * * * cd /path/to/stack && docker compose exec -T postgres pg_dump -U mroki mroki | gzip > /backup/mroki_$(date +\%Y\%m\%d).sql.gz
```

## Updating

```bash
docker compose pull
docker compose up -d
```

Compose recreates only the containers whose images changed.

## What's Next

- [Kubernetes](KUBERNETES.md) — multi-node and high-availability deployments
- [Security](SECURITY.md) — TLS, network policies, secret management
- [Monitoring](MONITORING.md) — metrics, alerts, and dashboards
- [Configuration](CONFIGURATION.md) — full environment variable reference
