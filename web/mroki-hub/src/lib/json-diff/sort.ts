/**
 * Deterministic sort key for JSON values.
 * Mirrors the Go toSortKey function in pkg/diff/options.go.
 */
function toSortKey(v: unknown): string {
  if (v === null || v === undefined) return 'z:null'
  if (typeof v === 'string') return 's:' + v
  if (typeof v === 'number') return 'n:' + v
  if (typeof v === 'boolean') return 'b:' + v
  // For complex types (objects, arrays), produce canonical JSON.
  // JSON.stringify sorts nothing by default, but we need deterministic output,
  // so we sort object keys ourselves.
  return 'j:' + canonicalJson(v)
}

function canonicalJson(v: unknown): string {
  if (v === null || v === undefined) return 'null'
  if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') {
    return JSON.stringify(v)
  }
  if (Array.isArray(v)) {
    return '[' + v.map(canonicalJson).join(',') + ']'
  }
  if (typeof v === 'object' && v !== null) {
    const obj = v as Record<string, unknown>
    const keys = Object.keys(obj).sort()
    return '{' + keys.map((k) => JSON.stringify(k) + ':' + canonicalJson(obj[k])).join(',') + '}'
  }
  return String(v)
}

/**
 * Recursively sorts all arrays in a JSON tree using a deterministic key.
 * Maps and other values are traversed but not reordered.
 * Returns a deep-cloned, sorted copy — the original is not mutated.
 *
 * Mirrors the Go SortArraysInTree function in pkg/diff/options.go.
 */
export function sortArraysInTree(v: unknown): unknown {
  if (Array.isArray(v)) {
    // Recursively sort nested structures first, then sort this array
    const sorted = v.map(sortArraysInTree)
    sorted.sort((a, b) => {
      const ka = toSortKey(a)
      const kb = toSortKey(b)
      if (ka < kb) return -1
      if (ka > kb) return 1
      return 0
    })
    return sorted
  }
  if (v !== null && typeof v === 'object') {
    const obj = v as Record<string, unknown>
    const result: Record<string, unknown> = {}
    for (const key of Object.keys(obj)) {
      result[key] = sortArraysInTree(obj[key])
    }
    return result
  }
  return v
}
