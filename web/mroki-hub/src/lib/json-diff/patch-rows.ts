import type { PatchOp } from '@/api'
import type { Token } from './types'
import { tokenizeJson } from './tokens'
import { buildContext, classifyArray, alignArray, type AlignEntry, type DiffContext } from './align'

/**
 * Per-`buildPatchRows`-call cache of array alignments, keyed by the parent
 * array's shadow pointer. Each entry maps a shadow (y) index to its aligned
 * slot so old-value resolution is O(1) per segment instead of re-running the
 * O(N) `alignArray` for every op that descends through the same array (#5).
 */
type AlignCache = Map<string, Map<number, AlignEntry>>

// Strings longer than this get an expandable detail block in the patch list.
const LONG_STRING = 48

export interface PatchRow {
  op: PatchOp['op']
  path: string
  /** Path segments before the leaf (pointer-unescaped), rendered de-emphasized. */
  pathPrefix: string[]
  /** Final path segment, emphasized. Array indices are wrapped as `[n]`. */
  leafLabel: string
  leafIsIndex: boolean
  hasOld: boolean
  oldValue: unknown
  /** Compact, single-line highlighted tokens for the old value. */
  oldInline: Token[]
  hasNew: boolean
  newValue: unknown
  /** Compact, single-line highlighted tokens for the new value. */
  newInline: Token[]
  /** Plain-text "old → new" summary for a hover title. */
  valueTitle: string
  /** Whether the value is complex/long enough to warrant an expandable detail block. */
  expandable: boolean
}

function unescapeToken(t: string): string {
  return t.replace(/~1/g, '/').replace(/~0/g, '~')
}

/** Resolve an RFC 6901 JSON Pointer against a document. */
export function resolvePointer(doc: unknown, pointer: string): { found: boolean; value: unknown } {
  if (pointer === '' || pointer === '/') return { found: true, value: doc }
  const parts = pointer.split('/').slice(1).map(unescapeToken)
  let cur: unknown = doc
  for (const part of parts) {
    if (Array.isArray(cur)) {
      const idx = Number(part)
      if (!Number.isInteger(idx) || idx < 0 || idx >= cur.length) {
        return { found: false, value: undefined }
      }
      cur = cur[idx]
    } else if (cur !== null && typeof cur === 'object') {
      const rec = cur as Record<string, unknown>
      if (!(part in rec)) return { found: false, value: undefined }
      cur = rec[part]
    } else {
      return { found: false, value: undefined }
    }
  }
  return { found: true, value: cur }
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}

/**
 * Recursively sort object keys into a single, stable order.
 *
 * A Patch row draws its two sides from different serialisations: the NEW value
 * comes from the diff content (stored as TEXT, produced by Go's `json.Marshal`,
 * which orders map keys alphabetically) while the OLD value is reconstructed
 * from the live body (stored as JSONB, which Postgres reorders by key length
 * then byte order). The same logical object can therefore render with two
 * different key orders. Canonicalising both sides here keeps the inline tokens,
 * the expandable detail, and the hover title aligned. Purely cosmetic — the
 * stored diff and the API contract are unaffected.
 *
 * Already-canonical input is returned by reference: a new container is
 * allocated lazily, and only at the levels whose keys are actually out of order
 * (or that contain a rebuilt child). The Go-sorted `add` side therefore costs
 * nothing, and the JSONB-ordered old value is rebuilt only where it differs.
 */
function canonicalizeKeys(v: unknown): unknown {
  if (Array.isArray(v)) {
    let out: unknown[] | null = null
    for (let i = 0; i < v.length; i++) {
      const c = canonicalizeKeys(v[i])
      if (!out && c !== v[i]) out = v.slice(0, i)
      out?.push(c)
    }
    return out ?? v
  }
  if (isRecord(v)) {
    const keys = Object.keys(v)
    const inOrder = keys.every((k, i) => i === 0 || keys[i - 1]! <= k)
    const order = inOrder ? keys : [...keys].sort()
    let out: Record<string, unknown> | null = inOrder ? null : {}
    for (let i = 0; i < order.length; i++) {
      const k = order[i]!
      const c = canonicalizeKeys(v[k])
      if (!out && c !== v[k]) {
        out = {}
        for (let j = 0; j < i; j++) out[order[j]!] = v[order[j]!]
      }
      if (out) out[k] = c
    }
    return out ?? v
  }
  return v
}

/** Return (and memoize) the shadow-index → aligned-slot map for one array. */
function alignmentFor(
  ctx: DiffContext,
  shadowPath: string,
  liveLen: number,
  shadowLen: number,
  cache: AlignCache
): Map<number, AlignEntry> {
  const hit = cache.get(shadowPath)
  if (hit) return hit
  const byShadow = new Map<number, AlignEntry>()
  for (const e of alignArray(liveLen, shadowLen, classifyArray(ctx, shadowPath))) {
    if (e.shadowIndex !== null) byShadow.set(e.shadowIndex, e)
  }
  cache.set(shadowPath, byShadow)
  return byShadow
}

/**
 * Resolve the OLD (live-side) value for a remove/replace op.
 *
 * Intermediate array segments of an op path are addressed in shadow (y) space,
 * so they must be translated to live (x) space via the array alignment before
 * indexing into liveDoc — otherwise a reorder/insert shifts every subsequent
 * element and the old value resolves against the wrong entry (issue #1). The
 * final segment of a `remove` is already live-indexed and is read directly.
 *
 * Alignments are memoized per array path in `cache` so resolving many ops under
 * the same reordered array stays linear overall rather than O(ops·N) (#5).
 */
function resolveOldValue(
  op: PatchOp,
  liveDoc: unknown,
  shadowDoc: unknown,
  ctx: DiffContext,
  cache: AlignCache
): { found: boolean; value: unknown } {
  const notFound = { found: false, value: undefined }
  const segs = op.path.split('/').slice(1)
  const translateCount = op.op === 'remove' ? segs.length - 1 : segs.length
  let liveCur: unknown = liveDoc
  let shadowCur: unknown = shadowDoc
  let shadowPath = ''
  for (let s = 0; s < translateCount; s++) {
    const rawSeg = segs[s]!
    if (Array.isArray(shadowCur)) {
      const y = Number(rawSeg)
      if (!Number.isInteger(y)) return notFound
      const liveArr = Array.isArray(liveCur) ? liveCur : []
      const entry = alignmentFor(ctx, shadowPath, liveArr.length, shadowCur.length, cache).get(y)
      if (!entry || entry.liveIndex === null) return notFound
      liveCur = liveArr[entry.liveIndex]
      shadowCur = shadowCur[y]
      shadowPath += `/${y}`
    } else {
      const key = unescapeToken(rawSeg)
      liveCur = isRecord(liveCur) ? liveCur[key] : undefined
      shadowCur = isRecord(shadowCur) ? shadowCur[key] : undefined
      shadowPath += `/${rawSeg}`
    }
  }
  if (op.op === 'remove') {
    const rawLeaf = segs[segs.length - 1]!
    if (Array.isArray(liveCur)) {
      const x = Number(rawLeaf)
      if (!Number.isInteger(x) || x < 0 || x >= liveCur.length) return notFound
      return { found: true, value: liveCur[x] }
    }
    const key = unescapeToken(rawLeaf)
    if (isRecord(liveCur) && key in liveCur) return { found: true, value: liveCur[key] }
    return notFound
  }
  return liveCur === undefined ? notFound : { found: true, value: liveCur }
}

function isComplex(v: unknown): boolean {
  return v !== null && typeof v === 'object'
}

function isLongString(v: unknown): boolean {
  return typeof v === 'string' && v.length > LONG_STRING
}

function plainText(v: unknown): string {
  if (v === undefined) return ''
  if (typeof v === 'string') return v
  return JSON.stringify(v)
}

function splitPath(path: string): {
  pathPrefix: string[]
  leafLabel: string
  leafIsIndex: boolean
} {
  const segs = path.split('/').filter(Boolean).map(unescapeToken)
  const leaf = segs.pop() ?? ''
  const leafIsIndex = /^\d+$/.test(leaf)
  return { pathPrefix: segs, leafLabel: leafIsIndex ? `[${leaf}]` : leaf, leafIsIndex }
}

/**
 * Build flat, render-ready rows from a list of RFC 6902 patch ops.
 *
 * The patch transforms liveDoc -> shadowDoc, so `op.value` is the NEW value
 * (add/replace) while the OLD value (remove/replace) is resolved from liveDoc.
 *
 * `shadowDoc` is required: the OLD value is resolved through the array alignment
 * so reordered/inserted array elements map back to their real live index (issue
 * #1). Resolving positionally against a shadow (y) indexed pointer would shift
 * onto the wrong element, so callers must always supply the shadow document.
 */
export function buildPatchRows(ops: PatchOp[], liveDoc: unknown, shadowDoc: unknown): PatchRow[] {
  const ctx = buildContext(ops)
  const cache: AlignCache = new Map()
  return ops.map((op) => {
    const { pathPrefix, leafLabel, leafIsIndex } = splitPath(op.path)

    const hasNew = op.op === 'add' || op.op === 'replace'
    const newValue = hasNew ? canonicalizeKeys(op.value) : undefined

    const oldRes =
      op.op === 'add'
        ? { found: false, value: undefined }
        : resolveOldValue(op, liveDoc, shadowDoc, ctx, cache)
    const hasOld = oldRes.found
    const oldValue = canonicalizeKeys(oldRes.value)

    const valueTitle =
      op.op === 'replace'
        ? `${plainText(oldValue)}  →  ${plainText(newValue)}`
        : op.op === 'add'
          ? plainText(newValue)
          : plainText(oldValue)

    return {
      op: op.op,
      path: op.path,
      pathPrefix,
      leafLabel,
      leafIsIndex,
      hasOld,
      oldValue,
      oldInline: hasOld ? tokenizeJson(oldValue) : [],
      hasNew,
      newValue,
      newInline: hasNew ? tokenizeJson(newValue) : [],
      valueTitle,
      expandable:
        isComplex(oldValue) ||
        isComplex(newValue) ||
        isLongString(oldValue) ||
        isLongString(newValue),
    }
  })
}
