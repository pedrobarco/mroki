import { test, expect } from './fixtures'
import { seedScreenshotData } from './seed'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const ASSETS_DIR = path.resolve(__dirname, '../../../docs/assets/screenshots')
const VIEWPORT = { width: 1280, height: 800 }

// ── Screenshot tests ────────────────────────────────────────────────
// All tests share a single seeded dataset. The seed is created lazily
// on the first test that needs it, so dev-reset + screenshots is all
// you need.

let seed: Awaited<ReturnType<typeof seedScreenshotData>> | null = null

test.describe('@screenshots', () => {
  test.use({ viewport: VIEWPORT })
  test.describe.configure({ mode: 'serial' })

  test('hub-gates', async ({ page, api }) => {
    seed = await seedScreenshotData(api)
    await page.goto('/gates')
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gates.png'), fullPage: true })
  })

  test('hub-create-gate', async ({ page }) => {
    await page.goto('/gates')
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()

    // Open the Create Gate dialog
    await page.getByRole('button', { name: /new gate/i }).click()
    await expect(page.getByRole('heading', { name: 'Create New Gate' })).toBeVisible()
    await page.waitForTimeout(300)

    // Fill in realistic values so the screenshot looks complete
    await page.getByLabel('Name').fill('checkout-api')
    await page.getByLabel('Live URL').fill('https://api.acme.io/v1/checkout')
    await page.getByLabel('Shadow URL').fill('https://api-canary.acme.io/v1/checkout')

    await page.screenshot({
      path: path.join(ASSETS_DIR, 'hub-create-gate.png'),
      clip: { x: 0, y: 0, width: VIEWPORT.width, height: VIEWPORT.height },
    })
  })

  test('hub-gate-detail', async ({ page }) => {
    expect(seed).not.toBeNull()
    const { gate } = seed!.ordersGate

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText(gate.id)).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gate-detail.png'), fullPage: true })
  })

  test('hub-gate-config', async ({ page }) => {
    expect(seed).not.toBeNull()
    const { gate } = seed!.ordersGate

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText(gate.id)).toBeVisible()

    // Open the Configure dialog
    await page.getByRole('button', { name: /configure/i }).click()
    await expect(page.getByRole('heading', { name: 'Configure Gate' })).toBeVisible()
    await page.waitForTimeout(300)

    // Fill in diff config fields so the screenshot shows them populated
    await page.getByLabel('Ignored Fields').fill('timestamp, request_id, trace_id')
    await page.getByLabel('Float Tolerance').fill('0.001')

    await page.screenshot({ path: path.join(ASSETS_DIR, 'hub-gate-config.png'), fullPage: true })
  })

  test('hub-request-detail-unified', async ({ page }) => {
    expect(seed).not.toBeNull()
    const { gate, requests } = seed!.ordersGate

    await page.goto(`/gates/${gate.id}/requests/${requests[0].id}`)
    await expect(page.getByRole('heading', { name: 'Request Detail' })).toBeVisible()
    await page.waitForTimeout(500)
    await page.screenshot({
      path: path.join(ASSETS_DIR, 'hub-request-detail-unified.png'),
      fullPage: true,
    })
  })

  test('hub-request-detail-split', async ({ page }) => {
    expect(seed).not.toBeNull()
    const { gate, requests } = seed!.ordersGate

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
