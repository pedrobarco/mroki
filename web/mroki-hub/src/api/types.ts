// Core API entities matching mroki-api contracts

/**
 * Gate represents a live/shadow service pair
 */
export interface Gate {
  id: string
  name: string
  live_url: string
  shadow_url: string
  created_at: string
}

/**
 * Request represents a captured HTTP request
 */
export interface Request {
  id: string
  gate_id: string
  method: string
  path: string
  headers: Record<string, string[]>
  body: string
  created_at: string
}

/**
 * Response represents a single HTTP response (live or shadow)
 */
export interface Response {
  id: string
  type: 'live' | 'shadow'
  status_code: number
  headers: Record<string, string[]>
  body: string
  created_at: string
}

/**
 * PatchOp represents a single RFC 6902 JSON Patch operation
 */
export interface PatchOp {
  op: 'add' | 'remove' | 'replace'
  path: string
  value?: unknown
}

/**
 * Diff contains the computed difference between responses
 */
export interface Diff {
  content: PatchOp[]
}

/**
 * RequestDetail represents a request with full response details and diff
 * Note: This has fewer fields than Request (used in listings)
 */
export interface RequestDetail {
  id: string
  method: string
  path: string
  created_at: string
  responses: Response[]
  diff: Diff
}

// API Response Wrappers

/**
 * Generic API response wrapper
 */
export interface ApiResponse<T> {
  data: T
}

/**
 * Pagination metadata
 */
export interface PaginationMeta {
  limit: number
  offset: number
  total: number
  has_more: boolean
}

/**
 * Paginated API response wrapper
 */
export interface PaginatedResponse<T> {
  data: T
  pagination: PaginationMeta
}

// Error Types (RFC 7807)

/**
 * API error following RFC 7807 Problem Details format
 */
export interface ApiError {
  type: string
  title: string
  status: number
  detail: string
  instance?: string
}

/**
 * Custom error class for API errors
 */
export class ApiErrorException extends Error {
  error: ApiError

  constructor(error: ApiError) {
    super(error.detail)
    this.name = 'ApiErrorException'
    this.error = error
  }
}

// Request Payloads

/**
 * Payload for creating a new gate
 */
export interface CreateGatePayload {
  name: string
  live_url: string
  shadow_url: string
}

/**
 * Valid sort fields for gate listing
 */
export type GateSortField = 'id' | 'name' | 'live_url' | 'shadow_url' | 'created_at'

/**
 * Query parameters for listing gates
 */
export interface ListGatesParams {
  limit?: number
  offset?: number
  name?: string
  live_url?: string
  shadow_url?: string
  sort?: GateSortField
  order?: SortOrder
}

/**
 * Valid sort fields for request listing
 */
export type RequestSortField = 'created_at' | 'method' | 'path'

/**
 * Valid sort directions
 */
export type SortOrder = 'asc' | 'desc'

/**
 * Query parameters for listing requests
 */
export interface ListRequestsParams {
  limit?: number
  offset?: number
  method?: string[]
  path?: string
  has_diff?: boolean
  sort?: RequestSortField
  order?: SortOrder
}
