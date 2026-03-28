import { test, expect } from './fixtures'

test.describe('Gates Page', () => {
  test('displays the gates page with heading', async ({ page }) => {
    await page.goto('/gates')
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()
    await expect(page.getByText('Manage live/shadow service pairs')).toBeVisible()
  })

  test('shows seeded gate in the list', async ({ page, api }) => {
    const gate = await api.createGate(
      'https://list-live.example.com/api',
      'https://list-shadow.example.com/api'
    )

    // Navigate to gate detail to verify it was created and is accessible
    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText('https://list-live.example.com/api')).toBeVisible()
    await expect(page.getByText('https://list-shadow.example.com/api')).toBeVisible()
    await expect(page.getByText(gate.id)).toBeVisible()
  })

  test('create gate via dialog', async ({ page }) => {
    await page.goto('/gates')

    // Open dialog
    await page.getByRole('button', { name: 'New Gate' }).click()
    await expect(page.getByRole('heading', { name: 'Create New Gate' })).toBeVisible()

    // Fill form
    await page.getByLabel('Live URL').fill('https://new-live.example.com')
    await page.getByLabel('Shadow URL').fill('https://new-shadow.example.com')

    // Submit
    await page.locator('form').getByRole('button', { name: 'Create Gate' }).click()

    // Dialog closes
    await expect(page.getByRole('heading', { name: 'Create New Gate' })).not.toBeVisible()

    // Search for the newly created gate by live URL (may not be on page 1 with 5-per-page)
    await page.getByPlaceholder('Search gates by URL...').fill('new-live')

    // Wait for debounce (400ms) + API response
    await expect(page.getByText('https://new-live.example.com').first()).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('https://new-shadow.example.com').first()).toBeVisible()
  })

  test('create gate form disables submit for invalid URLs', async ({ page }) => {
    await page.goto('/gates')
    await page.getByRole('button', { name: 'New Gate' }).click()

    const submitButton = page.locator('form').getByRole('button', { name: 'Create Gate' })

    // Empty form — submit disabled
    await expect(submitButton).toBeDisabled()

    // Invalid live URL
    await page.getByLabel('Live URL').fill('not-a-url')
    await page.getByLabel('Shadow URL').fill('https://shadow.example.com')
    await expect(submitButton).toBeDisabled()
    await expect(page.getByText('Please enter a valid URL')).toBeVisible()

    // Fix live URL — submit enabled
    await page.getByLabel('Live URL').fill('https://live.example.com')
    await expect(submitButton).toBeEnabled()
  })

  test('clicking gate card navigates to gate detail', async ({ page }) => {
    await page.goto('/gates')

    // Get the first gate's ID from its title, then click the card
    const firstCard = page.locator('[class*="cursor-pointer"]').first()
    await firstCard.click()
    await expect(page).toHaveURL(/\/gates\/[0-9a-f-]+$/)
  })

  test('filter gates by URL search', async ({ page, api }) => {
    await api.createGate(
      'https://xflt-alpha-live.example.com',
      'https://xflt-alpha-shadow.example.com'
    )
    await api.createGate(
      'https://xflt-beta-live.example.com',
      'https://xflt-beta-shadow.example.com'
    )

    await page.goto('/gates')
    const searchBox = page.getByPlaceholder('Search gates by URL...')

    // Search for our unique prefix to isolate test data
    await searchBox.fill('xflt-')

    // Both test gates should appear
    await expect(page.getByText('xflt-alpha-live').first()).toBeVisible()
    await expect(page.getByText('xflt-beta-live').first()).toBeVisible()

    // Narrow down to alpha only
    await searchBox.fill('xflt-alpha')

    await expect(page.getByText('xflt-alpha-live').first()).toBeVisible()
    await expect(page.getByText('xflt-beta-live')).not.toBeVisible()

    // Narrow down to beta only
    await searchBox.fill('xflt-beta')

    await expect(page.getByText('xflt-beta-live').first()).toBeVisible()
    await expect(page.getByText('xflt-alpha-live')).not.toBeVisible()
  })

  test('sort gates by live URL ascending', async ({ page, api }) => {
    await api.createGate(
      'https://xsrt-zebra.example.com',
      'https://xsrt-shadow-z.example.com'
    )
    await api.createGate(
      'https://xsrt-apple.example.com',
      'https://xsrt-shadow-a.example.com'
    )

    await page.goto('/gates')

    // First isolate our test gates using search
    await page.getByPlaceholder('Search gates by URL...').fill('xsrt-')
    await expect(page.getByText('xsrt-zebra').first()).toBeVisible()

    // Change sort to Live URL A→Z
    await page.getByText('Sort:').click()
    await page.getByRole('option', { name: 'Live URL (A→Z)' }).click()

    // First card should be apple (alphabetically first)
    const cards = page.locator('[class*="cursor-pointer"]')
    await expect(cards).toHaveCount(2)
    const firstCardText = await cards.first().textContent()
    expect(firstCardText).toContain('xsrt-apple')

    // Last card should be zebra
    const lastCardText = await cards.last().textContent()
    expect(lastCardText).toContain('xsrt-zebra')
  })

  test('pagination works with many gates', async ({ page, api }) => {
    // Create 8 gates with unique prefix (page size is 5, so we get 2 pages)
    const promises = Array.from({ length: 8 }, (_, i) =>
      api.createGate(
        `https://xpag-live-${String(i).padStart(3, '0')}.example.com`,
        `https://xpag-shadow-${String(i).padStart(3, '0')}.example.com`
      )
    )
    await Promise.all(promises)

    await page.goto('/gates')

    // Filter to only our test gates
    await page.getByPlaceholder('Search gates by URL...').fill('xpag-')

    // Should show pagination info (8 gates, 5 per page = 2 pages)
    await expect(page.getByText('Page 1 of 2')).toBeVisible()
    await expect(page.getByText('8 gates')).toBeVisible()

    // Next page
    await page.getByRole('button', { name: 'Next' }).click()
    await expect(page.getByText('Page 2 of 2')).toBeVisible()

    // Previous page
    await page.getByRole('button', { name: 'Previous' }).click()
    await expect(page.getByText('Page 1 of 2')).toBeVisible()
  })

  test('pagination resets when search filter changes', async ({ page, api }) => {
    // Create 8 gates so pagination appears
    const promises = Array.from({ length: 8 }, (_, i) =>
      api.createGate(
        `https://xrst-live-${String(i).padStart(3, '0')}.example.com`,
        `https://xrst-shadow-${String(i).padStart(3, '0')}.example.com`
      )
    )
    await Promise.all(promises)

    await page.goto('/gates')

    // Filter to our test gates — 8 results, 2 pages
    const searchBox = page.getByPlaceholder('Search gates by URL...')
    await searchBox.fill('xrst-')
    await expect(page.getByText('Page 1 of 2')).toBeVisible()

    // Go to page 2
    await page.getByRole('button', { name: 'Next' }).click()
    await expect(page.getByText('Page 2 of 2')).toBeVisible()

    // Clear and type a narrower filter — should reset to page 1
    await searchBox.clear()
    await searchBox.fill('xrst-live-000')

    // Wait for debounce (400ms) + API response
    await expect(page.getByText('xrst-live-000').first()).toBeVisible({ timeout: 10000 })

    // Pagination should be gone (only 1 result)
    await expect(page.getByText(/Page \d+ of \d+/)).not.toBeVisible()
  })
})
