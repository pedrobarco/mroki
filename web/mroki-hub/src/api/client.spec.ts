import { describe, it, expect, vi, beforeEach } from 'vitest'
import { request } from './client'
import { ApiErrorException, type ApiError } from './types'

// The real config module reads window/import.meta env at import time and throws
// when unset, so we stub it with deterministic values for the client under test.
vi.mock('@/config', () => ({
  config: { apiBaseUrl: 'https://api.test', apiKey: 'test-key' },
}))

interface FakeResponseInit {
  ok: boolean
  status: number
  statusText?: string
  contentType?: string | null
  contentLength?: string | null
  json?: unknown
}

function fakeResponse(init: FakeResponseInit): Response {
  const headers = new Map<string, string | null>()
  if (init.contentType !== undefined) headers.set('content-type', init.contentType)
  if (init.contentLength !== undefined) headers.set('content-length', init.contentLength)
  return {
    ok: init.ok,
    status: init.status,
    statusText: init.statusText ?? '',
    headers: { get: (key: string) => headers.get(key.toLowerCase()) ?? null },
    json: async () => init.json,
  } as unknown as Response
}

const fetchMock = vi.fn()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

describe('request success parsing', () => {
  it('resolves with the parsed JSON body and calls the composed URL with auth headers', async () => {
    fetchMock.mockResolvedValue(
      fakeResponse({ ok: true, status: 200, contentType: 'application/json', json: { data: 42 } })
    )

    const result = await request<{ data: number }>('/gates')

    expect(result).toEqual({ data: 42 })
    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toBe('https://api.test/gates')
    expect(options.headers).toMatchObject({
      'Content-Type': 'application/json',
      Authorization: 'Bearer test-key',
    })
  })

  it('merges caller-supplied headers and preserves the method', async () => {
    fetchMock.mockResolvedValue(
      fakeResponse({ ok: true, status: 200, contentType: 'application/json', json: {} })
    )

    await request('/gates', { method: 'POST', headers: { 'X-Trace': 'abc' } })

    const [, options] = fetchMock.mock.calls[0]
    expect(options.method).toBe('POST')
    expect(options.headers).toMatchObject({ 'X-Trace': 'abc', Authorization: 'Bearer test-key' })
  })

  it('returns undefined for 204 No Content without parsing a body', async () => {
    const json = vi.fn()
    fetchMock.mockResolvedValue({
      ok: true,
      status: 204,
      statusText: 'No Content',
      headers: { get: () => null },
      json,
    } as unknown as Response)

    const result = await request('/gates/1', { method: 'DELETE' })

    expect(result).toBeUndefined()
    expect(json).not.toHaveBeenCalled()
  })

  it('returns undefined when content-length is zero', async () => {
    fetchMock.mockResolvedValue(fakeResponse({ ok: true, status: 200, contentLength: '0' }))
    expect(await request('/empty')).toBeUndefined()
  })
})

describe('request error parsing (RFC 7807)', () => {
  it('throws ApiErrorException carrying the problem-details payload for JSON errors', async () => {
    const problem: ApiError = {
      type: 'about:blank',
      title: 'Not Found',
      status: 404,
      detail: 'gate not found',
    }
    fetchMock.mockResolvedValue(
      fakeResponse({ ok: false, status: 404, contentType: 'application/json', json: problem })
    )

    const err = await request('/gates/x').catch((e) => e)
    expect(err).toBeInstanceOf(ApiErrorException)
    expect((err as ApiErrorException).error).toEqual(problem)
    expect((err as ApiErrorException).message).toBe('gate not found')
  })

  it('surfaces the HTTP status for non-JSON error responses, wrapped by the generic handler', async () => {
    fetchMock.mockResolvedValue(
      fakeResponse({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        contentType: 'text/plain',
      })
    )

    const err = await request('/gates').catch((e) => e)
    expect(err).toBeInstanceOf(Error)
    expect(err).not.toBeInstanceOf(ApiErrorException)
    // The non-JSON branch throws a plain Error, which the catch re-wraps with context.
    expect((err as Error).message).toBe('API request failed: HTTP 500: Internal Server Error')
    expect(((err as Error).cause as Error).message).toBe('HTTP 500: Internal Server Error')
  })
})

describe('request network failure handling', () => {
  it('wraps a rejected fetch with context and preserves the cause', async () => {
    const cause = new Error('connection refused')
    fetchMock.mockRejectedValue(cause)

    const err = await request('/gates').catch((e) => e)
    expect(err).toBeInstanceOf(Error)
    expect((err as Error).message).toBe('API request failed: connection refused')
    expect((err as Error).cause).toBe(cause)
  })
})
