import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { configQuery, type AppConfig } from '@/api'

/**
 * Session-cached access to read-only server settings (e.g. the global
 * retention floor).
 *
 * Backed by TanStack Query: the value is fetched at most once per session and
 * shared across every caller through the `config` query key. Concurrent
 * first-time callers are deduplicated by the query cache, so no bespoke
 * in-flight tracking is needed. `staleTime: Infinity` means the config is never
 * considered stale during a session (it only changes on a server restart), so
 * navigating between pages never refetches it.
 *
 * Must be called from a component `setup()` so the shared QueryClient can be
 * injected.
 */
export function useAppConfig() {
  const query = useQuery({
    ...configQuery(),
    staleTime: Infinity,
    gcTime: Infinity,
  })

  /** Reactive cached config, or null until the first successful load. */
  const config = computed<AppConfig | null>(() => query.data.value ?? null)

  return {
    config,
    /** The underlying query, exposing `refetch`, `error`, `isPending`, etc. */
    query,
  }
}
