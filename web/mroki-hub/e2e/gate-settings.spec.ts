import { test, expect } from './fixtures'

test.describe('Gate Settings Page', () => {
  test('displays gate settings with correct sections', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `settings-gate-${suffix}`,
      `https://settings-live-${suffix}.example.com`,
      `https://settings-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)
    await expect(page.getByText('Gate Settings')).toBeVisible()
    await expect(page.getByText('General')).toBeVisible()
    await expect(page.getByText('Header Scrubbing')).toBeVisible()
    await expect(page.getByText('Diff Configuration')).toBeVisible()
    await expect(page.getByText('Danger Zone')).toBeVisible()
  })

  test('shows default scrub fields as read-only', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `scrub-defaults-gate-${suffix}`,
      `https://scrub-live-${suffix}.example.com`,
      `https://scrub-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)
    await expect(page.getByText('headers.Authorization')).toBeVisible()
    await expect(page.getByText('headers.Cookie')).toBeVisible()
    await expect(page.getByText('headers.Set-Cookie')).toBeVisible()
    await expect(page.getByText('headers.X-Api-Key')).toBeVisible()
    await expect(page.getByText('Always active')).toBeVisible()
  })

  test('updates gate name', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `name-gate-${suffix}`,
      `https://name-live-${suffix}.example.com`,
      `https://name-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)
    const nameInput = page.getByLabel('Name')
    await nameInput.clear()
    await nameInput.fill(`renamed-gate-${suffix}`)
    await page.getByRole('button', { name: 'Save Changes' }).click()

    // Should show save confirmation
    await expect(page.getByRole('button', { name: 'Saved' })).toBeVisible()

    // Verify via API
    const updated = await api.updateGate(gate.id, {})
    expect(updated.name).toBe(`renamed-gate-${suffix}`)
  })

  test('adds and removes scrub additional fields', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `scrub-edit-gate-${suffix}`,
      `https://scrub-live-${suffix}.example.com`,
      `https://scrub-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)

    // Add a scrub field using placeholder to target the correct input
    const scrubInput = page.getByPlaceholder('e.g. headers.X-Internal-Token')
    await scrubInput.fill('headers.X-Internal-Token')
    await scrubInput.press('Enter')

    // Field should appear
    await expect(page.getByText('headers.X-Internal-Token')).toBeVisible()

    // Save
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.getByRole('button', { name: 'Saved' })).toBeVisible()

    // Verify via API
    const updated = await api.updateGate(gate.id, {})
    expect(updated.scrub_config.additional_fields).toContain('headers.X-Internal-Token')
  })

  test('updates diff configuration', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `diff-cfg-gate-${suffix}`,
      `https://diffcfg-live-${suffix}.example.com`,
      `https://diffcfg-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)

    // Add ignored field using placeholder to target the correct input
    const ignoredInput = page.getByPlaceholder('e.g. timestamp, request_id')
    await ignoredInput.fill('timestamp')
    await ignoredInput.press('Enter')

    // Set float tolerance
    const toleranceInput = page.locator('input[type="number"]')
    await toleranceInput.fill('0.001')

    // Save
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.getByRole('button', { name: 'Saved' })).toBeVisible()

    // Verify via API
    const updated = await api.updateGate(gate.id, {})
    expect(updated.diff_config.ignored_fields).toContain('timestamp')
    expect(updated.diff_config.float_tolerance).toBe(0.001)
  })

  test('delete gate from settings and navigates to gates list', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `delete-gate-${suffix}`,
      `https://delete-live-${suffix}.example.com`,
      `https://delete-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)

    // Click Delete Gate button
    await page.getByRole('button', { name: 'Delete Gate' }).click()

    // Confirmation dialog appears
    const dialog = page.getByRole('alertdialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByText(gate.name)).toBeVisible()

    // Cancel first
    await page.getByRole('button', { name: 'Cancel' }).click()
    await expect(dialog).not.toBeVisible()

    // Delete for real
    await page.getByRole('button', { name: 'Delete Gate' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Delete' }).click()

    await expect(page).toHaveURL('/gates')
  })

  test('back button navigates to gate detail', async ({ page, api }) => {
    const suffix = Date.now()
    const gate = await api.createGate(
      `back-settings-gate-${suffix}`,
      `https://back-live-${suffix}.example.com`,
      `https://back-shadow-${suffix}.example.com`
    )

    await page.goto(`/gates/${gate.id}/settings`)
    await page.getByText('Back to Gate').click()
    await expect(page).toHaveURL(`/gates/${gate.id}`)
  })
})
