import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import GateSettings from './GateSettings.vue'
import type { Gate } from '@/api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'gate-1' } }),
  useRouter: () => ({ push }),
  onBeforeRouteLeave: vi.fn(),
}))

const getGate = vi.fn()
const updateGate = vi.fn()
const deleteGate = vi.fn()
const getConfig = vi.fn()
// Mock the underlying API modules so the real query-key factory and query
// adapters (re-exported from '@/api') keep working while the network is stubbed.
vi.mock('@/api/gates', () => ({
  getGate: (...a: unknown[]) => getGate(...a),
  updateGate: (...a: unknown[]) => updateGate(...a),
  deleteGate: (...a: unknown[]) => deleteGate(...a),
}))
vi.mock('@/api/config', () => ({
  getConfig: (...a: unknown[]) => getConfig(...a),
}))

// A fresh, retry-free client per mount isolates the gate/config cache per test.
function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function makeGate(overrides: Partial<Gate> = {}): Gate {
  return {
    id: 'gate-1',
    name: 'checkout-api',
    live_url: 'https://live.example.com',
    shadow_url: 'https://shadow.example.com',
    diff_config: {
      ignored_fields: [],
      included_fields: [],
      float_tolerance: 0,
      sort_arrays: false,
    },
    redacted_fields: [],
    retention: '',
    created_at: '2026-03-29T09:00:00Z',
    stats: { request_count_24h: 0, diff_count_24h: 0, diff_rate: 0, last_active: null },
    ...overrides,
  }
}

function mountShallow() {
  return mount(GateSettings, {
    shallow: true,
    global: { plugins: [[VueQueryPlugin, { queryClient: makeQueryClient() }]] },
  })
}

async function mountSettings() {
  getGate.mockResolvedValue({ data: makeGate() })
  const wrapper = mountShallow()
  await flushPromises()
  return wrapper
}

describe('GateSettings hardening', () => {
  beforeEach(() => {
    push.mockClear()
    getGate.mockReset()
    updateGate.mockReset()
    deleteGate.mockReset()
    getConfig.mockReset()
    getConfig.mockResolvedValue({ data: { retention: '720h0m0s' } })
  })

  it('surfaces delete errors and keeps the confirm dialog open on failure', async () => {
    deleteGate.mockRejectedValueOnce(new Error('cannot delete gate'))
    const wrapper = await mountSettings()

    wrapper.vm.deleteDialogOpen = true
    await wrapper.vm.handleDelete()
    await flushPromises()

    expect(wrapper.vm.deleteError).toBe('cannot delete gate')
    expect(wrapper.vm.deleteDialogOpen).toBe(true)
    expect(push).not.toHaveBeenCalled()
  })

  it('closes the dialog and navigates away on a successful delete', async () => {
    deleteGate.mockResolvedValueOnce(undefined)
    const wrapper = await mountSettings()

    wrapper.vm.deleteDialogOpen = true
    await wrapper.vm.handleDelete()
    await flushPromises()

    expect(wrapper.vm.deleteError).toBeNull()
    expect(wrapper.vm.deleteDialogOpen).toBe(false)
    expect(push).toHaveBeenCalledWith('/gates')
  })

  it('keeps the form clean (not dirty) right after load', async () => {
    const wrapper = await mountSettings()
    expect(wrapper.vm.isDirty).toBe(false)
  })

  it('marks the form dirty once a field changes', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.name = 'renamed-gate'
    await nextTick()
    expect(wrapper.vm.isDirty).toBe(true)
  })

  it('opens the discard-changes dialog instead of navigating when dirty', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.name = 'renamed-gate'
    await nextTick()

    wrapper.vm.goBack()
    expect(wrapper.vm.leaveDialogOpen).toBe(true)
    expect(push).not.toHaveBeenCalled()
  })

  it('navigates back directly when the form is clean', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.goBack()
    expect(push).toHaveBeenCalledWith('/gates/gate-1')
    expect(wrapper.vm.leaveDialogOpen).toBe(false)
  })

  it('populates retention from the gate and stays clean', async () => {
    getGate.mockResolvedValue({ data: makeGate({ retention: '168h' }) })
    const wrapper = mountShallow()
    await flushPromises()

    expect(wrapper.vm.retention).toBe('168h')
    expect(wrapper.vm.isDirty).toBe(false)
  })

  it('marks the form dirty once retention changes', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = '240h'
    await nextTick()
    expect(wrapper.vm.isDirty).toBe(true)
  })

  it('sends the trimmed retention in the update payload', async () => {
    const wrapper = await mountSettings()
    updateGate.mockResolvedValueOnce({ data: makeGate({ retention: '800h' }) })

    // Above the 720h global floor so validation permits the save.
    wrapper.vm.retention = '  800h  '
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()

    expect(updateGate).toHaveBeenCalledWith(
      'gate-1',
      expect.objectContaining({ retention: '800h' })
    )
  })

  it('sends an empty retention to reset to the global default', async () => {
    getGate.mockResolvedValue({ data: makeGate({ retention: '168h' }) })
    const wrapper = mountShallow()
    await flushPromises()
    updateGate.mockResolvedValueOnce({ data: makeGate({ retention: '' }) })

    wrapper.vm.retention = ''
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()

    expect(updateGate).toHaveBeenCalledWith('gate-1', expect.objectContaining({ retention: '' }))
  })

  it('exposes the loaded global retention floor', async () => {
    const wrapper = await mountSettings()
    expect(wrapper.vm.globalRetention).toBe('720h0m0s')
  })

  it('surfaces the global retention floor in the retention copy', async () => {
    const wrapper = await mountSettings()
    expect(wrapper.text()).toContain('720h0m0s')
  })

  it('mounts and stays editable when the config load rejects', async () => {
    // The floor is best-effort guidance; a rejected /config must never break
    // the page. (The session cache may already hold a floor from an earlier
    // test, so this asserts resilience, not a null floor.)
    getConfig.mockRejectedValue(new Error('config unavailable'))
    const wrapper = await mountSettings()
    expect(wrapper.vm.isDirty).toBe(false)
    wrapper.vm.retention = '240h'
    await nextTick()
    expect(wrapper.vm.isDirty).toBe(true)
  })

  it('accepts an empty retention (reset) with no error', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = ''
    await nextTick()
    expect(wrapper.vm.retentionError).toBeNull()
  })

  it('rejects an invalid duration format', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = 'abc'
    await nextTick()
    expect(wrapper.vm.retentionError).toContain('Enter a duration')
    expect(wrapper.vm.canSave).toBe(false)
  })

  it('accepts a day shorthand at or above the floor and sends Go units', async () => {
    const wrapper = await mountSettings()
    updateGate.mockResolvedValueOnce({ data: makeGate({ retention: '720h' }) })

    // 30d == 720h == the floor.
    wrapper.vm.retention = '30d'
    await nextTick()
    expect(wrapper.vm.retentionError).toBeNull()
    expect(wrapper.vm.canSave).toBe(true)

    await wrapper.vm.handleSave()
    await flushPromises()
    expect(updateGate).toHaveBeenCalledWith(
      'gate-1',
      expect.objectContaining({ retention: '720h' })
    )
  })

  it('accepts a week shorthand above the floor', async () => {
    const wrapper = await mountSettings()
    // 5w == 840h, above the 720h floor.
    wrapper.vm.retention = '5w'
    await nextTick()
    expect(wrapper.vm.retentionError).toBeNull()
    expect(wrapper.vm.canSave).toBe(true)
  })

  it('rejects a day shorthand below the floor', async () => {
    const wrapper = await mountSettings()
    // 10d == 240h, below the 720h floor.
    wrapper.vm.retention = '10d'
    await nextTick()
    expect(wrapper.vm.retentionError).toContain('at least the global retention')
    expect(wrapper.vm.canSave).toBe(false)
  })

  it('rejects keep-forever (0)', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = '0'
    await nextTick()
    expect(wrapper.vm.retentionError).toContain('Keep-forever')
    expect(wrapper.vm.canSave).toBe(false)
  })

  it('rejects a value below the global floor', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = '24h'
    await nextTick()
    expect(wrapper.vm.retentionError).toContain('at least the global retention')
    expect(wrapper.vm.canSave).toBe(false)
  })

  it('accepts a value at or above the global floor', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = '720h'
    await nextTick()
    expect(wrapper.vm.retentionError).toBeNull()
    expect(wrapper.vm.canSave).toBe(true)
  })

  it('does not call the API when saving an invalid retention', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = '24h'
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()
    expect(updateGate).not.toHaveBeenCalled()
  })

  it('describes the retention input with its guidance when valid', async () => {
    const wrapper = await mountSettings()
    expect(wrapper.vm.retentionDescribedBy).toBe('gate-retention-desc')
  })

  it('appends the error id to the description once the field is touched and invalid', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = 'abc'
    await nextTick()
    // Untouched: no visible error yet, so the description stays clean.
    expect(wrapper.vm.retentionDescribedBy).toBe('gate-retention-desc')
    wrapper.vm.onRetentionBlur()
    await nextTick()
    expect(wrapper.vm.retentionDescribedBy).toBe('gate-retention-desc gate-retention-error')
  })

  it('holds the inline error until the field is blurred', async () => {
    const wrapper = await mountSettings()
    wrapper.vm.retention = 'abc'
    await nextTick()
    // Validity gate fires immediately, but the visible field error waits.
    expect(wrapper.vm.retentionError).toContain('Enter a duration')
    expect(wrapper.vm.retentionFieldError).toBeNull()
    wrapper.vm.onRetentionBlur()
    await nextTick()
    expect(wrapper.vm.retentionFieldError).toContain('Enter a duration')
  })

  it('surfaces the inline error immediately on a paste of an invalid value', async () => {
    const wrapper = await mountSettings()
    // Paste marks the field touched before the value settles.
    wrapper.vm.onRetentionPaste()
    wrapper.vm.retention = 'abc'
    await nextTick()
    expect(wrapper.vm.retentionFieldError).toContain('Enter a duration')
  })

  it('humanizes a whole-day global floor in the copy', async () => {
    const wrapper = await mountSettings()
    expect(wrapper.vm.globalRetentionLabel).toBe('30d (720h0m0s)')
    expect(wrapper.text()).toContain('30d (720h0m0s)')
  })

  it('co-locates a retention rejection from the server under the field', async () => {
    const wrapper = await mountSettings()
    updateGate.mockRejectedValueOnce(new Error('retention must be at least the global retention'))

    wrapper.vm.retention = '800h'
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()

    expect(wrapper.vm.retentionFieldError).toContain('retention')
    expect(wrapper.vm.saveError).toBeNull()
  })

  it('co-locates a floor rejection phrased without the word "retention"', async () => {
    const wrapper = await mountSettings()
    updateGate.mockRejectedValueOnce(new Error('value must be at least 720h0m0s'))

    wrapper.vm.retention = '800h'
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()

    expect(wrapper.vm.retentionServerError).toContain('at least')
    expect(wrapper.vm.saveError).toBeNull()
  })

  it('keeps non-retention server errors in the top save alert', async () => {
    const wrapper = await mountSettings()
    updateGate.mockRejectedValueOnce(new Error('name already in use'))

    wrapper.vm.retention = '800h'
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()

    expect(wrapper.vm.saveError).toBe('name already in use')
    expect(wrapper.vm.retentionServerError).toBeNull()
  })

  it('clears a server retention error when the value is edited', async () => {
    const wrapper = await mountSettings()
    updateGate.mockRejectedValueOnce(new Error('retention below the global floor'))

    wrapper.vm.retention = '800h'
    await nextTick()
    await wrapper.vm.handleSave()
    await flushPromises()
    expect(wrapper.vm.retentionServerError).not.toBeNull()

    wrapper.vm.onRetentionInput()
    await nextTick()
    expect(wrapper.vm.retentionServerError).toBeNull()
  })
})
