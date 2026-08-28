import { describe, it, expect } from 'vitest'
import { queryKeys } from './query-keys'
import type { ListGatesParams, ListRequestsParams } from './types'

describe('queryKeys.gates', () => {
  it('builds hierarchical keys that share a common prefix for invalidation', () => {
    expect(queryKeys.gates.all).toEqual(['gates'])
    expect(queryKeys.gates.lists()).toEqual(['gates', 'list'])
    expect(queryKeys.gates.details()).toEqual(['gates', 'detail'])
    expect(queryKeys.gates.list()).toEqual(['gates', 'list', {}])
    expect(queryKeys.gates.detail('abc')).toEqual(['gates', 'detail', 'abc'])
  })

  it('embeds list params verbatim so equal params hash to the same key', () => {
    const params: ListGatesParams = { limit: 20, offset: 0, sort: 'name', order: 'asc' }
    expect(queryKeys.gates.list(params)).toEqual(['gates', 'list', params])
    expect(queryKeys.gates.list(params)).toEqual(queryKeys.gates.list({ ...params }))
  })
})

describe('queryKeys.requests', () => {
  it('builds hierarchical keys scoped by gate id', () => {
    expect(queryKeys.requests.all).toEqual(['requests'])
    expect(queryKeys.requests.lists()).toEqual(['requests', 'list'])
    expect(queryKeys.requests.details()).toEqual(['requests', 'detail'])
    expect(queryKeys.requests.list('g1')).toEqual(['requests', 'list', 'g1', {}])
    expect(queryKeys.requests.detail('g1', 'r1')).toEqual(['requests', 'detail', 'g1', 'r1'])
  })

  it('embeds gate id and list params so distinct filters resolve to distinct keys', () => {
    const params: ListRequestsParams = { limit: 20, offset: 40, method: ['GET'], has_diff: true }
    expect(queryKeys.requests.list('g1', params)).toEqual(['requests', 'list', 'g1', params])
    expect(queryKeys.requests.list('g1', params)).not.toEqual(queryKeys.requests.list('g2', params))
  })
})

describe('queryKeys.stats and config', () => {
  it('builds stable keys for the singleton queries', () => {
    expect(queryKeys.stats.global).toEqual(['stats', 'global'])
    expect(queryKeys.config.all).toEqual(['config'])
  })
})
