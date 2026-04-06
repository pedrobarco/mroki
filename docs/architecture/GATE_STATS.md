# Gate Stats Design

> **Status:** ✅ Implemented (2026-04-06)

This document describes the design for adding statistics to gate responses, covering both per-gate stats and global stats.

## Problem

The UI displays hardcoded dummy data for gate metrics (request counts, diff rates, etc.). The API currently returns no statistics — the frontend would need to call multiple endpoints per page and compute aggregates client-side.

## Goals

1. **Per-gate stats** embedded in existing gate responses (`GET /gates`, `GET /gates/{id}`)
2. **Global stats** via a new endpoint (`GET /stats`) for the dashboard stats bar
3. Single API call per page — no N+1 requests from the frontend

## Non-Goals

- Real-time / WebSocket stats
- Historical time-series data


---

## API Design

### Per-Gate Stats — Embedded in Gate Responses

Stats are included as a nested `stats` object on every gate response. No new endpoints needed.

**`GET /gates/{id}` response:**

```json
{
  "data": {
    "id": "a1b2c3d4-...",
    "name": "payments-api",
    "live_url": "https://api.example.com",
    "shadow_url": "https://shadow.example.com",
    "created_at": "2026-03-15T10:00:00Z",
    "stats": {
      "request_count_24h": 347,
      "diff_count_24h": 87,
      "diff_rate": 25.07,
      "last_active": "2026-03-29T14:32:05Z"
    }
  }
}
```

**`GET /gates` response** — same `stats` object per gate in the list.

**Stats fields:**

| Field | Type | Description |
|-------|------|-------------|
| `request_count_24h` | `int64` | Request count in the last 24 hours |
| `diff_count_24h` | `int64` | Number of requests with diffs in the last 24 hours |
| `diff_rate` | `float64` | `diff_count_24h / request_count_24h * 100`, `0.0` when no requests |
| `last_active` | `string?` | RFC 3339 timestamp of the most recent request, `null` if no requests |

### Global Stats — New Endpoint

**`GET /stats`**

Returns cross-gate aggregate statistics for the dashboard stats bar.

```json
{
  "data": {
    "total_gates": 4,
    "total_requests_24h": 5241,
    "total_diff_rate": 4.48
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `total_gates` | `int64` | Total number of gates |
| `total_requests_24h` | `int64` | Request count in the last 24 hours across all gates |
| `total_diff_rate` | `float64` | `total_diffs_24h / total_requests_24h * 100`, `0.0` when no requests |

**Errors:** Only `500 Internal Server Error` (no user input to validate).

---

## Data Sources

All stats are derived from existing tables. No new schema required.

| Stat | Source |
|------|--------|
| `request_count_24h` | ent: `Request.Query().Where(GateIDIn(...), CreatedAtGTE(since24h)).GroupBy(FieldGateID).Aggregate(Count())` |
| `diff_count_24h` | ent: `Request.Query().Where(GateIDIn(...), CreatedAtGTE(since24h), HasDiff()).GroupBy(FieldGateID).Aggregate(Count())` |
| `diff_rate` | Computed in Go: `diff_count_24h / request_count_24h * 100` |
| `last_active` | ent: `Request.Query().Where(GateIDIn(...)).GroupBy(FieldGateID).Aggregate(Max(FieldCreatedAt))` |

**Existing indexes that support these queries:**

- `requests(gate_id)` — filters by gate
- `requests(gate_id, created_at)` — time-windowed counts

Global stats use similar queries without `GroupBy`/`GateIDIn` — just `Count()` on the full table.

---

## Implementation — Layer by Layer

### Domain Layer (`internal/domain/traffictesting/`)

**`gate_stats.go`** — per-gate stats value object:

```go
type GateStats struct {
    RequestCount24h int64
    DiffCount24h    int64
    DiffRate        float64
    LastActive      *time.Time
}
```

**`gate.go`** — `Stats GateStats` field added to `Gate` struct. Zero-valued by default, populated by persistence layer.

**`global_stats.go`** — cross-gate aggregates:

```go
type GlobalStats struct {
    TotalGates       int64
    TotalRequests24h int64
    TotalDiffRate    float64
}
```

**`stats_repository.go`** — new interface:

```go
type StatsRepository interface {
    GetGlobalStats(ctx context.Context) (*GlobalStats, error)
}
```

No changes to `GateRepository` interface — `GetByID` and `GetAll` populate `Stats` as part of their existing return values.

### Persistence Layer (`internal/infrastructure/persistence/ent/`)

**`gate_repository.go`** — private `attachStats(ctx, []*Gate)` method runs 3 batch ent queries:

1. Request count 24h per gate — `GroupBy(FieldGateID).Aggregate(Count())`
2. Diff count 24h per gate — same with `HasDiff()` predicate
3. Last active per gate — `GroupBy(FieldGateID).Aggregate(Max(FieldCreatedAt))`

Both `GetByID` and `GetAll` call `attachStats` after fetching gates. Gates with zero requests keep zero-valued stats.

**`stats_repository.go`** — implements `StatsRepository` with 3 simple ent queries (gate count, request count 24h, diff count 24h). DiffRate computed in Go.

### Application Layer (`internal/application/queries/`)

**No changes** to `GetGateHandler` or `ListGatesHandler` — stats flow through `Gate.Stats` automatically.

**`get_global_stats.go`** — thin handler delegating to `StatsRepository.GetGlobalStats`.

### DTO Layer (`pkg/dto/`)

**`gate.go`** — `GateStats` struct with JSON tags, embedded in `Gate`:

```go
type GateStats struct {
    RequestCount24h int64   `json:"request_count_24h"`
    DiffCount24h    int64   `json:"diff_count_24h"`
    DiffRate        float64 `json:"diff_rate"`
    LastActive      *string `json:"last_active"`
}
```

**`stats.go`** — `GlobalStats` struct:

```go
type GlobalStats struct {
    TotalGates       int64   `json:"total_gates"`
    TotalRequests24h int64   `json:"total_requests_24h"`
    TotalDiffRate    float64 `json:"total_diff_rate"`
}
```

### HTTP Handler Layer (`internal/interfaces/http/handlers/`)

**`gate.go`** — `mapGateToDTO` maps `Gate.Stats` → `dto.GateStats`, formatting `LastActive` as RFC 3339 string pointer.

**`stats.go`** — `GetGlobalStats` handler for `GET /stats`, returns `dto.Response[dto.GlobalStats]`.

**`cmd/mroki-api/main.go`** — wires `NewStatsRepository` → `NewGetGlobalStatsHandler` → `GetGlobalStats` handler, registers `GET /stats` route.

---

## Frontend Wiring

All hardcoded dummy data has been replaced with real API data:

| Component | Wired To |
|-----------|----------|
| `GateCard.vue` — requests 24h | `gate.stats.request_count_24h` |
| `GateCard.vue` — diff count | `gate.stats.diff_count_24h` |
| `GateCard.vue` — diff rate | `gate.stats.diff_rate` |
| `GateCard.vue` — last active | `gate.stats.last_active` (formatted as relative time) |
| `GateDetail.vue` — requests 24h | `gate.stats.request_count_24h` |
| `GateDetail.vue` — diff rate | `gate.stats.diff_rate` |
| `Gates.vue` — total gates | `globalStats.total_gates` via `GET /stats` |
| `Gates.vue` — requests 24h | `globalStats.total_requests_24h` via `GET /stats` |
| `Gates.vue` — diff rate | `globalStats.total_diff_rate` via `GET /stats` |

---

## Performance

### Current Approach (v1)

Stats are computed on-the-fly from `requests` + `diffs` tables using ent GroupBy aggregate queries. Acceptable at current scale.

- `GetByID`: 1 gate query + 3 stats queries (request count, diff count, last active)
- `GetAll`: 1 gate list query + 1 count query + 3 batched stats queries
- `GET /stats`: 3 queries (gate count, request count, diff count)

All stats queries are batch-scoped to the gate IDs on the current page — no N+1 from the frontend.

**Indexes already in place:** `requests(gate_id)`, `requests(gate_id, created_at)`.

### Future Optimization (if needed)

If aggregate queries become a bottleneck at scale, introduce a `gate_stats` table with precomputed counters updated on writes:

- Counters (`request_count_24h`, `diff_count_24h`) incremented atomically in the `CreateRequest` transaction
- Timestamps (`last_active`) updated on each write
- Time-windowed stats either refreshed periodically or kept as the only live-computed stat
- Turns read path into a simple `LEFT JOIN` with no aggregation
