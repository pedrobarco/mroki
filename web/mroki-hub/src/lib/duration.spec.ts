import { describe, it, expect } from 'vitest'
import { parseGoDuration, humanizeGoDuration, normalizeToGoDuration } from './duration'

const H = 3600 * 1e9
const M = 60 * 1e9
const S = 1e9

describe('parseGoDuration', () => {
  it('parses a single hour unit', () => {
    expect(parseGoDuration('168h')).toBe(168 * H)
  })

  it('parses compound durations', () => {
    expect(parseGoDuration('1h30m')).toBe(H + 30 * M)
    expect(parseGoDuration('1h30m15s')).toBe(H + 30 * M + 15 * S)
  })

  it('parses fractional values', () => {
    expect(parseGoDuration('1.5h')).toBe(1.5 * H)
    expect(parseGoDuration('.5h')).toBe(0.5 * H)
  })

  it('accepts a trailing decimal point like Go (e.g. "1.h")', () => {
    expect(parseGoDuration('1.h')).toBe(H)
  })

  it('rejects values that overflow int64 nanoseconds like Go', () => {
    expect(parseGoDuration('100000000000000h')).toBeNull()
    expect(parseGoDuration('9999999999999999999h')).toBeNull()
  })

  it('parses sub-second units', () => {
    expect(parseGoDuration('300ms')).toBe(300 * 1e6)
    expect(parseGoDuration('500us')).toBe(500 * 1e3)
    expect(parseGoDuration('500ns')).toBe(500)
  })

  it('parses the micro sign unit', () => {
    expect(parseGoDuration('500µs')).toBe(500 * 1e3)
  })

  it('parses bare zero and zero with a unit', () => {
    expect(parseGoDuration('0')).toBe(0)
    expect(parseGoDuration('0s')).toBe(0)
  })

  it('honors a leading sign', () => {
    expect(parseGoDuration('-1h')).toBe(-H)
    expect(parseGoDuration('+1h')).toBe(H)
  })

  it('trims surrounding whitespace', () => {
    expect(parseGoDuration('  168h  ')).toBe(168 * H)
  })

  it('rejects the empty string', () => {
    expect(parseGoDuration('')).toBeNull()
    expect(parseGoDuration('   ')).toBeNull()
  })

  it('rejects numbers without a unit', () => {
    expect(parseGoDuration('168')).toBeNull()
    expect(parseGoDuration('7')).toBeNull()
  })

  it('rejects units Go does not support (d, w, y)', () => {
    expect(parseGoDuration('7d')).toBeNull()
    expect(parseGoDuration('1w')).toBeNull()
    expect(parseGoDuration('1y')).toBeNull()
  })

  it('rejects non-duration text', () => {
    expect(parseGoDuration('abc')).toBeNull()
    expect(parseGoDuration('168hh')).toBeNull()
    expect(parseGoDuration('h168')).toBeNull()
  })
})

describe('humanizeGoDuration', () => {
  it('renders a whole number of days with the raw value', () => {
    expect(humanizeGoDuration('720h0m0s')).toBe('30d (720h0m0s)')
    expect(humanizeGoDuration('168h')).toBe('7d (168h)')
    expect(humanizeGoDuration('24h')).toBe('1d (24h)')
  })

  it('returns the trimmed raw value when not a whole number of days', () => {
    expect(humanizeGoDuration('1h30m')).toBe('1h30m')
    expect(humanizeGoDuration('  36h  ')).toBe('36h')
    expect(humanizeGoDuration('720h1m')).toBe('720h1m')
  })

  it('returns the input unchanged when it cannot be parsed', () => {
    expect(humanizeGoDuration('abc')).toBe('abc')
    expect(humanizeGoDuration('0')).toBe('0')
  })
})

describe('normalizeToGoDuration', () => {
  it('expands day and week shorthands into hours', () => {
    expect(normalizeToGoDuration('30d')).toBe('720h')
    expect(normalizeToGoDuration('2w')).toBe('336h')
    expect(normalizeToGoDuration('1d')).toBe('24h')
  })

  it('expands mixed day/week/Go-unit values', () => {
    expect(normalizeToGoDuration('1w3d')).toBe('168h72h')
    expect(normalizeToGoDuration('1d12h')).toBe('24h12h')
  })

  it('leaves plain Go durations unchanged', () => {
    expect(normalizeToGoDuration('168h')).toBe('168h')
    expect(normalizeToGoDuration('1h30m')).toBe('1h30m')
  })

  it('honors a leading sign', () => {
    expect(normalizeToGoDuration('-1d')).toBe('-24h')
  })

  it('parses to the same nanoseconds as the raw shorthand implies', () => {
    expect(parseGoDuration(normalizeToGoDuration('30d')!)).toBe(720 * H)
    expect(parseGoDuration(normalizeToGoDuration('2w')!)).toBe(336 * H)
  })

  it('trims whitespace and keeps bare zero', () => {
    expect(normalizeToGoDuration('  7d  ')).toBe('168h')
    expect(normalizeToGoDuration('0')).toBe('0')
  })

  it('returns null for unparseable input', () => {
    expect(normalizeToGoDuration('')).toBeNull()
    expect(normalizeToGoDuration('abc')).toBeNull()
    expect(normalizeToGoDuration('1y')).toBeNull()
    expect(normalizeToGoDuration('30dd')).toBeNull()
  })
})
