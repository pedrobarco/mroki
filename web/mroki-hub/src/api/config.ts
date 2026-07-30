import { request } from './client'
import type { AppConfig, ApiResponse } from './types'

/**
 * Get read-only, server-wide settings (e.g. the global retention floor).
 *
 * @returns Server configuration
 *
 * @example
 * const response = await getConfig()
 * console.log(response.data.retention) // "720h0m0s"
 */
export async function getConfig(): Promise<ApiResponse<AppConfig>> {
  return request<ApiResponse<AppConfig>>('/config')
}
