import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ProfileCommandPalette from './ProfileCommandPalette.vue'
import * as appApi from '../services/appApi'

const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

const mockProfiles = [
  { id: 'prof-1', name: 'Mathematics', is_active: true, pending_tasks: 2 },
  { id: 'prof-2', name: 'Biology', is_active: false, pending_tasks: 0 },
]

describe('ProfileCommandPalette.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(appApi, 'getProfiles').mockResolvedValue({ profiles: mockProfiles })
    vi.spyOn(appApi, 'setActiveProfileID').mockResolvedValue({ ok: true })
  })

  it('renders modal with actions and study profiles when opened', async () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    expect(document.body.querySelector('.palette-shell')).toBeNull()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(document.body.querySelector('.palette-shell')).not.toBeNull()
    const items = document.body.querySelectorAll('.palette-item')
    // 2 action items (Pomodoro, Settings) + 2 profiles = 4 items
    expect(items.length).toBe(4)
    expect(items[0].textContent).toContain('Pomodoro')
    expect(items[1].textContent).toContain('Settings')
    expect(items[2].textContent).toContain('Mathematics')
    expect(items[3].textContent).toContain('Biology')

    wrapper.unmount()
  })

  it('switches profile and fires settings-updated and profile-switched when selecting profile 2 directly via key 2 in search input', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const input = document.body.querySelector('.palette-search-input')
    expect(input).not.toBeNull()

    // Press '2' while focused in search input to directly select 2nd profile (Biology)
    input.dispatchEvent(new KeyboardEvent('keydown', { key: '2', bubbles: true }))
    await flushPromises()

    expect(appApi.setActiveProfileID).toHaveBeenCalledWith('prof-2')
    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'profile-switched', detail: { profileId: 'prof-2' } })
    )
    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'settings-updated' })
    )
    expect(wrapper.emitted('profile-switched')[0]).toEqual(['prof-2'])
    expect(wrapper.emitted('update:modelValue')[0]).toEqual([false])

    wrapper.unmount()
  })

  it('switches profile via numpad digit code Numpad2', async () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const input = document.body.querySelector('.palette-search-input')
    input.dispatchEvent(new KeyboardEvent('keydown', { code: 'Numpad2', bubbles: true }))
    await flushPromises()

    expect(appApi.setActiveProfileID).toHaveBeenCalledWith('prof-2')
    wrapper.unmount()
  })

  it('executes Pomodoro toggle when selecting the first action item via Enter', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const input = document.body.querySelector('.palette-search-input')
    // Press Enter on the first item (focusedIdx = 0 by default, which is Pomodoro action)
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()

    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'pomodoro-toggle' })
    )
    expect(wrapper.emitted('update:modelValue')[0]).toEqual([false])

    wrapper.unmount()
  })

  it('navigates to Settings when selecting Open Settings action', async () => {
    const wrapper = mount(ProfileCommandPalette, {
      props: { modelValue: false },
      attachTo: document.body,
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const input = document.body.querySelector('.palette-search-input')
    // Down arrow to Settings item
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()

    expect(mockPush).toHaveBeenCalledWith('/settings')
    expect(wrapper.emitted('update:modelValue')[0]).toEqual([false])

    wrapper.unmount()
  })

  it('filters commands and profiles when typing into search input', async () => {
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
