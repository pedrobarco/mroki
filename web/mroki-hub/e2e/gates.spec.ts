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

    // Dialog closes and new gate appears
    await expect(page.getByRole('heading', { name: 'Create New Gate' })).not.toBeVisible()
    await expect(page.getByText('https://new-live.example.com').first()).toBeVisible()
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
})
