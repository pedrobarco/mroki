import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import GateList from './GateList.vue'
import type { GateFilterState } from './GateFilters.vue'
import type { Gate, PaginatedResponse } from '@/api'

const getGates = vi.fn()
// Mock the underlying API module so the real query adapter (gatesQuery, from
// '@/api') keeps working while the network call is stubbed.
vi.mock('@/api/gates', () => ({
  getGates: (...args: unknown[]) => getGates(...args),
}))

// A fresh, retry-free client per mount isolates the gate-list cache per test so
// paging assertions always observe a real fetch.
function mountOpts(props: Record<string, unknown>) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return { props, global: { plugins: [[VueQueryPlugin, { queryClient }]] } }
}

// GateCard renders router links; stub it so the list can mount standalone. The
// stub echoes the gate id so tests can assert which page's data is on screen.
vi.mock('./GateCard.vue', () => ({
  default: {
    name: 'GateCard',
    props: ['gate'],
    template: '<div class="gate-card-stub">{{ gate.id }}</div>',
  },
}))

function makeGate(overrides: Partial<Gate> = {}): Gate {
  return {
    id: 'gate-1',
    name: 'checkout-api',
    live_url: 'https://live.example.com',
    shadow_url: 'https://shadow.example.com',
    diff_config: {
      ignored_fields: [],
      included_fields: [],
      float_tolerance: 0,
      sort_arrays: false,
    },
    redacted_fields: [],
    retention: '',
    created_at: '2026-03-29T09:00:00Z',
    stats: { request_count_24h: 0, diff_count_24h: 0, diff_rate: 0, last_active: null },
    ...overrides,
  }
}

function response(gates: Gate[]): PaginatedResponse<Gate[]> {
  return {
    data: gates,
    pagination: { limit: 5, offset: 0, total: gates.length, has_more: false },
  }
}

function makeFilters(overrides: Partial<GateFilterState> = {}): GateFilterState {
  return { liveUrl: '', shadowUrl: '', sort: 'created_at', order: 'desc', ...overrides }
}

async function mountList(gates: Gate[], filters: GateFilterState) {
  getGates.mockResolvedValue(response(gates))
  const wrapper = mount(GateList, mountOpts({ filters }))
  await flushPromises()
  return wrapper
}

describe('GateList empty states', () => {
  beforeEach(() => {
    getGates.mockReset()
  })

  it('shows the first-run empty state with a create CTA when there are no gates and no filter', async () => {
    const wrapper = await mountList([], makeFilters())

    expect(wrapper.text()).toContain('Create your first gate')
    const cta = wrapper.findAll('button').find((b) => b.text().includes('New gate'))
    expect(cta).toBeTruthy()

    await cta!.trigger('click')
    expect(wrapper.emitted('create')).toHaveLength(1)
  })

  it('shows the no-results state with a clear-filter action when a filter is active', async () => {
    const wrapper = await mountList([], makeFilters({ liveUrl: 'nomatch' }))

    expect(wrapper.text()).toContain('No gates match your current filter')
    expect(wrapper.text()).not.toContain('Create your first gate')

    const clear = wrapper.findAll('button').find((b) => b.text().includes('Clear filter'))
    expect(clear).toBeTruthy()

    await clear!.trigger('click')
    expect(wrapper.emitted('clearFilters')).toHaveLength(1)
  })

  it('renders gate cards and neither empty state when gates exist', async () => {
    const wrapper = await mountList([makeGate(), makeGate({ id: 'gate-2' })], makeFilters())

    expect(wrapper.findAll('.gate-card-stub')).toHaveLength(2)
    expect(wrapper.text()).not.toContain('Create your first gate')
    expect(wrapper.text()).not.toContain('No gates match your current filter')
  })
})

// A page holds 5 gates; total 12 => 3 pages. Used for pagination + reset tests.
function pagedResponse(gates: Gate[], total: number, hasMore: boolean): PaginatedResponse<Gate[]> {
  return { data: gates, pagination: { limit: 5, offset: 0, total, has_more: hasMore } }
}

function lastGetGatesParams(): { offset?: number; live_url?: string } {
  return getGates.mock.calls[getGates.mock.calls.length - 1][0] as {
    offset?: number
    live_url?: string
  }
}

function nextButton(wrapper: Awaited<ReturnType<typeof mountList>>) {
  return wrapper.findAll('button').find((b) => b.text() === 'Next')
}

describe('GateList pagination', () => {
  beforeEach(() => {
    getGates.mockReset()
  })

  it('renders the pager with a page indicator when more than one page exists', async () => {
    getGates.mockResolvedValue(pagedResponse([makeGate()], 12, true))
    const wrapper = mount(GateList, mountOpts({ filters: makeFilters() }))
    await flushPromises()

    expect(wrapper.text()).toContain('Page 1 of 3')
    expect(nextButton(wrapper)).toBeTruthy()
  })

  it('advances the offset by the page size when Next is clicked', async () => {
    getGates.mockResolvedValue(pagedResponse([makeGate()], 12, true))
    const wrapper = mount(GateList, mountOpts({ filters: makeFilters() }))
    await flushPromises()

    await nextButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(lastGetGatesParams().offset).toBe(5)
    expect(wrapper.text()).toContain('Page 2 of 3')
  })
})

describe('GateList pagination stability', () => {
  beforeEach(() => {
    getGates.mockReset()
  })

  // Criterion 5 of #179: paging must not flash empty/loading or render a stale,
  // out-of-order page. placeholderData: keepPreviousData holds the current page
  // while the next one loads, and the query re-keys on the new offset.
  it('keeps the previous page visible while the next page loads and re-keys on offset', async () => {
    // Page 1 resolves immediately; page 2 is deferred so the in-flight window is
    // observable. Each page carries a distinct id so we can tell them apart.
    let resolvePage2!: (value: PaginatedResponse<Gate[]>) => void
    const page2 = new Promise<PaginatedResponse<Gate[]>>((resolve) => {
      resolvePage2 = resolve
    })
    getGates.mockImplementation((params: { offset?: number }) => {
      if ((params.offset ?? 0) === 0) {
        return Promise.resolve(pagedResponse([makeGate({ id: 'gate-p1' })], 12, true))
      }
      return page2
    })

    const wrapper = mount(GateList, mountOpts({ filters: makeFilters() }))
    await flushPromises()
    expect(wrapper.text()).toContain('gate-p1')

    // Advance to page 2. The fetch is still pending, so the query must re-key on
    // the new offset while keepPreviousData keeps page-1 data on screen.
    await nextButton(wrapper)!.trigger('click')
    await flushPromises()
    expect(lastGetGatesParams().offset).toBe(5)
    expect(wrapper.text()).toContain('gate-p1')
    expect(wrapper.text()).not.toContain('gate-p2')

    // Resolving the page-2 fetch swaps the list to the fresh page; the stale
    // page-1 data is gone (no out-of-order render).
    resolvePage2(pagedResponse([makeGate({ id: 'gate-p2' })], 12, true))
    await flushPromises()
    expect(wrapper.text()).toContain('gate-p2')
    expect(wrapper.text()).not.toContain('gate-p1')
  })
})

describe('GateList reset-on-watch', () => {
  beforeEach(() => {
    getGates.mockReset()
  })

  it('resets pagination to the first page and reloads when filters change', async () => {
    getGates.mockResolvedValue(pagedResponse([makeGate()], 12, true))
    const wrapper = mount(GateList, mountOpts({ filters: makeFilters() }))
    await flushPromises()

    // Move to page 2 so the reset is observable.
    await nextButton(wrapper)!.trigger('click')
    await flushPromises()
    expect(lastGetGatesParams().offset).toBe(5)

    // Changing the filters must reset the offset and refetch with the new filter.
    await wrapper.setProps({ filters: makeFilters({ liveUrl: 'checkout' }) })
    await flushPromises()

    const params = lastGetGatesParams()
    expect(params.offset).toBe(0)
    expect(params.live_url).toBe('checkout')
    expect(wrapper.text()).toContain('Page 1 of 3')
  })
})
