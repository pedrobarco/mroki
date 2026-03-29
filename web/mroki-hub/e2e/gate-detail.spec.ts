import { test, expect } from './fixtures'

test.describe('Gate Detail Page', () => {
  test('displays gate info', async ({ page, api }) => {
    const gate = await api.createGate(
      'detail-gate',
      'https://detail-live.example.com',
      'https://detail-shadow.example.com'
    )

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText('https://detail-live.example.com')).toBeVisible()
    await expect(page.getByText('https://detail-shadow.example.com')).toBeVisible()
    await expect(page.getByText(gate.id)).toBeVisible()
  })

  test('shows empty state when no requests', async ({ page, api }) => {
    const gate = await api.createGate(
      'empty-gate',
      'https://empty-live.example.com',
      'https://empty-shadow.example.com'
    )

    await page.goto(`/gates/${gate.id}`)
    await expect(
      page.getByText('No requests captured yet. Send traffic through this gate')
    ).toBeVisible()
  })

  test('displays seeded requests in table', async ({ page, api }) => {
    const gate = await api.createGate(
      'reqs-gate',
      'https://reqs-live.example.com',
      'https://reqs-shadow.example.com'
    )
    await api.seedRequest(gate.id, { method: 'GET', path: '/api/users' })
    await api.seedRequest(gate.id, { method: 'POST', path: '/api/orders' })

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText('/api/users')).toBeVisible()
    await expect(page.getByText('/api/orders')).toBeVisible()

    // Request total
    await expect(page.getByText('2 requests')).toBeVisible()
  })

  test('filter by HTTP method', async ({ page, api }) => {
    const gate = await api.createGate(
      'filter-gate',
      'https://filter-live.example.com',
      'https://filter-shadow.example.com'
    )
    await api.seedRequest(gate.id, { method: 'GET', path: '/api/filter-get' })
    await api.seedRequest(gate.id, { method: 'POST', path: '/api/filter-post' })

    await page.goto(`/gates/${gate.id}`)

    // Both visible initially
    await expect(page.getByText('/api/filter-get')).toBeVisible()
    await expect(page.getByText('/api/filter-post')).toBeVisible()

    // Click POST method button in filters
    await page.getByRole('button', { name: 'POST' }).click()

    // Only POST visible
    await expect(page.getByText('/api/filter-post')).toBeVisible()
    await expect(page.getByText('/api/filter-get')).not.toBeVisible()
  })

  test('filter by path', async ({ page, api }) => {
    const gate = await api.createGate(
      'pathf-gate',
      'https://pathf-live.example.com',
      'https://pathf-shadow.example.com'
    )
    await api.seedRequest(gate.id, { method: 'GET', path: '/api/alpha' })
    await api.seedRequest(gate.id, { method: 'GET', path: '/api/beta' })

    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText('/api/alpha')).toBeVisible()
    await expect(page.getByText('/api/beta')).toBeVisible()

    // Type in path filter
    await page.getByPlaceholder('Filter by path...').fill('/api/alpha')

    // Wait for debounce + reload
    await expect(page.getByText('/api/alpha')).toBeVisible()
    await expect(page.getByText('/api/beta')).not.toBeVisible()
  })

  test('pagination works with many requests', async ({ page, api }) => {
    const gate = await api.createGate(
      'page-gate',
      'https://page-live.example.com',
      'https://page-shadow.example.com'
    )

    // Seed 25 requests (page size is 20)
    const promises = Array.from({ length: 25 }, (_, i) =>
      api.seedRequest(gate.id, {
        method: 'GET',
        path: `/api/item-${String(i).padStart(3, '0')}`,
        createdAt: new Date(2026, 0, 1, 0, 0, i).toISOString(),
      })
    )
    await Promise.all(promises)

    await page.goto(`/gates/${gate.id}`)

    // Should show request total and pagination info
    await expect(page.getByText('25 requests')).toBeVisible()
    await expect(page.getByText('Page 1 of 2')).toBeVisible()

    // Next page
    await page.getByRole('button', { name: 'Next' }).click()
    await expect(page.getByText('Page 2 of 2')).toBeVisible()

    // Previous page
    await page.getByRole('button', { name: 'Previous' }).click()
    await expect(page.getByText('Page 1 of 2')).toBeVisible()
  })

  test('click request navigates to detail', async ({ page, api }) => {
    const gate = await api.createGate(
      'nav-gate',
      'https://nav-live.example.com',
      'https://nav-shadow.example.com'
    )
    const req = await api.seedRequest(gate.id, { method: 'GET', path: '/api/nav-test' })

    await page.goto(`/gates/${gate.id}`)
    await page.getByText('/api/nav-test').click()
    await expect(page).toHaveURL(`/gates/${gate.id}/requests/${req.id}`)
  })

  test('back button navigates to gates list', async ({ page, api }) => {
    const gate = await api.createGate(
      'back-gate-detail',
      'https://backdet-live.example.com',
      'https://backdet-shadow.example.com'
    )

    await page.goto(`/gates/${gate.id}`)
    await page.getByText('Back to Gates').click()
    await expect(page).toHaveURL(/\/gates$/)
  })
})
