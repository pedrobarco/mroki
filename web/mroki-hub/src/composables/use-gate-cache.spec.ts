import { describe, it, expect, beforeEach } from 'vitest'
import { useGateCache } from './use-gate-cache'
import type { Gate } from '@/api'

function makeGate(overrides: Partial<Gate> = {}): Gate {
  return {
    id: 'gate-1',
    name: 'test-gate',
    live_url: 'https://live.example.com',
    shadow_url: 'https://shadow.example.com',
    diff_config: {
      ignored_fields: [],
      included_fields: [],
      float_tolerance: 0,
    },
    scrub_config: {
      additional_fields: [],
    },
    created_at: '2026-03-29T09:00:00Z',
    stats: {
      request_count_24h: 0,
      diff_count_24h: 0,
      diff_rate: 0,
      last_active: null,
    },
    ...overrides,
  }
}

describe('useGateCache', () => {
  beforeEach(() => {
    const { clearCache } = useGateCache()
    clearCache()
  })

  it('getCachedGate returns null when cache is empty', () => {
    const { getCachedGate } = useGateCache()
    expect(getCachedGate('gate-1')).toBeNull()
  })

  it('getCachedGate returns the gate when ID matches', () => {
    const { setGate, getCachedGate } = useGateCache()
    const gate = makeGate({ id: 'gate-1' })

    setGate(gate)

    expect(getCachedGate('gate-1')).toEqual(gate)
  })

  it('getCachedGate returns null when cached ID does not match', () => {
    const { setGate, getCachedGate } = useGateCache()
    setGate(makeGate({ id: 'gate-1' }))

    expect(getCachedGate('gate-2')).toBeNull()
  })

  it('setGate replaces previously cached gate', () => {
    const { setGate, getCachedGate } = useGateCache()
    setGate(makeGate({ id: 'gate-1', name: 'first' }))
    setGate(makeGate({ id: 'gate-2', name: 'second' }))

    expect(getCachedGate('gate-1')).toBeNull()
    expect(getCachedGate('gate-2')?.name).toBe('second')
  })

  it('clearCache resets the cache', () => {
    const { setGate, getCachedGate, clearCache } = useGateCache()
    setGate(makeGate({ id: 'gate-1' }))

    clearCache()

    expect(getCachedGate('gate-1')).toBeNull()
  })

  it('cache is shared across multiple useGateCache calls (module-level singleton)', () => {
    const instance1 = useGateCache()
    const instance2 = useGateCache()
    const gate = makeGate()

    instance1.setGate(gate)

    expect(instance2.getCachedGate(gate.id)).toEqual(gate)
  })

  it('getCachedGate returns the cached data (not a copy)', () => {
    const { setGate, getCachedGate } = useGateCache()
    const gate = makeGate({ id: 'gate-1' })

    setGate(gate)

    const cached = getCachedGate('gate-1')
    expect(cached).toStrictEqual(gate)
    // Mutating the cache result should be reflected in subsequent lookups
    // (proves it's the same underlying reactive data, not a deep clone)
    if (cached) {
      cached.name = 'mutated'
      expect(getCachedGate('gate-1')?.name).toBe('mutated')
    }
  })

  it('does not expose cachedGate ref directly', () => {
    const cache = useGateCache()
    // The refactored API should not leak the internal ref
    expect(cache).not.toHaveProperty('cachedGate')
  })

  it('does not expose getGateById (no async fetch)', () => {
    const cache = useGateCache()
    expect(cache).not.toHaveProperty('getGateById')
  })
})
