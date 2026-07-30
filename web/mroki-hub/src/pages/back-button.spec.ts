import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import GateDetail from './GateDetail.vue'
import RequestDetail from './RequestDetail.vue'
import GateSettings from './GateSettings.vue'

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
  useRoute: () => ({ params: { id: 'gate-1', rid: 'req-1' } }),
}))

// Reject API calls so each page settles into its error branch, keeping the
// mount lightweight while still rendering the back button in the template.
vi.mock('@/api', () => ({
  getGate: vi.fn().mockRejectedValue(new Error('stub')),
  getRequest: vi.fn().mockRejectedValue(new Error('stub')),
  updateGate: vi.fn(),
  deleteGate: vi.fn(),
}))

vi.mock('@/composables/use-gate-cache', () => ({
  useGateCache: () => ({ getCachedGate: () => null, setGate: vi.fn() }),
}))

beforeEach(() => {
  push.mockClear()
})

async function mountPage(component: unknown) {
  const wrapper = mount(component as never, { shallow: true })
  await flushPromises()
  return wrapper
}

// The back controls are native <button type="button"> elements, which the
// browser makes keyboard-operable (Enter/Space -> click) with no extra JS.
// These tests lock in the native-button semantics and the navigation target.
describe('page back buttons are keyboard-operable native buttons', () => {
  const cases = [
    { name: 'GateDetail', component: GateDetail, label: 'Back to Gates', target: '/gates' },
    {
      name: 'RequestDetail',
      component: RequestDetail,
      label: 'Back to Gate',
      target: '/gates/gate-1',
    },
    {
      name: 'GateSettings',
      component: GateSettings,
      label: 'Back to Gate',
      target: '/gates/gate-1',
    },
  ]

  cases.forEach(({ name, component, label, target }) => {
    it(`${name}: back control is a <button type="button">`, async () => {
      const wrapper = await mountPage(component)
      const button = wrapper.findAll('button').find((b) => b.text().includes(label))
      expect(button, `expected a back button labelled "${label}"`).toBeTruthy()
      expect(button!.element.tagName).toBe('BUTTON')
      expect(button!.attributes('type')).toBe('button')
    })

    it(`${name}: activating the back control navigates to ${target}`, async () => {
      const wrapper = await mountPage(component)
      const button = wrapper.findAll('button').find((b) => b.text().includes(label))!
      await button.trigger('click')
      expect(push).toHaveBeenCalledWith(target)
    })
  })
})
