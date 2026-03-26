import { request } from './client'
import type {
  Request,
  RequestDetail,
  ApiResponse,
  PaginatedResponse,
  ListRequestsParams,
} from './types'

/**
 * List requests for a gate with pagination, filtering, and sorting
 *
 * @param gateId - Gate UUID
 * @param params - Pagination, filter, and sort parameters
 * @returns Paginated list of requests
 *
 * @example
 * const response = await getRequests('550e8400-e29b-41d4-a716-446655440000', {
 *   limit: 20,
 *   offset: 0,
 *   method: ['GET', 'POST'],
 *   sort: 'created_at',
 *   order: 'desc',
 * })
 * console.log(response.data) // Request[]
 * console.log(response.pagination.has_more) // boolean
 */
export async function getRequests(
  gateId: string,
  params?: ListRequestsParams
): Promise<PaginatedResponse<Request[]>> {
  const searchParams = new URLSearchParams()

  if (params?.limit !== undefined) {
    searchParams.set('limit', params.limit.toString())
  }
  if (params?.offset !== undefined) {
    searchParams.set('offset', params.offset.toString())
  }
  if (params?.method !== undefined && params.method.length > 0) {
    searchParams.set('method', params.method.join(','))
  }
  if (params?.path !== undefined && params.path !== '') {
    searchParams.set('path', params.path)
  }
  if (params?.has_diff !== undefined) {
    searchParams.set('has_diff', params.has_diff.toString())
  }
  if (params?.sort !== undefined) {
    searchParams.set('sort', params.sort)
  }
  if (params?.order !== undefined) {
    searchParams.set('order', params.order)
  }

  const query = searchParams.toString()
  const endpoint = query ? `/gates/${gateId}/requests?${query}` : `/gates/${gateId}/requests`

  return request<PaginatedResponse<Request[]>>(endpoint)
}

/**
 * Get a single request with full details (responses and diff)
 *
 * @param gateId - Gate UUID
 * @param requestId - Request UUID
 * @returns Request details with responses and diff
 *
 * @example
 * const response = await getRequest(
 *   '550e8400-e29b-41d4-a716-446655440000',
 *   '6ba7b810-9dad-11d1-80b4-00c04fd430c8'
 * )
 * console.log(response.data.responses) // [live, shadow]
 * console.log(response.data.diff.content) // Diff content
 */
export async function getRequest(
  gateId: string,
  requestId: string
): Promise<ApiResponse<RequestDetail>> {
  return request<ApiResponse<RequestDetail>>(`/gates/${gateId}/requests/${requestId}`)
}
