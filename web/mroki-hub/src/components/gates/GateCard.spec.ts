import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import GateCard from './GateCard.vue'
import type { Gate } from '@/api'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
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

describe('GateCard keyboard operability', () => {
  beforeEach(() => {
    push.mockClear()
  })

  it('exposes button semantics on the clickable root', () => {
    const wrapper = mount(GateCard, { props: { gate: makeGate() } })
    const control = wrapper.get('[role="button"]')
    expect(control.attributes('tabindex')).toBe('0')
    expect(control.attributes('aria-label')).toBe('View gate checkout-api')
  })

  it('navigates to the gate on Enter', async () => {
    const wrapper = mount(GateCard, { props: { gate: makeGate({ id: 'gate-9' }) } })
    await wrapper.get('[role="button"]').trigger('keydown.enter')
    expect(push).toHaveBeenCalledWith('/gates/gate-9')
  })

  it('navigates to the gate on Space', async () => {
    const wrapper = mount(GateCard, { props: { gate: makeGate({ id: 'gate-7' }) } })
    await wrapper.get('[role="button"]').trigger('keydown.space')
    expect(push).toHaveBeenCalledWith('/gates/gate-7')
  })

  it('navigates to the gate on click', async () => {
    const wrapper = mount(GateCard, { props: { gate: makeGate({ id: 'gate-3' }) } })
    await wrapper.get('[role="button"]').trigger('click')
    expect(push).toHaveBeenCalledWith('/gates/gate-3')
  })
})
