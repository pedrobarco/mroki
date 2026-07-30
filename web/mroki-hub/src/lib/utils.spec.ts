import { describe, it, expect } from 'vitest'
import { methodColorClass, pointerToGjson, formatLatency } from './utils'

describe('methodColorClass', () => {
  it('maps the known HTTP verbs to their semantic classes', () => {
    expect(methodColorClass('GET')).toBe('bg-info/15 text-info')
    expect(methodColorClass('POST')).toBe('bg-success/15 text-success')
    expect(methodColorClass('PUT')).toBe('bg-warning/15 text-warning')
    expect(methodColorClass('PATCH')).toBe('bg-warning/15 text-warning')
    expect(methodColorClass('DELETE')).toBe('bg-danger/15 text-danger')
  })

  it('is case-insensitive', () => {
    expect(methodColorClass('get')).toBe('bg-info/15 text-info')
    expect(methodColorClass('delete')).toBe('bg-danger/15 text-danger')
  })

  it('falls back to a muted neutral for unknown verbs', () => {
    expect(methodColorClass('OPTIONS')).toBe('bg-muted text-muted-foreground')
    expect(methodColorClass('')).toBe('bg-muted text-muted-foreground')
  })
})

describe('pointerToGjson', () => {
  it('converts an RFC 6901 pointer to a gjson dot-path', () => {
    expect(pointerToGjson('/body/user/name')).toBe('body.user.name')
    expect(pointerToGjson('/body/items/0/price')).toBe('body.items.0.price')
  })

  it('decodes the RFC 6901 escape sequences', () => {
    expect(pointerToGjson('/headers/X~1Trace')).toBe('headers.X/Trace')
    expect(pointerToGjson('/body/a~0b')).toBe('body.a~b')
  })
})

describe('formatLatency', () => {
  it('renders an em dash for missing values', () => {
    expect(formatLatency(null)).toBe('—')
    expect(formatLatency(undefined)).toBe('—')
  })

  it('collapses sub-millisecond timings to <1ms', () => {
    expect(formatLatency(0.4)).toBe('<1ms')
  })

  it('renders whole-millisecond timings with a unit', () => {
    expect(formatLatency(0)).toBe('0ms')
    expect(formatLatency(12)).toBe('12ms')
  })
})
