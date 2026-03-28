#!/bin/bash
# Apply all Atlas migration files in order.
# This script runs inside the PostgreSQL container as part of
# docker-entrypoint-initdb.d (before 99-seed.sql).

set -euo pipefail

for f in /migrations/*.sql; do
  echo "Applying migration: $(basename "$f")"
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$f"
done
