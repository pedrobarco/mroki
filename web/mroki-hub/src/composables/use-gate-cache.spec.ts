import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useGateCache } from './use-gate-cache'
import type { Gate } from '@/api'

vi.mock('@/api', () => ({
  getGate: vi.fn(),
}))

import { getGate } from '@/api'

const mockedGetGate = vi.mocked(getGate)

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
    vi.clearAllMocks()
  })

  it('setGate caches the gate', () => {
    const { setGate, cachedGate } = useGateCache()
    const gate = makeGate()

    setGate(gate)

    expect(cachedGate.value).toEqual(gate)
  })

  it('getGateById returns cached gate without API call when ID matches', async () => {
    const { setGate, getGateById } = useGateCache()
    const gate = makeGate({ id: 'gate-1' })
    setGate(gate)

    const result = await getGateById('gate-1')

    expect(result).toEqual(gate)
    expect(mockedGetGate).not.toHaveBeenCalled()
  })

  it('getGateById fetches from API when cache is empty', async () => {
    const { getGateById } = useGateCache()
    const gate = makeGate({ id: 'gate-2' })
    mockedGetGate.mockResolvedValue({ data: gate })

    const result = await getGateById('gate-2')

    expect(result).toEqual(gate)
    expect(mockedGetGate).toHaveBeenCalledWith('gate-2')
    expect(mockedGetGate).toHaveBeenCalledTimes(1)
  })

  it('getGateById fetches from API when cached ID does not match', async () => {
    const { setGate, getGateById } = useGateCache()
    const cachedGate = makeGate({ id: 'gate-1' })
    setGate(cachedGate)

    const newGate = makeGate({ id: 'gate-3', name: 'other-gate' })
    mockedGetGate.mockResolvedValue({ data: newGate })

    const result = await getGateById('gate-3')

    expect(result).toEqual(newGate)
    expect(mockedGetGate).toHaveBeenCalledWith('gate-3')
  })

  it('getGateById updates cache after fetching', async () => {
    const { getGateById, cachedGate } = useGateCache()
    const gate = makeGate({ id: 'gate-4', name: 'fetched-gate' })
    mockedGetGate.mockResolvedValue({ data: gate })

    await getGateById('gate-4')

    expect(cachedGate.value).toEqual(gate)
  })

  it('clearCache resets the cached gate to null', () => {
    const { setGate, clearCache, cachedGate } = useGateCache()
    setGate(makeGate())

    clearCache()

    expect(cachedGate.value).toBeNull()
  })

  it('cache is shared across multiple useGateCache calls', () => {
    const instance1 = useGateCache()
    const instance2 = useGateCache()
    const gate = makeGate()

    instance1.setGate(gate)

    expect(instance2.cachedGate.value).toEqual(gate)
  })
})
