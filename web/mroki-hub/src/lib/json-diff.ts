import type { PatchOp } from '@/api'

export type TokenType = 'key' | 'string' | 'number' | 'boolean' | 'null' | 'bracket' | 'punctuation'
export interface Token {
  text: string
  type: TokenType
}
export type LineType = 'normal' | 'added' | 'removed' | 'replaced-old' | 'replaced-new'
export interface DiffLine {
  tokens: Token[]
  type: LineType
  indent: number
  path: string
}
export interface SplitRow {
  left: DiffLine | null
  right: DiffLine | null
}

export function buildDiffLines(live: unknown, shadow: unknown, ops: PatchOp[]): DiffLine[] {
  const opMap = new Map<string, PatchOp>()
  for (const op of ops) opMap.set(op.path, op)
  const lines: DiffLine[] = []
  walkMerged(live, shadow, '', 0, opMap, lines)
  return lines
}

export function stripPathPrefix(ops: PatchOp[], prefix: string): PatchOp[] {
  return ops
    .filter((op) => op.path.startsWith(prefix + '/') || op.path === prefix)
    .map((op) => ({ ...op, path: op.path === prefix ? '/' : op.path.slice(prefix.length) }))
}

export function buildSplitRows(lines: DiffLine[]): SplitRow[] {
  const rows: SplitRow[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]!
    if (line.type === 'normal') {
      rows.push({ left: line, right: line })
      i++
    } else if (line.type === 'replaced-old') {
      const next = i + 1 < lines.length ? lines[i + 1]! : null
      if (next && next.type === 'replaced-new' && next.path === line.path) {
        rows.push({ left: line, right: next })
        i += 2
      } else {
        rows.push({ left: line, right: null })
        i++
      }
    } else if (line.type === 'removed') {
      rows.push({ left: line, right: null })
      i++
    } else {
      rows.push({ left: null, right: line })
      i++
    }
  }
  return rows
}

// --- Token helpers ---
function valTok(v: unknown): Token[] {
  if (v === null) return [{ text: 'null', type: 'null' }]
  if (v === undefined) return [{ text: 'undefined', type: 'null' }]
  if (typeof v === 'string') return [{ text: `"${v}"`, type: 'string' }]
  if (typeof v === 'number') return [{ text: String(v), type: 'number' }]
  if (typeof v === 'boolean') return [{ text: String(v), type: 'boolean' }]
  return [{ text: JSON.stringify(v), type: 'string' }]
}
function keyTok(k: string): Token[] {
  return [
    { text: `"${k}"`, type: 'key' },
    { text: ': ', type: 'punctuation' },
  ]
}
function comma(): Token {
  return { text: ',', type: 'punctuation' }
}
function br(t: string): Token {
  return { text: t, type: 'bracket' }
}
function appendComma(lines: DiffLine[]): void {
  const last = lines[lines.length - 1]
  if (last) last.tokens = [...last.tokens, comma()]
}

// --- Helpers ---
function isCont(v: unknown): boolean {
  return isObj(v) || Array.isArray(v)
}
function isObj(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}
function hasOpsUnder(path: string, opMap: Map<string, PatchOp>): boolean {
  for (const p of opMap.keys()) if (p === path || p.startsWith(path + '/')) return true
  return false
}
function escPtr(t: string): string {
  return t.replace(/~/g, '~0').replace(/\//g, '~1')
}

// --- Core walk ---
function walkMerged(
  live: unknown,
  shadow: unknown,
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const op = opMap.get(path)
  if (op && !isCont(live) && !isCont(shadow)) {
    if (op.op === 'replace') {
      lines.push({ tokens: valTok(live), type: 'replaced-old', indent, path })
      lines.push({ tokens: valTok(shadow), type: 'replaced-new', indent, path })
    } else if (op.op === 'remove') {
      lines.push({ tokens: valTok(live), type: 'removed', indent, path })
    } else if (op.op === 'add') {
      lines.push({ tokens: valTok(shadow), type: 'added', indent, path })
    }
    return
  }
  if (isObj(live) && isObj(shadow)) {
    walkObject(live, shadow, path, indent, opMap, lines)
    return
  }
  if (Array.isArray(live) && Array.isArray(shadow)) {
    walkArray(live, shadow, path, indent, opMap, lines)
    return
  }
  if (op) {
    renderBlock(live, indent, 'removed', path, lines)
    renderBlock(shadow, indent, 'added', path, lines)
  } else {
    lines.push({ tokens: valTok(live), type: 'normal', indent, path })
  }
}

function walkObject(
  live: Record<string, unknown>,
  shadow: Record<string, unknown>,
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const allKeys = [...new Set([...Object.keys(live), ...Object.keys(shadow)])]
  lines.push({ tokens: [br('{')], type: 'normal', indent, path })
  allKeys.forEach((key, i) => {
    const cp = `${path}/${escPtr(key)}`
    const last = i === allKeys.length - 1
    const inL = key in live,
      inS = key in shadow
    if (inL && inS) {
      if (hasOpsUnder(cp, opMap)) {
        if (isCont(live[key]) && isCont(shadow[key])) {
          lines.push({ tokens: keyTok(key), type: 'normal', indent: indent + 1, path: cp })
          walkMerged(live[key], shadow[key], cp, indent + 1, opMap, lines)
          if (!last) appendComma(lines)
        } else {
          const op = opMap.get(cp)
          if (op && op.op === 'replace') {
            lines.push({
              tokens: [...keyTok(key), ...valTok(live[key])],
              type: 'replaced-old',
              indent: indent + 1,
              path: cp,
            })
            lines.push({
              tokens: [...keyTok(key), ...valTok(shadow[key]), ...(last ? [] : [comma()])],
              type: 'replaced-new',
              indent: indent + 1,
              path: cp,
            })
          } else {
            lines.push({ tokens: keyTok(key), type: 'normal', indent: indent + 1, path: cp })
            walkMerged(live[key], shadow[key], cp, indent + 1, opMap, lines)
            if (!last) appendComma(lines)
          }
        }
      } else {
        lines.push({
          tokens: [...keyTok(key), ...valTok(live[key]), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    } else if (inL) {
      lines.push({
        tokens: [...keyTok(key), ...valTok(live[key]), ...(last ? [] : [comma()])],
        type: 'removed',
        indent: indent + 1,
        path: cp,
      })
    } else {
      lines.push({
        tokens: [...keyTok(key), ...valTok(shadow[key]), ...(last ? [] : [comma()])],
        type: 'added',
        indent: indent + 1,
        path: cp,
      })
    }
  })
  lines.push({ tokens: [br('}')], type: 'normal', indent, path })
}

function walkArray(
  live: unknown[],
  shadow: unknown[],
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const max = Math.max(live.length, shadow.length)
  lines.push({ tokens: [br('[')], type: 'normal', indent, path })
  for (let i = 0; i < max; i++) {
    const cp = `${path}/${i}`
    const last = i === max - 1
    const inL = i < live.length,
      inS = i < shadow.length
    if (inL && inS) {
      if (hasOpsUnder(cp, opMap)) {
        walkMerged(live[i], shadow[i], cp, indent + 1, opMap, lines)
        if (!last) appendComma(lines)
      } else {
        lines.push({
          tokens: [...valTok(live[i]), ...(last ? [] : [comma()])],
          type: 'normal',
          indent: indent + 1,
          path: cp,
        })
      }
    } else if (inL) {
      lines.push({
        tokens: [...valTok(live[i]), ...(last ? [] : [comma()])],
        type: 'removed',
        indent: indent + 1,
        path: cp,
      })
    } else {
      lines.push({
        tokens: [...valTok(shadow[i]), ...(last ? [] : [comma()])],
        type: 'added',
        indent: indent + 1,
        path: cp,
      })
    }
  }
  lines.push({ tokens: [br(']')], type: 'normal', indent, path })
}

function renderBlock(
  value: unknown,
  indent: number,
  type: LineType,
  path: string,
  lines: DiffLine[]
): void {
  const formatted = JSON.stringify(value, null, 2)
  for (const line of formatted.split('\n')) {
    lines.push({ tokens: [{ text: line.trim(), type: 'string' }], type, indent, path })
  }
}
