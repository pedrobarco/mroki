# Kubernetes Deployment

> **Full docs:** see the [Deployment Guide](../../docs/guides/DEPLOYMENT.md) for all deployment options.

Production-grade Kubernetes deployment for mroki.

## Files

- `namespace.yaml` - Namespace creation
- `secrets.yaml` - Secrets and ConfigMaps
- `postgres.yaml` - PostgreSQL StatefulSet with persistent storage
- `api.yaml` - mroki-api Deployment and Service
- `agent.yaml` - mroki-agent Deployment and Service

## Prerequisites

- Kubernetes 1.20+
- kubectl configured
- Persistent volume provisioner (for PostgreSQL storage)

## Deployment

1. **Edit secrets** in `secrets.yaml`:
   ```yaml
   stringData:
     db-password: "your_actual_secure_password"
     database-url: "postgres://postgres:your_actual_secure_password@postgres:5432/mroki"
   ```

2. **Edit configuration** in `secrets.yaml`:
   ```yaml
   data:
     live-url: "https://api.production.example.com"
     shadow-url: "https://api.shadow.example.com"
     gate-id: "your-gate-id-here"
   ```

3. **Deploy in order:**
   ```bash
   kubectl apply -f namespace.yaml
   kubectl apply -f secrets.yaml
   kubectl apply -f postgres.yaml
   kubectl apply -f api.yaml
   kubectl apply -f agent.yaml
   ```

4. **Verify deployment:**
   ```bash
   kubectl get pods -n mroki
   kubectl get services -n mroki
   ```

## Scaling

Scale API replicas:
```bash
kubectl scale deployment mroki-api --replicas=5 -n mroki
```

Scale agent replicas:
```bash
kubectl scale deployment mroki-agent --replicas=10 -n mroki
```

## Monitoring

View logs:
```bash
kubectl logs -n mroki -l app=mroki-api -f
kubectl logs -n mroki -l app=mroki-agent -f
```

Check pod status:
```bash
kubectl describe pod -n mroki <pod-name>
```

## Health Checks

The API deployment includes:
- **Liveness probe**: `/health/live` on port 8081
- **Readiness probe**: `/health/ready` on port 8081

Kubernetes automatically restarts unhealthy pods.

## Resource Limits

**mroki-api:**
- Request: 128Mi memory, 100m CPU
- Limit: 512Mi memory, 500m CPU

**mroki-agent:**
- Request: 64Mi memory, 50m CPU
- Limit: 256Mi memory, 200m CPU

Adjust these values based on your workload in the respective YAML files.

## Storage

PostgreSQL uses a StatefulSet with 20Gi persistent volume. Data persists across pod restarts.

## Uninstall

```bash
kubectl delete -f agent.yaml
kubectl delete -f api.yaml
kubectl delete -f postgres.yaml
kubectl delete -f secrets.yaml
kubectl delete -f namespace.yaml
```

**Warning:** This will delete all data including the database.
