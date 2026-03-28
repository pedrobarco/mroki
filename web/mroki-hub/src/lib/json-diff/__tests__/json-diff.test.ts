import { describe, it, expect } from 'vitest'
import { buildDiffLines, buildSplitRows, stripPathPrefix, expandCollapsed } from '../index'
import type { DiffLine } from '../types'
import type { PatchOp } from '@/api'

// Helper to extract text from tokens on a line
function lineText(line: DiffLine): string {
  return line.tokens.map((t) => t.text).join('')
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
