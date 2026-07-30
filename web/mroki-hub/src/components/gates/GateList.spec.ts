import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GateList from './GateList.vue'
import type { GateFilterState } from './GateFilters.vue'
import type { Gate, PaginatedResponse } from '@/api'

const getGates = vi.fn()
vi.mock('@/api', () => ({
  getGates: (...args: unknown[]) => getGates(...args),
}))

// GateCard renders router links; stub it so the list can mount standalone.
vi.mock('./GateCard.vue', () => ({
  default: { name: 'GateCard', props: ['gate'], template: '<div class="gate-card-stub" />' },
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
  const wrapper = mount(GateList, { props: { filters } })
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
