import { ref } from 'vue'
import { getConfig, type AppConfig } from '@/api'

// Server config is static for the lifetime of the session, so it is cached at
// module scope: the first caller triggers a single fetch, concurrent callers
// share the same in-flight promise, and every later caller reads the cache.
const config = ref<AppConfig | null>(null)
let inflight: Promise<AppConfig> | null = null

/**
 * Session-cached access to read-only server settings (e.g. the global
 * retention floor).
 *
 * The value is fetched at most once per session. `load()` is idempotent and
 * safe to call from multiple components; it returns the cached value on
 * subsequent calls and deduplicates concurrent first-time fetches.
 */
export function useAppConfig() {
  async function load(): Promise<AppConfig> {
    if (config.value) {
      return config.value
    }
    if (!inflight) {
      inflight = getConfig()
        .then((response) => {
          config.value = response.data
          return response.data
        })
        .finally(() => {
          inflight = null
        })
    }
    return inflight
  }

  return {
    /** Reactive cached config, or null until the first successful load. */
    config,
    load,
  }
}
