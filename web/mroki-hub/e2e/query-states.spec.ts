import { test, expect } from './fixtures'
import type { Page } from '@playwright/test'

// Coverage for the unified TanStack Query loading/error/empty states (#142).
// The `api` fixture seeds data through Node-side fetch (never intercepted),
// while page.route() intercepts the SPA's browser fetches so we can force the
// error and loading branches deterministically.
//
// Interception is scoped to the API origin (not just the pathname): the SPA
// document is served from :5173, so matching on pathname alone would also
// hijack the page navigation. A 500 is served *persistently* (not just once)
// because the app's QueryClient retries once — a single failure would silently
// recover — and a toggle lets us re-enable the real backend before Retry.

const API_ORIGIN = 'http://localhost:8090'

// Unique per test-process so re-runs against a persistent dev database never
// collide on the gate name uniqueness constraint.
const RUN = Math.random().toString(36).slice(2, 8)
function gateNames(base: string) {
  const name = `${base}-${RUN}`
  return {
    name,
    live: `https://${name}-live.example.com`,
    shadow: `https://${name}-shadow.example.com`,
  }
}

const RFC7807_500 = {
  status: 500,
  contentType: 'application/json',
  body: JSON.stringify({
    type: 'about:blank',
    title: 'Internal Server Error',
    status: 500,
    detail: 'internal error',
  }),
}

// Fail every matching API request until `active` is flipped off, then pass
// through to the real backend so a Retry can recover.
async function fail500(page: Page, pathname: string) {
  const state = { active: true }
  await page.route(
    (u) => u.origin === API_ORIGIN && u.pathname === pathname,
    async (route) => {
      if (state.active) await route.fulfill(RFC7807_500)
      else await route.continue()
    }
  )
  return state
}

test.describe('Query loading, error, and empty states', () => {
  test('gates list shows the error state and recovers on Retry', async ({ page, api }) => {
    const g = gateNames('qs-gates-err')
    await api.createGate(g.name, g.live, g.shadow)
    const route = await fail500(page, '/gates')

    await page.goto('/gates')

    // Unified error branch: RFC 7807 detail + a Retry action.
    await expect(page.getByText('internal error')).toBeVisible()
    const retry = page.getByRole('button', { name: 'Retry' })
    await expect(retry).toBeVisible()

    // Recover: let the refetch reach the backend, then confirm the list renders.
    route.active = false
    await retry.click()
    await expect(page.getByText('internal error')).not.toBeVisible()
    await page.getByPlaceholder('Search gates by live URL...').fill(g.name)
    await expect(page.getByText(g.name).first()).toBeVisible()
  })

  test('gates list shows the loading state while the first page is in flight', async ({ page }) => {
    // Hold the API fetch (only) open long enough to observe the loading branch,
    // then pass it through to the real backend.
    await page.route(
      (u) => u.origin === API_ORIGIN && u.pathname === '/gates',
      async (route) => {
        await new Promise((r) => setTimeout(r, 1200))
        await route.continue()
      }
    )

    await page.goto('/gates')
    await expect(page.getByText('Loading gates...')).toBeVisible()
    await expect(page.getByText('Loading gates...')).toBeHidden({ timeout: 10000 })
  })

  test('gate detail shows the error state and recovers on Retry', async ({ page, api }) => {
    const g = gateNames('qs-gate-err')
    const gate = await api.createGate(g.name, g.live, g.shadow)
    const route = await fail500(page, `/gates/${gate.id}`)

    await page.goto(`/gates/${gate.id}`)

    await expect(page.getByText('internal error')).toBeVisible()
    const retry = page.getByRole('button', { name: 'Retry' })
    await expect(retry).toBeVisible()

    route.active = false
    await retry.click()
    await expect(page.getByText('internal error')).not.toBeVisible()
    await expect(page.getByText(g.live)).toBeVisible()
  })

  test('request list shows the error state and recovers on Retry', async ({ page, api }) => {
    const g = gateNames('qs-reqs-err')
    const gate = await api.createGate(g.name, g.live, g.shadow)
    // The gate detail loads normally; only the nested request list fetch fails.
    const route = await fail500(page, `/gates/${gate.id}/requests`)

    await page.goto(`/gates/${gate.id}`)

    await expect(page.getByText('internal error')).toBeVisible()
    const retry = page.getByRole('button', { name: 'Retry' })
    await expect(retry).toBeVisible()

    // Retry recovers to the empty state (this gate has no captured requests).
    route.active = false
    await retry.click()
    await expect(page.getByText('internal error')).not.toBeVisible()
    await expect(page.getByText('No requests captured yet')).toBeVisible()
  })

  test('request list shows the empty state when a gate has no requests', async ({ page, api }) => {
    const g = gateNames('qs-reqs-empty')
    const gate = await api.createGate(g.name, g.live, g.shadow)

    await page.goto(`/gates/${gate.id}`)
    await expect(
      page.getByText('No requests captured yet. Send traffic through this gate')
    ).toBeVisible()
  })

  test('request detail shows the error state when the request fails to load', async ({
    page,
    api,
  }) => {
    const g = gateNames('qs-reqdetail-err')
    const gate = await api.createGate(g.name, g.live, g.shadow)
    const req = await api.seedRequest(gate.id, { method: 'GET', path: '/api/qs-detail' })
    await fail500(page, `/gates/${gate.id}/requests/${req.id}`)

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Request detail surfaces the same unified error alert (no Retry affordance).
    await expect(page.getByText('internal error')).toBeVisible()
  })
})
