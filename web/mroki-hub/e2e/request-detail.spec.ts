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

    // Configure every diff setting away from its default so the snapshot
    // exercises each branch of the popover (sort arrays, float tolerance,
    // ignored fields, included fields).
    await api.updateGate(gate.id, {
      diff_config: {
        ignored_fields: ['timestamp', 'trace_id'],
        included_fields: ['body.user'],
        float_tolerance: 0.01,
        sort_arrays: true,
      },
    })

    // Seed a request — the diff config snapshot captures the non-default config.
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
    // captured settings with every non-default value surfaced. Scope the
    // assertions to the popover so field names don't collide with labels
    // elsewhere on the page (e.g. the "Timestamp" request metadata field).
    await page.getByRole('button', { name: 'Diff configuration' }).click()
    const popover = page.locator('[data-slot="popover-content"]')
    await expect(popover.getByText('Snapshot used to compute this diff')).toBeVisible()
    await expect(popover.getByText('Sort arrays')).toBeVisible()
    await expect(popover.getByText('On', { exact: true })).toBeVisible()
    await expect(popover.getByText('±0.01')).toBeVisible()
    await expect(popover.getByText('timestamp')).toBeVisible()
    await expect(popover.getByText('trace_id')).toBeVisible()
    await expect(popover.getByText('body.user')).toBeVisible()
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
    const popover = page.locator('[data-slot="popover-content"]')
    await expect(popover.getByText('Snapshot used to compute this diff')).toBeVisible()
    await expect(popover.getByText('Off', { exact: true })).toBeVisible()
    await expect(popover.getByText('Exact', { exact: true })).toBeVisible()
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

  test('patch view lists operations and filters by op type', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `patch-gate-${suffix}`,
      `https://patch-live-${suffix}.example.com`,
      `https://patch-shadow-${suffix}.example.com`
    )
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/patch-test',
      liveBody: btoa(JSON.stringify({ status: 'processing', legacy_token: 'tok_old' })),
      shadowBody: btoa(JSON.stringify({ status: 'completed', coupon: 'SAVE10' })),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [
        { op: 'replace', path: '/body/status', value: 'completed' },
        { op: 'add', path: '/body/coupon', value: 'SAVE10' },
        { op: 'remove', path: '/body/legacy_token' },
      ],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)

    // Switch to the Patch view
    await page.getByRole('button', { name: 'Patch' }).click()
    await expect(page.getByText(/3 shown/)).toBeVisible()

    // All three operations render with their leaf paths and values
    await expect(page.getByText('coupon')).toBeVisible()
    await expect(page.getByText('legacy_token')).toBeVisible()
    await expect(page.getByText('completed')).toBeVisible()
    await expect(page.getByText('"processing"')).toBeVisible() // struck-through old value
    await expect(page.getByText('"tok_old"')).toBeVisible()

    // Filtering by Removed shows only the remove op
    await page.getByRole('button', { name: /Removed/ }).click()
    await expect(page.getByText(/1 shown/)).toBeVisible()
    await expect(page.getByText('legacy_token')).toBeVisible()
    await expect(page.getByText('coupon')).not.toBeVisible()
    await expect(page.getByText('completed')).not.toBeVisible()

    // Filtering by Added shows only the add op
    await page.getByRole('button', { name: /Added/ }).click()
    await expect(page.getByText('coupon')).toBeVisible()
    await expect(page.getByText('legacy_token')).not.toBeVisible()
  })

  // Regression for #114: an array-of-objects reorder must render valid array
  // indices in the Patch view (never the pre-fix [-1] sentinel), and the moved
  // object must show one key order on both its add and remove rows even though
  // the add value comes from the diff (Go's alphabetical json.Marshal) while the
  // removed old value is reconstructed from the live body (Postgres JSONB, which
  // reorders object keys by length then byte order).
  test('renders array reorder with valid indices and canonical key order', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `reorder-gate-${suffix}`,
      `https://reorder-live-${suffix}.example.com`,
      `https://reorder-shadow-${suffix}.example.com`
    )
    const moved = { name: 'USB Cable', qty: 3, sku: 'CABLE-009' }
    const widget = { name: 'Standard Widget', qty: 2, sku: 'WIDGET-001' }
    const gadget = { name: 'Pro Gadget', qty: 1, sku: 'GADGET-042' }
    const req = await api.seedRequest(gate.id, {
      method: 'GET',
      path: '/api/reorder-test',
      // Live keys are sent in a non-alphabetical order so the JSONB round-trip
      // differs from the diff's Go-sorted add value.
      liveBody: btoa(
        JSON.stringify({
          items: [
            { qty: 2, sku: 'WIDGET-001', name: 'Standard Widget' },
            { qty: 1, sku: 'GADGET-042', name: 'Pro Gadget' },
            { qty: 3, sku: 'CABLE-009', name: 'USB Cable' },
          ],
        })
      ),
      shadowBody: btoa(JSON.stringify({ items: [moved, widget, gadget] })),
      liveStatus: 200,
      shadowStatus: 200,
      diffContent: [
        { op: 'add', path: '/body/items/0', value: moved },
        { op: 'remove', path: '/body/items/2' },
      ],
    })

    await page.goto(`/gates/${gate.id}/requests/${req.id}`)
    await page.getByRole('button', { name: 'Patch' }).click()
    await expect(page.getByText(/2 shown/)).toBeVisible()

    // Valid array indices on both rows; the broken /-1 index never renders.
    await expect(page.getByText('[0]')).toBeVisible()
    await expect(page.getByText('[2]')).toBeVisible()
    await expect(page.getByText('[-1]')).toHaveCount(0)

    // The add row (new value from the diff) and the remove row (old value
    // reconstructed from the live body) serialise the moved object with the same
    // canonical key order, so the value div title matches on both rows.
    const movedTitle = JSON.stringify(moved)
    await expect(page.locator(`[title='${movedTitle}']`)).toHaveCount(2)
  })
})
