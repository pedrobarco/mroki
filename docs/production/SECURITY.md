# Security

Security considerations and hardening guidance for production mroki deployments.

See [Configuration](CONFIGURATION.md) for the full environment variable reference.

---

## API Key Authentication

Every request to the mroki API must include a valid API key via the `Authorization` header:

```
Authorization: Bearer <key>
```

The key is configured with the `MROKI_APP_API_KEY` environment variable and **must be at least 16 characters**. Keys are compared using `crypto/subtle.ConstantTimeCompare`, preventing timing side-channel attacks.

**Generating a strong key:**

```bash
# 32-byte random key, base64-encoded (44 characters)
openssl rand -base64 32
```

Unauthenticated or invalid requests receive an RFC 7807 error response with HTTP 401.

---

## Field Redaction

mroki replaces sensitive field values with `[REDACTED]` before storage. The following fields are redacted by default:

- `headers.Authorization`
- `headers.Cookie`
- `headers.Set-Cookie`
- `headers.X-Api-Key`

Fields use gjson path notation with a `headers.` or `body.` prefix (e.g. `body.user.password`).

Redacted fields are automatically excluded from diff computation so they don't produce false positives.

**Adding fields per gate (API mode):**

```bash
# Add extra redacted fields to an existing gate
curl -X PATCH /gates/{id} \
  -d '{"redacted_fields": ["headers.X-Internal-Token", "body.secret"]}'
```

**Adding fields in standalone proxy mode:**

```bash
# Comma-separated list — adds to the default set
MROKI_APP_REDACTED_FIELDS=headers.X-Internal-Token,body.user.password
```

---

## Rate Limiting

The API enforces a token-bucket rate limit of **1000 requests per minute per IP** (configurable via `MROKI_APP_RATE_LIMIT`). When exceeded, the API responds with:

- HTTP `429 Too Many Requests`
- `Retry-After` header indicating when to retry

---

## TLS / Network Security

mroki does **not** terminate TLS itself. Use a reverse proxy or load balancer (nginx, Caddy, cloud LB) to terminate HTTPS in front of the API.

**Example nginx snippet:**

```nginx
server {
    listen 443 ssl;
    server_name mroki.example.com;

    ssl_certificate     /etc/ssl/certs/mroki.crt;
    ssl_certificate_key /etc/ssl/private/mroki.key;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

The internal network between the proxy and the API should be trusted (e.g. same Kubernetes pod, private subnet) or secured with mutual TLS (mTLS). Proxy-to-API mTLS is planned but not yet implemented.

---

## Database Security

```bash
# Use a strong, randomly generated password
POSTGRES_PASSWORD=$(openssl rand -base64 32)

# Enable SSL for the database connection
MROKI_APP_DATABASE_URL=postgres://user:pass@host:5432/mroki?sslmode=require

# Restrict network access in pg_hba.conf
host    mroki    mroki    10.0.0.0/8    scram-sha-256
```

Additional recommendations:
- Create a dedicated database user with only the permissions mroki requires (SELECT, INSERT, UPDATE, DELETE on its tables)
- Store database credentials in a secrets manager (Kubernetes Secrets, AWS Secrets Manager, etc.)

---

## CORS

Cross-origin requests are controlled via `MROKI_APP_CORS_ORIGINS`. Set it to a comma-separated list of allowed origins:

```bash
MROKI_APP_CORS_ORIGINS=https://hub.example.com,https://admin.example.com
```

When configured, the API sets:

```
Access-Control-Allow-Origin: <configured origin>
Access-Control-Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 86400
```

If `MROKI_APP_CORS_ORIGINS` is empty or unset, CORS is disabled entirely — no `Access-Control-*` headers are sent.

---

## Production Checklist

- **Strong API key** — at least 16 characters, randomly generated (`openssl rand -base64 32`)
- **TLS termination** — place a reverse proxy or load balancer in front of the API
- **Database SSL** — append `?sslmode=require` to `MROKI_APP_DATABASE_URL`
- **Restrict network access** — firewall the API and database; deploy the proxy in an isolated network
- **Configure CORS origins** — set `MROKI_APP_CORS_ORIGINS` to only the domains that need access
- **Enable field redaction** — review default redacted fields; add application-specific fields per gate or via `MROKI_APP_REDACTED_FIELDS`
- **Monitor logs** — watch for 401/429 responses and unusual traffic patterns
