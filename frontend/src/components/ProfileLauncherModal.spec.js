import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ProfileLauncherModal from './ProfileLauncherModal.vue'

describe('ProfileLauncherModal.vue', () => {
  const dummyProfiles = [
    {
      id: 'prof-1',
      name: 'October Placement',
      deadline_at: Math.floor(Date.now() / 1000) + 86400 * 17,
      pending_tasks: 2,
      pace: { feasibility_status: 'ON_TRACK' },
    },
    {
      id: 'prof-2',
      name: 'Kannada Basics',
      deadline_at: 0,
      pending_tasks: 5,
      pace: null,
    },
  ]

  it('renders correctly when open with multiple profiles', () => {
    const wrapper = mount(ProfileLauncherModal, {
      props: {
        isOpen: true,
        profiles: dummyProfiles,
        activeProfileId: 'prof-1',
        askProfileOnStartup: true,
      },
      global: {
        stubs: {
          Teleport: true,
          BaseIcon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Who is studying today?')
    expect(wrapper.text()).toContain('October Placement')
    expect(wrapper.text()).toContain('Kannada Basics')
    expect(wrapper.text()).toContain('17d left')
    expect(wrapper.text()).toContain('No target exam')
    expect(wrapper.text()).toContain('2 pending study tasks')
    expect(wrapper.text()).toContain('5 pending study tasks')
  })

  it('emits select-profile when a profile card or button is clicked', async () => {
    const wrapper = mount(ProfileLauncherModal, {
      props: {
        isOpen: true,
        profiles: dummyProfiles,
        activeProfileId: 'prof-1',
        askProfileOnStartup: true,
      },
      global: {
        stubs: {
          Teleport: true,
          BaseIcon: true,
        },
      },
    })

    const cards = wrapper.findAll('.profile-picker-card')
    expect(cards.length).toBe(2)

    await cards[1].trigger('click')
    expect(wrapper.emitted('select-profile')).toBeTruthy()
    expect(wrapper.emitted('select-profile')[0]).toEqual(['prof-2'])
  })

  it('emits update-startup-preference when toggling checkbox', async () => {
    const wrapper = mount(ProfileLauncherModal, {
      props: {
        isOpen: true,
        profiles: dummyProfiles,
        activeProfileId: 'prof-1',
        askProfileOnStartup: true,
      },
      global: {
        stubs: {
          Teleport: true,
          BaseIcon: true,
        },
      },
    })

    const checkbox = wrapper.find('.checkbox-input')
    expect(checkbox.exists()).toBe(true)
    // Initially unchecked because askProfileOnStartup is true
    expect(checkbox.element.checked).toBe(false)

    // User checks "Don't ask on startup" -> updates askProfileOnStartup to false
    await checkbox.setValue(true)
    expect(wrapper.emitted('update-startup-preference')).toBeTruthy()
    expect(wrapper.emitted('update-startup-preference')[0]).toEqual([false])
  })

  it('emits select-profile on number keypress (1 or 2)', async () => {
    const wrapper = mount(ProfileLauncherModal, {
      props: {
        isOpen: true,
        profiles: dummyProfiles,
        activeProfileId: 'prof-1',
        askProfileOnStartup: true,
      },
      global: {
        stubs: {
          Teleport: true,
          BaseIcon: true,
        },
      },
    })

    window.dispatchEvent(new KeyboardEvent('keydown', { key: '2' }))
    expect(wrapper.emitted('select-profile')).toBeTruthy()
    expect(wrapper.emitted('select-profile')[0]).toEqual(['prof-2'])
  })
})
