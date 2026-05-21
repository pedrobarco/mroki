import { ref } from 'vue'
import type { Gate } from '@/api'

const cachedGate = ref<Gate | null>(null)

/**
 * Pure in-memory gate cache composable.
 *
 * Pages that always need fresh stats (GateDetail) should fetch from
 * the API and call `setGate` to populate the cache. Pages that only
 * need config data (GateSettings, RequestDetail) can use
 * `getCachedGate` for an instant synchronous lookup and fall back to
 * a fetch only on a cache miss.
 *
 * The composable never makes API calls — callers own the fetch logic.
 */
export function useGateCache() {
  /** Write a gate into the cache, replacing any previously cached gate. */
  function setGate(gate: Gate) {
    cachedGate.value = gate
  }

  /**
   * Synchronous cache lookup. Returns the cached gate if its ID matches,
   * or null if the cache is empty / holds a different gate.
   */
  function getCachedGate(id: string): Gate | null {
    return cachedGate.value?.id === id ? cachedGate.value : null
  }

  /** Clear the cache (e.g. on gate deletion). */
  function clearCache() {
    cachedGate.value = null
  }

  return {
    setGate,
    getCachedGate,
    clearCache,
  }
}
