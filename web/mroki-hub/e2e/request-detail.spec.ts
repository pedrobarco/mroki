import { test, expect } from './fixtures'

test.describe('Request Detail Page', () => {
  test('displays request info and diff viewer', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://reqdetail-live.example.com',
      'https://reqdetail-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, {
      method: 'POST',
      path: '/api/detail-test',
      liveBody: btoa(JSON.stringify({ user: 'alice', id: 1 })),
      shadowBody: btoa(JSON.stringify({ user: 'alice', id: 2 })),
      liveStatus: 200,
      shadowStatus: 201,
      diffContent: 'id: 1 != 2',
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Request info card
    await expect(page.getByRole('heading', { name: 'Request Detail' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Request Information' })).toBeVisible()
    await expect(page.getByText('POST')).toBeVisible()
    await expect(page.getByText('/api/detail-test')).toBeVisible()
    await expect(page.getByText(req.id)).toBeVisible()

    // Diff viewer sections
    await expect(page.getByRole('heading', { name: 'Response Comparison' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Live Response' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Shadow Response' })).toBeVisible()
  })

  test('shows live and shadow status codes', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://status-live.example.com',
      'https://status-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/status-test',
      liveBody: btoa('{"ok":true}'),
      shadowBody: btoa('{"ok":false}'),
      liveStatus: 200,
      shadowStatus: 500,
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Status codes should be visible in the diff viewer
    await expect(page.getByText('200')).toBeVisible()
    await expect(page.getByText('500')).toBeVisible()
  })

  test('back button navigates to gate detail', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://back-live.example.com',
      'https://back-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, { method: 'GET', path: '/api/back-test' })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)
    await page.getByRole('button', { name: '← Back to Gate' }).click()
    await expect(page).toHaveURL(`/gates/${gate.id}`)
  })
})
