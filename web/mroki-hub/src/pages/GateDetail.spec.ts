import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import GateDetail from './GateDetail.vue'
import type { Gate } from '@/api'

const getGate = vi.fn()
// Mock the underlying API module so the real query adapter (gateQuery, from
// '@/api') keeps working while the network call is stubbed.
vi.mock('@/api/gates', () => ({
  getGate: (...a: unknown[]) => getGate(...a),
}))

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'gate-1' } }),
  useRouter: () => ({ push }),
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
    stats: { request_count_24h: 3, diff_count_24h: 1, diff_rate: 12.5, last_active: null },
    ...overrides,
  }
}

// The error branch renders the Alert (title + message) and a Retry button, so
// those are passthrough stubs (slots rendered); the request list/filters are
// stubbed away so the page exercises only its own gate-query states.
const passthrough = { template: '<div><slot /></div>' }
function stateGlobal() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return {
    stubs: {
      RequestList: true,
      RequestFilters: true,
      Alert: passthrough,
      AlertDescription: passthrough,
      AlertTitle: passthrough,
      Button: { template: '<button><slot /></button>' },
      ChevronLeft: true,
      Settings: true,
    },
    plugins: [[VueQueryPlugin, { queryClient }]],
  }
}

function mountDetail() {
  return mount(GateDetail, { global: stateGlobal() })
}

describe('GateDetail loading, error, and data states', () => {
  beforeEach(() => {
    push.mockClear()
    getGate.mockReset()
  })

  it('shows the loading state while the gate is in flight', () => {
    // A never-resolving fetch keeps the query pending so the loading branch is
    // observable.
    getGate.mockReturnValue(new Promise(() => {}))
    const wrapper = mountDetail()

    expect(wrapper.text()).toContain('Loading gate details...')
    expect(wrapper.findComponent({ name: 'RequestList' }).exists()).toBe(false)
  })

  it('renders the gate info and the request list once loaded', async () => {
    getGate.mockResolvedValue({ data: makeGate() })
    const wrapper = mountDetail()
    await flushPromises()

    expect(wrapper.text()).toContain('checkout-api')
    expect(wrapper.text()).toContain('https://live.example.com')
    expect(wrapper.text()).toContain('https://shadow.example.com')
    expect(wrapper.text()).not.toContain('Loading gate details...')
    expect(wrapper.findComponent({ name: 'RequestList' }).exists()).toBe(true)
  })

  it('shows the error alert with the failure message when the fetch rejects', async () => {
    getGate.mockRejectedValue(new Error('gate boom'))
    const wrapper = mountDetail()
    await flushPromises()

    expect(wrapper.text()).toContain('Error')
    expect(wrapper.text()).toContain('gate boom')
    expect(wrapper.findComponent({ name: 'RequestList' }).exists()).toBe(false)
  })

  it('refetches when Retry is clicked and recovers to the gate on success', async () => {
    getGate.mockRejectedValueOnce(new Error('gate boom'))
    const wrapper = mountDetail()
    await flushPromises()
    expect(wrapper.text()).toContain('gate boom')

    // The retry path calls query.refetch(); the next fetch resolves, so the
    // error clears and the gate info renders.
    getGate.mockResolvedValue({ data: makeGate() })
    const retry = wrapper.findAll('button').find((b) => b.text().includes('Retry'))
    expect(retry).toBeTruthy()
    await retry!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('gate boom')
    expect(wrapper.text()).toContain('checkout-api')
  })
})
