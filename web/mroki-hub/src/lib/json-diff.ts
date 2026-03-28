import type { PatchOp } from '@/api'

export type LineType = 'normal' | 'added' | 'removed'

export interface DiffLine {
  content: string
  type: LineType
  indent: number
  path: string
}

/**
 * Build annotated diff lines from live and shadow JSON values using pre-computed PatchOps.
 * Produces a unified view: normal lines for unchanged content, removed (red) for live-only,
 * added (green) for shadow-only values.
 */
export function buildDiffLines(live: unknown, shadow: unknown, ops: PatchOp[]): DiffLine[] {
  const opMap = new Map<string, PatchOp>()
  for (const op of ops) {
    opMap.set(op.path, op)
  }
  const lines: DiffLine[] = []
  walkMerged(live, shadow, '', 0, opMap, lines)
  return lines
}

/**
 * Strip a prefix from PatchOp paths (e.g., "/body" -> makes "/body/id" become "/id").
 * Ops that don't match the prefix are excluded.
 */
export function stripPathPrefix(ops: PatchOp[], prefix: string): PatchOp[] {
  return ops
    .filter((op) => op.path.startsWith(prefix + '/') || op.path === prefix)
    .map((op) => ({
      ...op,
      path: op.path === prefix ? '/' : op.path.slice(prefix.length),
    }))
}

// --- Core walk logic ---

function walkMerged(
  live: unknown,
  shadow: unknown,
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const op = opMap.get(path)

  // Leaf with a direct op
  if (op && !isContainer(live) && !isContainer(shadow)) {
    if (op.op === 'replace') {
      lines.push({ content: formatValue(live), type: 'removed', indent, path })
      lines.push({ content: formatValue(shadow), type: 'added', indent, path })
    } else if (op.op === 'remove') {
      lines.push({ content: formatValue(live), type: 'removed', indent, path })
    } else if (op.op === 'add') {
      lines.push({ content: formatValue(shadow), type: 'added', indent, path })
    }
    return
  }

  if (isObject(live) && isObject(shadow)) {
    walkObject(live, shadow, path, indent, opMap, lines)
    return
  }

  if (Array.isArray(live) && Array.isArray(shadow)) {
    walkArray(live, shadow, path, indent, opMap, lines)
    return
  }

  // Type mismatch with op
  if (op) {
    renderBlock(live, indent, 'removed', path, lines)
    renderBlock(shadow, indent, 'added', path, lines)
  } else {
    lines.push({ content: formatValue(live), type: 'normal', indent, path })
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
  lines.push({ content: '{', type: 'normal', indent, path })

  allKeys.forEach((key, i) => {
    const childPath = `${path}/${escapePointer(key)}`
    const comma = i < allKeys.length - 1 ? ',' : ''
    const hasInLive = key in live
    const hasInShadow = key in shadow

    if (hasInLive && hasInShadow) {
      if (hasOpsUnder(childPath, opMap)) {
        lines.push({ content: `"${key}":`, type: 'normal', indent: indent + 1, path: childPath })
        walkMerged(live[key], shadow[key], childPath, indent + 1, opMap, lines)
        appendComma(lines, comma)
      } else {
        lines.push({
          content: `"${key}": ${formatValue(live[key])}${comma}`,
          type: 'normal',
          indent: indent + 1,
          path: childPath,
        })
      }
    } else if (hasInLive) {
      lines.push({
        content: `"${key}": ${formatValue(live[key])}${comma}`,
        type: 'removed',
        indent: indent + 1,
        path: childPath,
      })
    } else {
      lines.push({
        content: `"${key}": ${formatValue(shadow[key])}${comma}`,
        type: 'added',
        indent: indent + 1,
        path: childPath,
      })
    }
  })

  lines.push({ content: '}', type: 'normal', indent, path })
}

function walkArray(
  live: unknown[],
  shadow: unknown[],
  path: string,
  indent: number,
  opMap: Map<string, PatchOp>,
  lines: DiffLine[]
): void {
  const maxLen = Math.max(live.length, shadow.length)
  lines.push({ content: '[', type: 'normal', indent, path })

  for (let i = 0; i < maxLen; i++) {
    const childPath = `${path}/${i}`
    const comma = i < maxLen - 1 ? ',' : ''
    const hasInLive = i < live.length
    const hasInShadow = i < shadow.length

    if (hasInLive && hasInShadow) {
      if (hasOpsUnder(childPath, opMap)) {
        walkMerged(live[i], shadow[i], childPath, indent + 1, opMap, lines)
        appendComma(lines, comma)
      } else {
        lines.push({
          content: `${formatValue(live[i])}${comma}`,
          type: 'normal',
          indent: indent + 1,
          path: childPath,
        })
      }
    } else if (hasInLive) {
      lines.push({
        content: `${formatValue(live[i])}${comma}`,
        type: 'removed',
        indent: indent + 1,
        path: childPath,
      })
    } else {
      lines.push({
        content: `${formatValue(shadow[i])}${comma}`,
        type: 'added',
        indent: indent + 1,
        path: childPath,
      })
    }
  }

  lines.push({ content: ']', type: 'normal', indent, path })
}

// --- Helpers ---

function isContainer(v: unknown): boolean {
  return isObject(v) || Array.isArray(v)
}

function isObject(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function hasOpsUnder(path: string, opMap: Map<string, PatchOp>): boolean {
  for (const opPath of opMap.keys()) {
    if (opPath === path || opPath.startsWith(path + '/')) {
      return true
    }
  }
  return false
}

function escapePointer(token: string): string {
  return token.replace(/~/g, '~0').replace(/\//g, '~1')
}

function formatValue(v: unknown): string {
  if (v === undefined) return 'undefined'
  return JSON.stringify(v, null, isContainer(v) ? 2 : undefined)
}

function appendComma(lines: DiffLine[], comma: string): void {
  const last = lines[lines.length - 1]
  if (comma && last) {
    last.content += comma
  }
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
    lines.push({ content: line.trim(), type, indent, path })
  }
}
