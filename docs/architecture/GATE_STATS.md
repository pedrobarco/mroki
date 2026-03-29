# Gate Stats Design

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
      "total_requests": 1250,
      "recent_requests": 347,
      "diff_count": 87,
      "diff_rate": 6.96,
      "last_active": "2026-03-29T14:32:05Z"
    }
  }
}
```

**`GET /gates` response** — same `stats` object per gate in the list.

**Stats fields:**

| Field | Type | Description |
|-------|------|-------------|
| `total_requests` | `int64` | All-time request count for this gate |
| `recent_requests` | `int64` | Request count in the last 24 hours |
| `diff_count` | `int64` | Number of requests that produced a diff |
| `diff_rate` | `float64` | `diff_count / total_requests * 100`, `0.0` when no requests |
| `last_active` | `string?` | ISO 8601 timestamp of the most recent request, `null` if no requests |

### Global Stats — New Endpoint

**`GET /stats`**

Returns cross-gate aggregate statistics for the dashboard stats bar.

```json
{
  "data": {
    "total_gates": 4,
    "total_requests": 12847,
    "recent_requests": 5241,
    "total_diffs": 576,
    "diff_rate": 4.48,
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `total_gates` | `int64` | Total number of gates |
| `total_requests` | `int64` | All-time request count across all gates |
| `recent_requests` | `int64` | Request count in the last 24 hours across all gates |
| `total_diffs` | `int64` | Total diffs across all gates |
| `diff_rate` | `float64` | `total_diffs / total_requests * 100` |

**Errors:** Only `500 Internal Server Error` (no user input to validate).

---

## Data Sources

All stats are derived from existing tables. No new schema required.

| Stat | SQL Source |
|------|-----------|
| `total_requests` | `COUNT(*) FROM requests WHERE gate_id = ?` |
| `recent_requests` | `COUNT(*) FROM requests WHERE gate_id = ? AND created_at >= now() - 24h` |
| `diff_count` | `COUNT(*) FROM diffs d JOIN requests r ON d.request_id = r.id WHERE r.gate_id = ?` |
| `diff_rate` | Computed in Go: `diff_count / total_requests * 100` |
| `last_active` | `MAX(created_at) FROM requests WHERE gate_id = ?` |

**Existing indexes that support these queries:**

- `requests(gate_id)` — filters by gate
- `requests(gate_id, created_at)` — time-windowed counts

Global stats use the same queries without the `WHERE gate_id = ?` filter.

---

## Implementation — Layer by Layer

### Domain Layer (`internal/domain/traffictesting/`)

Add `GateStats` struct and embed in `Gate`:

```go
// gate_stats.go
type GateStats struct {
    TotalRequests  int64
    RecentRequests int64
    DiffCount      int64
    DiffRate       float64
    LastActive     *time.Time
}
```

Add `Stats` field to `Gate` struct:

```go
// gate.go
type Gate struct {
    ID        GateID
    Name      GateName
    LiveURL   GateURL
    ShadowURL GateURL
    CreatedAt time.Time
    Stats     GateStats  // zero-valued by default

    Requests []Request
}
```

Add `GlobalStats` struct for the `/stats` endpoint:

```go
// global_stats.go
type GlobalStats struct {
    TotalGates     int64
    TotalRequests  int64
    RecentRequests int64
    TotalDiffs     int64
    DiffRate       float64
}
```

No changes to `GateRepository` interface — `GetByID` and `GetAll` populate `Stats` as part of their existing return values.

### Persistence Layer (`internal/infrastructure/persistence/ent/`)

**`gate_repository.go`** — update `GetByID` and `GetAll`:

1. Fetch gate(s) as before
2. Collect gate IDs from the result
3. Run a single batched stats query for all IDs
4. Attach stats to each domain `Gate`

Private helper on the repository:

```go
func (r *gateRepository) queryStats(ctx context.Context, gateIDs []uuid.UUID) (map[uuid.UUID]*traffictesting.GateStats, error) {
    // Single SQL query:
    // SELECT r.gate_id, COUNT(*), COUNT(created_at >= ...), COUNT(d.id),
    //        MAX(r.created_at)
    // FROM requests r LEFT JOIN diffs d ON d.request_id = r.id
    // WHERE r.gate_id IN (...)
    // GROUP BY r.gate_id
}
```

- `GetByID`: calls `queryStats` with a single-element slice
- `GetAll`: calls `queryStats` with all gate IDs from the page

Gates with zero requests will not appear in the stats result — their `GateStats` stays zero-valued.

**Global stats** — add a new repository or a standalone query function (not on `GateRepository`):

```go
type StatsRepository interface {
    GetGlobalStats(ctx context.Context) (*traffictesting.GlobalStats, error)
}
```

Implementation runs the same aggregate query without a `WHERE gate_id` filter, plus a `COUNT(*)` on the gates table.

### Application Layer (`internal/application/queries/`)

**No changes** to `GetGateHandler` or `ListGatesHandler` — stats flow through `Gate.Stats` automatically.

**New** `GetGlobalStatsHandler`:

```go
type GetGlobalStatsHandler struct {
    repo traffictesting.StatsRepository
}

func (h *GetGlobalStatsHandler) Handle(ctx context.Context) (*traffictesting.GlobalStats, error) {
    return h.repo.GetGlobalStats(ctx)
}
```

### DTO Layer (`pkg/dto/gate.go`)

```go
type GateStats struct {
    TotalRequests  int64   `json:"total_requests"`
    RecentRequests int64   `json:"recent_requests"`
    DiffCount      int64   `json:"diff_count"`
    DiffRate       float64 `json:"diff_rate"`
    LastActive     *string `json:"last_active"`
}

// Embedded in existing Gate DTO
type Gate struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    LiveURL   string    `json:"live_url"`
    ShadowURL string    `json:"shadow_url"`
    CreatedAt string    `json:"created_at"`
    Stats     GateStats `json:"stats"`
}

type GlobalStats struct {
    TotalGates     int64   `json:"total_gates"`
    TotalRequests  int64   `json:"total_requests"`
    RecentRequests int64   `json:"recent_requests"`
    TotalDiffs     int64   `json:"total_diffs"`
    DiffRate       float64 `json:"diff_rate"`
}
```

### HTTP Handler Layer (`internal/interfaces/http/handlers/`)

**`gate.go`** — update `mapGateToDTO` to include stats:

```go
func mapGateToDTO(gate *traffictesting.Gate) dto.Gate {
    var lastActive *string
    if gate.Stats.LastActive != nil {
        t := gate.Stats.LastActive.Format(time.RFC3339)
        lastActive = &t
    }
    return dto.Gate{
        // ... existing fields ...
        Stats: dto.GateStats{
            TotalRequests:  gate.Stats.TotalRequests,
            RecentRequests: gate.Stats.RecentRequests,
            DiffCount:      gate.Stats.DiffCount,
            DiffRate:       gate.Stats.DiffRate,
            LastActive:     lastActive,
        },
    }
}
```

**New `stats.go`** handler for `GET /stats`:

```go
func GetGlobalStats(handler *queries.GetGlobalStatsHandler) AppHandler {
    return func(w http.ResponseWriter, r *http.Request) error {
        stats, err := handler.Handle(r.Context())
        // ... map to dto.Response[dto.GlobalStats] ...
    }
}
```

**`cmd/mroki-api/main.go`** — register the new route:

```go
mux.Handle("GET /stats", baseChain.Then(getGlobalStats))
```

---

## Frontend Wiring

Once the API returns stats, the frontend changes are:

| Component | Current | Wired To |
|-----------|---------|----------|
| `GateCard.vue` — requests 24h | Hardcoded `"5,241"` | `gate.stats.recent_requests` |
| `GateCard.vue` — diff count | Hardcoded `"162"` | `gate.stats.diff_count` |
| `GateCard.vue` — diff rate | Hardcoded `"3.1%"` | `gate.stats.diff_rate` |
| `GateCard.vue` — last active | Hardcoded `"2 min ago"` | `gate.stats.last_active` |
| `GateDetail.vue` — requests 24h | Hardcoded `"5,241"` | `gate.stats.recent_requests` |
| `GateDetail.vue` — diff rate | Hardcoded `"3.1%"` | `gate.stats.diff_rate` |
| `Gates.vue` — total gates | Hardcoded `"4"` | `globalStats.total_gates` |
| `Gates.vue` — requests 24h | Hardcoded `"12,847"` | `globalStats.recent_requests` |
| `Gates.vue` — diff rate | Hardcoded `"4.2%"` | `globalStats.diff_rate` |

---

## Performance

### Current Approach (v1)

Stats are computed on-the-fly from `requests` + `diffs` tables using aggregate queries. Acceptable at current scale.

- `GetByID`: 1 gate query + 1 stats query (single gate ID)
- `GetAll`: 1 gate list query + 1 batched stats query (all gate IDs on page)
- Global stats: 1 aggregate query across all requests

**Indexes already in place:** `requests(gate_id)`, `requests(gate_id, created_at)`.

### Future Optimization (if needed)

If aggregate queries become a bottleneck at scale, introduce a `gate_stats` table with precomputed counters updated on writes:

- Counters (`total_requests`, `diff_count`) incremented atomically in the `CreateRequest` transaction
- Timestamps (`last_active`) updated on each write
- Time-windowed stats (`recent_requests`) either refreshed periodically or kept as the only live-computed stat
- Turns read path into a simple `LEFT JOIN` with no aggregation
