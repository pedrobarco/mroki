import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import RequestList from './RequestList.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { FilterState } from './RequestFilters.vue'
import type { PaginatedResponse, Request } from '@/api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

const getRequests = vi.fn()
// Mock the underlying API module so the real query adapter (requestsQuery, from
// '@/api') keeps working while the network call is stubbed.
vi.mock('@/api/requests', () => ({
  getRequests: (...args: unknown[]) => getRequests(...args),
}))

function makeRequest(overrides: Partial<Request> = {}): Request {
  return {
    id: 'req-1',
    method: 'GET',
    path: '/api/users',
    created_at: '2026-03-29T09:00:00Z',
    live_response: { status_code: 200, latency_ms: 10 },
    shadow_response: { status_code: 200, latency_ms: 12 },
    has_diff: false,
    ...overrides,
  }
}

function makeResponse(requests: Request[]): PaginatedResponse<Request[]> {
  return {
    data: requests,
    pagination: { limit: 20, offset: 0, total: requests.length, has_more: false },
  }
}

const filters: FilterState = {
  methods: [],
  path: '',
  hasDiff: undefined,
  sort: 'created_at',
  order: 'desc',
}

// Stub UI components so mounting does not require their internals. A fresh,
// retry-free client per mount isolates the request-list cache per test so paging
// assertions always observe a real fetch.
function makeGlobal() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return {
    stubs: {
      Alert: true,
      AlertDescription: true,
      AlertTitle: true,
      Button: true,
      Tooltip: true,
      TooltipContent: true,
      TooltipProvider: false,
      TooltipTrigger: true,
      ChevronRight: true,
    },
    plugins: [[VueQueryPlugin, { queryClient }]],
  }
}

async function mountList(requests: Request[]) {
  getRequests.mockResolvedValue(makeResponse(requests))
  const wrapper = mount(RequestList, { props: { gateId: 'gate-1', filters }, global: makeGlobal() })
  await flushPromises()
  return wrapper
}

describe('RequestList row keyboard operability', () => {
  beforeEach(() => {
    push.mockClear()
    getRequests.mockReset()
  })

  it('exposes button semantics on each request row', async () => {
    const wrapper = await mountList([makeRequest({ method: 'POST', path: '/api/orders' })])
    const row = wrapper.get('[role="button"]')
    expect(row.attributes('tabindex')).toBe('0')
    expect(row.attributes('aria-label')).toBe('View request POST /api/orders')
  })

  it('navigates to the request on Enter', async () => {
    const wrapper = await mountList([makeRequest({ id: 'req-9' })])
    await wrapper.get('[role="button"]').trigger('keydown.enter')
    expect(push).toHaveBeenCalledWith('/gates/gate-1/requests/req-9')
  })

  it('navigates to the request on Space', async () => {
    const wrapper = await mountList([makeRequest({ id: 'req-7' })])
    await wrapper.get('[role="button"]').trigger('keydown.space')
    expect(push).toHaveBeenCalledWith('/gates/gate-1/requests/req-7')
  })

  it('navigates to the request on click', async () => {
    const wrapper = await mountList([makeRequest({ id: 'req-3' })])
    await wrapper.get('[role="button"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/gates/gate-1/requests/req-3')
  })
})

describe('RequestList pagination', () => {
  beforeEach(() => {
    push.mockClear()
    getRequests.mockReset()
  })

  it('hides the pager when there is a single page', async () => {
    const wrapper = await mountList([makeRequest()])
    expect(wrapper.findComponent(Pagination).exists()).toBe(true)
    // The pager component mounts but renders nothing while totalPages <= 1.
    expect(wrapper.findComponent(Pagination).find('button').exists()).toBe(false)
  })

  it('renders the pager when more than one page exists', async () => {
    getRequests.mockResolvedValue({
      data: [makeRequest()],
      pagination: { limit: 20, offset: 0, total: 40, has_more: true },
    } satisfies PaginatedResponse<Request[]>)
    const wrapper = mount(RequestList, {
      props: { gateId: 'gate-1', filters },
      global: makeGlobal(),
    })
    await flushPromises()
    const pager = wrapper.findComponent(Pagination)
    expect(pager.text()).toContain('Page 1 of 2')
    expect(pager.findAll('button')).toHaveLength(2)
  })
})

describe('RequestList reset-on-watch', () => {
  beforeEach(() => {
    push.mockClear()
    getRequests.mockReset()
  })

  function lastParams(): { offset?: number; path?: string } {
    return getRequests.mock.calls[getRequests.mock.calls.length - 1][1] as {
      offset?: number
      path?: string
    }
  }

  it('resets the offset to the first page and refetches when filters change', async () => {
    // 40 rows over a page size of 20 => two pages, so the pager is interactive.
    getRequests.mockResolvedValue({
      data: [makeRequest()],
      pagination: { limit: 20, offset: 0, total: 40, has_more: true },
    } satisfies PaginatedResponse<Request[]>)
    const wrapper = mount(RequestList, {
      props: { gateId: 'gate-1', filters },
      global: makeGlobal(),
    })
    await flushPromises()

    // Advance to page 2 so the reset is observable.
    const next = wrapper.findComponent(Pagination).findAll('button')[1]
    await next.trigger('click')
    await flushPromises()
    expect(lastParams().offset).toBe(20)

    // Changing the filters must reset the offset and refetch with the new path.
    await wrapper.setProps({ filters: { ...filters, path: '/api/orders' } })
    await flushPromises()

    const params = lastParams()
    expect(params.offset).toBe(0)
    expect(params.path).toBe('/api/orders')
    expect(wrapper.findComponent(Pagination).text()).toContain('Page 1 of 2')
  })
})

describe('RequestList showing count', () => {
  beforeEach(() => {
    push.mockClear()
    getRequests.mockReset()
  })

  it('emits the showing count from the rendered table row model', async () => {
    const wrapper = await mountList([makeRequest({ id: 'req-1' }), makeRequest({ id: 'req-2' })])
    // The emitted count matches the rows actually rendered from the row model.
    expect(wrapper.findAll('[role="button"]')).toHaveLength(2)
    const showing = wrapper.emitted('update:showing')
    expect(showing).toBeTruthy()
    expect(showing!.at(-1)).toEqual([2])
  })
})

describe('RequestList latency formatting', () => {
  beforeEach(() => {
    push.mockClear()
    getRequests.mockReset()
  })

  it('renders formatted latencies with a single unit', async () => {
    const wrapper = await mountList([
      makeRequest({
        live_response: { status_code: 200, latency_ms: 10 },
        shadow_response: { status_code: 200, latency_ms: 12 },
      }),
    ])
    expect(wrapper.text()).toContain('10ms / 12ms')
  })

  it('renders an em dash when a latency is missing', async () => {
    const wrapper = await mountList([
      makeRequest({
        live_response: { status_code: 200, latency_ms: 10 },
        shadow_response: null,
      }),
    ])
    expect(wrapper.text()).toContain('10ms / —')
  })
})

describe('RequestList loading, error, and empty states', () => {
  beforeEach(() => {
    push.mockClear()
    getRequests.mockReset()
  })

  // The error branch renders the Alert (title + message) and a Retry button, so
  // these are passthrough stubs (slots rendered) rather than the `true` stubs
  // used elsewhere; the loading/empty branches are plain divs and need none.
  const passthrough = { template: '<div><slot /></div>' }
  function stateGlobal() {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    return {
      stubs: {
        Alert: passthrough,
        AlertDescription: passthrough,
        AlertTitle: passthrough,
        Button: { template: '<button><slot /></button>' },
        Tooltip: true,
        TooltipContent: true,
        TooltipProvider: false,
        TooltipTrigger: true,
        ChevronRight: true,
      },
      plugins: [[VueQueryPlugin, { queryClient }]],
    }
  }

  it('shows the loading state while the first page is in flight', () => {
    // A never-resolving fetch keeps the query pending so the loading branch is
    // observable (no previous data for keepPreviousData to hold).
    getRequests.mockReturnValue(new Promise(() => {}))
    const wrapper = mount(RequestList, {
      props: { gateId: 'gate-1', filters },
      global: stateGlobal(),
    })

    expect(wrapper.text()).toContain('Loading requests...')
  })

  it('shows the empty state when the gate has no captured requests', async () => {
    getRequests.mockResolvedValue(makeResponse([]))
    const wrapper = mount(RequestList, {
      props: { gateId: 'gate-1', filters },
      global: stateGlobal(),
    })
    await flushPromises()

    expect(wrapper.text()).toContain('No requests captured yet')
    expect(wrapper.find('[role="button"]').exists()).toBe(false)
  })

  it('shows the error alert with the failure message when the fetch rejects', async () => {
    getRequests.mockRejectedValue(new Error('kaboom'))
    const wrapper = mount(RequestList, {
      props: { gateId: 'gate-1', filters },
      global: stateGlobal(),
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Error')
    expect(wrapper.text()).toContain('kaboom')
  })

  it('refetches when Retry is clicked and recovers to the rows on success', async () => {
    getRequests.mockRejectedValueOnce(new Error('kaboom'))
    const wrapper = mount(RequestList, {
      props: { gateId: 'gate-1', filters },
      global: stateGlobal(),
    })
    await flushPromises()
    expect(wrapper.text()).toContain('kaboom')

    getRequests.mockResolvedValue(makeResponse([makeRequest({ id: 'req-ok' })]))
    const retry = wrapper.findAll('button').find((b) => b.text().includes('Retry'))
    expect(retry).toBeTruthy()
    await retry!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('kaboom')
    expect(wrapper.findAll('[role="button"]')).toHaveLength(1)
  })
})
