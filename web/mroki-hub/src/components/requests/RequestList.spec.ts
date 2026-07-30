import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import RequestList from './RequestList.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { FilterState } from './RequestFilters.vue'
import type { PaginatedResponse, Request } from '@/api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

const getRequests = vi.fn()
vi.mock('@/api', () => ({
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

// Stub UI components so mounting does not require their internals.
const global = {
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
}

async function mountList(requests: Request[]) {
  getRequests.mockResolvedValue(makeResponse(requests))
  const wrapper = mount(RequestList, { props: { gateId: 'gate-1', filters }, global })
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
    const wrapper = mount(RequestList, { props: { gateId: 'gate-1', filters }, global })
    await flushPromises()
    const pager = wrapper.findComponent(Pagination)
    expect(pager.text()).toContain('Page 1 of 2')
    expect(pager.findAll('button')).toHaveLength(2)
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
