import type { ApiError } from './types'
import { ApiErrorException } from './types'

/**
 * Base API client configuration
 */
const getBaseURL = (): string => {
  const baseURL = import.meta.env.VITE_API_BASE_URL
  if (!baseURL) {
    throw new Error('VITE_API_BASE_URL environment variable is not set')
  }
  return baseURL
}

const getApiKey = (): string => {
  const apiKey = import.meta.env.VITE_API_KEY
  if (!apiKey) {
    throw new Error('VITE_API_KEY environment variable is not set')
  }
  return apiKey
}

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
  const baseURL = getBaseURL()
  const apiKey = getApiKey()

  const url = `${baseURL}${endpoint}`

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${apiKey}`,
    ...options.headers,
  }

  const config: RequestInit = {
    ...options,
    headers,
  }

  try {
    const response = await fetch(url, config)

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
