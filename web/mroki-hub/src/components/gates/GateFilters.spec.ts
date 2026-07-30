import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import GateFilters, { type GateFilterState } from './GateFilters.vue'

function makeFilters(overrides: Partial<GateFilterState> = {}): GateFilterState {
  return { liveUrl: '', shadowUrl: '', sort: 'created_at', order: 'desc', ...overrides }
}

// A Select stub that surfaces the bound value and lets us drive a selection,
// so we can assert the sort encode/decode without reka-ui's runtime.
const SelectStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template:
    '<div class="select-stub" :data-value="modelValue" @click="$emit(\'update:modelValue\', \'name-asc\')" />',
}

const global = {
  stubs: {
    Select: SelectStub,
    SelectContent: true,
    SelectItem: true,
    SelectTrigger: true,
    SelectValue: true,
    Search: true,
  },
}

function mountFilters(modelValue: GateFilterState) {
  return mount(GateFilters, { props: { modelValue }, global })
}

function lastUpdate(wrapper: ReturnType<typeof mountFilters>): GateFilterState {
  const events = wrapper.emitted('update:modelValue') as GateFilterState[][] | undefined
  if (!events?.length) throw new Error('no update:modelValue emitted')
  return events[events.length - 1][0]
}

describe('GateFilters debounced search', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('emits the live URL update only after the 400ms debounce window', async () => {
    vi.useFakeTimers()
    const wrapper = mountFilters(makeFilters())

    await wrapper.find('#gate-url-search').setValue('checkout')

    vi.advanceTimersByTime(399)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    vi.advanceTimersByTime(1)
    expect(lastUpdate(wrapper).liveUrl).toBe('checkout')
  })

  it('debounces rapid keystrokes into a single trailing update', async () => {
    vi.useFakeTimers()
    const wrapper = mountFilters(makeFilters())
    const input = wrapper.find('#gate-url-search')

    await input.setValue('a')
    vi.advanceTimersByTime(200)
    await input.setValue('ab')
    vi.advanceTimersByTime(400)

    const events = wrapper.emitted('update:modelValue') as GateFilterState[][]
    expect(events).toHaveLength(1)
    expect(events[0][0].liveUrl).toBe('ab')
  })
})

describe('GateFilters sort', () => {
  it('reflects the current sort/order as a combined field-order value', () => {
    const wrapper = mountFilters(makeFilters({ sort: 'name', order: 'desc' }))
    expect(wrapper.find('.select-stub').attributes('data-value')).toBe('name-desc')
  })

  it('decodes a selected field-order value into separate sort and order updates', async () => {
    const wrapper = mountFilters(makeFilters())
    await wrapper.find('.select-stub').trigger('click')

    const update = lastUpdate(wrapper)
    expect(update.sort).toBe('name')
    expect(update.order).toBe('asc')
  })
})
