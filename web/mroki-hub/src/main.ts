import { createApp } from 'vue'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'
import './style.css'
import App from './App.vue'
import router from './router'

/**
 * Shared TanStack Query client. Defaults are chosen to preserve the hub's
 * current behaviour while enabling caching:
 * - `staleTime: 30_000` — dashboard-style data tolerates brief staleness, so a
 *   30s window avoids redundant refetches when navigating between pages.
 * - `retry: 1` — one retry smooths over transient blips without masking real
 *   failures (matches the existing fetch-once-then-surface-error behaviour).
 * - `refetchOnWindowFocus: false` — pages historically fetched only on mount;
 *   disabling focus refetch keeps that behaviour and avoids surprise reloads.
 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

createApp(App).use(router).use(VueQueryPlugin, { queryClient }).mount('#app')
