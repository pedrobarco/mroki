# Deployment Guide

Guide for deploying mroki to production environments.

## Deployment Options

### 1. Docker Compose (Simple)

Best for: Small deployments, single-server setups

See [`/deployments/docker-compose/`](../../deployments/docker-compose/) for complete manifests.

**Quick start:**
```bash
# Create .env file
cat > .env <<EOF
DB_PASSWORD=your_secure_password
API_KEY=your-api-key-min-16-chars
GATE_ID=550e8400-e29b-41d4-a716-446655440000
EOF

# Deploy
docker-compose -f deployments/docker-compose/full-stack.yaml up -d

# Check status
docker-compose -f deployments/docker-compose/full-stack.yaml ps
```

---

### 2. Kubernetes (Production)

Best for: Large deployments, high availability, auto-scaling

See [`/deployments/kubernetes/`](../../deployments/kubernetes/) for complete manifests.

**Quick start:**
```bash
# Edit secrets and config in deployments/kubernetes/secrets.yaml

# Deploy all components
kubectl apply -f deployments/kubernetes/namespace.yaml
kubectl apply -f deployments/kubernetes/secrets.yaml
kubectl apply -f deployments/kubernetes/postgres.yaml
kubectl apply -f deployments/kubernetes/api.yaml
kubectl apply -f deployments/kubernetes/agent.yaml

# Check status
kubectl get pods -n mroki
kubectl get services -n mroki
```

**Manifests included:**
- `namespace.yaml` - Creates mroki namespace
- `secrets.yaml` - Secrets and ConfigMaps for credentials
- `postgres.yaml` - PostgreSQL StatefulSet with persistent storage
- `api.yaml` - mroki-api Deployment (3 replicas) and Service
- `agent.yaml` - mroki-agent Deployment (2 replicas) and Service

**Health checks:**
- API includes liveness probe on `/health/live`
- API includes readiness probe on `/health/ready`
- Automatic pod restart on failures

---

### 3. Systemd (Traditional Servers)

Best for: VMs, traditional server deployments

**`/etc/systemd/system/mroki-api.service`:**
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
Environment="MROKI_APP_PORT=8081"
Environment="MROKI_APP_DATABASE_URL=postgres://mroki:password@localhost:5432/mroki"
ExecStart=/opt/mroki/mroki-api
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**`/etc/systemd/system/mroki-agent.service`:**
```ini
[Unit]
Description=mroki Agent
After=network.target mroki-api.service
Requires=mroki-api.service

[Service]
Type=simple
User=mroki
Group=mroki
WorkingDirectory=/opt/mroki
Environment="MROKI_APP_PORT=8080"
Environment="MROKI_APP_API_URL=http://localhost:8081"
Environment="MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000"
Environment="MROKI_APP_API_KEY=your-secret-key-min-16-chars"
Environment="MROKI_APP_DIFF_IGNORED_FIELDS=timestamp,created_at"
ExecStart=/opt/mroki/mroki-agent
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Setup:**
```bash
# Create user
sudo useradd -r -s /bin/false mroki

# Copy binaries
sudo mkdir -p /opt/mroki
sudo cp mroki-api mroki-agent /opt/mroki/
sudo chown -R mroki:mroki /opt/mroki

# Enable and start services
sudo systemctl daemon-reload
sudo systemctl enable mroki-api mroki-agent
sudo systemctl start mroki-api mroki-agent

# Check status
sudo systemctl status mroki-api
sudo systemctl status mroki-agent
```

---

## Security Considerations

### PostgreSQL

```bash
# Use strong password
POSTGRES_PASSWORD=$(openssl rand -base64 32)

# Enable SSL in production
MROKI_APP_DATABASE_URL=postgres://user:pass@host:5432/mroki?sslmode=require

# Restrict network access (pg_hba.conf)
host    mroki    mroki    10.0.0.0/8    scram-sha-256
```

### API

```bash
# Use reverse proxy (nginx) for HTTPS
# Future: Add API key authentication
```

### Agent

```bash
# Restrict network access (firewall)
# Only allow traffic from trusted sources
# Future: mTLS for agent-to-API communication
```

---

## Monitoring

### Logs

**Centralized logging with journald:**
```bash
# View API logs
journalctl -u mroki-api -f

# View agent logs
journalctl -u mroki-agent -f
```

**Kubernetes logs:**
```bash
kubectl logs -n mroki -l app=mroki-api -f
kubectl logs -n mroki -l app=mroki-agent -f
```

### Health Checks

```bash
# API health
curl http://localhost:8081/health/live
curl http://localhost:8081/health/ready

# Kubernetes probes handle this automatically
```

### Metrics (Future)

Prometheus metrics planned for future release.

---

## Backup & Recovery

### Database Backup

```bash
# Backup
pg_dump -U postgres mroki > mroki_backup_$(date +%Y%m%d).sql

# Restore
psql -U postgres mroki < mroki_backup_20260131.sql
```

**Automated backups:**
```bash
# Cron job
0 2 * * * /usr/bin/pg_dump -U postgres mroki | gzip > /backup/mroki_$(date +\%Y\%m\%d).sql.gz
```

---

## Scaling

### Horizontal Scaling

**API:** Stateless - scale freely
```bash
# Kubernetes
kubectl scale deployment mroki-api --replicas=5 -n mroki
```

**Agent:** Stateless - scale freely
```bash
# Kubernetes
kubectl scale deployment mroki-agent --replicas=10 -n mroki
```

**Database:** Use read replicas for queries
```yaml
# Future: Configure read replica endpoints
MROKI_APP_DATABASE_READ_URL=postgres://replica:5432/mroki
```

---

## Troubleshooting

### Check Service Status

```bash
# Docker Compose
docker-compose ps
docker-compose logs mroki-api
docker-compose logs mroki-agent

# Kubernetes
kubectl get pods -n mroki
kubectl describe pod mroki-api-xxx -n mroki
kubectl logs mroki-api-xxx -n mroki

# Systemd
systemctl status mroki-api
systemctl status mroki-agent
```

### Common Issues

**Database connection fails:**
```bash
# Test connection
psql -U postgres -h localhost -d mroki

# Check firewall
sudo ufw status
```

**Agent can't reach API:**
```bash
# Test connectivity
curl http://mroki-api:8081/health/live
```

---

## Related Documentation

- [Quick Start Guide](QUICK_START.md)
- [Development Guide](DEVELOPMENT.md)
- [Architecture Overview](../architecture/OVERVIEW.md)
