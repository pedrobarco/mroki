import { ref } from 'vue'
import { getGate } from '@/api'
import type { Gate } from '@/api'

const cachedGate = ref<Gate | null>(null)

/**
 * Simple gate cache composable.
 *
 * GateDetail sets the cache on load. RequestDetail reads it — if the
 * gate ID matches, no extra API call is made. On a direct deep-link
 * to a request, it falls back to fetching.
 */
export function useGateCache() {
  function setGate(gate: Gate) {
    cachedGate.value = gate
  }

  async function getGateById(id: string): Promise<Gate> {
    if (cachedGate.value?.id === id) {
      return cachedGate.value
    }

    const response = await getGate(id)
    cachedGate.value = response.data
    return response.data
  }

  function clearCache() {
    cachedGate.value = null
  }

  return {
    cachedGate,
    setGate,
    getGateById,
    clearCache,
  }
}
