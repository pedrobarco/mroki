import type { ListGatesParams, ListRequestsParams } from './types'

/**
 * Typed query-key factory: the single source of truth for every TanStack Query
 * cache key used by the hub.
 *
 * Keys are hierarchical so that broad invalidation works out of the box — e.g.
 * invalidating `queryKeys.gates.all` also matches every gate list and detail
 * key, and `queryKeys.gates.lists()` matches every filtered list variant. Never
 * hand-write a key array in a component; always derive it from this factory so
 * cache reads, writes, and invalidations stay consistent.
 *
 * Filter/param objects are embedded verbatim; TanStack Query hashes keys
 * structurally, so two calls with equal params resolve to the same cache entry.
 *
 * Convention: resource root keys and true singletons are plain array values
 * (`gates.all`, `stats.global`, `config.all`); everything else is a function.
 * That includes the intermediate grouping keys (`lists()`, `details()`), which
 * take no params but stay functions so they read consistently with their
 * parameterized siblings (`list(params)`, `detail(id)`) and keep the
 * `all → lists() → list() → details() → detail()` hierarchy uniform.
 */
export const queryKeys = {
  gates: {
    /** Root key for all gate-related queries. */
    all: ['gates'] as const,
    /** Root key for all gate list queries (any filter set). */
    lists: () => [...queryKeys.gates.all, 'list'] as const,
    /** Key for a gate list narrowed by the given filter/pagination params. */
    list: (params?: ListGatesParams) => [...queryKeys.gates.lists(), params ?? {}] as const,
    /** Root key for all single-gate detail queries. */
    details: () => [...queryKeys.gates.all, 'detail'] as const,
    /** Key for a single gate by id. */
    detail: (id: string) => [...queryKeys.gates.details(), id] as const,
  },
  requests: {
    /** Root key for all request-related queries. */
    all: ['requests'] as const,
    /** Root key for all request list queries (any gate / filter set). */
    lists: () => [...queryKeys.requests.all, 'list'] as const,
    /**
     * Key for a request list scoped to a gate and narrowed by the given
     * filter/pagination params (page is carried inside `params.offset`).
     */
    list: (gateId: string, params?: ListRequestsParams) =>
      [...queryKeys.requests.lists(), gateId, params ?? {}] as const,
    /** Root key for all single-request detail queries. */
    details: () => [...queryKeys.requests.all, 'detail'] as const,
    /** Key for a single request by gate id + request id. */
    detail: (gateId: string, requestId: string) =>
      [...queryKeys.requests.details(), gateId, requestId] as const,
  },
  stats: {
    /** Key for the global statistics query (singleton, no params). */
    global: ['stats', 'global'] as const,
  },
  config: {
    /** Key for the read-only server config query (singleton, no params). */
    all: ['config'] as const,
  },
} as const
