import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import GateForm from './GateForm.vue'

const createGate = vi.fn()
vi.mock('@/api', () => ({
  createGate: (...args: unknown[]) => createGate(...args),
}))

function mountForm() {
  return mount(GateForm)
}

async function fillValid(wrapper: ReturnType<typeof mountForm>) {
  await wrapper.find('#gate-name').setValue('checkout-api')
  await wrapper.find('#live-url').setValue('https://live.example.com')
  await wrapper.find('#shadow-url').setValue('https://shadow.example.com')
}

function submitButton(wrapper: ReturnType<typeof mountForm>) {
  return wrapper.find('button[type="submit"]')
}

describe('GateForm validation gating', () => {
  beforeEach(() => {
    createGate.mockReset()
  })

  it('disables submit until the name and both URLs are valid', async () => {
    const wrapper = mountForm()
    expect(submitButton(wrapper).attributes('disabled')).toBeDefined()

    await fillValid(wrapper)
    expect(submitButton(wrapper).attributes('disabled')).toBeUndefined()
  })

  it('flags an invalid URL and keeps submit disabled', async () => {
    const wrapper = mountForm()
    await wrapper.find('#gate-name').setValue('checkout-api')
    await wrapper.find('#live-url').setValue('not-a-url')
    await wrapper.find('#shadow-url').setValue('https://shadow.example.com')

    expect(wrapper.text()).toContain('Please enter a valid URL')
    expect(submitButton(wrapper).attributes('disabled')).toBeDefined()
  })
})

describe('GateForm submission', () => {
  beforeEach(() => {
    createGate.mockReset()
  })

  it('creates the gate, emits success, and resets the form on success', async () => {
    createGate.mockResolvedValue({ data: {} })
    const wrapper = mountForm()
    await fillValid(wrapper)

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(createGate).toHaveBeenCalledWith({
      name: 'checkout-api',
      live_url: 'https://live.example.com',
      shadow_url: 'https://shadow.example.com',
    })
    expect(wrapper.emitted('success')).toHaveLength(1)
    expect((wrapper.find('#gate-name').element as HTMLInputElement).value).toBe('')
  })

  it('surfaces the API error message and keeps the form populated on failure', async () => {
    createGate.mockRejectedValue(new Error('name already taken'))
    const wrapper = mountForm()
    await fillValid(wrapper)

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('name already taken')
    expect(wrapper.emitted('success')).toBeUndefined()
    expect((wrapper.find('#gate-name').element as HTMLInputElement).value).toBe('checkout-api')
  })
})
