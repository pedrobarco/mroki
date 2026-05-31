import type { PatchOp } from '@/api'
import type { Token } from './types'
import { tokenizeJson } from './tokens'

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
 * (add/replace) while the OLD value (remove/replace) is resolved from liveDoc
 * via its JSON pointer.
 */
export function buildPatchRows(ops: PatchOp[], liveDoc: unknown): PatchRow[] {
  return ops.map((op) => {
    const { pathPrefix, leafLabel, leafIsIndex } = splitPath(op.path)

    const hasNew = op.op === 'add' || op.op === 'replace'
    const newValue = hasNew ? op.value : undefined

    const oldRes =
      op.op === 'add' ? { found: false, value: undefined } : resolvePointer(liveDoc, op.path)
    const hasOld = oldRes.found
    const oldValue = oldRes.value

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
