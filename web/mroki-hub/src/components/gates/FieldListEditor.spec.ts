import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FieldListEditor from './FieldListEditor.vue'

interface Props {
  fields: string[]
  placeholder?: string
  disabled?: boolean
}

function mountEditor(props: Props) {
  return mount(FieldListEditor, { props })
}

function addButton(wrapper: ReturnType<typeof mountEditor>) {
  return wrapper.findAll('button').find((b) => b.text().includes('Add'))!
}

describe('FieldListEditor add', () => {
  it('trims input and emits add with the trimmed field, then clears the input', async () => {
    const wrapper = mountEditor({ fields: [] })
    await wrapper.find('input').setValue('  headers.X-Trace  ')
    await addButton(wrapper).trigger('click')

    expect(wrapper.emitted('add')).toHaveLength(1)
    expect(wrapper.emitted('add')![0]).toEqual(['headers.X-Trace'])
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('')
  })

  it('adds on Enter as well as via the button', async () => {
    const wrapper = mountEditor({ fields: [] })
    const input = wrapper.find('input')
    await input.setValue('body.id')
    await input.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('add')![0]).toEqual(['body.id'])
  })

  it('does not emit add for a duplicate field', async () => {
    const wrapper = mountEditor({ fields: ['body.id'] })
    await wrapper.find('input').setValue('body.id')
    await addButton(wrapper).trigger('click')

    expect(wrapper.emitted('add')).toBeUndefined()
  })

  it('disables the Add button when the input is empty or whitespace-only', async () => {
    const wrapper = mountEditor({ fields: [] })
    expect(addButton(wrapper).attributes('disabled')).toBeDefined()

    await wrapper.find('input').setValue('   ')
    expect(addButton(wrapper).attributes('disabled')).toBeDefined()
  })
})

describe('FieldListEditor remove', () => {
  it('renders each field and emits remove with its index', async () => {
    const wrapper = mountEditor({ fields: ['body.a', 'body.b'] })
    const removeB = wrapper.find('button[aria-label="Remove body.b"]')
    expect(removeB.exists()).toBe(true)

    await removeB.trigger('click')
    expect(wrapper.emitted('remove')![0]).toEqual([1])
  })
})

describe('FieldListEditor disabled state', () => {
  it('disables the input and both action buttons', () => {
    const wrapper = mountEditor({ fields: ['body.a'], disabled: true })
    expect(wrapper.find('input').attributes('disabled')).toBeDefined()
    expect(addButton(wrapper).attributes('disabled')).toBeDefined()
    expect(wrapper.find('button[aria-label="Remove body.a"]').attributes('disabled')).toBeDefined()
  })
})
