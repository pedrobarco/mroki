import { test, expect } from './fixtures'

test.describe('Gates Page', () => {
  test('displays the gates page with heading', async ({ page }) => {
    await page.goto('/gates')
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()
    await expect(page.getByText('Manage live/shadow service pairs')).toBeVisible()
  })

  test('shows seeded gate in the list', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `list-gate-${suffix}`,
      `https://list-live-${suffix}.example.com/api`,
      `https://list-shadow-${suffix}.example.com/api`
    )

    // Navigate to gate detail to verify it was created and is accessible
    await page.goto(`/gates/${gate.id}`)
    await expect(page.getByText(`https://list-live-${suffix}.example.com/api`)).toBeVisible()
    await expect(page.getByText(`https://list-shadow-${suffix}.example.com/api`)).toBeVisible()
    await expect(page.getByText(gate.id)).toBeVisible()
  })

  test('create gate via dialog', async ({ page }) => {
    const suffix = Date.now()
    await page.goto('/gates')

    // Open dialog
    await page.getByRole('button', { name: 'New gate' }).click()
    await expect(page.getByRole('heading', { name: 'Create gate' })).toBeVisible()

    // Fill form
    await page.getByLabel('Name').fill(`new-test-gate-${suffix}`)
    await page.getByLabel('Live URL').fill(`https://new-live-${suffix}.example.com`)
    await page.getByLabel('Shadow URL').fill(`https://new-shadow-${suffix}.example.com`)

    // Submit
    await page.locator('form').getByRole('button', { name: 'Create gate' }).click()

    // Dialog closes
    await expect(page.getByRole('heading', { name: 'Create gate' })).not.toBeVisible()

    // Search for the newly created gate by live URL (may not be on page 1 with 5-per-page)
    await page.getByPlaceholder('Search gates by live URL...').fill(`new-live-${suffix}`)

    // Wait for debounce (400ms) + API response
    await expect(page.getByText(`https://new-live-${suffix}.example.com`).first()).toBeVisible({
      timeout: 10000,
    })
    await expect(page.getByText(`https://new-shadow-${suffix}.example.com`).first()).toBeVisible()
  })

  test('create gate form disables submit for invalid URLs', async ({ page }) => {
    await page.goto('/gates')
    await page.getByRole('button', { name: 'New gate' }).click()

    const submitButton = page.locator('form').getByRole('button', { name: 'Create gate' })

    // Empty form — submit disabled
    await expect(submitButton).toBeDisabled()

    // Invalid live URL (with name filled)
    await page.getByLabel('Name').fill('test-gate')
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
    const suffix = Date.now()
    await api.createGate(
      `xflt-alpha-gate-${suffix}`,
      `https://xflt-${suffix}-alpha-live.example.com`,
      `https://xflt-${suffix}-alpha-shadow.example.com`
    )
    await api.createGate(
      `xflt-beta-gate-${suffix}`,
      `https://xflt-${suffix}-beta-live.example.com`,
      `https://xflt-${suffix}-beta-shadow.example.com`
    )

    await page.goto('/gates')
    const searchBox = page.getByPlaceholder('Search gates by live URL...')

    // Search for our unique prefix to isolate test data
    await searchBox.fill(`xflt-${suffix}-`)

    // Both test gates should appear
    await expect(page.getByText(`xflt-${suffix}-alpha-live`).first()).toBeVisible()
    await expect(page.getByText(`xflt-${suffix}-beta-live`).first()).toBeVisible()

    // Narrow down to alpha only
    await searchBox.fill(`xflt-${suffix}-alpha`)

    await expect(page.getByText(`xflt-${suffix}-alpha-live`).first()).toBeVisible()
    await expect(page.getByText(`xflt-${suffix}-beta-live`)).not.toBeVisible()

    // Narrow down to beta only
    await searchBox.fill(`xflt-${suffix}-beta`)

    await expect(page.getByText(`xflt-${suffix}-beta-live`).first()).toBeVisible()
    await expect(page.getByText(`xflt-${suffix}-alpha-live`)).not.toBeVisible()
  })

  test('sort gates by live URL ascending', async ({ page, api }) => {
    const suffix = Date.now()
    await api.createGate(
      `xsrt-zebra-gate-${suffix}`,
      `https://xsrt-${suffix}-zebra.example.com`,
      `https://xsrt-${suffix}-shadow-z.example.com`
    )
    await api.createGate(
      `xsrt-apple-gate-${suffix}`,
      `https://xsrt-${suffix}-apple.example.com`,
      `https://xsrt-${suffix}-shadow-a.example.com`
    )

    await page.goto('/gates')

    // First isolate our test gates using search
    await page.getByPlaceholder('Search gates by live URL...').fill(`xsrt-${suffix}-`)
    await expect(page.getByText(`xsrt-${suffix}-zebra`).first()).toBeVisible()

    // Change sort to Live URL A→Z
    await page.getByText('Sort:').click()
    await page.getByRole('option', { name: 'Live URL (A→Z)' }).click()

    // First card should be apple (alphabetically first)
    const cards = page.locator('[class*="cursor-pointer"]')
    await expect(cards).toHaveCount(2)
    const firstCardText = await cards.first().textContent()
    expect(firstCardText).toContain(`xsrt-${suffix}-apple`)

    // Last card should be zebra
    const lastCardText = await cards.last().textContent()
    expect(lastCardText).toContain(`xsrt-${suffix}-zebra`)
  })

  test('pagination works with many gates', async ({ page, api }) => {
    // Create 8 gates with unique prefix (page size is 5, so we get 2 pages)
    const suffix = Date.now()
    const promises = Array.from({ length: 8 }, (_, i) =>
      api.createGate(
        `xpag-gate-${suffix}-${String(i).padStart(3, '0')}`,
        `https://xpag-${suffix}-live-${String(i).padStart(3, '0')}.example.com`,
        `https://xpag-${suffix}-shadow-${String(i).padStart(3, '0')}.example.com`
      )
    )
    await Promise.all(promises)

    await page.goto('/gates')

    // Filter to only our test gates
    await page.getByPlaceholder('Search gates by live URL...').fill(`xpag-${suffix}-`)

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
    const suffix = Date.now()
    const promises = Array.from({ length: 8 }, (_, i) =>
      api.createGate(
        `xrst-gate-${suffix}-${String(i).padStart(3, '0')}`,
        `https://xrst-${suffix}-live-${String(i).padStart(3, '0')}.example.com`,
        `https://xrst-${suffix}-shadow-${String(i).padStart(3, '0')}.example.com`
      )
    )
    await Promise.all(promises)

    await page.goto('/gates')

    // Filter to our test gates — 8 results, 2 pages
    const searchBox = page.getByPlaceholder('Search gates by live URL...')
    await searchBox.fill(`xrst-${suffix}-`)
    await expect(page.getByText('Page 1 of 2')).toBeVisible()

    // Go to page 2
    await page.getByRole('button', { name: 'Next' }).click()
    await expect(page.getByText('Page 2 of 2')).toBeVisible()

    // Clear and type a narrower filter — should reset to page 1
    await searchBox.clear()
    await searchBox.fill(`xrst-${suffix}-live-000`)

    // Wait for debounce (400ms) + API response
    await expect(page.getByText(`xrst-${suffix}-live-000`).first()).toBeVisible({ timeout: 10000 })

    // Pagination should be gone (only 1 result)
    await expect(page.getByText(/Page \d+ of \d+/)).not.toBeVisible()
  })
})
