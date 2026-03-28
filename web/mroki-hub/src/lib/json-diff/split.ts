import type { DiffLine, SplitRow } from './types'

export function buildSplitRows(lines: DiffLine[]): SplitRow[] {
  const rows: SplitRow[] = []
  let i = 0
  while (i < lines.length) {
    const line = lines[i]!
    if (line.type === 'normal' || line.type === 'collapsed') {
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
