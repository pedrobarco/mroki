import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
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
vi.mock('@/api', () => ({
  getGate: (...a: unknown[]) => getGate(...a),
  updateGate: (...a: unknown[]) => updateGate(...a),
  deleteGate: (...a: unknown[]) => deleteGate(...a),
}))

const setGate = vi.fn()
vi.mock('@/composables/use-gate-cache', () => ({
  useGateCache: () => ({ setGate, getCachedGate: () => null }),
}))

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
    created_at: '2026-03-29T09:00:00Z',
    stats: { request_count_24h: 0, diff_count_24h: 0, diff_rate: 0, last_active: null },
    ...overrides,
  }
}

async function mountSettings() {
  getGate.mockResolvedValue({ data: makeGate() })
  const wrapper = mount(GateSettings, { shallow: true })
  await flushPromises()
  return wrapper
}

describe('GateSettings hardening', () => {
  beforeEach(() => {
    push.mockClear()
    getGate.mockReset()
    updateGate.mockReset()
    deleteGate.mockReset()
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
})
