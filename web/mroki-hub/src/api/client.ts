import { config } from '@/config'
import type { ApiError } from './types'
import { ApiErrorException } from './types'

/**
 * Generic HTTP request wrapper with authentication and error handling
 *
 * @param endpoint - API endpoint path (e.g., '/gates')
 * @param options - Fetch options (method, body, headers, etc.)
 * @returns Parsed JSON response
 * @throws ApiErrorException for API errors (RFC 7807)
 * @throws Error for network or other errors
 */
export async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = `${config.apiBaseUrl}${endpoint}`

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${config.apiKey}`,
    ...options.headers,
  }

  const fetchConfig: RequestInit = {
    ...options,
    headers,
  }

  try {
    const response = await fetch(url, fetchConfig)

    // Handle error responses (RFC 7807)
    if (!response.ok) {
      const contentType = response.headers.get('content-type')
      if (contentType?.includes('application/json')) {
        const errorData = (await response.json()) as ApiError
        throw new ApiErrorException(errorData)
      }

      // Fallback for non-JSON errors
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    // Handle empty responses (e.g. 204 No Content)
    if (response.status === 204 || response.headers.get('content-length') === '0') {
      return undefined as T
    }

    // Parse successful response
    return (await response.json()) as T
  } catch (error) {
    // Re-throw ApiErrorException as-is
    if (error instanceof ApiErrorException) {
      throw error
    }

    // Wrap other errors
    if (error instanceof Error) {
      throw new Error(`API request failed: ${error.message}`, { cause: error })
    }

    throw new Error('Unknown API error occurred', { cause: error })
  }
}
