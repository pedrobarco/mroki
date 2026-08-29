import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { createGate, updateGate, deleteGate } from './gates'
import { queryKeys } from './query-keys'
import type { CreateGatePayload, UpdateGatePayload } from './types'

/**
 * Gate mutation composables over the `gates` API module. Each pairs a
 * `useMutation` with the shared query cache: on success it invalidates the
 * hierarchical keys from {@link queryKeys} so active reads refetch the canonical
 * server state, rather than hand-writing entities into the cache.
 *
 * This is the template for later mutation work — keep new mutations here,
 * co-located with their invalidation, and let components own only UI concerns
 * (dialogs, navigation, error routing). Rollback is intentionally left simple:
 * no optimistic updates, so a failed mutation leaves the cache untouched.
 */

/**
 * Create a gate. On success, refreshes every gate list variant and the global
 * stats; a new gate has no detail cache entry yet, so none is touched.
 */
export function useCreateGateMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateGatePayload) => createGate(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.gates.lists() })
      queryClient.invalidateQueries({ queryKey: queryKeys.stats.global })
    },
  })
}

/**
 * Update a gate by id. On success, invalidates that gate's detail (so every
 * consumer — settings, detail, request detail — refetches) and all list
 * variants (name/config changes surface in listings).
 */
export function useUpdateGateMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateGatePayload }) =>
      updateGate(id, payload),
    onSuccess: (_response, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.gates.detail(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.gates.lists() })
    },
  })
}

/**
 * Delete a gate by id. On success, drops the now-gone gate's detail from the
 * cache and refreshes the list and global stats reads.
 */
export function useDeleteGateMutation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteGate(id),
    onSuccess: (_response, id) => {
      queryClient.removeQueries({ queryKey: queryKeys.gates.detail(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.gates.lists() })
      queryClient.invalidateQueries({ queryKey: queryKeys.stats.global })
    },
  })
}
