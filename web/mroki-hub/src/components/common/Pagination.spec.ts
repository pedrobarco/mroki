import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Pagination from './Pagination.vue'

interface Props {
  currentPage: number
  totalPages: number
  disabledPrev?: boolean
  disabledNext?: boolean
}

function mountPager(props: Props, slots: Record<string, string> = {}) {
  return mount(Pagination, { props, slots })
}

describe('Pagination', () => {
  it('renders nothing when there is a single page', () => {
    const wrapper = mountPager({ currentPage: 1, totalPages: 1 })
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('renders the page indicator and Prev/Next when multiple pages exist', () => {
    const wrapper = mountPager({ currentPage: 2, totalPages: 5 })
    expect(wrapper.text()).toContain('Page 2 of 5')
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(2)
    expect(buttons[0].text()).toBe('Previous')
    expect(buttons[1].text()).toBe('Next')
  })

  it('disables Previous and Next per the disabled flags', () => {
    const wrapper = mountPager({
      currentPage: 1,
      totalPages: 3,
      disabledPrev: true,
      disabledNext: false,
    })
    const [prev, next] = wrapper.findAll('button')
    expect(prev.attributes('disabled')).toBeDefined()
    expect(next.attributes('disabled')).toBeUndefined()
  })

  it('emits prev and next when the buttons are clicked', async () => {
    const wrapper = mountPager({ currentPage: 2, totalPages: 3 })
    const [prev, next] = wrapper.findAll('button')
    await prev.trigger('click')
    await next.trigger('click')
    expect(wrapper.emitted('prev')).toHaveLength(1)
    expect(wrapper.emitted('next')).toHaveLength(1)
  })

  it('renders the meta slot alongside the page indicator', () => {
    const wrapper = mountPager({ currentPage: 1, totalPages: 2 }, { meta: ' · 12 gates' })
    expect(wrapper.text()).toContain('Page 1 of 2')
    expect(wrapper.text()).toContain('· 12 gates')
  })
})
