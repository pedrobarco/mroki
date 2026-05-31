import { test, expect } from './fixtures'

test.describe('Request Detail Page', () => {
  test('displays request info and diff viewer', async ({ page, api }) => {
    const gate = await api.createGate(
      'reqdetail-gate',
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
      'status-gate',
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
      'back-gate',
      'https://back-live.example.com',
      'https://back-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, { method: 'GET', path: '/api/back-test' })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)
    await page.getByText('Back to Gate').click()
    await expect(page).toHaveURL(`/gates/${gate.id}`)
  })

  test('copy cURL dropdown shows live and shadow options', async ({ page, api, context }) => {
    // Grant clipboard permissions
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])

    const suffix = Date.now()
    const gate = await api.createGate(
      `curl-gate-${suffix}`,
      `https://curl-live-${suffix}.example.com`,
      `https://curl-shadow-${suffix}.example.com`
    )
    const req = await api.seedRequest(gate.id, {
      method: 'POST',
      path: '/api/curl-test',
      liveBody: btoa(JSON.stringify({ test: true })),
      shadowBody: btoa(JSON.stringify({ test: false })),
      liveStatus: 200,
      shadowStatus: 200,
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Click Copy cURL dropdown trigger
    await page.getByRole('button', { name: 'Copy cURL' }).click()

    // Verify dropdown options
    await expect(page.getByText('Live endpoint')).toBeVisible()
    await expect(page.getByText('Shadow endpoint')).toBeVisible()

    // Click live endpoint option
    await page.getByText('Live endpoint').click()

    // Button should show "Copied!" feedback
    await expect(page.getByText('Copied!')).toBeVisible()

    // Verify clipboard contains cURL with live URL
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText())
    expect(clipboardText).toContain(`curl -X POST`)
    expect(clipboardText).toContain(`curl-live-${suffix}.example.com/api/curl-test`)
  })

  test('displays query parameters in request header', async ({ page, api }) => {
    const gate = await api.createGate(
      'qp-detail-gate',
      'https://qp-live.example.com',
      'https://qp-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/search',
      rawQuery: 'q=hello&page=1',
      liveBody: btoa('{"results":[]}'),
      shadowBody: btoa('{"results":[]}'),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Path and query params should be visible in the header (in separate elements)
    await expect(page.getByText('/api/search')).toBeVisible()
    await expect(page.getByText('?q=hello&page=1')).toBeVisible()
  })

  test('copy cURL includes query parameters', async ({ page, api, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])

    const suffix = Date.now()
    const gate = await api.createGate(
      `curl-qp-gate-${suffix}`,
      `https://curl-qp-live-${suffix}.example.com`,
      `https://curl-qp-shadow-${suffix}.example.com`
    )
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/items',
      rawQuery: 'category=books&limit=25',
      liveBody: btoa('{"items":[]}'),
      shadowBody: btoa('{"items":[]}'),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    await page.getByRole('button', { name: 'Copy cURL' }).click()
    await page.getByText('Live endpoint').click()
    await expect(page.getByText('Copied!')).toBeVisible()

    const clipboardText = await page.evaluate(() => navigator.clipboard.readText())
    expect(clipboardText).toContain('/api/items?category=books&limit=25')
    expect(clipboardText).toContain(`curl-qp-live-${suffix}.example.com`)
  })

  test('export JSON includes query parameters', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `export-qp-gate-${suffix}`,
      `https://export-qp-live-${suffix}.example.com`,
      `https://export-qp-shadow-${suffix}.example.com`
    )
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/export-qp',
      rawQuery: 'format=csv&fields=name,email',
      liveBody: btoa('{}'),
      shadowBody: btoa('{}'),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: 'Export JSON' }).click()
    const download = await downloadPromise

    const filePath = await download.path()
    expect(filePath).toBeTruthy()

    const fs = await import('fs')
    const content = JSON.parse(fs.readFileSync(filePath!, 'utf-8'))
    expect(content.path).toBe('/api/export-qp')
    expect(content.raw_query).toBe('format=csv&fields=name,email')
  })

  test('config snapshot reflects non-default settings', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `sort-badge-gate-${suffix}`,
      `https://sort-badge-live-${suffix}.example.com`,
      `https://sort-badge-shadow-${suffix}.example.com`
    )

    // Enable sort_arrays on the gate
    await api.updateGate(gate.id, {
      diff_config: {
        ignored_fields: [],
        included_fields: [],
        float_tolerance: 0,
        sort_arrays: true,
      },
    })

    // Seed a request — the diff config snapshot will include sort_arrays: true
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/sorted-test',
      liveBody: btoa(JSON.stringify({ items: ['b', 'a', 'c'] })),
      shadowBody: btoa(JSON.stringify({ items: ['b', 'a', 'c'] })),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // The config snapshot icon is always available; opening it reveals the
    // captured settings, with the non-default Sort arrays = On.
    await page.getByRole('button', { name: 'Diff configuration' }).click()
    await expect(page.getByText('Snapshot used to compute this diff')).toBeVisible()
    await expect(page.getByText('Sort arrays')).toBeVisible()
    await expect(page.getByText('On', { exact: true })).toBeVisible()
  })

  test('config snapshot shows defaults when config is all-default', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `no-sort-badge-gate-${suffix}`,
      `https://no-sort-live-${suffix}.example.com`,
      `https://no-sort-shadow-${suffix}.example.com`
    )

    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/unsorted-test',
      liveBody: btoa(JSON.stringify({ items: ['b', 'a', 'c'] })),
      shadowBody: btoa(JSON.stringify({ items: ['b', 'a', 'c'] })),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // The config snapshot icon is always available, even with all-default
    // config; opening it shows the default values.
    await page.getByRole('button', { name: 'Diff configuration' }).click()
    await expect(page.getByText('Snapshot used to compute this diff')).toBeVisible()
    await expect(page.getByText('Off', { exact: true })).toBeVisible()
    await expect(page.getByText('Exact', { exact: true })).toBeVisible()
  })

  test('export JSON downloads request data', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `export-gate-${suffix}`,
      `https://export-live-${suffix}.example.com`,
      `https://export-shadow-${suffix}.example.com`
    )
    const req = await api.seedRequest(gate.id, {
      method: 'POST',
      path: '/api/export-test',
      liveBody: btoa(JSON.stringify({ status: 'ok' })),
      shadowBody: btoa(JSON.stringify({ status: 'error' })),
      liveStatus: 200,
      shadowStatus: 500,
      diffContent: [{ op: 'replace', path: '/status', value: 'error' }],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Listen for the download event
    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: 'Export JSON' }).click()
    const download = await downloadPromise

    // Verify filename
    expect(download.suggestedFilename()).toMatch(/^request-.*\.json$/)

    // Read downloaded file and verify content
    const filePath = await download.path()
    expect(filePath).toBeTruthy()

    const fs = await import('fs')
    const content = JSON.parse(fs.readFileSync(filePath!, 'utf-8'))
    expect(content.id).toBe(req.id)
    expect(content.method).toBe('POST')
    expect(content.path).toBe('/api/export-test')
    expect(content.live_response.status_code).toBe(200)
    expect(content.shadow_response.status_code).toBe(500)
    expect(content.diff.content).toHaveLength(1)
  })
})
