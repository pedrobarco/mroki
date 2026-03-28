import type { DiffLine } from './types'
import { appendComma } from './tokens'
import { renderNormal } from './render'

/**
 * Expand a collapsed line in-place, replacing it with fully-tokenised normal lines.
 * Returns the number of new lines inserted (so callers can adjust indices).
 */
export function expandCollapsed(lines: DiffLine[], index: number): number {
  const line = lines[index]
  if (!line || line.type !== 'collapsed') return 0
  const expanded: DiffLine[] = []
  renderNormal(line.collapsedValue, line.indent, line.path, expanded)
  if (line.collapsedTrailingComma && expanded.length > 0) {
    appendComma(expanded)
  }
  lines.splice(index, 1, ...expanded)
  return expanded.length - 1 // net lines added
}
