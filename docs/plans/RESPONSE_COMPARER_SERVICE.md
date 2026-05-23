# Plan: `ResponseComparer` Application Service

## Problem

The standalone callback (`cmd/mroki-proxy/handlers/proxy.go`) and the API handler (`internal/application/commands/create_request.go`) duplicate the same redact → envelope → diff pipeline. They diverge only at the boundaries (base64 encoding, persistence vs logging, error policy).

## Approach

Extract the shared core into `internal/application/services/ResponseComparer` — a small service that owns the redact+diff cycle. Both callers become thin wrappers around it.

## Steps

| # | Task | File | Description |
|---|------|------|-------------|
| 1 | Create service package + `ResponseComparer` | `internal/application/services/response_comparer.go` | `ResponseData` input struct, `CompareResult` output struct, `NewResponseComparer(redactor, diffOpts)`, `Compare(req, live, shadow)` method |
| 2 | Refactor `CreateRequestHandler` | `internal/application/commands/create_request.go` | Replace inline redact×3 + BuildEnvelope + diff.Parsed with `comparer.Compare(...)`. Keep base64 decode/encode, gate lookup, `cmd.Diff != nil` short-circuit |
| 3 | Refactor standalone callback | `cmd/mroki-proxy/handlers/proxy.go` | Replace inline redact×3 + optimized diff path with `comparer.Compare(...)`. Keep `if redactor != nil` guard + byte-level fallback + `logDiffResult` |
| 4 | Update existing tests | `create_request_test.go`, `handlers_test.go` | Ensure existing tests still pass with the delegated path |
| 5 | Build + test | — | `go build` + `go test` all affected packages |

## Service Interface

```go
// internal/application/services/response_comparer.go
package services

type ResponseData struct {
    StatusCode int
    Headers    http.Header
    Body       []byte
}

type CompareResult struct {
    Request traffictesting.RedactResult
    Live    traffictesting.RedactResult
    Shadow  traffictesting.RedactResult
    Ops     []diff.PatchOp
}

type ResponseComparer struct {
    redactor *traffictesting.Redactor
    diffOpts []diff.Option
}

func NewResponseComparer(redactor *traffictesting.Redactor, diffOpts []diff.Option) *ResponseComparer

// Compare redacts all three inputs and diffs live vs shadow.
// Redaction errors are fatal (returned). Diff errors are non-fatal (empty ops).
func (c *ResponseComparer) Compare(req, live, shadow ResponseData) (*CompareResult, error)
```

## Caller Integration

### API handler (`create_request.go`)

```go
comparer := services.NewResponseComparer(redactor, diffOpts)
result, err := comparer.Compare(
    services.ResponseData{0, cmd.Headers, reqBodyDecoded},
    services.ResponseData{cmd.LiveResponse.StatusCode, cmd.LiveResponse.Headers, liveBodyDecoded},
    services.ResponseData{cmd.ShadowResponse.StatusCode, cmd.ShadowResponse.Headers, shadowBodyDecoded},
)
// result.Live.Body → re-encode to base64 for storage
// result.Ops → store as diff (unless cmd.Diff was pre-provided)
```

### Standalone callback (`handlers/proxy.go`)

```go
comparer := services.NewResponseComparer(redactor, cfg.DiffOptions)
result, err := comparer.Compare(
    services.ResponseData{0, req.Headers, req.Body},
    services.ResponseData{live.StatusCode, live.Response.Header, live.Body},
    services.ResponseData{shadow.StatusCode, shadow.Response.Header, shadow.Body},
)
// result.Ops → logDiffResult
// Apply redacted headers/body back to live/shadow for logging
```

## Design Decisions

- **`RedactResult` exposed in `CompareResult`** — callers need `Body` (for storage/logging) and `Headers` (to update the original structs). `BodyParsed` is consumed internally by the service but also available if needed.
- **No `BodyParsed` nil check** — the service always calls `BuildEnvelope` + `diff.Parsed`. Nil `BodyParsed` → `null` in envelope → valid comparison. Aligns both paths.
- **Byte-level fallback stays outside** — only needed when there's no redactor (standalone mode without config). The service always has a redactor, so it always uses the optimized path.
- **Request redaction included** — both callers redact the request too, so the service handles all three in one call.

## Error Handling

| Error source | Policy | Rationale |
|-------------|--------|-----------|
| Redaction failure | Fatal — `Compare` returns error | Caller decides how to handle (API fails request, proxy skips diff) |
| Diff computation failure | Non-fatal — returns empty `[]PatchOp{}` | Matches current API behavior; diff errors shouldn't block storage/logging |

## File Changes Summary

| Action | File |
|--------|------|
| Create | `internal/application/services/response_comparer.go` |
| Create | `internal/application/services/response_comparer_test.go` |
| Modify | `internal/application/commands/create_request.go` |
| Modify | `cmd/mroki-proxy/handlers/proxy.go` |

## Testing

- New unit tests for `ResponseComparer` in `response_comparer_test.go`
- Existing tests in `create_request_test.go` and `handlers_test.go` must pass unchanged
- `go test ./internal/application/services/... ./internal/application/commands/... ./cmd/mroki-proxy/handlers/...`
