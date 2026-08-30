---
title: API Reference
---

# API Reference

The REST API provided by `mroki-api`. It manages traffic-testing gates and the
requests captured against them, and exposes server configuration and aggregate
statistics.

This reference is generated from the multi-file **OpenAPI 3.1** spec
(`docs/api/openapi/`), bundled at prebuild time with every external `$ref`
resolved. Use the **sidebar** to open any single operation on its own page,
grouped by tag.

## Conventions

- **Responses** — all successful responses are wrapped in a `data` envelope.
- **Errors** — error responses follow [RFC 7807](https://www.rfc-editor.org/rfc/rfc7807)
  (Problem Details for HTTP APIs).
- **Authentication** — every endpoint except the infrastructure endpoints
  (`/health/*`, `/metrics`) requires a bearer token, supplied as
  `Authorization: Bearer <your-api-key>`.

## Base URL

| Environment       | URL                     |
| ----------------- | ----------------------- |
| Local development | `http://localhost:8090` |

## Operations by tag

| Tag          | Description                                                 |
| ------------ | ----------------------------------------------------------- |
| **Config**   | Read-only, server-wide configuration.                       |
| **Stats**    | Cross-gate aggregate statistics.                            |
| **Gates**    | Traffic-testing gates with live and shadow URLs.            |
| **Requests** | Requests captured and compared against a gate.              |
| **Health**   | Kubernetes liveness and readiness probes (unauthenticated). |
| **Metrics**  | Prometheus metrics scrape endpoint (unauthenticated).       |
