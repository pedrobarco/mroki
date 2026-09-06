# Compose Deployment

> **Full guide:** [Production: Docker Compose](../../docs/production/DOCKER_COMPOSE.md) — configuration, systemd, backup/recovery, and updates.

Ready-to-use Docker Compose stack for mroki: PostgreSQL, DB migrator, API, proxy, and hub.

## Files

- `full-stack.yaml` — complete stack (PostgreSQL, DB migrator, API, proxy, hub)

## Quick start

Provide `DB_PASSWORD`, `LIVE_URL`, `SHADOW_URL`, and `GATE_ID` (plus optional `MROKI_APP_API_BASE_URL` and `MROKI_APP_API_KEY`) via a `.env` file or the environment, then:

```bash
docker compose -f full-stack.yaml up -d
docker compose -f full-stack.yaml ps
```

See the [full guide](../../docs/production/DOCKER_COMPOSE.md) for configuration, backups, and updates.
