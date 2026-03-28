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
      diffContent: [{ op: 'replace', path: '/id', value: 2 }],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Request info card
    await expect(page.getByRole('heading', { name: 'Request Detail' })).toBeVisible()
    await expect(page.getByText('POST')).toBeVisible()
    await expect(page.getByText('/api/detail-test')).toBeVisible()

    // Diff viewer section
    await expect(page.getByText('Response Comparison')).toBeVisible()
    await expect(page.getByText('Live Status').first()).toBeVisible()
    await expect(page.getByText('Shadow Status').first()).toBeVisible()
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
    await expect(page.getByText('200').first()).toBeVisible()
    await expect(page.getByText('500').first()).toBeVisible()
  })

  test('back button navigates to gate detail', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://back-live.example.com',
      'https://back-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, { method: 'GET', path: '/api/back-test' })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)
    await page.getByText('Back to Gate').click()
    await expect(page).toHaveURL(`/gates/${gate.id}`)
  })
})
