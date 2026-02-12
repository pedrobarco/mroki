import { request } from './client'
import type { Gate, ApiResponse, PaginatedResponse, CreateGatePayload } from './types'

/**
 * List all gates with pagination
 *
 * @param params - Pagination parameters (limit, offset)
 * @returns Paginated list of gates
 *
 * @example
 * const response = await getGates({ limit: 20, offset: 0 })
 * console.log(response.data) // Gate[]
 * console.log(response.pagination.total) // Total count
 */
export async function getGates(params?: {
  limit?: number
  offset?: number
}): Promise<PaginatedResponse<Gate[]>> {
  const searchParams = new URLSearchParams()

  if (params?.limit !== undefined) {
    searchParams.set('limit', params.limit.toString())
  }
  if (params?.offset !== undefined) {
    searchParams.set('offset', params.offset.toString())
  }

  const query = searchParams.toString()
  const endpoint = query ? `/gates?${query}` : '/gates'

  return request<PaginatedResponse<Gate[]>>(endpoint)
}

/**
 * Get a single gate by ID
 *
 * @param id - Gate UUID
 * @returns Gate details
 *
 * @example
 * const response = await getGate('550e8400-e29b-41d4-a716-446655440000')
 * console.log(response.data.live_url)
 */
export async function getGate(id: string): Promise<ApiResponse<Gate>> {
  return request<ApiResponse<Gate>>(`/gates/${id}`)
}

/**
 * Create a new gate
 *
 * @param payload - Gate creation payload (live_url, shadow_url)
 * @returns Created gate
 *
 * @example
 * const response = await createGate({
 *   live_url: 'https://api.production.example.com',
 *   shadow_url: 'https://api.shadow.example.com'
 * })
 * console.log(response.data.id)
 */
export async function createGate(payload: CreateGatePayload): Promise<ApiResponse<Gate>> {
  return request<ApiResponse<Gate>>('/gates', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}
