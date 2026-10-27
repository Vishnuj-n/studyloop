import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ProfileCommandPalette from './ProfileCommandPalette.vue'
import * as appApi from '../services/appApi'

const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

describe('ProfileCommandPalette.vue', () => {
  const dummyProfiles = [
    { id: 'p-1', name: 'Computer Science', is_active: true, deadline_at: 0 },
    { id: 'p-2', name: 'Mathematics', is_active: false, deadline_at: 0 },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(appApi, 'getProfiles').mockResolvedValue({ profiles: dummyProfiles })
    vi.spyOn(appApi, 'setActiveProfileID').mockResolvedValue({ ok: true })
    document.body.innerHTML = ''
  })

  it('renders nothing when modelValue is false', () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
    })
    expect(wrapper.find('.palette-shell').exists()).toBe(false)
  })

  it('renders modal and displays profiles when opened', async () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const shell = document.body.querySelector('.palette-shell')
    expect(shell).not.toBeNull()

    const items = document.body.querySelectorAll('.palette-item')
    expect(items.length).toBe(2)
    expect(items[0].textContent).toContain('Computer Science')
    expect(items[1].textContent).toContain('Mathematics')

    wrapper.unmount()
  })

  it('filters profiles when typing into search input', async () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const input = document.body.querySelector('.palette-search-input')
    expect(input).not.toBeNull()

    input.value = 'Math'
    input.dispatchEvent(new Event('input'))
    await flushPromises()

    const items = document.body.querySelectorAll('.palette-item')
    expect(items.length).toBe(1)
    expect(items[0].textContent).toContain('Mathematics')

    wrapper.unmount()
  })

  it('emits update:modelValue false on esc key', async () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const input = document.body.querySelector('.palette-search-input')
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')[0]).toEqual([false])

    wrapper.unmount()
  })
})
