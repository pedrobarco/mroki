import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import Gates from './Gates.vue'
import type { GlobalStats } from '@/api'

const getGlobalStats = vi.fn()
vi.mock('@/api', () => ({
  getGlobalStats: (...args: unknown[]) => getGlobalStats(...args),
}))

function makeStats(overrides: Partial<GlobalStats> = {}): GlobalStats {
  return { total_gates: 12, total_requests_24h: 3456, total_diff_rate: 4.2, ...overrides }
}

// Stub the child feature components and the Dialog primitives so mounting the
// page exercises only its own stats/polling logic.
const global = {
  stubs: {
    GateList: true,
    GateForm: true,
    GateFilters: true,
    Dialog: true,
    DialogContent: true,
    DialogDescription: true,
    DialogHeader: true,
    DialogTitle: true,
    DialogTrigger: true,
    Button: true,
    Plus: true,
    RefreshCw: true,
  },
}

function mountGates() {
  return mount(Gates, { global })
}

describe('Gates stats bar', () => {
  beforeEach(() => {
    getGlobalStats.mockReset()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads and renders the global stats on mount', async () => {
    getGlobalStats.mockResolvedValue({ data: makeStats() })
    const wrapper = mountGates()
    await flushPromises()

    expect(getGlobalStats).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('3,456')
    expect(wrapper.text()).toContain('4.2%')
    expect(wrapper.text()).toContain('updated just now')
  })

  it('surfaces a retry note when stats fail to load', async () => {
    getGlobalStats.mockRejectedValue(new Error('boom'))
    const wrapper = mountGates()
    await flushPromises()

    expect(wrapper.text()).toContain('Stats unavailable — retry')
    // Placeholder dashes keep the layout stable when stats are missing.
    expect(wrapper.text()).toContain('—')
  })

  it('refetches stats when the manual refresh button is clicked', async () => {
    getGlobalStats.mockResolvedValue({ data: makeStats() })
    const wrapper = mountGates()
    await flushPromises()
    expect(getGlobalStats).toHaveBeenCalledTimes(1)

    await wrapper.find('button[type="button"]').trigger('click')
    await flushPromises()
    expect(getGlobalStats).toHaveBeenCalledTimes(2)
  })
})

describe('Gates polling and clock', () => {
  beforeEach(() => {
    getGlobalStats.mockReset()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('advances the relative age label on the 1s clock without refetching', async () => {
    vi.useFakeTimers()
    getGlobalStats.mockResolvedValue({ data: makeStats() })
    const wrapper = mountGates()
    await flushPromises()
    expect(wrapper.text()).toContain('updated just now')

    vi.advanceTimersByTime(7000)
    await flushPromises()
    expect(wrapper.text()).toContain('updated 7s ago')
    // The ticking clock must not trigger extra fetches.
    expect(getGlobalStats).toHaveBeenCalledTimes(1)
  })

  it('polls stats every 30s', async () => {
    vi.useFakeTimers()
    getGlobalStats.mockResolvedValue({ data: makeStats() })
    mountGates()
    await flushPromises()
    expect(getGlobalStats).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(30000)
    await flushPromises()
    expect(getGlobalStats).toHaveBeenCalledTimes(2)
  })

  it('clears both timers on unmount so no polling continues', async () => {
    vi.useFakeTimers()
    getGlobalStats.mockResolvedValue({ data: makeStats() })
    const wrapper = mountGates()
    await flushPromises()

    wrapper.unmount()
    vi.advanceTimersByTime(60000)
    await flushPromises()
    expect(getGlobalStats).toHaveBeenCalledTimes(1)
  })
})
