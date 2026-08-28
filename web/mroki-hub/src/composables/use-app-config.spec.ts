import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import type { ApiResponse, AppConfig } from '@/api'
import { useAppConfig } from './use-app-config'

const getConfig = vi.fn<() => Promise<ApiResponse<AppConfig>>>()

vi.mock('@/api/config', () => ({
  getConfig: () => getConfig(),
}))

// A fresh QueryClient per test isolates the cache so each starts empty. Retries
// are disabled so a rejected fetch surfaces immediately.
function makeClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

// Mounts a throwaway component that calls the composable inside setup() (a
// requirement for useQuery) and exposes its return value on the instance.
function mountComposable(queryClient = makeClient()) {
  let api!: ReturnType<typeof useAppConfig>
  const Harness = defineComponent({
    setup() {
      api = useAppConfig()
      return () => null
    },
  })
  const wrapper = mount(Harness, { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  return { api, wrapper, queryClient }
}

beforeEach(() => {
  getConfig.mockReset()
})

describe('useAppConfig', () => {
  it('config is null before the first load resolves', () => {
    getConfig.mockReturnValue(new Promise(() => {}))
    const { api } = mountComposable()
    expect(api.config.value).toBeNull()
  })

  it('fetches and exposes the config once loaded', async () => {
    getConfig.mockResolvedValue({ data: { retention: '720h0m0s' } })
    const { api } = mountComposable()
    await flushPromises()

    expect(api.config.value).toEqual({ retention: '720h0m0s' })
    expect(getConfig).toHaveBeenCalledTimes(1)
  })

  it('does not refetch a session-cached config on remount', async () => {
    getConfig.mockResolvedValue({ data: { retention: '168h0m0s' } })
    const client = makeClient()

    const first = mountComposable(client)
    await flushPromises()
    first.wrapper.unmount()

    // A second consumer sharing the same client reads the cached value with no
    // new network call (staleTime: Infinity).
    const { api } = mountComposable(client)
    await flushPromises()

    expect(api.config.value).toEqual({ retention: '168h0m0s' })
    expect(getConfig).toHaveBeenCalledTimes(1)
  })

  it('deduplicates concurrent first-time consumers into a single fetch', async () => {
    getConfig.mockResolvedValue({ data: { retention: '720h0m0s' } })
    const client = makeClient()

    const a = mountComposable(client)
    const b = mountComposable(client)
    await flushPromises()

    expect(a.api.config.value).toEqual({ retention: '720h0m0s' })
    expect(b.api.config.value).toEqual({ retention: '720h0m0s' })
    expect(getConfig).toHaveBeenCalledTimes(1)
  })

  it('surfaces the error and leaves config null when the fetch fails', async () => {
    getConfig.mockRejectedValue(new Error('network'))
    const { api } = mountComposable()
    await flushPromises()

    expect(api.config.value).toBeNull()
    expect(api.query.isError.value).toBe(true)
  })
})
