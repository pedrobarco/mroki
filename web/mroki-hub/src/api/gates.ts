import { request } from './client'
import type {
  Gate,
  GlobalStats,
  ApiResponse,
  PaginatedResponse,
  CreateGatePayload,
  UpdateGatePayload,
  ListGatesParams,
} from './types'

/**
 * List all gates with pagination, filtering, and sorting
 *
 * @param params - Pagination, filter, and sort parameters
 * @returns Paginated list of gates
 *
 * @example
 * const response = await getGates({ limit: 20, offset: 0, sort: 'live_url', order: 'asc' })
 * console.log(response.data) // Gate[]
 * console.log(response.pagination.total) // Total count
 */
export async function getGates(params?: ListGatesParams): Promise<PaginatedResponse<Gate[]>> {
  const searchParams = new URLSearchParams()

  if (params?.limit !== undefined) {
    searchParams.set('limit', params.limit.toString())
  }
  if (params?.offset !== undefined) {
    searchParams.set('offset', params.offset.toString())
  }
  if (params?.name) {
    searchParams.set('name', params.name)
  }
  if (params?.live_url) {
    searchParams.set('live_url', params.live_url)
  }
  if (params?.shadow_url) {
    searchParams.set('shadow_url', params.shadow_url)
  }
  if (params?.sort) {
    searchParams.set('sort', params.sort)
  }
  if (params?.order) {
    searchParams.set('order', params.order)
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
 * @param payload - Gate creation payload (name, live_url, shadow_url)
 * @returns Created gate
 *
 * @example
 * const response = await createGate({
 *   name: 'checkout-api',
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

/**
 * Update an existing gate
 *
 * @param id - Gate UUID
 * @param payload - Fields to update (name and/or diff_config)
 * @returns Updated gate
 *
 * @example
 * const response = await updateGate('550e8400-...', { name: 'checkout-api-v2' })
 * console.log(response.data.name) // 'checkout-api-v2'
 */
export async function updateGate(
  id: string,
  payload: UpdateGatePayload
): Promise<ApiResponse<Gate>> {
  return request<ApiResponse<Gate>>(`/gates/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

/**
 * Delete a gate and all its associated requests
 *
 * @param id - Gate UUID
 */
export async function deleteGate(id: string): Promise<void> {
  await request(`/gates/${id}`, { method: 'DELETE' })
}

/**
 * Get global statistics across all gates
 *
 * @returns Global stats (total gates, requests 24h, diff rate)
 *
 * @example
 * const response = await getGlobalStats()
 * console.log(response.data.total_requests_24h)
 */
export async function getGlobalStats(): Promise<ApiResponse<GlobalStats>> {
  return request<ApiResponse<GlobalStats>>('/stats')
}
