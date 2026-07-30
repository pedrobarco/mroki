import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { ApiResponse, AppConfig } from '@/api'

const getConfig = vi.fn<() => Promise<ApiResponse<AppConfig>>>()

vi.mock('@/api', () => ({
  getConfig: () => getConfig(),
}))

// The composable caches at module scope, so each test re-imports it fresh to
// start from an empty cache.
async function freshComposable() {
  vi.resetModules()
  const mod = await import('./use-app-config')
  return mod.useAppConfig()
}

beforeEach(() => {
  getConfig.mockReset()
})

describe('useAppConfig', () => {
  it('config is null before the first load', async () => {
    const { config } = await freshComposable()
    expect(config.value).toBeNull()
  })

  it('load fetches and caches the config', async () => {
    getConfig.mockResolvedValue({ data: { retention: '720h0m0s' } })
    const { config, load } = await freshComposable()

    const result = await load()

    expect(result).toEqual({ retention: '720h0m0s' })
    expect(config.value).toEqual({ retention: '720h0m0s' })
    expect(getConfig).toHaveBeenCalledTimes(1)
  })

  it('load returns the cached value without refetching', async () => {
    getConfig.mockResolvedValue({ data: { retention: '168h0m0s' } })
    const { load } = await freshComposable()

    await load()
    await load()

    expect(getConfig).toHaveBeenCalledTimes(1)
  })

  it('deduplicates concurrent first-time loads into a single fetch', async () => {
    getConfig.mockResolvedValue({ data: { retention: '720h0m0s' } })
    const { load } = await freshComposable()

    const [a, b] = await Promise.all([load(), load()])

    expect(a).toEqual({ retention: '720h0m0s' })
    expect(b).toEqual({ retention: '720h0m0s' })
    expect(getConfig).toHaveBeenCalledTimes(1)
  })

  it('retries on the next load after a failed fetch', async () => {
    getConfig.mockRejectedValueOnce(new Error('network')).mockResolvedValueOnce({
      data: { retention: '720h0m0s' },
    })
    const { load } = await freshComposable()

    await expect(load()).rejects.toThrow('network')
    const result = await load()

    expect(result).toEqual({ retention: '720h0m0s' })
    expect(getConfig).toHaveBeenCalledTimes(2)
  })

  it('shares the cache across multiple useAppConfig calls', async () => {
    getConfig.mockResolvedValue({ data: { retention: '720h0m0s' } })
    vi.resetModules()
    const mod = await import('./use-app-config')

    const first = mod.useAppConfig()
    await first.load()
    const second = mod.useAppConfig()

    expect(second.config.value).toEqual({ retention: '720h0m0s' })
  })
})
