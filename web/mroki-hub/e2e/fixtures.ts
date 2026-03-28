import { test as base } from '@playwright/test'

const API_BASE = 'http://localhost:8090'
const API_KEY = 'mroki-dev-api-key-16'

interface Gate {
  id: string
  live_url: string
  shadow_url: string
}

interface RequestSummary {
  id: string
  method: string
  path: string
  created_at: string
}

interface CreateRequestPayload {
  agent_id: string
  method: string
  path: string
  headers: Record<string, string[]>
  body: string
  created_at: string
  responses: {
    type: 'live' | 'shadow'
    status_code: number
    headers: Record<string, string[]>
    body: string
    created_at: string
  }[]
  diff: { content: { op: string; path: string; value?: unknown }[] }
}

export interface ApiHelper {
  createGate(liveUrl: string, shadowUrl: string): Promise<Gate>
  createRequest(gateId: string, data: CreateRequestPayload): Promise<RequestSummary>
  seedRequest(
    gateId: string,
    options?: {
      method?: string
      path?: string
      liveBody?: string
      shadowBody?: string
      liveStatus?: number
      shadowStatus?: number
      diffContent?: { op: string; path: string; value?: unknown }[]
      createdAt?: string
    }
  ): Promise<RequestSummary>
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
      async createGate(liveUrl, shadowUrl) {
        return apiRequest<Gate>('/gates', {
          method: 'POST',
          body: JSON.stringify({ live_url: liveUrl, shadow_url: shadowUrl }),
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
          liveBody = '{"result":"live"}',
          shadowBody = '{"result":"shadow"}',
          liveStatus = 200,
          shadowStatus = 200,
          diffContent = [{ op: 'replace', path: '/result', value: 'shadow' }],
          createdAt = new Date().toISOString(),
        } = options

        return this.createRequest(gateId, {
          agent_id: 'e2e-agent-a1b2c3d4',
          method,
          path,
          headers: { 'Content-Type': ['application/json'] },
          body: '',
          created_at: createdAt,
          responses: [
            {
              type: 'live',
              status_code: liveStatus,
              headers: { 'Content-Type': ['application/json'] },
              body: liveBody,
              created_at: createdAt,
            },
            {
              type: 'shadow',
              status_code: shadowStatus,
              headers: { 'Content-Type': ['application/json'] },
              body: shadowBody,
              created_at: createdAt,
            },
          ],
          diff: { content: diffContent },
        })
      },
    }
    await use(helper)
  },
})

export { expect } from '@playwright/test'
