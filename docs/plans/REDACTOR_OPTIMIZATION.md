# Redactor Optimization & JSONB Storage — Implementation Plan

Eliminate redundant JSON parse/serialize cycles in the redact+diff pipeline, and migrate
body storage from `bytea` (base64) to native `JSONB`.

## Problem

### 1. Redundant parsing

The pipeline parses and serializes JSON bodies **multiple times** per response:

```
Redactor.redactBody()
  json.Unmarshal(body)           ← PARSE 1
  walk tree, set [REDACTED]
  json.Marshal(tree)             ← SERIALIZE 1, tree discarded

computeDiffFromDecoded()
  json.Marshal(headers) × 2     ← headers already in memory
  fmt.Sprintf(envelope)          ← string concat

diff.JSON()
  gjson.ValidBytes()             ← redundant validation
  NormalizeBytes()               ← per-field gjson/sjson byte ops
  gjson.ParseBytes()             ← PARSE 2
  cmp.Equal()                    ← compare
```

### 2. Opaque storage

Bodies are stored as base64-encoded `bytea`. The frontend must `atob()` + `JSON.parse()`
on every view. The API can't return inline JSON. PostgreSQL can't query body content.

### 3. Duplicated tree-walk logic

The redactor (`setRedacted`) and normalizer (`NormalizeBytes`) both walk dot-separated
paths with `#` array wildcards — but one operates on `map[string]any`, the other on
`[]byte` via gjson/sjson. Two implementations of the same semantics.

---

## Target Architecture

```
Proxy → base64 body → API
                        ↓
  decodeBase64 → json.Unmarshal (ONCE)
                        ↓
  redactor.Redact() → BodyParsed (map[string]any, retained)
                        ↓
  ┌─ json.Marshal(BodyParsed) → JSONB column (storage)
  │
  └─ BuildEnvelope(status, headers, BodyParsed) → map[string]any
       ↓
     NormalizeTree() → map ops (no gjson/sjson)
       ↓
     cmp.Equal() → []PatchOp
                        ↓
  API GET → body is inline JSON object (no base64)
                        ↓
  Frontend → typeof body === 'object' ? <JsonTree/> : <TextBlock/>
             (no atob, no JSON.parse)
```

---

## Schema Changes

### Current

```
Request:  ..., headers JSONB, body BYTEA (base64-encoded)
Response: ..., headers JSONB, body BYTEA (base64-encoded)
```

### Target

```
Request:  ..., headers JSONB, body JSONB (object / string / null)
Response: ..., headers JSONB, body JSONB (object / string / null)
```

`body JSONB` stores:
- **JSON content** → native JSON object: `{"user": {"name": "Alice"}}`
- **Text content** → JSON string: `"<html>hello</html>"`
- **Binary content** → base64 JSON string: `"iVBORw0KGgo..."`
- **Empty / no body** → `null`

Content type is determined at runtime from `headers["Content-Type"]` — no extra column.
`jsonb_typeof(body)` can discriminate object vs string if needed for queries.

No `content_type` column. No functional indexes (ent doesn't support them natively).
If filtering by content type becomes a hot query, add a plain column + B-tree later.

---

## Phases

### Phase 1: Shared tree-walk utility

**Files:** `internal/domain/traffictesting/treewalk.go` (new), tests

Extract the path-walking logic shared between `setRedacted` and the future `NormalizeTree`
into a reusable utility:

```go
// WalkPath navigates a map[string]any / []any tree to the node at `path`.
// Calls visitor(parent, key) at the leaf. Supports "#" wildcard for arrays.
func WalkPath(tree any, path string, visitor func(parent map[string]any, key string))
```

**Changes:**
- Refactor `setRedacted` to use `WalkPath` internally
- Same path syntax: dot-separated, `#` for array iteration
- Visitor pattern: caller decides what to do at the leaf (redact, delete, copy)

**Tests:**
- Existing `redactor_test.go` tests must pass unchanged (refactor, not behavior change)
- Unit tests for `WalkPath` with edge cases: missing keys, nested arrays, empty paths

### Phase 2: Extend `RedactResult` with `BodyParsed`

**Files:** `redactor.go`, `redactor_test.go`

**Changes:**
- Add `BodyParsed any` field to `RedactResult`
- In `redactBody()`, after `setRedacted(root, ...)`, retain `root` as `BodyParsed`
- `json.Marshal(root)` still runs (produces `Body` for storage) — no behavior change yet
- Non-JSON / empty body: `BodyParsed = nil`
- Existing callers unaffected — `BodyParsed` is additive

**Tests:**
- Assert `BodyParsed` is non-nil for JSON bodies and reflects redacted values
- Assert `BodyParsed` is nil for non-JSON / empty bodies
- Assert `Body` bytes match `json.Marshal(BodyParsed)` (consistency check)

### Phase 3: Tree-based field normalization

**Files:** `pkg/diff/normalizer.go`, `pkg/diff/normalizer_test.go`

**Changes:**
- Add `NormalizeTree(tree any) any` to `FieldNormalizer`
- Uses `WalkPath` from Phase 1 internally
- Whitelist: build new `map[string]any` keeping only included paths
- Blacklist: delete keys in-place from the tree
- `#` wildcard: iterate `[]any` slices, apply to each element
- No gjson/sjson — pure Go map/slice operations

**Tests:**
- Mirror every existing `NormalizeBytes` test with a `NormalizeTree` equivalent
- Assert identical output between `NormalizeBytes` and `NormalizeTree` on same inputs
- Benchmark: `NormalizeTree` vs `NormalizeBytes` on 1KB, 100KB, 1MB bodies

### Phase 4: `diff.Parsed` entry point + envelope builder

**Files:** `pkg/diff/parsed.go` (new), `pkg/diff/envelope.go` (new), tests

#### 4a: Envelope builder

Build the synthetic diff envelope as a Go value tree instead of `fmt.Sprintf`:

```go
func BuildEnvelope(statusCode int, headers http.Header, bodyParsed any) map[string]any
```

**Critical type matching:** `json.Unmarshal` produces `float64` for numbers and
`[]any` for arrays. The envelope must match:
- `statusCode` → `float64(statusCode)` (not `int`)
- `headers` → `map[string]any` where values are `[]any{string, ...}` (match `json.Marshal`→`gjson.ParseBytes` shape)
- `bodyParsed` → as-is from `json.Unmarshal` (already correct types)

#### 4b: `diff.Parsed`

```go
func Parsed(a, b any, opts ...Option) ([]PatchOp, error)
```

- Applies `NormalizeTree` instead of `NormalizeBytes`
- Passes normalized trees directly to `cmp.Equal` with `patchReporter`
- Skips: `gjson.ValidBytes`, `gjson.ParseBytes`, byte→string→byte conversions
- Existing `diff.JSON()` remains unchanged

**Tests:**
- Same test cases as `diff.JSON` — assert identical `[]PatchOp` output
- `BuildEnvelope` output matches `gjson.ParseBytes(fmt.Sprintf(...)).Value()` for:
  - nil body, empty headers, multi-value headers, numeric status codes
- Property test: for N random JSON docs, `diff.JSON(a, b) == diff.Parsed(unmarshal(a), unmarshal(b))`

### Phase 5: Wire up optimized diff path

**Files:**
- `internal/application/commands/create_request.go`
- `pkg/proxy/diff.go`

#### 5a: `CreateRequestHandler`

Replace `computeDiffFromDecoded()` with `computeDiffFromParsed()`:

```go
func computeDiffFromParsed(
    liveStatus int, liveHeaders http.Header, liveBodyParsed any,
    shadowStatus int, shadowHeaders http.Header, shadowBodyParsed any,
    opts ...diff.Option,
) ([]diff.PatchOp, error) {
    liveEnvelope := diff.BuildEnvelope(liveStatus, liveHeaders, liveBodyParsed)
    shadowEnvelope := diff.BuildEnvelope(shadowStatus, shadowHeaders, shadowBodyParsed)
    return diff.Parsed(liveEnvelope, shadowEnvelope, opts...)
}
```

Pass `RedactResult.BodyParsed` directly. No more `json.Marshal(headers)`,
`fmt.Sprintf`, `jsonBodyOrNull`.

#### 5b: `proxyResponseDiffer`

Update `Diff()` to use `diff.Parsed` + `BuildEnvelope` when possible.
Fall back to `diff.JSON` for non-JSON bodies (where `BodyParsed` is nil).

**Tests:**
- Existing diff tests must produce identical `[]PatchOp` output
- Integration: redact → diff pipeline produces same results end-to-end

### Phase 6: Storage migration — `body bytea` → `body JSONB`

**Files:**
- `ent/schema/request.go`, `ent/schema/response.go`
- `ent/migrate/migrations/` (new migration)
- `internal/domain/traffictesting/request.go`, `response.go`
- `internal/infrastructure/persistence/ent/mapper.go`
- `internal/application/commands/create_request.go`
- `pkg/dto/request.go`

#### 6a: Schema change

```go
// ent/schema/request.go — change:
field.Bytes("body").Optional()
// to:
field.JSON("body", json.RawMessage{}).Optional()

// ent/schema/response.go — same change
```

Using `json.RawMessage` as the Go type lets ent store any valid JSON value
(object, string, null) as PostgreSQL `jsonb`.

#### 6b: Domain model change

```go
// request.go, response.go — change:
Body []byte
// to:
Body json.RawMessage
```

`json.RawMessage` is `[]byte` underneath, so most code works as-is.
The semantic difference: it's now always valid JSON, not arbitrary bytes.

#### 6c: Write path (ingestion)

In `CreateRequestHandler`:
- Decode base64 from proxy wire format (unchanged)
- Redact → get `BodyParsed` (Phase 2)
- For JSON bodies: `json.Marshal(BodyParsed)` → store as `json.RawMessage`
- For non-JSON bodies: `json.Marshal(string(rawBytes))` → store as JSON string
- For empty bodies: `nil` → `NULL` in DB
- **Drop** `encodeBase64Body` — no more base64 for storage

#### 6d: Migration

```sql
-- Convert existing base64 bytea to JSONB
-- Bodies are base64-encoded JSON. Decode base64 → parse JSON → store as jsonb.
ALTER TABLE requests
  ALTER COLUMN body TYPE jsonb
  USING CASE
    WHEN body IS NULL THEN NULL
    ELSE convert_from(decode(body::text, 'base64'), 'UTF-8')::jsonb
  END;

ALTER TABLE responses
  ALTER COLUMN body TYPE jsonb
  USING CASE
    WHEN body IS NULL THEN NULL
    ELSE convert_from(decode(body::text, 'base64'), 'UTF-8')::jsonb
  END;
```

**Note:** This migration transforms existing data. Test on a copy first.
Non-JSON bodies that were base64-encoded will need special handling
(wrap in a JSON string before casting).

### Phase 7: API response changes

**Files:**
- `pkg/dto/request.go`
- `internal/interfaces/http/handlers/request.go`

#### 7a: DTO changes

```go
// Change Body from string (base64) to json.RawMessage (inline JSON)
type RequestDetail struct {
    // ...
    Body json.RawMessage `json:"body"` // was: string
}

type ResponseDetail struct {
    // ...
    Body json.RawMessage `json:"body"` // was: string
}
```

The API response goes from:
```json
{"body": "eyJ1c2VyIjp7Im5hbWUiOiJBbGljZSJ9fQ=="}
```
to:
```json
{"body": {"user": {"name": "Alice"}}}
```

#### 7b: Handler mapping

```go
// Change:
Body: string(resp.Body)  // was: cast base64 []byte to string
// To:
Body: resp.Body           // json.RawMessage passes through directly
```

#### 7c: Input DTO (proxy → API)

The wire format from proxy to API still sends base64 (proxy doesn't change).
The command handler decodes it (Phase 6c). No change to `CreateRequestPayload`.

**Tests:**
- API handler tests: verify body is returned as inline JSON, not base64
- Verify non-JSON bodies are returned as JSON strings

### Phase 8: Frontend updates

**Files:**
- `web/mroki-hub/src/components/diff/DiffViewer.vue`
- `web/mroki-hub/src/pages/RequestDetail.vue`
- Related TypeScript types

**Changes:**
- Remove `atob()` / `decodeBody()` — body is already decoded
- Remove `JSON.parse()` / `tryParseJson()` — body is already an object for JSON
- Update type: `body: string` → `body: unknown` (object | string | null)
- Rendering logic:
  ```typescript
  const contentType = response.headers?.['Content-Type']?.[0] ?? ''
  if (contentType.startsWith('application/json') && typeof body === 'object') {
    // render as JSON tree
  } else if (typeof body === 'string') {
    // render as text
  } else {
    // no body
  }
  ```
- Update cURL builder: body is now an object, need `JSON.stringify()` for `-d` flag
- Update diff viewer: bodies are already objects, no decode step

**Tests:**
- Verify JSON body renders as tree
- Verify text body renders as text block
- Verify null body shows "no body"
- Verify cURL builder outputs correct body

---

## Phase Dependencies

```
Phase 1 (tree-walk utility)
  ↓
Phase 2 (BodyParsed)     Phase 3 (NormalizeTree)
  ↓                        ↓
Phase 4 (diff.Parsed + envelope)
  ↓
Phase 5 (wire up callers)
  ↓
Phase 6 (JSONB storage)
  ↓
Phase 7 (API changes)
  ↓
Phase 8 (frontend)
```

Phases 2 and 3 can be done in parallel. Phases 6-8 are sequential.
Each phase is independently deployable — no big-bang migration.

---

## Equivalence & Verification

The optimized path must produce **identical** `[]PatchOp` output as the current path.

| Concern | Risk | Mitigation |
|---------|------|------------|
| Number types | `gjson` and `json.Unmarshal` both produce `float64` | Verify edge cases: large ints, scientific notation |
| Header shape | `json.Marshal(http.Header)` → arrays; `BuildEnvelope` must match | Test multi-value headers, single-value headers |
| Map key ordering | `cmp.Equal` is order-independent | No issue |
| Null vs missing body | Current: `"body": null` via `jsonBodyOrNull` | `BuildEnvelope`: `envelope["body"] = nil` |
| Normalizer path syntax | `NormalizeTree` must handle `#` wildcards identically | Mirror all `NormalizeBytes` tests |

### Benchmarks

```
BenchmarkPipeline_Current   — redact → marshal envelope → diff.JSON
BenchmarkPipeline_Optimized — redact → BuildEnvelope → diff.Parsed
```

Matrix: 1KB, 100KB, 1MB, 10MB bodies × 0/5/20 redacted fields × 0/5/10 ignored fields.

Expected: ~50% fewer allocs, 30-50% faster on large bodies, ~40% fewer bytes/op.

---

## Caddy Module Gaps

The Caddy module (`pkg/caddymodule`) currently does **not** support:

| Feature | Status | Notes |
|---------|--------|-------|
| Redaction (RedactedFields / Redactor) | ❌ Missing | No `redacted_fields` Caddyfile directive; bodies are diffed without redaction |
| Optimized diff path (BuildEnvelope + diff.Parsed) | ❌ Missing | Always uses byte-level `diff.JSON` fallback since no redactor → no parsed trees |
| Gate-level configuration | ❌ Missing | Caddy uses static Caddyfile config; no gate concept |

These are tracked for future work. The Caddy module remains functional — it uses the
byte-level diff path, which is correct but slower for large JSON bodies.

---

## What stays the same

- `headers.Clone()` in `Redact()` — mutation safety
- `cmp.Equal()` + `patchReporter` — comparison engine
- `diff.JSON()` — unchanged, still available
- `NormalizeBytes` — unchanged, still available
- Proxy → API wire format — still sends base64 (proxy is stateless, no DB)

## What gets removed

- `encodeBase64Body` / `decodeBase64Body` in the storage path
- `jsonBodyOrNull` helper
- `fmt.Sprintf` envelope construction in diff path
- `gjson.ValidBytes` validation of self-constructed JSON
- Frontend `atob()` / `JSON.parse()` on body display
