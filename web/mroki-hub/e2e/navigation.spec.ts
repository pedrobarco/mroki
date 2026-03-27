import { test, expect } from './fixtures'

test.describe('Navigation', () => {
  test('root redirects to /gates', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/gates$/)
  })

  test('unknown route shows 404 page', async ({ page }) => {
    await page.goto('/nonexistent-route')
    await expect(page.getByText('404')).toBeVisible()
    await expect(page.getByText('Page not found')).toBeVisible()
  })

  test('404 page "Go to Gates" link navigates to /gates', async ({ page }) => {
    await page.goto('/nonexistent-route')
    await page.getByRole('link', { name: 'Go to Gates' }).click()
    await expect(page).toHaveURL(/\/gates$/)
    await expect(page.getByRole('heading', { name: 'Gates' })).toBeVisible()
  })

  test('header "Gates" link navigates to /gates', async ({ page }) => {
    await page.goto('/nonexistent-route')
    await page.locator('nav').getByRole('link', { name: 'Gates' }).click()
    await expect(page).toHaveURL(/\/gates$/)
  })
})
