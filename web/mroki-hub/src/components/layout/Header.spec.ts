import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Header from './Header.vue'

// A mutable route stand-in; each test sets the path before mounting.
const routeState = vi.hoisted(() => ({ path: '/gates' }))
vi.mock('vue-router', () => ({ useRoute: () => routeState }))

// Keep class/`to` bindings observable by rendering a plain anchor.
const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

function mountHeader(path: string) {
  routeState.path = path
  return mount(Header, { global: { stubs: { RouterLink: RouterLinkStub } } })
}

function gatesLink(wrapper: ReturnType<typeof mountHeader>) {
  return wrapper.findAll('a').find((a) => a.text() === 'Gates')!
}

describe('Header active navigation', () => {
  it('highlights the Gates link when on the gates index', () => {
    const link = gatesLink(mountHeader('/gates'))
    expect(link.classes()).toContain('bg-accent')
    expect(link.classes()).toContain('text-foreground')
    expect(link.classes()).toContain('font-medium')
  })

  it('keeps the Gates link active on a nested gates route', () => {
    const link = gatesLink(mountHeader('/gates/abc-123'))
    expect(link.classes()).toContain('bg-accent')
  })

  it('renders the Gates link as inactive on an unrelated route', () => {
    const link = gatesLink(mountHeader('/something-else'))
    expect(link.classes()).toContain('text-muted-foreground')
    expect(link.classes()).not.toContain('bg-accent')
  })
})
