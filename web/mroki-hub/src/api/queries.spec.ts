import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  gatesQuery,
  gateQuery,
  requestsQuery,
  requestQuery,
  globalStatsQuery,
  configQuery,
} from './queries'
import { queryKeys } from './query-keys'
import type {
  Gate,
  GlobalStats,
  AppConfig,
  Request,
  RequestDetail,
  PaginatedResponse,
} from './types'

const getGates = vi.fn()
const getGate = vi.fn()
const getGlobalStats = vi.fn()
const getRequests = vi.fn()
const getRequest = vi.fn()
const getConfig = vi.fn()

vi.mock('./gates', () => ({
  getGates: (...a: unknown[]) => getGates(...a),
  getGate: (...a: unknown[]) => getGate(...a),
  getGlobalStats: (...a: unknown[]) => getGlobalStats(...a),
}))
vi.mock('./requests', () => ({
  getRequests: (...a: unknown[]) => getRequests(...a),
  getRequest: (...a: unknown[]) => getRequest(...a),
}))
vi.mock('./config', () => ({
  getConfig: (...a: unknown[]) => getConfig(...a),
}))

beforeEach(() => {
  vi.resetAllMocks()
})

const gate = { id: 'g1', name: 'checkout' } as Gate
const paginatedGates: PaginatedResponse<Gate[]> = {
  data: [gate],
  pagination: { limit: 20, offset: 0, total: 1, has_more: false },
}

describe('gate query options', () => {
  it('gatesQuery pairs the list key with a queryFn that returns the full paginated response', async () => {
    getGates.mockResolvedValue(paginatedGates)
    const opts = gatesQuery({ limit: 20 })

    expect(opts.queryKey).toEqual(queryKeys.gates.list({ limit: 20 }))
    await expect(opts.queryFn!({} as never)).resolves.toEqual(paginatedGates)
    expect(getGates).toHaveBeenCalledWith({ limit: 20 })
  })

  it('gateQuery pairs the detail key with a queryFn that unwraps .data to the entity', async () => {
    getGate.mockResolvedValue({ data: gate })
    const opts = gateQuery('g1')

    expect(opts.queryKey).toEqual(queryKeys.gates.detail('g1'))
    await expect(opts.queryFn!({} as never)).resolves.toEqual(gate)
    expect(getGate).toHaveBeenCalledWith('g1')
  })
})

describe('request query options', () => {
  const req = { id: 'r1', method: 'GET' } as Request
  const paginatedRequests: PaginatedResponse<Request[]> = {
    data: [req],
    pagination: { limit: 20, offset: 0, total: 1, has_more: false },
  }

  it('requestsQuery pairs the scoped list key with a queryFn returning the full paginated response', async () => {
    getRequests.mockResolvedValue(paginatedRequests)
    const opts = requestsQuery('g1', { limit: 20 })

    expect(opts.queryKey).toEqual(queryKeys.requests.list('g1', { limit: 20 }))
    await expect(opts.queryFn!({} as never)).resolves.toEqual(paginatedRequests)
    expect(getRequests).toHaveBeenCalledWith('g1', { limit: 20 })
  })

  it('requestQuery pairs the detail key with a queryFn that unwraps .data', async () => {
    const detail = { id: 'r1', method: 'GET' } as RequestDetail
    getRequest.mockResolvedValue({ data: detail })
    const opts = requestQuery('g1', 'r1')

    expect(opts.queryKey).toEqual(queryKeys.requests.detail('g1', 'r1'))
    await expect(opts.queryFn!({} as never)).resolves.toEqual(detail)
    expect(getRequest).toHaveBeenCalledWith('g1', 'r1')
  })
})

describe('stats and config query options', () => {
  it('globalStatsQuery unwraps .data to the GlobalStats entity', async () => {
    const stats = { total_gates: 3, total_requests_24h: 10, total_diff_rate: 0.1 } as GlobalStats
    getGlobalStats.mockResolvedValue({ data: stats })
    const opts = globalStatsQuery()

    expect(opts.queryKey).toEqual(queryKeys.stats.global)
    await expect(opts.queryFn!({} as never)).resolves.toEqual(stats)
    expect(getGlobalStats).toHaveBeenCalledTimes(1)
  })

  it('configQuery unwraps .data to the AppConfig entity', async () => {
    const cfg = { retention: '720h0m0s' } as AppConfig
    getConfig.mockResolvedValue({ data: cfg })
    const opts = configQuery()

    expect(opts.queryKey).toEqual(queryKeys.config.all)
    await expect(opts.queryFn!({} as never)).resolves.toEqual(cfg)
    expect(getConfig).toHaveBeenCalledTimes(1)
  })
})
