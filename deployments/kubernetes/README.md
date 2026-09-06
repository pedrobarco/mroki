# Kubernetes Deployment

> **Full guide:** [Production: Kubernetes](../../docs/production/KUBERNETES.md) — prerequisites, Helm chart, raw manifests, scaling, health probes, and uninstall.

Deployable manifests for a production-grade mroki deployment on Kubernetes.

## Files

- `namespace.yaml` — Namespace
- `secrets.yaml` — Secrets and ConfigMaps
- `postgres.yaml` — PostgreSQL StatefulSet with a 20Gi persistent volume
- `api.yaml` — mroki-api Deployment and Service
- `proxy.yaml` — mroki-proxy Deployment and Service
- `charts/mroki/` — Helm chart (the recommended install method)

## Quick start

```bash
kubectl apply -f {namespace,secrets,postgres,api,proxy}.yaml
kubectl get pods -n mroki
```

Edit `secrets.yaml` first to set the database URL, API key, and gate configuration. See the [full guide](../../docs/production/KUBERNETES.md) for Helm installation, configuration, scaling, and uninstall.
