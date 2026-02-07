# Architecture Overview

This document provides a high-level overview of mroki's architecture, component interactions, and key design principles.

## System Architecture

mroki consists of four main components:

```
┌──────────────────────────────────────────────────────────────┐
│                      Production Environment                   │
│                                                               │
│  ┌─────────┐         ┌─────────────┐                        │
│  │ Client  │────────▶│ mroki-agent │                        │
│  └─────────┘         └──────┬──────┘                        │
│                             │                                │
│                    ┌────────┴────────┐                       │
│                    ↓                 ↓                       │
│            ┌──────────────┐   ┌─────────────┐               │
│            │     Live     │   │   Shadow    │               │
│            │   Service    │   │   Service   │               │
│            │ (Production) │   │ (Candidate) │               │
│            └──────────────┘   └─────────────┘               │
│                                                               │
└───────────────────────────────┬───────────────────────────────┘
                                │ HTTP/JSON
                                ↓
┌──────────────────────────────────────────────────────────────┐
│                      mroki Platform                          │
│                                                               │
│  ┌──────────────┐        ┌──────────────┐                   │
│  │  mroki-api   │◀───────│ PostgreSQL   │                   │
│  │  (REST API)  │        │  (Storage)   │                   │
│  └──────┬───────┘        └──────────────┘                   │
│         │                                                     │
│         │ REST/JSON                                          │
│         ↓                                                     │
│  ┌──────────────┐                                            │
│  │  mroki-hub   │                                            │
│  │  (Web UI)    │                                            │
│  └──────────────┘                                            │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

## Components

### 1. mroki-agent (Go)

**Purpose:** Transparent HTTP proxy that mirrors traffic to shadow services

**Responsibilities:**
- Intercept incoming HTTP requests
- Operate in dual modes: fetch gate config from API or use hardcoded URLs
- Forward to both live and shadow services in parallel
- Return live service response to client immediately
- Compute JSON response differences with configurable filtering in background
- Send captured data to mroki-api with retry logic
- Persist agent identity across restarts

**Technology:**
- Language: Go 1.24+
- HTTP Proxy: Custom `pkg/proxy`
- Diff Engine: Custom JSON differ `pkg/diff` (gjson/sjson + go-cmp) with field filtering and normalization
- API Client: `pkg/client` with exponential backoff

**Deployment:** Runs as a sidecar proxy or standalone service in production environment

---

### 2. mroki-api (Go)

**Purpose:** REST API for managing gates and persisting traffic diffs

**Responsibilities:**
- Gate CRUD operations (create, read, list)
- Store captured requests and responses
- Persist computed diffs
- Serve request/response data to hub
- Health check endpoints for Kubernetes

**Technology:**
- Language: Go 1.24+
- Framework: net/http (stdlib, Go 1.22+ routing)
- Database: PostgreSQL with pgx/v5
- Query Builder: sqlc (type-safe SQL)

**Deployment:** Stateless service, horizontally scalable

---

### 3. mroki-hub (Vue.js)

**Purpose:** Web interface for visualizing diffs and managing the system

**Responsibilities:**
- Display gate dashboard
- Browse captured requests
- Visualize response diffs with syntax highlighting
- Monitor agent health and status
- Manage gate configuration

**Technology:**
- Framework: Vue 3 + TypeScript + Composition API + `<script setup>`
- Build Tool: Vite
- HTTP Client: Native `fetch()`
- Diff Visualization: `vue-diff`
- Styling: TailwindCSS v4

**Deployment:** Static SPA served via CDN or web server

---

### 4. caddy-mroki (Go)

**Purpose:** Caddy server module for embedded mroki functionality

**Responsibilities:**
- Integrate mroki proxy into Caddy HTTP server
- Provide Caddyfile configuration syntax
- Enable mroki without standalone agent deployment

**Technology:**
- Language: Go 1.24+
- Integration: Caddy module system

**Deployment:** Compiled into Caddy binary

---

## Data Flow

### Request Capture Flow

```
1. Client Request
   │
   ↓
2. mroki-agent receives request
   │
   ├─────────────────┬─────────────────┐
   ↓                 ↓                 ↓
3. Forward to     Forward to      Start timer
   Live Service   Shadow Service
   │                 │
   ↓                 ↓
4. Receive         Receive
   Live Response   Shadow Response
   │                 │
   ↓                 │
5. Return Live ──────┘
   Response to
   Client
   │
   ↓
6. Background: Compute Diff
   │
   ↓
7. Send to mroki-api (with retry)
   │
   ↓
8. Store in PostgreSQL
```

### Key Properties

- **Non-blocking:** Live response returns immediately, shadow processing happens in background
- **Best-effort:** API failures are logged but don't affect live traffic
- **Idempotent:** Retries are safe (requests have unique IDs)
- **JSON-only:** Only JSON responses are diffed (Content-Type check)

---

## Data Model

### Core Entities

```go
// Gate represents a live/shadow service pair
type Gate struct {
    ID         GateID    // UUID
    LiveURL    URL       // Production service URL
    ShadowURL  URL       // Shadow service URL
    CreatedAt  time.Time
}

// Request represents a captured HTTP request
type Request struct {
    ID         RequestID  // UUID
    GateID     GateID     // Parent gate
    AgentID    AgentID    // Capturing agent
    Method     string     // HTTP method (GET, POST, etc.)
    Path       string     // Request path
    Headers    Headers    // HTTP headers
    Body       []byte     // Request body
    CreatedAt  time.Time
}

// Response represents a service response
type Response struct {
    ID          ResponseID  // UUID
    RequestID   RequestID   // Parent request
    StatusCode  int         // HTTP status code
    Headers     Headers     // Response headers
    Body        []byte      // Response body
    Duration    Duration    // Response time
    IsLive      bool        // true=live, false=shadow
}

// Diff represents computed difference
type Diff struct {
    RequestID   RequestID
    DiffJSON    []byte     // JSON diff format
    HasDiff     bool       // Quick check flag
}
```

### Database Schema

See [API Contracts](API_CONTRACTS.md#database-schema) for detailed schema.

---

## Key Design Decisions

### 1. Agent-Side Diffing

**Decision:** Compute diffs in the agent, not the API

**Rationale:**
- Reduces API processing load
- Keeps diff logic close to capture point
- Enables future sampling/filtering in agent
- Agent already has both responses in memory

### 2. Best-Effort Delivery

**Decision:** Agent never fails live traffic due to API issues

**Rationale:**
- Shadow testing should never impact production
- API outages shouldn't affect live service
- Failed captures can be logged and monitored
- Trade-off: Some diffs may be lost

### 3. JSON-Only Diffing

**Decision:** Only diff JSON responses, skip others

**Rationale:**
- JSON is structured and diffable
- Binary/HTML diffs less meaningful
- Reduces storage and processing costs
- Future: Can add support for other types

### 4. Agent ID Persistence

**Decision:** Persist agent ID to disk, not ephemeral

**Rationale:**
- Track which agent captured traffic across restarts
- Debugging and troubleshooting
- Agent health monitoring
- Format: `{hostname}-{uuid}` for human readability

### 5. Exponential Backoff Retry

**Decision:** Retry API requests with exponential backoff (1s, 2s, 4s)

**Rationale:**
- Handle temporary API unavailability
- Avoid thundering herd during outages
- Balance delivery reliability with resource usage
- 3 retries = ~8s total before giving up

### 6. Stateless API

**Decision:** API is fully stateless, all state in PostgreSQL

**Rationale:**
- Horizontal scalability
- Simple deployment model
- No session management needed
- Easy to load balance

### 7. Dual Operating Modes

**Decision:** Agent works in API mode (fetches config) or standalone mode (hardcoded URLs)

**Rationale:**
- API mode: Centralized configuration management
- Standalone mode: Useful for local testing and validation
- Graceful degradation when API unavailable
- Reduces operational dependencies for simple setups
- Agent can operate without API connection

---

## Security Considerations

### Implemented

- [x] API key authentication (`Authorization: Bearer <key>`)
- [x] Rate limiting (token bucket, 1000 req/min/IP default)
- [x] Request body size limits (10MB default)
- [x] Input validation via domain value objects
- [x] SQL injection prevention (parameterized queries via sqlc)
- [x] CORS with configurable allowed origins
- [x] HTTP timeouts and graceful shutdown
- [x] RFC 7807 structured error responses

### Not Yet Implemented

- **No TLS:** HTTP only (use reverse proxy for HTTPS)
- **No authorization:** All authenticated users have full access
- **No request filtering:** All traffic captured (may contain PII)

### Planned

- [ ] RBAC for multi-tenant usage
- [ ] PII redaction in captured requests
- [ ] Agent-to-API mutual TLS

---

## Scalability

### Bottlenecks

1. **PostgreSQL writes:** High traffic gates = many DB writes
2. **API processing:** Request parsing and validation
3. **Diff computation:** Large JSON responses

### Mitigation Strategies

1. **Batching:** Agents can batch multiple diffs per API request (future)
2. **Sampling:** Capture only N% of traffic (configurable per gate)
3. **Async processing:** Queue diffs for background processing
4. **Sharding:** Partition gates across multiple databases
5. **Read replicas:** Serve hub queries from replicas

### Current Limits (estimated)

- **Agent throughput:** ~1000 req/s per agent instance
- **API throughput:** ~500 req/s per API instance (DB-bound)
- **Storage:** ~1KB per request avg → 1M requests = ~1GB

---

## Observability

### Structured Logging

All components use structured logging (slog):

```go
log.Info("request captured",
    "method", "POST",
    "path", "/api/users",
    "has_diff", true,
    "agent_id", "host-abc123",
)
```

### Metrics (Planned)

- `mroki_agent_requests_total` - Total requests proxied
- `mroki_agent_diffs_detected` - Diffs found
- `mroki_agent_api_failures` - API send failures
- `mroki_api_requests_total` - API request count
- `mroki_api_request_duration` - API latency

### Health Checks

- **API:** `GET /health/ready` - DB connectivity
- **API:** `GET /health/live` - Service up
- **Agent:** HTTP server accepting connections

---

## Deployment Topology

### Option 1: Sidecar Proxy

```
┌────────────────────────────┐
│         Pod/Container      │
│  ┌──────────┐              │
│  │  App     │              │
│  │ Service  │              │
│  └────┬─────┘              │
│       │                    │
│  ┌────▼─────────┐          │
│  │ mroki-agent  │          │
│  └──────────────┘          │
└────────────────────────────┘
```

**Pros:** No app changes, transparent
**Cons:** Resource overhead per pod

### Option 2: Standalone Proxy

```
┌──────────┐     ┌──────────────┐
│  Client  │────▶│ mroki-agent  │
└──────────┘     └──────┬───────┘
                        │
                 ┌──────┴───────┐
                 ↓              ↓
            ┌────────┐    ┌──────────┐
            │  Live  │    │  Shadow  │
            └────────┘    └──────────┘
```

**Pros:** Centralized, lower resource usage
**Cons:** Single point of failure (use HA)

### Option 3: Caddy Module

```
┌──────────┐     ┌────────────────────┐
│  Client  │────▶│  Caddy (w/mroki)   │
└──────────┘     └──────┬─────────────┘
                        │
                 ┌──────┴───────┐
                 ↓              ↓
            ┌────────┐    ┌──────────┐
            │  Live  │    │  Shadow  │
            └────────┘    └──────────┘
```

**Pros:** Integrated with existing Caddy setup
**Cons:** Couples to Caddy lifecycle

---

## Technology Choices

### Why Go?

- Excellent HTTP/network performance
- Strong standard library (net/http)
- Easy concurrency (goroutines for parallel requests)
- Single binary deployment
- Great testing support

### Why Vue 3?

- Reactive and performant
- Excellent TypeScript support
- Composition API for reusable logic
- Strong ecosystem (Vite, TailwindCSS, etc.)
- Smaller bundle size than React

### Why PostgreSQL?

- JSONB support for flexible diff storage
- Strong consistency guarantees
- Excellent query performance
- Mature tooling and operations
- Native UUID support

### Why sqlc?

- Type-safe SQL queries
- Compile-time validation
- No reflection overhead
- Direct SQL control
- Simple integration with pgx

---

## Future Enhancements

### Phase 2 (Completed)
- [x] Agent fetches gate configuration from API
- [x] Dual operating modes (API vs standalone)
- [x] Configurable diff options (field filtering via normalizer, go-cmp based diffing)
- [x] API key authentication
- [x] Rate limiting (token bucket, configurable per IP)
- [x] CORS support (`rs/cors`)
- [x] TTL cleanup job for expired requests

### Phase 3
- [ ] Sampling configuration per gate
- [ ] Batch API requests for efficiency
- [ ] Prometheus metrics
- [ ] Request replay (send to shadow on-demand)
- [ ] Diff analysis algorithms (similarity scores)

### Phase 4
- [ ] Alerting on unexpected diffs
- [ ] Multi-region support
- [ ] Performance regression detection
- [ ] Advanced diff visualization

---

## Related Documentation

- [API Contracts](API_CONTRACTS.md) - Detailed API specifications
- [mroki-agent Component](../components/MROKI_AGENT.md)
- [mroki-api Component](../components/MROKI_API.md)
- [mroki-hub Component](../components/MROKI_HUB.md)
