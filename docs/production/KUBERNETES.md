# Production: Kubernetes

Deploy mroki on Kubernetes for team/org-wide rollouts with high availability, horizontal autoscaling, and fine-grained resource control.

## Overview

Use Kubernetes when you need:

- **High availability** — multiple replicas across nodes with automatic failover
- **Autoscaling** — HPA on API and proxy pods based on CPU/memory
- **Multi-team isolation** — namespace-scoped deployments with RBAC
- **Sidecar pattern** — embed mroki-proxy directly in application pods

For simpler single-host setups, see [Docker Compose](DOCKER_COMPOSE.md).

## Helm Chart

The recommended approach. Chart source: [`deployments/kubernetes/charts/mroki/`](../../deployments/kubernetes/charts/mroki/).

```bash
helm install mroki oci://ghcr.io/pedrobarco/mroki/charts/mroki \
  --namespace mroki --create-namespace \
  --set secrets.databaseUrl="postgres://user:pass@db:5432/mroki"
```

See the chart README for all configurable values.

### Database migrations

Schema migrations are applied by the `mroki-db-migrator` image (Atlas), not the API. The chart runs it as a Helm `pre-install,pre-upgrade` Job hook, so migrations execute exactly once per `helm install`/`helm upgrade` — before the API pods roll out.

- Enabled by default (`api.migration.enabled=true`). The Job is created when either `api.existingSecret` or `api.database.passwordSecret` is set.
- To apply a new migration, run `helm upgrade` with a chart/image version that bundles it.
- **Existing databases:** a database previously managed by the API's old auto-migration has no Atlas revision table, so the migrator fails with `not clean: ... baseline version ... required`. Set the baseline to the schema version already present to mark those migrations as applied without re-running them:

  ```bash
  helm upgrade mroki oci://ghcr.io/pedrobarco/mroki/charts/mroki \
    --set api.migration.baseline=20260328015306
  ```

#### GitOps consumers (`helm template` + `kubectl apply`)

Consumers that render the chart with `helm template` and apply it declaratively (Kustomize `helmCharts` with `--enable-helm`, Argo CD / Flux in template mode) do **not** run Helm's hook engine. With the default static Job name and inert `helm.sh/hook*` annotations, every upgrade fails because a Job `spec` is immutable. Set `api.migration.asHook=false` to emit the migration Job as a plain, declaratively-managed resource instead:

```bash
helm template mroki oci://ghcr.io/pedrobarco/mroki/charts/mroki \
  --set api.migration.asHook=false \
  --set api.migration.ttlSecondsAfterFinished=300 \
  | kubectl apply -f -
```

- `api.migration.asHook` (default `true`) — when `false`, the `helm.sh/hook*` annotations are dropped.
- The Job name is suffixed with the chart appVersion (override with `api.migration.nameSuffix`) so each version produces a fresh Job rather than mutating an immutable one on `kubectl apply`.
- `api.migration.ttlSecondsAfterFinished` — sets `Job.spec.ttlSecondsAfterFinished` so completed Jobs are garbage-collected automatically. Applies in both modes; leave null to keep finished Jobs.

The default (`asHook=true`) is unchanged for native `helm install`/`helm upgrade` users.

## Raw Manifests

Apply manifests from [`deployments/kubernetes/`](../../deployments/kubernetes/):

```bash
kubectl apply -f deployments/kubernetes/{namespace,secrets,postgres,api,proxy}.yaml
kubectl get pods -n mroki
```

### Secrets

Store credentials in a Kubernetes Secret (or use an external secrets operator):

```yaml
apiVersion: v1
kind: Secret
metadata: { name: mroki-secrets, namespace: mroki }
type: Opaque
stringData:
  database-url: "postgres://apiuser:pass@postgres:5432/mroki?sslmode=require"
  api-key: "your-api-key"
```

### mroki-api (3 replicas, port 8090) — [full manifest](../../deployments/kubernetes/api.yaml)

```yaml
containers:
- name: mroki-api
  image: mroki-api:latest
  ports:
  - containerPort: 8090
  env:
  - name: MROKI_APP_DATABASE_URL
    valueFrom:
      secretKeyRef: { name: mroki-secrets, key: database-url }
  - name: MROKI_APP_API_KEY
    valueFrom:
      secretKeyRef: { name: mroki-secrets, key: api-key }
  livenessProbe:
    httpGet: { path: /health/live, port: 8090 }
  readinessProbe:
    httpGet: { path: /health/ready, port: 8090 }
  resources:
    requests: { memory: "128Mi", cpu: "100m" }
    limits:   { memory: "512Mi", cpu: "500m" }
```

Service: `ClusterIP` port 80 → 8090.

### mroki-proxy (2 replicas, port 8080) — [full manifest](../../deployments/kubernetes/proxy.yaml)

```yaml
containers:
- name: mroki-proxy
  image: mroki-proxy:latest
  ports:
  - containerPort: 8080
  env:
  - name: MROKI_APP_API_URL
    value: "http://mroki-api:8090"
  - name: MROKI_APP_GATE_ID
    value: "550e8400-e29b-41d4-a716-446655440000"
  - name: MROKI_APP_API_KEY
    valueFrom:
      secretKeyRef: { name: mroki-secrets, key: api-key }
```

Service: `ClusterIP` port 80 → 8080.

### PostgreSQL

Use a managed database (Cloud SQL, RDS, Azure Database) in production. For an in-cluster instance, see [`postgres.yaml`](../../deployments/kubernetes/postgres.yaml).

## Sidecar Pattern

Inject mroki-proxy as a sidecar for per-pod proxying with localhost access and no extra network hop:

```yaml
spec:
  template:
    spec:
      containers:
      - name: app                  # your application
        image: my-app:latest
        ports: [{ containerPort: 3000 }]
      - name: mroki-proxy          # sidecar
        image: mroki-proxy:latest
        ports: [{ containerPort: 8080 }]
        env:
        - name: MROKI_APP_API_URL
          value: "http://mroki-api:8090"
        - name: MROKI_APP_GATE_ID
          valueFrom:
            configMapKeyRef: { name: mroki-config, key: gate-id }
        - name: MROKI_APP_API_KEY
          valueFrom:
            secretKeyRef: { name: mroki-secrets, key: api-key }
```

Then point your Service's `targetPort` at `8080` (the proxy) instead of `3000` (the app) so traffic flows through mroki-proxy.

## Health Probes

mroki-api exposes health endpoints for Kubernetes probes:

| Probe | Path | Port | Period | Failure Threshold |
|-------|------|------|--------|-------------------|
| Liveness | `/health/live` | 8090 | 10s | 3 |
| Readiness | `/health/ready` | 8090 | 5s | 2 |
| Startup | `/health/ready` | 8090 | 5s | 12 |

The startup probe gives the API up to 60s to initialize before liveness checks begin.

## Scaling

API and proxy are both **stateless** — scale horizontally:

```bash
kubectl scale deployment mroki-api --replicas=5 -n mroki
kubectl scale deployment mroki-proxy --replicas=10 -n mroki
```

Or use a HorizontalPodAutoscaler targeting CPU/memory. **PostgreSQL is the bottleneck** — use connection pooling (PgBouncer) and read replicas (`MROKI_APP_DATABASE_READ_URL`) for high-throughput workloads.

## What's Next

- [Docker Compose](DOCKER_COMPOSE.md) — simpler single-host deployment
- [Security](SECURITY.md) — TLS, authentication, network policies
- [Monitoring](MONITORING.md) — metrics, logging, alerting
- [Configuration](CONFIGURATION.md) — full environment variable reference
