# Getting Started: Standalone Proxy

Run mroki-proxy as a single binary — diffs printed to stdout, no database required.

## When to Use This

- **Quick local testing** — verify that a shadow service matches production behavior
- **CI/CD pipelines** — assert zero-diff between service versions in automated runs
- **One-off comparisons** — spot-check a new deployment without setting up infrastructure

> **Note:** Standalone mode does not persist diffs. Results are logged to stdout only. For persistence and a web UI, see [Full Stack](FULL_STACK.md).

## Prerequisites

- **Go 1.26+** (to build from source), _or_
- **Docker**

## Step 1: Build or Pull

**Option A — Build from source:**

```bash
go build -o mroki-proxy ./cmd/mroki-proxy
```

**Option B — Docker:**

```bash
docker pull ghcr.io/pedrobarco/mroki-proxy
```

## Step 2: Configure

Create a `.env` file in the directory where you'll run the binary:

```bash
cat > .env << 'EOF'
MROKI_APP_LIVE_URL=https://httpbin.org/anything?env=live
MROKI_APP_SHADOW_URL=https://httpbin.org/anything?env=shadow
MROKI_APP_PORT=8080
EOF
```

| Variable | Description |
|----------|-------------|
| `MROKI_APP_LIVE_URL` | The "production" upstream. Its response is returned to the caller. |
| `MROKI_APP_SHADOW_URL` | The "experimental" upstream. Its response is compared against live. |
| `MROKI_APP_PORT` | Port the proxy listens on (default `8080`). |

## Step 3: Run the Proxy

**From source build:**

```bash
./mroki-proxy
```

**With Docker:**

```bash
docker run --rm -p 8080:8080 \
  -e MROKI_APP_LIVE_URL=https://httpbin.org/anything?env=live \
  -e MROKI_APP_SHADOW_URL=https://httpbin.org/anything?env=shadow \
  ghcr.io/pedrobarco/mroki-proxy
```

Expected startup output:

```
INFO Starting in standalone mode live_url=https://httpbin.org/... shadow_url=https://httpbin.org/...
INFO Started server address=:8080
```

## Step 4: Send Traffic

```bash
curl http://localhost:8080/get
```

When the live and shadow responses differ, the proxy prints a diff to stdout:

```
INFO response diff detected method=GET path=/get live_status=200 shadow_status=200 changes=2
Diff:
 ~ /body/user: "alice" → "bob"
```

If the responses are identical, no diff output is produced.

## Optional: Tune Diff Options

Fine-tune which fields are compared via environment variables:

| Variable | Purpose |
|----------|---------|
| `MROKI_APP_DIFF_IGNORED_FIELDS` | Comma-separated field paths to skip (e.g. `timestamp,id`) |
| `MROKI_APP_DIFF_INCLUDED_FIELDS` | Whitelist — only these fields are compared |
| `MROKI_APP_DIFF_FLOAT_TOLERANCE` | Tolerance for float comparison (e.g. `0.001`) |
| `MROKI_APP_REDACTED_FIELDS` | Fields redacted before diff (e.g. `headers.X-Internal-Token`) |

Field paths use [gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md). See [Configuration](../production/CONFIGURATION.md) for the full reference.

## What's Next

- **[Full Stack](FULL_STACK.md)** — add mroki-api and the web UI for persistence and visual diffs
- **[Configuration](../production/CONFIGURATION.md)** — all environment variables and tuning options
- **[Troubleshooting](../TROUBLESHOOTING.md)** — common issues and solutions
