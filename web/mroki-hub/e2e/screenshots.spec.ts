import { test, expect } from './fixtures'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const ASSETS_DIR = path.resolve(__dirname, '../../../docs/assets')
const VIEWPORT = { width: 1280, height: 800 }

test.describe('@screenshots', () => {
  test.use({ viewport: VIEWPORT })

  test('capture gates list', async ({ page, api }) => {
    // Seed realistic gates
    await api.createGate(
      'https://api.acme.io/v2/checkout',
      'https://api-canary.acme.io/v2/checkout'
    )
    await api.createGate(
      'https://users.acme.io/api/v1/profile',
      'https://users-next.acme.io/api/v1/profile'
    )
    await api.createGate(
      'https://orders.acme.io/api/v3/fulfillment',
      'https://orders-exp.acme.io/api/v3/fulfillment'
    )

    await page.goto('/gates')
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gates.png') })
  })

  test('capture gate detail', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://api.acme.io/v2/checkout',
      'https://api-canary.acme.io/v2/checkout'
    )

    const requests = [
      { method: 'POST', path: '/v2/checkout/sessions' },
      { method: 'GET', path: '/v2/checkout/sessions/cs_12345' },
      { method: 'PUT', path: '/v2/checkout/sessions/cs_12345/confirm' },
      { method: 'GET', path: '/v2/checkout/prices' },
      { method: 'DELETE', path: '/v2/checkout/sessions/cs_98765' },
      { method: 'POST', path: '/v2/checkout/webhooks' },
      { method: 'GET', path: '/v2/checkout/config' },
    ]
    for (const r of requests) {
      await api.seedRequest(gate.id, {
        method: r.method,
        path: r.path,
        liveBody: btoa(JSON.stringify({ status: 'ok', service: 'live' })),
        shadowBody: btoa(JSON.stringify({ status: 'ok', service: 'shadow' })),
      })
    }

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText(gate.id)).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gate-detail.png') })
  })

  test('capture request detail with diff', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://api.acme.io/v2/checkout',
      'https://api-canary.acme.io/v2/checkout'
    )

    const liveResponse = {
      user: { id: 42, name: 'Alice Johnson', email: 'alice@acme.io' },
      cart: {
        items: [
          { sku: 'WIDGET-001', qty: 2, price: 29.99 },
          { sku: 'GADGET-042', qty: 1, price: 149.0 },
        ],
        subtotal: 208.98,
        currency: 'USD',
      },
      checkout: { session_id: 'cs_12345', status: 'pending', created_at: '2026-03-28T10:30:00Z' },
    }

    const shadowResponse = {
      user: { id: 42, name: 'Alice Johnson', email: 'alice@acme.io' },
      cart: {
        items: [
          { sku: 'WIDGET-001', qty: 2, price: 29.99 },
          { sku: 'GADGET-042', qty: 1, price: 149.0 },
          { sku: 'PROMO-100', qty: 1, price: 0.0 },
        ],
        subtotal: 208.98,
        discount: 15.0,
        total: 193.98,
        currency: 'USD',
      },
      checkout: { session_id: 'cs_12345', status: 'ready', created_at: '2026-03-28T10:30:00Z' },
    }

    const diffContent = [
      { op: 'add', path: '/cart/items/2', value: { sku: 'PROMO-100', qty: 1, price: 0.0 } },
      { op: 'add', path: '/cart/discount', value: 15.0 },
      { op: 'add', path: '/cart/total', value: 193.98 },
      { op: 'replace', path: '/checkout/status', value: 'ready' },
    ]

    const req = await api.seedRequest(gate.id, {
      method: 'POST',
      path: '/v2/checkout/sessions',
      liveBody: btoa(JSON.stringify(liveResponse)),
      shadowBody: btoa(JSON.stringify(shadowResponse)),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent,
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)
    await expect(page.getByRole('heading', { name: 'Request Detail' })).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({
      path: path.join(ASSETS_DIR, 'hub-request-detail.png'),
      fullPage: true,
    })
  })
})
