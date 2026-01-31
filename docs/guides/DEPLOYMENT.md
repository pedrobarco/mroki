# Deployment Guide

Guide for deploying mroki to production environments.

## Deployment Options

### 1. Docker Compose (Simple)

Best for: Small deployments, single-server setups

**`docker-compose.yml`:**
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: mroki
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  mroki-api:
    image: mroki-api:latest
    ports:
      - "8081:8081"
    environment:
      MROKI_APP_PORT: 8081
      MROKI_APP_DATABASE_URL: postgres://postgres:${DB_PASSWORD}@postgres:5432/mroki
    depends_on:
      - postgres
    restart: unless-stopped

  mroki-agent:
    image: mroki-agent:latest
    ports:
      - "8080:8080"
    environment:
      MROKI_APP_LIVE_URL: ${LIVE_URL}
      MROKI_APP_SHADOW_URL: ${SHADOW_URL}
      MROKI_APP_PORT: 8080
      MROKI_APP_API_URL: http://mroki-api:8081
      MROKI_APP_GATE_ID: ${GATE_ID}
    depends_on:
      - mroki-api
    restart: unless-stopped

volumes:
  postgres_data:
```

**`.env`:**
```bash
DB_PASSWORD=your_secure_password
LIVE_URL=https://api.production.example.com
SHADOW_URL=https://api.shadow.example.com
GATE_ID=550e8400-e29b-41d4-a716-446655440000
```

**Deploy:**
```bash
docker-compose up -d
```

---

### 2. Kubernetes (Production)

Best for: Large deployments, high availability, auto-scaling

**`kubernetes/namespace.yaml`:**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: mroki
```

**`kubernetes/postgres.yaml`:**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: mroki
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: mroki
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: db-password
        - name: POSTGRES_DB
          value: mroki
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 20Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: mroki
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
  clusterIP: None
```

**`kubernetes/api.yaml`:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mroki-api
  namespace: mroki
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mroki-api
  template:
    metadata:
      labels:
        app: mroki-api
    spec:
      containers:
      - name: mroki-api
        image: mroki-api:v1.0.0
        ports:
        - containerPort: 8081
        env:
        - name: MROKI_APP_PORT
          value: "8081"
        - name: MROKI_APP_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: mroki-secrets
              key: database-url
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: mroki-api
  namespace: mroki
spec:
  selector:
    app: mroki-api
  ports:
  - port: 80
    targetPort: 8081
  type: ClusterIP
```

**`kubernetes/agent.yaml`:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mroki-agent
  namespace: mroki
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mroki-agent
  template:
    metadata:
      labels:
        app: mroki-agent
    spec:
      containers:
      - name: mroki-agent
        image: mroki-agent:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: MROKI_APP_LIVE_URL
          valueFrom:
            configMapKeyRef:
              name: mroki-config
              key: live-url
        - name: MROKI_APP_SHADOW_URL
          valueFrom:
            configMapKeyRef:
              name: mroki-config
              key: shadow-url
        - name: MROKI_APP_PORT
          value: "8080"
        - name: MROKI_APP_API_URL
          value: "http://mroki-api"
        - name: MROKI_APP_GATE_ID
          valueFrom:
            configMapKeyRef:
              name: mroki-config
              key: gate-id
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "256Mi"
            cpu: "200m"
---
apiVersion: v1
kind: Service
metadata:
  name: mroki-agent
  namespace: mroki
spec:
  selector:
    app: mroki-agent
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

**`kubernetes/secrets.yaml`:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mroki-secrets
  namespace: mroki
type: Opaque
stringData:
  db-password: "your_secure_password"
  database-url: "postgres://postgres:your_secure_password@postgres:5432/mroki"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mroki-config
  namespace: mroki
data:
  live-url: "https://api.production.example.com"
  shadow-url: "https://api.shadow.example.com"
  gate-id: "550e8400-e29b-41d4-a716-446655440000"
```

**Deploy to Kubernetes:**
```bash
kubectl apply -f kubernetes/namespace.yaml
kubectl apply -f kubernetes/secrets.yaml
kubectl apply -f kubernetes/postgres.yaml
kubectl apply -f kubernetes/api.yaml
kubectl apply -f kubernetes/agent.yaml
```

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
Environment="MROKI_APP_LIVE_URL=https://api.production.example.com"
Environment="MROKI_APP_SHADOW_URL=https://api.shadow.example.com"
Environment="MROKI_APP_PORT=8080"
Environment="MROKI_APP_API_URL=http://localhost:8081"
Environment="MROKI_APP_GATE_ID=550e8400-e29b-41d4-a716-446655440000"
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
