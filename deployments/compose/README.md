# Compose Deployment

> **Full docs:** see the [Deployment Guide](../../docs/guides/DEPLOYMENT.md) for all deployment options.

Full-stack deployment of mroki using Docker Compose.

## Files

- `full-stack.yaml` - Complete deployment with PostgreSQL, API, and Proxy

## Usage

1. **Create `.env` file:**
   ```bash
   DB_PASSWORD=your_secure_password
   LIVE_URL=https://api.production.example.com
   SHADOW_URL=https://api.shadow.example.com
   GATE_ID=550e8400-e29b-41d4-a716-446655440000
   ```

2. **Deploy:**
   ```bash
   docker compose -f full-stack.yaml up -d
   ```

3. **Check status:**
   ```bash
   docker compose -f full-stack.yaml ps
   ```

4. **View logs:**
   ```bash
   docker compose -f full-stack.yaml logs -f
   ```

5. **Stop:**
   ```bash
   docker compose -f full-stack.yaml down
   ```

## Services

- **mroki-db**: PostgreSQL 15 database on port 5432
- **mroki-api**: API server on port 8081
- **mroki-proxy**: Proxy on port 8080

## Requirements

- Docker 20.10+
- Docker Compose v2+

## Notes

- Data persists in `mroki-db-data` volume
- All services restart automatically unless stopped
- Configure firewall rules for production deployments
