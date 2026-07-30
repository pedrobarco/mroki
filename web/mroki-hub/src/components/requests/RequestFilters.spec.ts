import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RequestFilters, { type FilterState } from './RequestFilters.vue'

function makeFilters(overrides: Partial<FilterState> = {}): FilterState {
  return {
    methods: [],
    path: '',
    hasDiff: undefined,
    sort: 'created_at',
    order: 'desc',
    ...overrides,
  }
}

// Stub the select/switch primitives so mounting does not require their runtime.
const global = {
  stubs: {
    Switch: true,
    Select: true,
    SelectContent: true,
    SelectItem: true,
    SelectTrigger: true,
    SelectValue: true,
    Search: true,
  },
}

function methodButton(
  wrapper: ReturnType<typeof mount>,
  label: string
): ReturnType<ReturnType<typeof mount>['get']> {
  const btn = wrapper.findAll('button').find((b) => b.text() === label)
  if (!btn) throw new Error(`method button "${label}" not found`)
  return btn
}

function lastMethods(wrapper: ReturnType<typeof mount>): string[] {
  const events = wrapper.emitted('update:modelValue') as FilterState[][] | undefined
  if (!events?.length) throw new Error('no update:modelValue emitted')
  return events[events.length - 1][0].methods
}

describe('RequestFilters method multi-select', () => {
  it('adds a method to the selection when toggled on', async () => {
    const wrapper = mount(RequestFilters, { props: { modelValue: makeFilters() }, global })
    await methodButton(wrapper, 'GET').trigger('click')
    expect(lastMethods(wrapper)).toEqual(['GET'])
  })

  it('keeps GET and POST active together (true multi-select)', async () => {
    const wrapper = mount(RequestFilters, {
      props: { modelValue: makeFilters({ methods: ['GET'] }) },
      global,
    })
    await methodButton(wrapper, 'POST').trigger('click')
    expect(lastMethods(wrapper)).toEqual(['GET', 'POST'])
  })

  it('removes an already-active method when toggled off', async () => {
    const wrapper = mount(RequestFilters, {
      props: { modelValue: makeFilters({ methods: ['GET', 'POST'] }) },
      global,
    })
    await methodButton(wrapper, 'GET').trigger('click')
    expect(lastMethods(wrapper)).toEqual(['POST'])
  })

  it('clears the selection when All is clicked', async () => {
    const wrapper = mount(RequestFilters, {
      props: { modelValue: makeFilters({ methods: ['GET', 'POST'] }) },
      global,
    })
    await methodButton(wrapper, 'All').trigger('click')
    expect(lastMethods(wrapper)).toEqual([])
  })

  it('reflects active state via aria-pressed', () => {
    const wrapper = mount(RequestFilters, {
      props: { modelValue: makeFilters({ methods: ['POST'] }) },
      global,
    })
    expect(methodButton(wrapper, 'All').attributes('aria-pressed')).toBe('false')
    expect(methodButton(wrapper, 'POST').attributes('aria-pressed')).toBe('true')
    expect(methodButton(wrapper, 'GET').attributes('aria-pressed')).toBe('false')
  })

  it('marks All as pressed when no methods are selected', () => {
    const wrapper = mount(RequestFilters, { props: { modelValue: makeFilters() }, global })
    expect(methodButton(wrapper, 'All').attributes('aria-pressed')).toBe('true')
  })
})
