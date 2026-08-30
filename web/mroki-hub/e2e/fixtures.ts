import { test as base } from '@playwright/test'

const API_BASE = 'http://localhost:8090'
const API_KEY = 'mroki-dev-api-key-16'

interface DiffConfig {
  ignored_fields: string[]
  included_fields: string[]
  float_tolerance: number
  sort_arrays: boolean
}

interface Gate {
  id: string
  name: string
  live_url: string
  shadow_url: string
  diff_config: DiffConfig
  redacted_fields: string[]
  retention: string
  created_at: string
}

interface UpdateGatePayload {
  name?: string
  diff_config?: DiffConfig
  redacted_fields?: string[]
  retention?: string
}

interface RequestSummary {
  id: string
  method: string
  path: string
  created_at: string
}

interface ResponsePayload {
  status_code: number
  headers: Record<string, string[]>
  body: string
  latency_ms: number
  created_at: string
}

interface CreateRequestPayload {
  method: string
  path: string
  raw_query?: string
  headers: Record<string, string[]>
  body: string
  created_at: string
  live_response: ResponsePayload
  shadow_response: ResponsePayload
  diff: { content: { op: string; path: string; value?: unknown }[] }
}

export interface ApiHelper {
  createGate(name: string, liveUrl: string, shadowUrl: string): Promise<Gate>
  getGate(gateId: string): Promise<Gate>
  updateGate(gateId: string, payload: UpdateGatePayload): Promise<Gate>
  createRequest(gateId: string, data: CreateRequestPayload): Promise<RequestSummary>
  seedRequest(
    gateId: string,
    options?: {
      method?: string
      path?: string
      rawQuery?: string
      liveBody?: string
      shadowBody?: string
      liveStatus?: number
      shadowStatus?: number
      diffContent?: { op: string; path: string; value?: unknown }[]
      createdAt?: string
    }
  ): Promise<RequestSummary>
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

// Encodes a top-level object key as an RFC 6901 JSON Pointer, escaping the
// reserved '~' and '/' characters ('~' first, per the spec).
function toJsonPointer(key: string): string {
  return `/${key.replace(/~/g, '~0').replace(/\//g, '~1')}`
}

// Order-insensitive JSON serialization: recursively sorts object keys so that a
// mere key reordering isn't mistaken for a value change during comparison.
function stableStringify(value: unknown): string {
  if (Array.isArray(value)) {
    return `[${value.map(stableStringify).join(',')}]`
  }
  if (isPlainObject(value)) {
    return `{${Object.keys(value)
      .sort()
      .map((k) => `${JSON.stringify(k)}:${stableStringify(value[k])}`)
      .join(',')}}`
  }
  return JSON.stringify(value)
}

// Derives a shallow RFC 6902 patch from base64-encoded JSON bodies so the
// default seeded diff stays consistent with overridden live/shadow bodies.
// Only top-level keys are compared: a nested difference collapses to a single
// replace of the whole subtree. Callers needing finer-grained or nested diffs
// should pass an explicit diffContent. Falls back to an empty patch for
// non-JSON or non-object bodies.
function deriveDiffContent(
  liveBody: string,
  shadowBody: string
): { op: string; path: string; value?: unknown }[] {
  let live: unknown
  let shadow: unknown
  try {
    live = JSON.parse(atob(liveBody))
    shadow = JSON.parse(atob(shadowBody))
  } catch {
    return []
  }
  if (!isPlainObject(live) || !isPlainObject(shadow)) {
    return []
  }
  const ops: { op: string; path: string; value?: unknown }[] = []
  for (const key of new Set([...Object.keys(live), ...Object.keys(shadow)])) {
    const path = toJsonPointer(key)
    if (!(key in shadow)) {
      ops.push({ op: 'remove', path })
    } else if (!(key in live)) {
      ops.push({ op: 'add', path, value: shadow[key] })
    } else if (stableStringify(live[key]) !== stableStringify(shadow[key])) {
      ops.push({ op: 'replace', path, value: shadow[key] })
    }
  }
  return ops
}

async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${API_KEY}`,
      ...options.headers,
    },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`API ${res.status}: ${text}`)
  }
  const json = await res.json()
  return json.data as T
}

export const test = base.extend<{ api: ApiHelper }>({
  api: async ({}, use) => {
    const helper: ApiHelper = {
      async createGate(name, liveUrl, shadowUrl) {
        return apiRequest<Gate>('/gates', {
          method: 'POST',
          body: JSON.stringify({ name, live_url: liveUrl, shadow_url: shadowUrl }),
        })
      },

      async getGate(gateId) {
        return apiRequest<Gate>(`/gates/${gateId}`)
      },

      async updateGate(gateId, payload) {
        return apiRequest<Gate>(`/gates/${gateId}`, {
          method: 'PATCH',
          body: JSON.stringify(payload),
        })
      },

      async createRequest(gateId, data) {
        return apiRequest<RequestSummary>(`/gates/${gateId}/requests`, {
          method: 'POST',
          body: JSON.stringify(data),
        })
      },

      async seedRequest(gateId, options = {}) {
        const {
          method = 'GET',
          path = '/api/test',
          rawQuery,
          liveBody = btoa('{"result":"live"}'),
          shadowBody = btoa('{"result":"shadow"}'),
          liveStatus = 200,
          shadowStatus = 200,
          diffContent = deriveDiffContent(liveBody, shadowBody),
          createdAt = new Date().toISOString(),
        } = options

        return this.createRequest(gateId, {
          method,
          path,
          raw_query: rawQuery,
          headers: { 'Content-Type': ['application/json'] },
          body: '',
          created_at: createdAt,
          live_response: {
            status_code: liveStatus,
            headers: { 'Content-Type': ['application/json'] },
            body: liveBody,
            latency_ms: Math.floor(Math.random() * 200) + 20,
            created_at: createdAt,
          },
          shadow_response: {
            status_code: shadowStatus,
            headers: { 'Content-Type': ['application/json'] },
            body: shadowBody,
            latency_ms: Math.floor(Math.random() * 300) + 30,
            created_at: createdAt,
          },
          diff: { content: diffContent },
        })
      },
    }
    await use(helper)
  },
})

export { expect } from '@playwright/test'
