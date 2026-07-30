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

describe('GateCard active dot recency', () => {
  // The dot is the small rounded-full span in the name row (w-1.5 h-1.5).
  function dotClasses(last_active: string | null): string[] {
    const stats = { ...makeGate().stats, last_active }
    const wrapper = mount(GateCard, { props: { gate: makeGate({ stats }) } })
    const dot = wrapper
      .findAll('span')
      .find((s) => s.classes().includes('w-1.5') && s.classes().includes('rounded-full'))
    return dot?.classes() ?? []
  }

  it('pulses when the gate was active within the recency window', () => {
    const recent = new Date(Date.now() - 60_000).toISOString()
    const classes = dotClasses(recent)
    expect(classes).toContain('bg-success')
    expect(classes).toContain('animate-pulse')
  })

  it('is a static success dot when activity is stale', () => {
    const stale = new Date(Date.now() - 60 * 60_000).toISOString()
    const classes = dotClasses(stale)
    expect(classes).toContain('bg-success')
    expect(classes).not.toContain('animate-pulse')
  })

  it('is a dim, non-pulsing dot when the gate has never been active', () => {
    const classes = dotClasses(null)
    expect(classes).toContain('bg-dim')
    expect(classes).not.toContain('animate-pulse')
  })
})

describe('GateCard name truncation', () => {
  it('truncates a long gate name so it cannot blow out the layout', () => {
    const wrapper = mount(GateCard, {
      props: { gate: makeGate({ name: 'a-very-long-gate-name-that-should-be-truncated' }) },
    })
    const nameSpan = wrapper
      .findAll('span')
      .find((s) => s.text().startsWith('a-very-long-gate-name'))
    expect(nameSpan?.classes()).toContain('truncate')
  })
})

describe('GateCard diff-rate color', () => {
  function rateClasses(diff_rate: number): string[] {
    const stats = { ...makeGate().stats, diff_rate }
    const wrapper = mount(GateCard, { props: { gate: makeGate({ stats }) } })
    const rateSpan = wrapper.findAll('span').find((s) => s.text().endsWith('%'))
    return rateSpan?.classes() ?? []
  }

  it('reads as success below 1%', () => {
    expect(rateClasses(0.4)).toContain('text-success')
  })

  it('reads as warning between 1% and 10%', () => {
    expect(rateClasses(5)).toContain('text-warning')
  })

  it('reads as danger at or above 10%', () => {
    expect(rateClasses(25)).toContain('text-danger')
  })
})
