import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import NotFound from './NotFound.vue'

// Render router-link as a plain anchor so the `to` target stays observable.
const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

function mountNotFound() {
  return mount(NotFound, { global: { stubs: { RouterLink: RouterLinkStub } } })
}

describe('NotFound', () => {
  it('renders the 404 code, heading, and explanatory copy', () => {
    const wrapper = mountNotFound()
    expect(wrapper.text()).toContain('404')
    expect(wrapper.text()).toContain('Page not found')
    expect(wrapper.text()).toContain("doesn't exist or has been moved")
  })

  it('links back to the gates index', () => {
    const wrapper = mountNotFound()
    const link = wrapper.find('a')
    expect(link.attributes('href')).toBe('/gates')
    expect(link.text()).toContain('Go to Gates')
  })
})
