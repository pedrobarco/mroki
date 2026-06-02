import { describe, it, expect } from 'vitest'
import {
  buildDiffLines,
  buildSplitRows,
  stripPathPrefix,
  expandCollapsed,
  tokenizeJson,
  buildPatchRows,
  resolvePointer,
} from './index'
import { buildContext, classifyArray } from './align'
import type { DiffLine, Token } from './types'
import type { PatchOp } from '@/api'

// Helper to extract text from tokens on a line
function lineText(line: DiffLine): string {
  return line.tokens.map((t) => t.text).join('')
}

// Helper to flatten a token list to plain text
function tokenText(tokens: Token[]): string {
  return tokens.map((t) => t.text).join('')
}

describe('stripPathPrefix', () => {
  it('strips prefix from matching ops', () => {
    const ops: PatchOp[] = [
      { op: 'replace', path: '/body/name', value: 'new' },
      { op: 'add', path: '/body/age', value: 30 },
      { op: 'remove', path: '/headers/x-foo' },
    ]
    const result = stripPathPrefix(ops, '/body')
    expect(result).toHaveLength(2)
    expect(result[0].path).toBe('/name')
    expect(result[1].path).toBe('/age')
  })

  it('handles exact prefix match', () => {
    const ops: PatchOp[] = [{ op: 'replace', path: '/body', value: 'new' }]
    const result = stripPathPrefix(ops, '/body')
    expect(result).toHaveLength(1)
    expect(result[0].path).toBe('/')
  })

  it('returns empty when no ops match', () => {
    const ops: PatchOp[] = [{ op: 'replace', path: '/headers/foo', value: 'bar' }]
    const result = stripPathPrefix(ops, '/body')
    expect(result).toHaveLength(0)
  })
})

describe('buildDiffLines', () => {
  it('renders identical primitive objects as normal lines', () => {
    const obj = { name: 'alice', age: 30 }
    const lines = buildDiffLines(obj, obj, [])
    expect(lines.every((l) => l.type === 'normal')).toBe(true)
    // Should have { + 2 key lines + }
    expect(lines).toHaveLength(4)
    expect(lineText(lines[0])).toBe('{')
    expect(lineText(lines[1])).toContain('"name"')
    expect(lineText(lines[1])).toContain('"alice"')
    expect(lineText(lines[2])).toContain('"age"')
    expect(lineText(lines[2])).toContain('30')
    expect(lineText(lines[3])).toBe('}')
  })

  it('marks replaced primitive values', () => {
    const live = { name: 'alice' }
    const shadow = { name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)
    const replaced = lines.filter((l) => l.type === 'replaced-old' || l.type === 'replaced-new')
    expect(replaced).toHaveLength(2)
    expect(lineText(replaced[0])).toContain('"alice"')
    expect(replaced[0].type).toBe('replaced-old')
    expect(lineText(replaced[1])).toContain('"bob"')
    expect(replaced[1].type).toBe('replaced-new')
  })

  it('marks added keys', () => {
    const live = { name: 'alice' }
    const shadow = { name: 'alice', age: 30 }
    const ops: PatchOp[] = [{ op: 'add', path: '/age', value: 30 }]
    const lines = buildDiffLines(live, shadow, ops)
    const added = lines.filter((l) => l.type === 'added')
    expect(added).toHaveLength(1)
    expect(lineText(added[0])).toContain('30')
  })

  it('marks removed keys', () => {
    const live = { name: 'alice', age: 30 }
    const shadow = { name: 'alice' }
    const ops: PatchOp[] = [{ op: 'remove', path: '/age' }]
    const lines = buildDiffLines(live, shadow, ops)
    const removed = lines.filter((l) => l.type === 'removed')
    expect(removed).toHaveLength(1)
    expect(lineText(removed[0])).toContain('30')
  })

  it('collapses unchanged nested objects', () => {
    const live = { meta: { x: 1, y: 2 }, name: 'alice' }
    const shadow = { meta: { x: 1, y: 2 }, name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)
    const collapsed = lines.filter((l) => l.type === 'collapsed')
    expect(collapsed).toHaveLength(1)
    expect(collapsed[0].collapsedValue).toEqual({ x: 1, y: 2 })
    expect(collapsed[0].collapsedLineCount).toBe(4) // { + x + y + }
    expect(lineText(collapsed[0])).toContain('2 fields')
  })

  it('collapses unchanged nested arrays', () => {
    const live = { items: [1, 2, 3], name: 'alice' }
    const shadow = { items: [1, 2, 3], name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)
    const collapsed = lines.filter((l) => l.type === 'collapsed')
    expect(collapsed).toHaveLength(1)
    expect(lineText(collapsed[0])).toContain('3 items')
  })

  it('walks into changed nested objects instead of collapsing', () => {
    const live = { meta: { x: 1, y: 2 } }
    const shadow = { meta: { x: 1, y: 99 } }
    const ops: PatchOp[] = [{ op: 'replace', path: '/meta/y', value: 99 }]
    const lines = buildDiffLines(live, shadow, ops)
    const collapsed = lines.filter((l) => l.type === 'collapsed')
    expect(collapsed).toHaveLength(0)
    // Should have replaced lines for y
    const replaced = lines.filter((l) => l.type === 'replaced-old' || l.type === 'replaced-new')
    expect(replaced).toHaveLength(2)
  })

  it('renders removed container values with proper structure', () => {
    const live = { data: { a: 1 } }
    const shadow = {}
    const ops: PatchOp[] = [{ op: 'remove', path: '/data' }]
    const lines = buildDiffLines(live, shadow, ops)
    const removed = lines.filter((l) => l.type === 'removed')
    // key line + { + "a": 1 + } = 4 removed lines
    expect(removed.length).toBeGreaterThanOrEqual(3)
  })

  it('renders added container values with proper structure', () => {
    const live = {}
    const shadow = { data: { a: 1 } }
    const ops: PatchOp[] = [{ op: 'add', path: '/data', value: { a: 1 } }]
    const lines = buildDiffLines(live, shadow, ops)
    const added = lines.filter((l) => l.type === 'added')
    expect(added.length).toBeGreaterThanOrEqual(3)
  })

  it('handles arrays with added elements', () => {
    const live = [1, 2]
    const shadow = [1, 2, 3]
    const ops: PatchOp[] = [{ op: 'add', path: '/2', value: 3 }]
    const lines = buildDiffLines(live, shadow, ops)
    const added = lines.filter((l) => l.type === 'added')
    expect(added).toHaveLength(1)
    expect(lineText(added[0])).toContain('3')
  })

  it('collapses unchanged object elements inside arrays', () => {
    const live = [
      { id: 1, name: 'a' },
      { id: 2, name: 'b' },
    ]
    const shadow = [
      { id: 1, name: 'a' },
      { id: 2, name: 'b' },
    ]
    const lines = buildDiffLines(live, shadow, [])
    const collapsed = lines.filter((l) => l.type === 'collapsed')
    expect(collapsed).toHaveLength(2)
  })
})

describe('expandCollapsed', () => {
  it('expands a collapsed line into normal lines', () => {
    const live = { meta: { x: 1, y: 2 }, name: 'alice' }
    const shadow = { meta: { x: 1, y: 2 }, name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)

    const collapsedIdx = lines.findIndex((l) => l.type === 'collapsed')
    expect(collapsedIdx).toBeGreaterThanOrEqual(0)

    const linesBefore = lines.length
    const net = expandCollapsed(lines, collapsedIdx)

    expect(net).toBeGreaterThan(0)
    expect(lines.length).toBe(linesBefore + net)
    // No more collapsed lines
    expect(lines.filter((l) => l.type === 'collapsed')).toHaveLength(0)
    // Expanded lines should all be normal
    const expandedRegion = lines.slice(collapsedIdx, collapsedIdx + net + 1)
    expect(expandedRegion.every((l) => l.type === 'normal')).toBe(true)
  })

  it('returns 0 for non-collapsed line', () => {
    const lines = buildDiffLines({ a: 1 }, { a: 1 }, [])
    const result = expandCollapsed(lines, 0)
    expect(result).toBe(0)
  })

  it('returns 0 for out-of-bounds index', () => {
    const lines = buildDiffLines({ a: 1 }, { a: 1 }, [])
    const result = expandCollapsed(lines, 999)
    expect(result).toBe(0)
  })

  it('preserves trailing comma on expanded content', () => {
    const live = { meta: { x: 1 }, other: { y: 2 }, name: 'alice' }
    const shadow = { meta: { x: 1 }, other: { y: 2 }, name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)

    // Find first collapsed (meta) — should have trailing comma
    const collapsedIdx = lines.findIndex((l) => l.type === 'collapsed')
    expect(lines[collapsedIdx].collapsedTrailingComma).toBe(true)

    expandCollapsed(lines, collapsedIdx)

    // After expansion, the last line of the expanded block should have a comma
    // The expanded block for {x:1} is: { , "x": 1, }  — the closing } should have comma
    const closingBrace = lines[collapsedIdx + 2] // { at idx, "x":1 at idx+1, } at idx+2
    expect(lineText(closingBrace)).toContain(',')
  })
})

describe('buildSplitRows', () => {
  it('pairs normal lines on both sides', () => {
    const lines = buildDiffLines({ a: 1 }, { a: 1 }, [])
    const rows = buildSplitRows(lines)
    rows.forEach((row) => {
      expect(row.left).toBe(row.right)
    })
  })

  it('pairs collapsed lines on both sides', () => {
    const live = { meta: { x: 1 }, name: 'alice' }
    const shadow = { meta: { x: 1 }, name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)
    const rows = buildSplitRows(lines)
    const collapsedRows = rows.filter((r) => r.left?.type === 'collapsed')
    expect(collapsedRows).toHaveLength(1)
    expect(collapsedRows[0].left).toBe(collapsedRows[0].right)
  })

  it('pairs replaced-old with replaced-new', () => {
    const live = { name: 'alice' }
    const shadow = { name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)
    const rows = buildSplitRows(lines)
    const replacedRows = rows.filter(
      (r) => r.left?.type === 'replaced-old' && r.right?.type === 'replaced-new'
    )
    expect(replacedRows).toHaveLength(1)
  })

  it('puts removed lines on left only', () => {
    const live = { name: 'alice', age: 30 }
    const shadow = { name: 'alice' }
    const ops: PatchOp[] = [{ op: 'remove', path: '/age' }]
    const lines = buildDiffLines(live, shadow, ops)
    const rows = buildSplitRows(lines)
    const removedRows = rows.filter((r) => r.left?.type === 'removed')
    expect(removedRows.length).toBeGreaterThan(0)
    removedRows.forEach((r) => expect(r.right).toBeNull())
  })

  it('puts added lines on right only', () => {
    const live = { name: 'alice' }
    const shadow = { name: 'alice', age: 30 }
    const ops: PatchOp[] = [{ op: 'add', path: '/age', value: 30 }]
    const lines = buildDiffLines(live, shadow, ops)
    const rows = buildSplitRows(lines)
    const addedRows = rows.filter((r) => r.right?.type === 'added')
    expect(addedRows.length).toBeGreaterThan(0)
    addedRows.forEach((r) => expect(r.left).toBeNull())
  })
})

describe('edge cases', () => {
  it('handles null values correctly', () => {
    const live = { val: null }
    const shadow = { val: 'hello' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/val', value: 'hello' }]
    const lines = buildDiffLines(live, shadow, ops)
    const old = lines.find((l) => l.type === 'replaced-old')
    expect(lineText(old!)).toContain('null')
  })

  it('handles boolean values correctly', () => {
    const live = { active: true }
    const shadow = { active: false }
    const ops: PatchOp[] = [{ op: 'replace', path: '/active', value: false }]
    const lines = buildDiffLines(live, shadow, ops)
    const old = lines.find((l) => l.type === 'replaced-old')
    expect(lineText(old!)).toContain('true')
    const newLine = lines.find((l) => l.type === 'replaced-new')
    expect(lineText(newLine!)).toContain('false')
  })

  it('handles deeply nested changes', () => {
    const live = { a: { b: { c: { d: 1 } } } }
    const shadow = { a: { b: { c: { d: 2 } } } }
    const ops: PatchOp[] = [{ op: 'replace', path: '/a/b/c/d', value: 2 }]
    const lines = buildDiffLines(live, shadow, ops)
    // Should NOT have any collapsed lines since the path leads through all levels
    expect(lines.filter((l) => l.type === 'collapsed')).toHaveLength(0)
    // Should have replaced lines
    const replaced = lines.filter((l) => l.type === 'replaced-old' || l.type === 'replaced-new')
    expect(replaced).toHaveLength(2)
  })

  it('handles empty objects', () => {
    const lines = buildDiffLines({}, {}, [])
    expect(lines).toHaveLength(2) // { and }
    expect(lineText(lines[0])).toBe('{')
    expect(lineText(lines[1])).toBe('}')
  })

  it('handles empty arrays', () => {
    const lines = buildDiffLines([], [], [])
    expect(lines).toHaveLength(2) // [ and ]
    expect(lineText(lines[0])).toBe('[')
    expect(lineText(lines[1])).toBe(']')
  })

  it('escapes JSON pointer characters in keys', () => {
    const live = { 'a/b': 1, 'c~d': 2 }
    const shadow = { 'a/b': 1, 'c~d': 2 }
    const lines = buildDiffLines(live, shadow, [])
    // Keys with / and ~ should still render correctly
    const keyLines = lines.filter((l) => lineText(l).includes('"a/b"'))
    expect(keyLines).toHaveLength(1)
  })
})

describe('tokenizeJson', () => {
  it('tokenizes primitives with the matching token type', () => {
    expect(tokenizeJson('hi')).toEqual([{ text: '"hi"', type: 'string' }])
    expect(tokenizeJson(42)).toEqual([{ text: '42', type: 'number' }])
    expect(tokenizeJson(true)).toEqual([{ text: 'true', type: 'boolean' }])
    expect(tokenizeJson(null)).toEqual([{ text: 'null', type: 'null' }])
  })

  it('tokenizes objects on a single compact line by default', () => {
    const tokens = tokenizeJson({ a: 1, b: 'x' })
    expect(tokenText(tokens)).toBe('{"a": 1, "b": "x"}')
    // Keys keep the key token type for highlighting
    expect(tokens.find((t) => t.text === '"a"')?.type).toBe('key')
    expect(tokens.some((t) => t.text.includes('\n'))).toBe(false)
  })

  it('emits newline tokens when pretty-printing', () => {
    const text = tokenText(tokenizeJson({ a: 1 }, true))
    expect(text).toBe('{\n  "a": 1\n}')
  })

  it('renders empty containers compactly', () => {
    expect(tokenText(tokenizeJson([]))).toBe('[]')
    expect(tokenText(tokenizeJson({}))).toBe('{}')
  })
})

describe('JSON string/key escaping', () => {
  it('escapes special characters in string values', () => {
    const value = 'he said "hi"\n\tC:\\tmp'
    const obj = { msg: value }
    const lines = buildDiffLines(obj, obj, [])
    const line = lines.find((l) => lineText(l).includes('"msg"'))
    expect(line).toBeDefined()
    // Rendered value must match JSON.stringify, not the raw unescaped string
    expect(lineText(line!)).toContain(JSON.stringify(value))
    expect(lineText(line!)).not.toContain('he said "hi"')
  })

  it('escapes special characters in object keys', () => {
    const key = 'a"b\\c'
    const obj = { [key]: 1 }
    const lines = buildDiffLines(obj, obj, [])
    const line = lines.find((l) => lineText(l).includes(JSON.stringify(key)))
    expect(line).toBeDefined()
    expect(lineText(line!)).not.toContain('"a"b\\c"')
  })

  it('escapes replaced values in split/unified lines', () => {
    const live = { msg: 'plain' }
    const shadow = { msg: 'he said "hi"' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/msg', value: 'he said "hi"' }]
    const lines = buildDiffLines(live, shadow, ops)
    const newLine = lines.find((l) => l.type === 'replaced-new')
    expect(newLine).toBeDefined()
    expect(lineText(newLine!)).toContain(JSON.stringify('he said "hi"'))
    expect(lineText(newLine!)).not.toContain('he said "hi"')
  })

  it('renders the same escaped string as tokenizeJson (patch/split parity)', () => {
    const value = 'tab\there'
    const lines = buildDiffLines({ msg: value }, { msg: value }, [])
    const line = lines.find((l) => lineText(l).includes('"msg"'))
    expect(lineText(line!)).toContain(tokenText(tokenizeJson(value)))
  })

  it('escapes special characters when expanding a collapsed node', () => {
    const value = 'he said "hi"'
    const live = { meta: { msg: value }, name: 'alice' }
    const shadow = { meta: { msg: value }, name: 'bob' }
    const ops: PatchOp[] = [{ op: 'replace', path: '/name', value: 'bob' }]
    const lines = buildDiffLines(live, shadow, ops)
    const collapsedIdx = lines.findIndex((l) => l.type === 'collapsed')
    expect(collapsedIdx).toBeGreaterThanOrEqual(0)
    expandCollapsed(lines, collapsedIdx)
    const line = lines.find((l) => lineText(l).includes('"msg"'))
    expect(line).toBeDefined()
    expect(lineText(line!)).toContain(JSON.stringify(value))
    expect(lineText(line!)).not.toContain('he said "hi"')
  })

  it('escapes special characters in array element strings', () => {
    const value = 'he said "hi"'
    const lines = buildDiffLines([value], [value], [])
    const line = lines.find((l) => lineText(l).includes('hi'))
    expect(line).toBeDefined()
    expect(lineText(line!)).toContain(JSON.stringify(value))
    expect(lineText(line!)).not.toContain('he said "hi"')
  })

  it('escapes special characters in added and removed scalar lines', () => {
    const value = 'he said "hi"'
    const added = buildDiffLines({ msg: 'plain' }, { msg: 'plain', extra: value }, [
      { op: 'add', path: '/extra', value },
    ])
    const addedLine = added.find((l) => l.type === 'added')
    expect(addedLine).toBeDefined()
    expect(lineText(addedLine!)).toContain(JSON.stringify(value))

    const removed = buildDiffLines({ msg: 'plain', extra: value }, { msg: 'plain' }, [
      { op: 'remove', path: '/extra' },
    ])
    const removedLine = removed.find((l) => l.type === 'removed')
    expect(removedLine).toBeDefined()
    expect(lineText(removedLine!)).toContain(JSON.stringify(value))
  })

  it('escapes special characters in keys/values of removed container blocks', () => {
    const key = 'a"b'
    const value = 'he said "hi"'
    const lines = buildDiffLines({ data: { [key]: value } }, {}, [{ op: 'remove', path: '/data' }])
    const text = lines.map(lineText).join('\n')
    expect(text).toContain(JSON.stringify(key))
    expect(text).toContain(JSON.stringify(value))
    expect(text).not.toContain('he said "hi"')
  })
})

describe('resolvePointer', () => {
  const doc = { body: { items: [{ price: 10 }, { price: 20 }], 'a/b': 1 }, headers: { x: ['v'] } }

  it('resolves nested object and array pointers', () => {
    expect(resolvePointer(doc, '/body/items/1/price')).toEqual({ found: true, value: 20 })
    expect(resolvePointer(doc, '/headers/x/0')).toEqual({ found: true, value: 'v' })
  })

  it('unescapes ~1 and ~0 pointer tokens', () => {
    expect(resolvePointer(doc, '/body/a~1b')).toEqual({ found: true, value: 1 })
  })

  it('returns the whole document for the root pointer', () => {
    expect(resolvePointer(doc, '')).toEqual({ found: true, value: doc })
  })

  it('reports not found for missing keys and out-of-range indices', () => {
    expect(resolvePointer(doc, '/body/missing')).toEqual({ found: false, value: undefined })
    expect(resolvePointer(doc, '/body/items/5')).toEqual({ found: false, value: undefined })
  })
})

describe('buildPatchRows', () => {
  const live = {
    body: { status: 'processing', items: [{ id: 1 }, { id: 2 }], legacy: 'old-token' },
    headers: { 'content-type': ['application/json'] },
  }

  it('derives old value from live and new value from op.value for replace', () => {
    const rows = buildPatchRows(
      [{ op: 'replace', path: '/body/status', value: 'completed' }],
      live,
      live
    )
    expect(rows).toHaveLength(1)
    const row = rows[0]
    expect(row.op).toBe('replace')
    expect(row.hasOld).toBe(true)
    expect(row.oldValue).toBe('processing')
    expect(row.hasNew).toBe(true)
    expect(row.newValue).toBe('completed')
    expect(row.leafLabel).toBe('status')
    expect(row.pathPrefix).toEqual(['body'])
    expect(row.valueTitle).toBe('processing  →  completed')
  })

  it('marks add ops with only a new value', () => {
    const rows = buildPatchRows([{ op: 'add', path: '/body/coupon', value: 'X1' }], live, live)
    expect(rows[0].hasOld).toBe(false)
    expect(rows[0].hasNew).toBe(true)
    expect(rows[0].newValue).toBe('X1')
  })

  it('marks remove ops with only the resolved old value', () => {
    const rows = buildPatchRows([{ op: 'remove', path: '/body/legacy' }], live, live)
    expect(rows[0].hasOld).toBe(true)
    expect(rows[0].oldValue).toBe('old-token')
    expect(rows[0].hasNew).toBe(false)
  })

  it('formats array index leaves and flags complex values as expandable', () => {
    const rows = buildPatchRows([{ op: 'remove', path: '/body/items/1' }], live, live)
    expect(rows[0].leafLabel).toBe('[1]')
    expect(rows[0].leafIsIndex).toBe(true)
    expect(rows[0].expandable).toBe(true)
  })

  // Issue #1: an insert shifts a later element that is also replaced. The
  // replace op is y-indexed (/items/2), which is out of range in the live array;
  // the old value must be recovered by mapping the shadow index back to live
  // index 1 through the array alignment.
  it('resolves the old value of a shifted replace via shadow alignment', () => {
    const liveDoc = { items: ['keep', 'target'] }
    const shadowDoc = { items: ['inserted', 'keep', 'TARGET'] }
    const ops: PatchOp[] = [
      { op: 'add', path: '/items/0', value: 'inserted' },
      { op: 'replace', path: '/items/2', value: 'TARGET' },
    ]
    const rows = buildPatchRows(ops, liveDoc, shadowDoc)
    const replaceRow = rows.find((r) => r.op === 'replace')!
    expect(replaceRow.hasOld).toBe(true)
    expect(replaceRow.oldValue).toBe('target')
    expect(replaceRow.newValue).toBe('TARGET')
  })

  // Resolving a shadow (y) indexed replace pointer positionally against the live
  // array falls out of range — the lossy behaviour #1 fixes by aligning through
  // the shadow document. buildPatchRows now *requires* the shadow document, so
  // this positional miss cannot happen via the public API; resolvePointer is the
  // remaining positional primitive and demonstrates the underlying hazard.
  it('positional pointer resolution misses a shifted replace (pre-#1 behaviour)', () => {
    const liveDoc = { items: ['keep', 'target'] }
    expect(resolvePointer(liveDoc, '/items/2')).toEqual({ found: false, value: undefined })
  })
})

describe('array reorder/insert/delete (issue #114)', () => {
  it('highlights a reordered element as added and removed in split/unified', () => {
    // ['a','b','c'] -> ['c','a','b']: 'c' is removed at live x-index 2 and
    // re-added at shadow y-index 0; 'a'/'b' stay equal.
    const liveArr = ['a', 'b', 'c']
    const shadowArr = ['c', 'a', 'b']
    const ops: PatchOp[] = [
      { op: 'add', path: '/0', value: 'c' },
      { op: 'remove', path: '/2' },
    ]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    const added = lines.filter((l) => l.type === 'added')
    const removed = lines.filter((l) => l.type === 'removed')
    expect(added).toHaveLength(1)
    expect(removed).toHaveLength(1)
    expect(lineText(added[0])).toContain('"c"')
    expect(lineText(removed[0])).toContain('"c"')
    // 'a' and 'b' are unchanged.
    expect(lines.filter((l) => l.type === 'normal' && lineText(l).includes('"a"'))).toHaveLength(1)
    expect(lines.filter((l) => l.type === 'normal' && lineText(l).includes('"b"'))).toHaveLength(1)

    // Split: added 'c' is right-only, removed 'c' is left-only.
    const rows = buildSplitRows(lines)
    const addedRow = rows.find((r) => r.right?.type === 'added')!
    const removedRow = rows.find((r) => r.left?.type === 'removed')!
    expect(addedRow.left).toBeNull()
    expect(removedRow.right).toBeNull()
  })

  it('renders an inserted array element as a single added line', () => {
    const liveArr = ['a', 'b']
    const shadowArr = ['a', 'x', 'b']
    const ops: PatchOp[] = [{ op: 'add', path: '/1', value: 'x' }]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    const added = lines.filter((l) => l.type === 'added')
    expect(added).toHaveLength(1)
    expect(lineText(added[0])).toContain('"x"')
    expect(lines.filter((l) => l.type === 'removed')).toHaveLength(0)
  })

  it('renders a deleted array element as a single removed line', () => {
    const liveArr = ['a', 'x', 'b']
    const shadowArr = ['a', 'b']
    const ops: PatchOp[] = [{ op: 'remove', path: '/1' }]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    const removed = lines.filter((l) => l.type === 'removed')
    expect(removed).toHaveLength(1)
    expect(lineText(removed[0])).toContain('"x"')
    expect(lines.filter((l) => l.type === 'added')).toHaveLength(0)
  })

  // #3: the Patch view (op count) and the Split/Unified view (highlighted lines)
  // must agree on the set of changes for a reorder — one add and one remove.
  it('keeps Split/Unified change counts in parity with the Patch view (#3)', () => {
    const liveDoc = { items: ['a', 'b', 'c'] }
    const shadowDoc = { items: ['c', 'a', 'b'] }
    const ops: PatchOp[] = [
      { op: 'add', path: '/items/0', value: 'c' },
      { op: 'remove', path: '/items/2' },
    ]
    const patchRows = buildPatchRows(ops, liveDoc, shadowDoc)
    const lines = buildDiffLines(liveDoc, shadowDoc, ops)
    const addedLines = lines.filter((l) => l.type === 'added')
    const removedLines = lines.filter((l) => l.type === 'removed')
    expect(patchRows.filter((r) => r.op === 'add')).toHaveLength(addedLines.length)
    expect(patchRows.filter((r) => r.op === 'remove')).toHaveLength(removedLines.length)
  })

  // #5: a single change inside a large array stays linear and renders exactly
  // one highlighted line, leaving the remaining elements as normal lines.
  it('handles a large array with a single change in linear fashion (#5)', () => {
    const n = 1000
    const liveArr = Array.from({ length: n }, (_, i) => i)
    const shadowArr = liveArr.slice()
    shadowArr[500] = -1
    const ops: PatchOp[] = [{ op: 'replace', path: '/500', value: -1 }]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    expect(lines.filter((l) => l.type === 'replaced-old')).toHaveLength(1)
    expect(lines.filter((l) => l.type === 'replaced-new')).toHaveLength(1)
    // The other 999 elements are unchanged normal lines (plus the [ and ] lines).
    expect(lines.filter((l) => l.type === 'normal')).toHaveLength(n - 1 + 2)
  })
})

describe('array alignment coverage (#114 follow-ups)', () => {
  // C6: a deeper field replace/remove under a reordered array element. The
  // intermediate index is shadow (y) indexed and must be translated back to the
  // live (x) element to read the correct old value (#1).
  it('resolves a nested-field replace old value under a reordered element', () => {
    const liveDoc = {
      items: [
        { id: 1, v: 'a' },
        { id: 2, v: 'b' },
      ],
    }
    const shadowDoc = {
      items: [
        { id: 2, v: 'B' },
        { id: 1, v: 'a' },
      ],
    }
    // go-cmp output for this reorder + deeper change.
    const ops: PatchOp[] = [
      { op: 'remove', path: '/items/0' },
      { op: 'replace', path: '/items/0/v', value: 'B' },
      { op: 'add', path: '/items/1', value: { id: 1, v: 'a' } },
    ]
    const rows = buildPatchRows(ops, liveDoc, shadowDoc)
    const replaceRow = rows.find((r) => r.op === 'replace')!
    expect(replaceRow.oldValue).toBe('b')
    expect(replaceRow.newValue).toBe('B')
  })

  it('resolves a nested-field remove old value under a reordered element', () => {
    const liveDoc = {
      items: [
        { id: 1, v: 'a' },
        { id: 2, v: 'b', extra: 'gone' },
      ],
    }
    const shadowDoc = {
      items: [
        { id: 2, v: 'b' },
        { id: 1, v: 'a' },
      ],
    }
    const ops: PatchOp[] = [
      { op: 'remove', path: '/items/0' },
      { op: 'remove', path: '/items/0/extra' },
      { op: 'add', path: '/items/1', value: { id: 1, v: 'a' } },
    ]
    const rows = buildPatchRows(ops, liveDoc, shadowDoc)
    const nested = rows.find((r) => r.path === '/items/0/extra')!
    expect(nested.hasOld).toBe(true)
    expect(nested.oldValue).toBe('gone')
    // The element-level remove reads its live x-index directly.
    const elem = rows.find((r) => r.path === '/items/0')!
    expect(elem.oldValue).toEqual({ id: 1, v: 'a' })
  })

  // C7: reordering an array of objects renders the moved object as an added and
  // a removed container block, with the untouched objects collapsed.
  it('renders an array-of-objects reorder as added/removed blocks + collapsed', () => {
    const liveArr = [
      { id: 1, name: 'a' },
      { id: 2, name: 'b' },
      { id: 3, name: 'c' },
    ]
    const shadowArr = [
      { id: 3, name: 'c' },
      { id: 1, name: 'a' },
      { id: 2, name: 'b' },
    ]
    const ops: PatchOp[] = [
      { op: 'add', path: '/0', value: { id: 3, name: 'c' } },
      { op: 'remove', path: '/2' },
    ]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    expect(lines.filter((l) => l.type === 'collapsed')).toHaveLength(2)
    const added = lines.filter((l) => l.type === 'added')
    const removed = lines.filter((l) => l.type === 'removed')
    expect(added.length).toBeGreaterThan(0)
    expect(removed.length).toBeGreaterThan(0)
    expect(added.map(lineText).join('\n')).toContain('"c"')
    expect(removed.map(lineText).join('\n')).toContain('"c"')
  })

  // C8: a reorder of an array with duplicate values still pairs one add with one
  // remove and leaves the duplicates as unchanged normal lines.
  it('handles a reorder with duplicate/equal values', () => {
    const liveArr = ['a', 'a', 'b']
    const shadowArr = ['b', 'a', 'a']
    const ops: PatchOp[] = [
      { op: 'add', path: '/0', value: 'b' },
      { op: 'remove', path: '/2' },
    ]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    expect(lines.filter((l) => l.type === 'added' && lineText(l).includes('"b"'))).toHaveLength(1)
    expect(lines.filter((l) => l.type === 'removed' && lineText(l).includes('"b"'))).toHaveLength(1)
    expect(lines.filter((l) => l.type === 'normal' && lineText(l).includes('"a"'))).toHaveLength(2)
  })

  // C9: degenerate arrays — all removed, all added, and empty.
  it('renders an all-removed array as only removed lines', () => {
    const ops: PatchOp[] = [
      { op: 'remove', path: '/0' },
      { op: 'remove', path: '/1' },
      { op: 'remove', path: '/2' },
    ]
    const lines = buildDiffLines(['a', 'b', 'c'], [], ops)
    expect(lines.filter((l) => l.type === 'removed')).toHaveLength(3)
    expect(lines.filter((l) => l.type === 'added')).toHaveLength(0)
  })

  it('renders an all-added array as only added lines', () => {
    const ops: PatchOp[] = [
      { op: 'add', path: '/0', value: 'a' },
      { op: 'add', path: '/1', value: 'b' },
      { op: 'add', path: '/2', value: 'c' },
    ]
    const lines = buildDiffLines([], ['a', 'b', 'c'], ops)
    expect(lines.filter((l) => l.type === 'added')).toHaveLength(3)
    expect(lines.filter((l) => l.type === 'removed')).toHaveLength(0)
  })

  it('renders two empty arrays as bare brackets', () => {
    const lines = buildDiffLines([], [], [])
    expect(lines).toHaveLength(2)
    expect(lineText(lines[0])).toBe('[')
    expect(lineText(lines[1])).toBe(']')
  })

  // C10: alignment recurses through arrays nested inside arrays.
  it('aligns a reorder of nested arrays-in-arrays', () => {
    const liveArr = [
      [1, 2],
      [3, 4],
    ]
    const shadowArr = [
      [3, 4],
      [1, 2],
    ]
    const ops: PatchOp[] = [
      { op: 'remove', path: '/0' },
      { op: 'add', path: '/1', value: [1, 2] },
    ]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    // [3,4] is unchanged → collapsed; [1,2] is removed then re-added.
    expect(lines.filter((l) => l.type === 'collapsed')).toHaveLength(1)
    expect(lines.filter((l) => l.type === 'removed').length).toBeGreaterThan(0)
    expect(lines.filter((l) => l.type === 'added').length).toBeGreaterThan(0)
  })

  // C11: a primitive replace inside an array pairs replaced-old/new in split view.
  it('pairs a replaced array element in the split view', () => {
    const ops: PatchOp[] = [{ op: 'replace', path: '/1', value: 'B' }]
    const lines = buildDiffLines(['a', 'b'], ['a', 'B'], ops)
    const rows = buildSplitRows(lines)
    const replaced = rows.find(
      (r) => r.left?.type === 'replaced-old' && r.right?.type === 'replaced-new'
    )!
    expect(replaced).toBeDefined()
    expect(lineText(replaced.left!)).toContain('"b"')
    expect(lineText(replaced.right!)).toContain('"B"')
  })

  // C12: an equal container that shifted position is collapsed using the LIVE
  // value but addressed by its SHADOW index.
  it('collapses an equal container at its shifted shadow index', () => {
    const liveArr = [{ keep: 1 }]
    const shadowArr = ['new', { keep: 1 }]
    const ops: PatchOp[] = [{ op: 'add', path: '/0', value: 'new' }]
    const lines = buildDiffLines(liveArr, shadowArr, ops)
    const collapsed = lines.find((l) => l.type === 'collapsed')!
    expect(collapsed.collapsedValue).toEqual({ keep: 1 })
    expect(collapsed.path).toBe('/1')
    expect(lines.filter((l) => l.type === 'added' && lineText(l).includes('"new"'))).toHaveLength(1)
  })

  // C13: classifyArray must not treat a bare (x-indexed) remove as a change of
  // the shadow element sharing that numeric index.
  it('does not mark a shadow element changed by a bare remove at the same index', () => {
    const bareRemove = classifyArray(buildContext([{ op: 'remove', path: '/1' }]), '')
    expect(bareRemove.removeX.has(1)).toBe(true)
    expect(bareRemove.isChangedY(1)).toBe(false)

    const replaced = classifyArray(buildContext([{ op: 'replace', path: '/1', value: 'x' }]), '')
    expect(replaced.isChangedY(1)).toBe(true)

    const deeper = classifyArray(buildContext([{ op: 'replace', path: '/1/x', value: 9 }]), '')
    expect(deeper.isChangedY(1)).toBe(true)
  })

  // C14: a y-indexed array op whose live ancestor is not an array resolves to
  // not-found rather than throwing or returning a wrong value.
  it('reports not-found when the live ancestor is not an array', () => {
    const rows = buildPatchRows(
      [{ op: 'replace', path: '/items/0', value: 'X' }],
      { items: { a: 1 } },
      { items: ['x'] }
    )
    expect(rows[0].hasOld).toBe(false)
  })

  // C15: the last element of an array carries no trailing comma; preceding
  // elements do — for add, remove, and replace alike.
  it('omits the trailing comma on the last array element', () => {
    const addLines = buildDiffLines(['a'], ['a', 'b'], [{ op: 'add', path: '/1', value: 'b' }])
    const firstA = addLines.find((l) => l.type === 'normal' && lineText(l).includes('"a"'))!
    const addedB = addLines.find((l) => l.type === 'added')!
    expect(lineText(firstA)).toContain(',')
    expect(lineText(addedB)).not.toContain(',')

    const removeLines = buildDiffLines(['a', 'b'], ['a'], [{ op: 'remove', path: '/1' }])
    const removedB = removeLines.find((l) => l.type === 'removed')!
    expect(lineText(removedB)).not.toContain(',')

    const replaceLines = buildDiffLines(
      ['a', 'b'],
      ['a', 'B'],
      [{ op: 'replace', path: '/1', value: 'B' }]
    )
    const newB = replaceLines.find((l) => l.type === 'replaced-new')!
    expect(lineText(newB)).not.toContain(',')
  })
})
