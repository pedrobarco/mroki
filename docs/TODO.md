# mroki Production Readiness TODO

**Last Updated:** 2026-02-07  
**Target:** Usable in dev environments immediately, production deployment within 2-3 weeks  
**Current Status:** **Phase 1: COMPLETE** | **Phase 4: COMPLETE** | **Hub: Next** | Overall: ~50% production-ready

---

## 📊 Overview

This document tracks tasks required to make mroki production-ready and highly usable in development environments. The codebase has excellent architecture and test coverage (62.8%), with Phases 1 and 4 complete.

**Phases:**
- **Phase 1:** Security & Stability (Week 1) ✅ **COMPLETE**
- **Phase 4:** Developer Experience ✅ **COMPLETE**
- **Phase 5:** mroki-hub (Web UI) 🔥 **CURRENT FOCUS**
- **Phase 2:** Observability & Resilience (Week 2)
- **Phase 3:** Production Hardening (Week 3)

---

## 🚨 Phase 1: Security & Stability (CRITICAL) ✅ COMPLETE

**Status:** ✅ **100% COMPLETE (7/7 tasks)**  
**Completion Date:** 2026-02-02

All critical security and stability tasks have been completed. The API is now production-ready from a security standpoint:

- ✅ **P0-1:** RFC 7807 error format (already implemented)
- ✅ **P0-2:** HTTP server timeouts (15s read, 30s write, 60s idle)
- ✅ **P0-3:** Request body size limits (10MB default)
- ✅ **P0-4:** Graceful shutdown (30s timeout for in-flight requests)
- ✅ **P0-5:** API key authentication (Bearer token)
- ✅ **P0-6:** Rate limiting (1000 req/min per IP)
- ✅ **P0-7:** Input validation (value objects at domain boundary)

**Security Posture:**
- 🛡️ Protected against Slowloris attacks (timeouts)
- 🛡️ Protected against memory exhaustion (body size limits)
- 🛡️ Protected against DoS attacks (rate limiting)
- 🛡️ Protected against unauthorized access (API key auth)
- 🛡️ Protected against invalid data (input validation)
- 🛡️ Graceful degradation (clean shutdown)

Must complete before production deployment.

### P0-1: Fix Error Response Format ⏱️ 2h
**Status:** ✅ COMPLETE (Already Implemented!)  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-01

**Original Problem:** Error responses don't match API_CONTRACTS.md specification.

**RESOLUTION:** Upon verification, the error format was **already correctly implemented** using RFC 7807 standard!

**Actual format (RFC 7807):**
```json
{
  "type": "/errors/invalid-request-body",
  "title": "Invalid Gate ID",
  "status": 400,
  "detail": "gate_id must be a valid UUID, got \"not-a-uuid\"",
  "instance": "/gates/not-a-uuid"
}
```

**Implementation:**
- ✅ `pkg/dto/error.go` - APIError struct implements RFC 7807 perfectly
- ✅ All error constructors use correct format (15 total)
- ✅ Auto-populates `instance` field for 4xx errors only
- ✅ Omits `instance` for 5xx errors (security best practice)
- ✅ All 410 tests passing
- ✅ API_CONTRACTS.md examples are accurate and match implementation

**Acceptance criteria:**
- [x] All error responses use RFC 7807 format
- [x] All tests pass (410/410)
- [x] API_CONTRACTS.md examples are accurate

**Note:** The "current format" and "required format" listed above were **incorrect assumptions**. The codebase was already production-ready with proper RFC 7807 error handling.

---

### P0-2: Add HTTP Server Timeouts ⏱️ 2h
**Status:** ✅ COMPLETE  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-01  
**Commit:** 32544d9

**Problem:** Servers vulnerable to Slowloris attacks and hanging connections.

**Implementation:**
```go
server := &http.Server{
    Addr:         fmt.Sprintf(":%d", cfg.App.Port),
    Handler:      mux,
    ReadTimeout:  15 * time.Second,  // Time to read request
    WriteTimeout: 30 * time.Second,  // Time to write response
    IdleTimeout:  60 * time.Second,  // Keep-alive timeout
}
```

**Files modified:**
- `cmd/mroki-api/main.go:92-95`
- `cmd/mroki-agent/main.go:68-71`

**Acceptance criteria:**
- [x] Both services have timeouts configured
- [x] Manual testing shows connections close after timeout
- [x] No impact on normal operations

---

### P0-3: Add Request Body Size Limits ⏱️ 2h
**Status:** ✅ COMPLETE  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-01  
**Commit:** 32544d9

**Problem:** No limits on request body size - memory exhaustion risk.

**Implementation:**
```go
// internal/interfaces/http/middleware/maxbodysize.go
func MaxBodySize(maxBytes int64) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```

**Configuration:**
- Default: 10MB (configurable via env var)
- `MROKI_API_MAX_BODY_SIZE=10485760` (bytes)

**Files created:**
- `internal/interfaces/http/middleware/maxbodysize.go`
- `internal/interfaces/http/middleware/maxbodysize_test.go`

**Files modified:**
- `cmd/mroki-api/main.go` - Apply to all POST endpoints
- `cmd/mroki-api/config/config.go` - Add config field

**Acceptance criteria:**
- [x] Requests larger than limit return 413 status
- [x] Tests cover edge cases (exactly at limit, over limit)
- [x] Documentation updated

---

### P0-4: Implement Graceful Shutdown ⏱️ 4h
**Status:** ✅ COMPLETE  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-01  
**Commit:** aaaab10

**Problem:** Servers don't handle SIGTERM/SIGINT - data loss on restart.

**Implementation:**
```go
// Start server in goroutine
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Error("Server error", "error", err)
    }
}()

// Wait for interrupt signal
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
<-stop

// Graceful shutdown with timeout
logger.Info("Shutting down server...")
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    logger.Error("Server shutdown error", "error", err)
}

// Close database pool
pool.Close()
logger.Info("Server stopped")
```

**Files modified:**
- `cmd/mroki-api/main.go` - Add shutdown logic
- `cmd/mroki-agent/main.go` - Add shutdown logic

**Testing:**
- Manual test: Send request, kill with SIGTERM during processing
- Verify: Request completes, logs show clean shutdown

**Acceptance criteria:**
- [x] In-flight requests complete before shutdown
- [x] Database connections closed cleanly
- [x] Clean log messages on shutdown
- [x] Works with both SIGTERM and SIGINT

---

### P0-5: Add API Key Authentication ⏱️ 1-2 days
**Status:** ✅ COMPLETE  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-01  
**Commit:** 3b34114

**Problem:** No authentication - anyone can access API.

**Design:**
- Simple bearer token authentication
- Keys configured via environment variable
- Header format: `Authorization: Bearer <api-key>`
- Skip health check endpoints

**Implementation:**
```go
// internal/interfaces/http/middleware/apikey.go
func APIKeyAuth(validKeys map[string]bool) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, `{"error":{"message":"missing authorization header"}}`, 
                    http.StatusUnauthorized)
                return
            }
            
            token := strings.TrimPrefix(authHeader, "Bearer ")
            if !validKeys[token] {
                http.Error(w, `{"error":{"message":"invalid API key"}}`, 
                    http.StatusUnauthorized)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

**Configuration:**
```bash
# API
MROKI_API_KEYS=key1,key2,key3

# Agent
MROKI_APP_API_KEY=key1
```

**Files created:**
- `internal/interfaces/http/middleware/apikey.go`
- `internal/interfaces/http/middleware/apikey_test.go`

**Files modified:**
- `cmd/mroki-api/main.go` - Apply to all endpoints except `/health/*`
- `cmd/mroki-api/config/config.go` - Add `APIKeys []string`
- `cmd/mroki-agent/config/config.go` - Add `APIKey string`
- `pkg/client/mroki_client.go` - Add key to requests

**Documentation:**
- `docs/architecture/API_CONTRACTS.md` - Add authentication section
- `docs/components/MROKI_API.md` - Document key management
- `docs/components/MROKI_AGENT.md` - Document configuration

**Acceptance criteria:**
- [x] Requests without auth return 401
- [x] Requests with invalid key return 401
- [x] Requests with valid key succeed
- [x] Health checks don't require auth
- [x] Agent can authenticate successfully
- [x] Documentation includes key rotation process

---

### P0-6: Implement Rate Limiting ⏱️ 1 day
**Status:** ✅ COMPLETE  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-02  
**Commit:** dd85ad8

**Problem:** No rate limiting - DoS vulnerability.

**Solution:** Implemented per-IP rate limiting with token bucket algorithm using `golang.org/x/time/rate`.

**Implementation:**
- Per-IP limiting with token bucket algorithm
- Configurable via `MROKI_API_RATE_LIMIT` (default: 1000 req/min = ~16.67 req/sec)
- Supports X-Forwarded-For header (proxy-aware via `ExtractIPWithForwardedFor`)
- Periodic cleanup goroutine (every 10min, removes limiters inactive >1hr)
- RFC 7807 error responses with Retry-After header
- Thread-safe concurrent access with `sync.Map`

**Architecture:**
```go
// Token bucket: allows bursts up to full minute limit
rps := float64(rpm) / 60.0  // Convert to req/sec
burst := rpm                 // Allow burst = full minute

limiter := rate.NewLimiter(rate.Limit(rps), burst)
```

**Middleware Order (CRITICAL):**
```
1. Logging
2. RateLimit  ← Positioned BEFORE auth to prevent auth bypass DoS
3. APIKeyAuth
4. MaxBodySize (POST only)
```

**Files created:**
- `internal/interfaces/http/middleware/ratelimit.go` (~170 lines)
- `internal/interfaces/http/middleware/ratelimit_test.go` (~290 lines, 14 test cases)

**Files modified:**
- `cmd/mroki-api/main.go` - Apply to both baseChain and postChain
- `cmd/mroki-api/config/config.go` - Add RateLimit field + validation
- `pkg/dto/error.go` - Add ErrRateLimitExceeded

**Configuration:**
```bash
# Default: 1000 requests per minute per IP
MROKI_API_RATE_LIMIT=1000

# Validation: must be positive, max 100000
```

**Acceptance criteria:**
- [x] Requests over limit return 429 with Retry-After header
- [x] Different IPs have independent limits
- [x] Limits refill over time (token bucket)
- [x] Health checks excluded (registered without middleware)
- [x] Concurrent requests handled safely (tested with race detector)
- [x] Config validation prevents invalid values
- [x] All 14 test cases passing
- [x] Custom IP extractor and error handler support

**Test Coverage:**
- 14 comprehensive test cases
- Concurrent access tested with race detector
- Burst behavior verified
- Token refill tested (time-sensitive test)
- Multiple IP extraction strategies tested

**Memory Management:**
- Cleanup goroutine runs every 10 minutes
- Removes limiters inactive for >1 hour
- Prevents memory leaks with many unique IPs

---

### P0-7: Add Input Validation ⏱️ 1 day
**Status:** ✅ COMPLETE  
**Priority:** ~~BLOCKER~~ RESOLVED  
**Completion Date:** 2026-02-02  
**Commit:** 1effcfc

**Problem:** Insufficient validation allows invalid data into system.

**Solution:** Implemented value objects at domain boundary to make illegal states unrepresentable.

**Validations implemented:**

1. **HTTP Method Validation**
   - File: `internal/domain/traffictesting/http_method.go` (already existed, now used)
   - Valid: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
   - Returns error for invalid methods

2. **Status Code Validation**
   - File: `internal/domain/traffictesting/status_code.go` (NEW)
   - Valid: 100-599
   - Returns error for out-of-range codes

3. **Path Validation**
   - File: `internal/domain/traffictesting/path.go` (NEW)
   - Non-empty
   - Must start with `/`
   - Max length: 2048 characters

4. **Headers Type Wrapper**
   - File: `internal/domain/traffictesting/headers.go` (NEW)
   - Type-safe wrapper around `http.Header`
   - No validation (trust standard library)

**Implementation approach:**
- Value objects enforce validation at construction time
- Service layer builds all value objects before entity construction
- Repository layer converts between value objects and primitives
- Clear, actionable error messages with context

**Files created:**
- `internal/domain/traffictesting/status_code.go`
- `internal/domain/traffictesting/status_code_test.go`
- `internal/domain/traffictesting/path.go`
- `internal/domain/traffictesting/path_test.go`
- `internal/domain/traffictesting/headers.go`

**Files modified:**
- `internal/domain/traffictesting/request.go` - Use HTTPMethod, Path, Headers value objects
- `internal/domain/traffictesting/response.go` - Use StatusCode, Headers value objects
- `internal/domain/traffictesting/errors.go` - Add new error constants
- `internal/application/commands/create_request.go` - Build value objects before entity construction
- `internal/infrastructure/persistence/postgres/request_repository.go` - Convert between value objects and primitives
- All test files updated to use value objects

**Acceptance criteria:**
- [x] Invalid HTTP methods rejected with clear error
- [x] Invalid status codes rejected with clear error
- [x] Invalid paths rejected with clear error
- [x] All validations have tests (98.7% domain coverage)
- [x] Error messages are helpful and include context
- [x] All 100+ tests passing with race detection

---

## 🎯 Phase 4: Developer Experience ✅ COMPLETE

**Status:** ✅ **100% COMPLETE (2/2 tasks)**  
**Priority:** CRITICAL for dev environment usability  
**Completion Date:** 2026-02-06

These features make mroki practical and usable in development environments. Without field ignoring, diffs are cluttered with irrelevant changes (timestamps, IDs). Without TTL, databases fill with test data.

**Why Phase 4 before Phase 2?**
- Phase 1 (security) provides safe foundation ✅
- Phase 4 (usability) makes the tool actually useful for developers 🔥
- Phase 2 (observability) matters more for production scale
- Phase 3 (hardening) is nice-to-have optimizations

---

### DEV-1: Diff Engine Rewrite + Field Filtering ⏱️ 3-4 days
**Status:** ✅ Phase 1 COMPLETE | Phases 2-3 DEFERRED  
**Priority:** CRITICAL (blocks useful diff analysis)  
**Effort:** 3-4 days (Phase 1 only)

**Problem:** Every diff shows irrelevant differences in timestamps, request IDs, and dynamic fields.

**What was implemented (Phase 1 - Core Diff Engine):**
- Replaced JD library with gjson/sjson + go-cmp (30%+ faster)
- `pkg/diff/normalizer.go` — Field filtering with whitelist/blacklist strategies and wildcard support (`users.#.created_at`)
- `pkg/diff/options.go` — Functional options pattern for diff configuration
- `pkg/diff/reporter.go` — Clean human-readable diff output
- `pkg/diff/json.go` — New diff engine using go-cmp
- Full test coverage for all new code

**What was NOT implemented (Phases 2-3 - Deferred):**
- Per-gate DiffConfig stored in database (migration, domain model, repository)
- API endpoints for configuring diff options per gate
- Agent integration to fetch and apply per-gate diff config
- These phases are deferred until there is a concrete need for per-gate configuration

**Acceptance criteria (Phase 1):**
- [x] JD library completely removed
- [x] New diff engine 30%+ faster than JD baseline
- [x] Field normalizer supports whitelist, blacklist, and wildcards
- [x] All existing tests pass (backward compatibility)
- [x] Benchmarks demonstrate performance improvement

---

### DEV-2: TTL for Diffs/Requests ⏱️ 6-8h
**Status:** ✅ COMPLETE  
**Priority:** HIGH (prevents database bloat)  
**Effort:** 6-8 hours  
**Completion Date:** 2026-02-05  
**Commit:** c584cb7

**Problem:** Test data accumulates forever. In dev environments, databases grow unbounded with experimental traffic.

**What was implemented:**
A global retention-based cleanup job that periodically deletes expired requests. Simpler than the originally planned per-gate `retention_days` design — uses global config instead.

**Configuration:**
```bash
# How long to keep requests (Go duration format)
MROKI_APP_RETENTION=168h  # 7 days

# How often to run cleanup
MROKI_APP_CLEANUP_INTERVAL=1h
```

**Architecture:**
- Consumer-defined `RequestCleaner` interface (follows existing codebase patterns)
- Background goroutine with configurable interval
- Deletes requests older than retention duration
- Graceful shutdown via context cancellation
- Structured logging of cleanup results

**Files created:**
- `internal/infrastructure/jobs/cleanup.go`
- `internal/infrastructure/jobs/cleanup_test.go`

**Files modified:**
- `cmd/mroki-api/config/config.go` — Added `Retention` and `CleanupInterval` fields
- `cmd/mroki-api/main.go` — Start cleanup job in background

**Acceptance criteria:**
- [x] Cleanup job runs on configurable interval
- [x] Records older than retention period are deleted
- [x] Cleanup logs deleted record count
- [x] Cleanup handles database errors gracefully
- [x] Graceful shutdown stops cleanup job
- [x] All tests passing

**Note:** The per-gate `retention_days` design originally described here (with database migration, per-gate override, etc.) was not implemented. The simpler global approach was chosen as sufficient for current needs.

---

## 🔄 Phase 2: Observability & Resilience (IMPORTANT)

Complete before scaling to multiple environments.

### P1-1: Add Request ID Middleware ⏱️ 4h
**Status:** 🔴 Not Started  
**Priority:** HIGH  
**Effort:** 4 hours

**Problem:** Can't trace individual requests through logs.

**Implementation:**
```go
// internal/interfaces/http/middleware/requestid.go
func RequestID() Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            requestID := r.Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = uuid.New().String()
            }
            
            w.Header().Set("X-Request-ID", requestID)
            ctx := context.WithValue(r.Context(), "request_id", requestID)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

**Files to create:**
- `internal/interfaces/http/middleware/requestid.go`
- `internal/interfaces/http/middleware/requestid_test.go`

**Files to modify:**
- `cmd/mroki-api/main.go` - Apply first in middleware chain
- `internal/interfaces/http/middleware/logging.go` - Include request ID in logs
- `pkg/client/mroki_client.go` - Send request ID from agent

**Acceptance criteria:**
- [ ] All requests have X-Request-ID header
- [ ] Request ID appears in all logs for that request
- [ ] Agent propagates request IDs
- [ ] Tests verify header presence

---

### P1-2: Implement Circuit Breaker in Agent ⏱️ 1 day
**Status:** 🔴 Not Started  
**Priority:** HIGH  
**Effort:** 1 day

**Problem:** Agent retries indefinitely if API is down - wastes resources.

**Implementation:**
```go
// pkg/client/mroki_client.go
// Use github.com/sony/gobreaker

type MrokiClient struct {
    // ... existing fields
    circuitBreaker *gobreaker.CircuitBreaker
}

func NewMrokiClient(...) *MrokiClient {
    cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        "mroki-api",
        MaxRequests: 3,
        Interval:    60 * time.Second,
        Timeout:     60 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= 5
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            logger.Info("circuit breaker state change",
                "from", from.String(),
                "to", to.String(),
            )
        },
    })
    
    client.circuitBreaker = cb
    return client
}

func (c *MrokiClient) SendRequest(...) error {
    _, err := c.circuitBreaker.Execute(func() (interface{}, error) {
        return nil, c.sendRequestOnce(ctx, req)
    })
    return err
}
```

**Dependencies:**
```bash
go get github.com/sony/gobreaker
```

**Files to modify:**
- `pkg/client/mroki_client.go`
- `pkg/client/mroki_client_test.go`

**Acceptance criteria:**
- [ ] Circuit opens after 5 failures
- [ ] Circuit half-opens after 60s
- [ ] State changes logged
- [ ] Tests verify circuit behavior
- [ ] Agent continues proxying when circuit is open

---

### P1-3: Optimize HTTP Connection Pooling ⏱️ 1 day
**Status:** 🔴 Not Started  
**Priority:** HIGH  
**Effort:** 1 day

**Problem:** Agent HTTP client uses default settings - inefficient.

**Implementation:**
```go
// pkg/client/mroki_client.go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        DisableKeepAlives:   false,
    },
}
```

**Configuration:**
```bash
MROKI_APP_MAX_IDLE_CONNS=100
MROKI_APP_MAX_IDLE_CONNS_PER_HOST=10
```

**Files to modify:**
- `pkg/client/mroki_client.go`
- `cmd/mroki-agent/config/config.go`

**Testing:**
- Load test with `hey` or `vegeta`
- Monitor connection reuse
- Compare before/after performance

**Acceptance criteria:**
- [ ] Connection pooling configured
- [ ] Load testing shows improved performance
- [ ] Documentation updated with tuning guide

---

### P1-4: Add CORS Support ⏱️ 2h
**Status:** ✅ COMPLETE  
**Priority:** MEDIUM  
**Effort:** 2 hours  
**Completion Date:** 2026-02-06  
**Commit:** 560ddef

**Problem:** Hub (web UI) can't call API from browser without CORS headers.

**What was implemented:**
- Used `github.com/rs/cors` v1.11.1 (battle-tested library instead of hand-rolled middleware)
- Configurable origins via `MROKI_APP_CORS_ORIGINS` env var (comma-separated, empty=disabled)
- `ParseCORSOrigins()` method on config splits into `[]string`
- Wraps entire mux with `cors.New(cors.Options{...}).Handler(mux)` when origins configured

**Configuration:**
```bash
# Comma-separated allowed origins (empty = CORS disabled)
MROKI_APP_CORS_ORIGINS=http://localhost:5173,https://hub.example.com
```

**Settings:**
- Allowed methods: GET, POST, OPTIONS
- Allowed headers: Content-Type, Authorization
- Max-age: 86400s (24h preflight cache)

**Acceptance criteria:**
- [x] OPTIONS requests handled correctly
- [x] CORS headers set for allowed origins
- [x] Configurable via environment variable
- [x] Disabled when no origins configured

---

### P1-5: Add Structured Error Logging ⏱️ 4h
**Status:** 🔴 Not Started  
**Priority:** MEDIUM  
**Effort:** 4 hours

**Problem:** Error logs lack context for troubleshooting.

**Implementation:**
```go
// internal/interfaces/http/handlers/handler.go
slog.Error("API error",
    slog.String("request_id", getRequestID(r)),
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.String("remote_addr", r.RemoteAddr),
    slog.String("user_agent", r.UserAgent()),
    slog.String("error.code", apiErr.Code),
    slog.String("error.message", apiErr.Message),
    slog.Any("error.details", apiErr.Details),
)
```

**Files to modify:**
- `internal/interfaces/http/handlers/handler.go`
- `internal/interfaces/http/middleware/logging.go`
- All handler files

**Acceptance criteria:**
- [ ] All error logs include request context
- [ ] Logs are easy to query in production
- [ ] Request ID present in all logs
- [ ] Sensitive data not logged

---

### P1-6: Update API_CONTRACTS.md ⏱️ 1h
**Status:** 🔴 Not Started  
**Priority:** MEDIUM  
**Effort:** 1 hour

**Problem:** Documentation doesn't match implementation.

**Changes needed:**
1. Document pagination (already implemented!)
2. Add authentication section
3. Add rate limiting section
4. Update error response examples
5. Add CORS section

**File to modify:**
- `docs/architecture/API_CONTRACTS.md`

**Sections to update:**
- Line 527-547: Update pagination from "Planned (v2)" to documented
- Line 503-522: Update authentication section
- Line 515-522: Update rate limiting section
- Add request ID header documentation

**Acceptance criteria:**
- [ ] All examples match actual API behavior
- [ ] No "Planned (v2)" for implemented features
- [ ] Authentication documented with examples
- [ ] Pagination examples included

---

## 🔧 Phase 3: Production Hardening (NICE-TO-HAVE)

Complete for optimal production operations.

### P2-1: TLS/HTTPS Support ⏱️ 1 day
**Status:** 🔴 Not Started  
**Priority:** MEDIUM  
**Effort:** 1 day

**Implementation:**
```go
// cmd/mroki-api/main.go
if cfg.App.TLS.Enabled {
    logger.Info("Starting HTTPS server", "address", server.Addr)
    err = server.ListenAndServeTLS(cfg.App.TLS.CertFile, cfg.App.TLS.KeyFile)
} else {
    logger.Info("Starting HTTP server", "address", server.Addr)
    err = server.ListenAndServe()
}
```

**Configuration:**
```bash
MROKI_API_TLS_ENABLED=true
MROKI_API_TLS_CERT_FILE=/path/to/cert.pem
MROKI_API_TLS_KEY_FILE=/path/to/key.pem
```

**Acceptance criteria:**
- [ ] Services support TLS configuration
- [ ] Documentation covers TLS termination at load balancer
- [ ] Certificate rotation documented

---

### P2-2: Request Deduplication ⏱️ 4h
**Status:** 🔴 Not Started  
**Priority:** LOW  
**Effort:** 4 hours

**Problem:** Duplicate request IDs cause errors instead of being idempotent.

**Implementation:**
- Add unique constraint on `requests.id` (already exists!)
- Handle `UNIQUE_VIOLATION` in repository
- Return 200 (success) for duplicates instead of error

**Files to modify:**
- `internal/infrastructure/persistence/postgres/request_repository.go`
- Tests

**Acceptance criteria:**
- [ ] Duplicate POST with same ID returns 200
- [ ] Original request returned in response
- [ ] Tests verify idempotency

---

### P2-3: Compression Middleware ⏱️ 2h
**Status:** 🔴 Not Started  
**Priority:** LOW  
**Effort:** 2 hours

**Problem:** Large responses (especially diffs) sent uncompressed.

**Implementation:**
```go
// Use github.com/klauspost/compress or stdlib
func Compression() Middleware {
    return func(next http.Handler) http.Handler {
        return gziphandler.GzipHandler(next)
    }
}
```

**Acceptance criteria:**
- [ ] Responses > 1KB compressed with gzip
- [ ] Client receives correct Content-Encoding header
- [ ] Compression optional (based on Accept-Encoding)

---

### P2-4: Config Hot-Reload ⏱️ 1 day
**Status:** 🔴 Not Started  
**Priority:** LOW  
**Effort:** 1 day

**Problem:** Config changes require server restart.

**Implementation:**
- Watch config file for changes
- Reload on SIGHUP signal
- Don't restart server
- Only reload safe settings (log levels, timeouts, etc.)

**Files to modify:**
- `internal/config/` package
- Both `cmd/*/main.go`

**Acceptance criteria:**
- [ ] SIGHUP reloads config
- [ ] Server continues running
- [ ] Unsafe changes require restart (logged)

---

## 📚 Documentation Tasks

### DOC-1: Create PRODUCTION_READINESS.md ⏱️ 2h
**Status:** 🟢 In Progress  
**Priority:** HIGH  
**Effort:** 2 hours

**File:** `docs/guides/PRODUCTION_READINESS.md`

**Content:**
- Pre-deployment checklist
- Phase-by-phase rollout plan
- Monitoring requirements
- Security hardening checklist
- Operations runbook
- Troubleshooting guide

---

### DOC-2: Update MROKI_API.md ⏱️ 1h
**Status:** 🔴 Not Started  
**Priority:** MEDIUM  
**Effort:** 1 hour

**File:** `docs/components/MROKI_API.md`

**Sections to add:**
- Production deployment section
- Security configuration
- Performance tuning guide
- Monitoring and metrics

---

### DOC-3: Update MROKI_AGENT.md ⏱️ 1h
**Status:** 🔴 Not Started  
**Priority:** MEDIUM  
**Effort:** 1 hour

**File:** `docs/components/MROKI_AGENT.md`

**Sections to add:**
- Production deployment patterns
- Circuit breaker behavior
- Connection pooling configuration
- Authentication setup

---

## 📊 Progress Tracking

### By Phase

**Phase 1: Security & Stability** ✅
- Total tasks: 7
- Completed: 7
- In Progress: 0
- Not Started: 0
- Progress: 100% ⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛

**Phase 4: Developer Experience** ✅
- Total tasks: 2
- Completed: 2
- In Progress: 0
- Not Started: 0
- Progress: 100% ⬛⬛⬛⬛⬛⬛⬛⬛⬛⬛

**Phase 2: Observability & Resilience** (P1-4 CORS moved to complete)
- Total tasks: 5
- Completed: 1 (P1-4 CORS)
- In Progress: 0
- Not Started: 4
- Progress: 20% ⬛⬛⬜⬜⬜⬜⬜⬜⬜⬜

**Phase 3: Production Hardening**
- Total tasks: 4
- Completed: 0
- In Progress: 0
- Not Started: 4
- Progress: 0% ⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜

**Documentation**
- Total tasks: 3
- Completed: 0
- In Progress: 1 (this file!)
- Not Started: 2
- Progress: 33% ⬛⬛⬛⬜⬜⬜⬜⬜⬜⬜

### Overall Progress
- Total tasks: 21
- Completed: 10 (7 Phase 1 + 2 Phase 4 + 1 Phase 2 CORS)
- Progress: ~48%
- **Phase 1 (Security):** ✅ Complete
- **Phase 4 (Dev Experience):** ✅ Complete
- **Next:** mroki-hub (Web UI) — see below
- **Phase 2 (Observability):** Queue (1 week)
- **Phase 3 (Hardening):** Nice-to-have (1 week)

---

## 🎯 Recommended Next Steps

### mroki-hub (Web UI) 🔥 CURRENT FOCUS

**Goal:** Build a Vue 3 SPA for visualizing diffs and managing gates.

**Tech Stack:**
- Vue 3 + Composition API + `<script setup>`
- TypeScript
- Vite
- TailwindCSS v4
- `vue-diff` for diff visualization
- Native `fetch()` (no Axios)
- `createWebHistory` router
- NO Pinia for v1 (component-local state only)

**Location:** `web/` directory (fresh start, NOT rebasing `feature/mroki-hub` branch)

**v1 Scope:**
1. Gate creation (create new gate, view gate list)
2. Request browser (list requests with filters, sort, pagination)
3. Diff visualization (side-by-side view of live vs shadow responses)

**NOT in v1:**
- Agent monitoring (no backend endpoint)
- Dashboard stats (no backend endpoint)
- Gate edit/delete (no backend endpoints)

**API Integration:**
- Base URL: `http://localhost:8090`
- Auth: `Authorization: Bearer <key>`
- Responses wrapped in `{"data": ...}` envelope
- Paginated responses add `{"pagination": {limit, offset, total, has_more}}`
- Errors: RFC 7807 format
- CORS: `MROKI_APP_CORS_ORIGINS=http://localhost:5173`

**Implementation Phases:**
1. Scaffold Vite + Vue 3 + TypeScript + TailwindCSS v4 project
2. API client + TypeScript types
3. Gate page (list + create)
4. Request browser (list + filters + pagination)
5. Diff viewer (request detail with side-by-side diff)

### For Production Readiness (Later)

After hub, focus on Phase 2 (Observability) before production deployment:

1. **P1-1: Request ID Middleware** (4 hours) — For debugging
2. **P1-5: Structured Logging** (4 hours) — For troubleshooting
3. **P1-6: Update API_CONTRACTS.md** (1 hour) — Document auth/rate limiting

---

## 🏁 Phase 1 Quick Wins (COMPLETE) ✅

All Phase 1 security tasks are complete:

1. ✅ **RFC 7807 error format** - Already implemented
2. ✅ **HTTP server timeouts** - Protects against Slowloris (commit 32544d9)
3. ✅ **Request body size limits** - 10MB default (commit 32544d9)
4. ✅ **Graceful shutdown** - Clean SIGTERM handling (commit aaaab10)
5. ✅ **API key authentication** - Bearer token (commit 3b34114)
6. ✅ **Rate limiting** - 1000 req/min per IP (commit dd85ad8)
7. ✅ **Input validation** - Value objects at boundary (commit 1effcfc)

**Security posture: Production-ready! 🛡️**

---

## 📝 Notes

### Dependencies Between Tasks

**Phase 4 (Dev Experience):** ✅ COMPLETE
- DEV-1 Phase 1 (core diff engine): Done
- DEV-2 (TTL cleanup): Done
- DEV-1 Phases 2-3 (per-gate config): Deferred

**Next: mroki-hub**
- Depends on CORS (done) and working API
- Can proceed independently of Phase 2/3

**Phase 2 (Observability):**
- P1-1 (Request ID) should come before P1-5 (Structured logging)
- P1-2 (Circuit Breaker) independent - can be done anytime
- P1-6 (Update docs) can be done alongside implementation

**Phase 3 (Hardening):**
- All tasks are independent
- Can be done in any order
- Not blockers for dev environment usage

**Cross-phase:**
- Phase 1 ✅ Complete — provides security foundation
- Phase 4 ✅ Complete — makes tool usable
- mroki-hub 🔥 Next — web interface for visualization
- Phase 2 — Needed before production scale
- Phase 3 — Nice-to-have optimizations

### Team Allocation Suggestions

**If working solo (current status):**
1. Week 1: ✅ Phase 1 complete
2. Week 2: ✅ Phase 4 complete (DEV-1 Phase 1 + DEV-2 + CORS)
3. Week 3: 🔥 mroki-hub v1 (gate page + request browser + diff viewer)
4. Week 4+: Phase 2 tasks based on production timeline

**If multiple developers:**
- **Dev 1:** mroki-hub — Web UI implementation
- **Dev 2:** P1-1 + P1-5 (Request ID + Logging) — Can start in parallel
- **Dev 3:** Documentation updates — P1-6, DOC-1, DOC-2

**Current priority:**
- ✅ Phase 1 and Phase 4 complete
- 🔥 mroki-hub v1 is the next high-impact deliverable
- ✅ CORS already done, enabling hub development

### Testing Strategy

**Phase 4 (Dev Experience):** ✅ All tests passing
- Unit tests for field normalization (whitelist + blacklist strategies)
- Wildcard pattern tests: `users.#.created_at`
- Performance benchmarks: gjson+go-cmp vs JD baseline
- Integration tests for TTL cleanup job

**Phase 2 (Observability):**
- Unit tests for all middleware
- Integration tests for auth + rate limiting
- Manual testing for graceful shutdown
- Load testing for rate limiting and connection pooling

**Phase 3 (Hardening):**
- TLS configuration tests
- Compression ratio measurements
- Idempotency tests for request deduplication

### Deployment Strategy

**For Dev Environments (Current):**
1. ✅ Phase 1 complete — API is secure and stable
2. ✅ Phase 4 complete — Diff engine rewritten, TTL cleanup active, CORS enabled
3. 🔥 Build mroki-hub — Web interface for visualization
4. Start using in local/staging environments
5. Gather feedback

**For Production (Later):**
1. Complete Phase 2 - Add observability and resilience
2. Deploy to staging environment
3. Run for 1 week, monitor logs and metrics
4. Load test with production-like traffic
5. Complete Phase 3 if needed (TLS, compression, etc.)
6. Deploy to production with phased rollout
7. Monitor closely for first 48 hours

---

## 📞 Support

**Questions?** Review these docs:
- [Architecture Overview](docs/architecture/OVERVIEW.md)
- [API Contracts](docs/architecture/API_CONTRACTS.md)
- [Development Guide](docs/guides/DEVELOPMENT.md)

**Ready to implement?** Each task has detailed implementation notes above.

**Need help?** Check the codebase for similar patterns (middleware, tests, config).
