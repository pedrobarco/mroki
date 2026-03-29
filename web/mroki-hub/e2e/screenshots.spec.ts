import { test, expect } from './fixtures'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const ASSETS_DIR = path.resolve(__dirname, '../../../docs/assets')
const VIEWPORT = { width: 1280, height: 800 }

// ── Showcase gate: httpbin.org v1 → canary ──────────────────────────
// A single gate comparing a stable API against its canary deployment.
// The diff viewer shows what changed between the two versions.

const LIVE_URL = 'https://httpbin.org/anything'
const SHADOW_URL = 'https://httpbin-canary.org/anything'

// A realistic httpbin-style response — the canary introduces a few changes
const liveBody = {
  args: { page: '1', limit: '20' },
  headers: {
    Accept: 'application/json',
    'Content-Type': 'application/json',
    Host: 'httpbin.org',
    'User-Agent': 'mroki-agent/1.0',
    'X-Request-Id': 'req-7f3a2b',
  },
  method: 'POST',
  origin: '203.0.113.42',
  url: 'https://httpbin.org/anything/v1/orders',
  json: {
    customer: { id: 'cust_9k3m', name: 'Alice Johnson', tier: 'premium' },
    items: [
      { sku: 'WIDGET-001', qty: 2, unit_price: 29.99 },
      { sku: 'GADGET-042', qty: 1, unit_price: 149.0 },
    ],
    totals: { subtotal: 208.98, tax: 18.81, total: 227.79, currency: 'USD' },
    shipping: { method: 'express', estimated_days: 2, tracking_id: null },
  },
}

const shadowBody = {
  args: { page: '1', limit: '20' },
  headers: {
    Accept: 'application/json',
    'Content-Type': 'application/json',
    Host: 'httpbin-canary.org',
    'User-Agent': 'mroki-agent/1.0',
    'X-Request-Id': 'req-7f3a2b',
  },
  method: 'POST',
  origin: '203.0.113.42',
  url: 'https://httpbin-canary.org/anything/v1/orders',
  json: {
    customer: { id: 'cust_9k3m', name: 'Alice Johnson', tier: 'premium' },
    items: [
      { sku: 'WIDGET-001', qty: 2, unit_price: 29.99 },
      { sku: 'GADGET-042', qty: 1, unit_price: 149.0 },
      { sku: 'PROMO-100', qty: 1, unit_price: 0.0 },
    ],
    totals: { subtotal: 208.98, discount: 15.0, tax: 17.46, total: 211.44, currency: 'USD' },
    shipping: { method: 'express', estimated_days: 2, tracking_id: null },
  },
}

const diffOps = [
  { op: 'replace', path: '/body/headers/Host', value: 'httpbin-canary.org' },
  { op: 'replace', path: '/body/url', value: 'https://httpbin-canary.org/anything/v1/orders' },
  { op: 'add', path: '/body/json/items/2', value: { sku: 'PROMO-100', qty: 1, unit_price: 0.0 } },
  { op: 'add', path: '/body/json/totals/discount', value: 15.0 },
  { op: 'replace', path: '/body/json/totals/tax', value: 17.46 },
  { op: 'replace', path: '/body/json/totals/total', value: 211.44 },
]

// ── Helpers ──────────────────────────────────────────────────────────

/** Seed the showcase gate with a handful of varied requests. */
async function seedShowcaseGate(api: import('./fixtures').ApiHelper) {
  const gate = await api.createGate('showcase-gate', LIVE_URL, SHADOW_URL)

  const reqs = [
    { method: 'POST', path: '/anything/v1/orders', live: liveBody, shadow: shadowBody, ops: diffOps },
    { method: 'GET', path: '/anything/v1/orders/ord_4a8f', live: liveBody, shadow: liveBody, ops: [] },
    { method: 'PUT', path: '/anything/v1/orders/ord_4a8f/confirm', live: liveBody, shadow: shadowBody, ops: diffOps },
    { method: 'GET', path: '/anything/v1/products?page=1', live: liveBody, shadow: liveBody, ops: [] },
    { method: 'DELETE', path: '/anything/v1/orders/ord_9z2x', live: liveBody, shadow: liveBody, ops: [] },
  ]

  const seeded = []
  for (const r of reqs) {
    seeded.push(
      await api.seedRequest(gate.id, {
        method: r.method,
        path: r.path,
        liveBody: btoa(JSON.stringify(r.live)),
        shadowBody: btoa(JSON.stringify(r.shadow)),
        liveStatus: 200,
        shadowStatus: 200,
        diffContent: r.ops,
      })
    )
  }
  return { gate, requests: seeded }
}

// ── Screenshot tests ────────────────────────────────────────────────

test.describe('@screenshots', () => {
  test.use({ viewport: VIEWPORT })

  test('hub-gates', async ({ page, api }) => {
    await api.createGate('main-gate', LIVE_URL, SHADOW_URL)
    await api.createGate(
      'users-gate',
      'https://api.acme.io/v1/users',
      'https://api-canary.acme.io/v1/users'
    )
    await api.createGate(
      'payments-gate',
      'https://payments.stripe.dev/v2/charges',
      'https://payments-next.stripe.dev/v2/charges'
    )

    await page.goto('/gates')
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gates.png'), fullPage: true })
  })

  test('hub-gate-detail', async ({ page, api }) => {
    const { gate } = await seedShowcaseGate(api)

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText(gate.id)).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gate-detail.png'), fullPage: true })
  })

  test('hub-request-detail-unified', async ({ page, api }) => {
    const { gate, requests } = await seedShowcaseGate(api)

    await page.goto(`/gates/${gate.id}/requests/${requests[0].id}`)
    await expect(page.getByRole('heading', { name: 'Request Detail' })).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({
      path: path.join(ASSETS_DIR, 'hub-request-detail-unified.png'),
      fullPage: true,
    })
  })

  test('hub-request-detail-split', async ({ page, api }) => {
    const { gate, requests } = await seedShowcaseGate(api)

    await page.goto(`/gates/${gate.id}/requests/${requests[0].id}`)
    await expect(page.getByRole('heading', { name: 'Request Detail' })).toBeVisible()
    await page.waitForTimeout(500)

    const splitBtn = page.getByRole('button', { name: /split/i })
    if (await splitBtn.isVisible()) {
      await splitBtn.click()
      await page.waitForTimeout(300)
    }

    await page.screenshot({
      path: path.join(ASSETS_DIR, 'hub-request-detail-split.png'),
      fullPage: true,
    })
  })
})


