import { queryOptions } from '@tanstack/vue-query'
import { getGate, getGates, getGlobalStats } from './gates'
import { getRequest, getRequests } from './requests'
import { getConfig } from './config'
import { queryKeys } from './query-keys'
import type { ListGatesParams, ListRequestsParams } from './types'

/**
 * Thin TanStack Query adapters over the `gates`, `requests`, and `config` API
 * modules. Each factory pairs a key from {@link queryKeys} with a `queryFn`
 * that calls the existing typed API function and unwraps its response envelope.
 *
 * Envelope handling:
 * - Single-resource endpoints return `ApiResponse<T> = { data: T }`; the adapter
 *   unwraps `.data` so consumers receive the bare entity `T`.
 * - List endpoints return `PaginatedResponse<T> = { data: T; pagination }`. Here
 *   `pagination` is meaningful sibling data (not a redundant wrapper), so the
 *   adapter returns the full paginated response unchanged.
 *
 * These are pure option factories — importing them triggers no network calls and
 * no component is migrated by adding this module. Use with `useQuery`, e.g.
 * `const { data } = useQuery(gateQuery(id))`.
 */

/** Query options for the paginated, filterable gate list. */
export function gatesQuery(params?: ListGatesParams) {
  return queryOptions({
    queryKey: queryKeys.gates.list(params),
    queryFn: () => getGates(params),
  })
}

/** Query options for a single gate by id (unwrapped to `Gate`). */
export function gateQuery(id: string) {
  return queryOptions({
    queryKey: queryKeys.gates.detail(id),
    queryFn: async () => (await getGate(id)).data,
  })
}

/** Query options for the paginated, filterable request list of a gate. */
export function requestsQuery(gateId: string, params?: ListRequestsParams) {
  return queryOptions({
    queryKey: queryKeys.requests.list(gateId, params),
    queryFn: () => getRequests(gateId, params),
  })
}

/** Query options for a single request detail (unwrapped to `RequestDetail`). */
export function requestQuery(gateId: string, requestId: string) {
  return queryOptions({
    queryKey: queryKeys.requests.detail(gateId, requestId),
    queryFn: async () => (await getRequest(gateId, requestId)).data,
  })
}

/** Query options for global statistics (unwrapped to `GlobalStats`). */
export function globalStatsQuery() {
  return queryOptions({
    queryKey: queryKeys.stats.global,
    queryFn: async () => (await getGlobalStats()).data,
  })
}

/** Query options for the read-only server config (unwrapped to `AppConfig`). */
export function configQuery() {
  return queryOptions({
    queryKey: queryKeys.config.all,
    queryFn: async () => (await getConfig()).data,
  })
}
