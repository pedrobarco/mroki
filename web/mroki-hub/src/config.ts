/**
 * Runtime configuration for mroki-hub.
 *
 * In development (Vite dev server), values come from import.meta.env.VITE_*.
 * In production (Docker), values come from window.__MROKI__, which is injected
 * by the container entrypoint script via /config.js before the app loads.
 */

interface MrokiConfig {
  apiBaseUrl: string
  apiKey: string
}

interface MrokiWindowConfig {
  API_BASE_URL?: string
  API_KEY?: string
}

declare global {
  interface Window {
    __MROKI__?: MrokiWindowConfig
  }
}

function loadConfig(): MrokiConfig {
  const runtime = window.__MROKI__

  const apiBaseUrl = runtime?.API_BASE_URL || import.meta.env.VITE_API_BASE_URL
  if (!apiBaseUrl) {
    throw new Error(
      'API base URL is not configured. ' +
        'Set VITE_API_BASE_URL (dev) or MROKI_API_BASE_URL (production).'
    )
  }

  const apiKey = runtime?.API_KEY || import.meta.env.VITE_API_KEY
  if (!apiKey) {
    throw new Error(
      'API key is not configured. ' + 'Set VITE_API_KEY (dev) or MROKI_API_KEY (production).'
    )
  }

  return { apiBaseUrl, apiKey }
}

export const config = loadConfig()
