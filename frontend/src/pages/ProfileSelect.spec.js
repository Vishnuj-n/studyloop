import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ProfileSelect from './ProfileSelect.vue'

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

const mockPush = vi.fn()
const mockReplace = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
    replace: mockReplace,
  }),
}))

vi.mock('../services/appApi', () => ({
  getProfiles: vi.fn().mockResolvedValue({ profiles: dummyProfiles }),
  getUserSettings: vi.fn().mockResolvedValue({
    active_profile_id: 'prof-1',
    ask_profile_on_startup: true,
  }),
  setActiveProfileID: vi.fn().mockResolvedValue({ success: true }),
  updateUserSettings: vi.fn().mockResolvedValue({ success: true }),
}))

describe('ProfileSelect.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    sessionStorage.clear()
  })

  it('renders correctly with profiles loaded', async () => {
    const wrapper = mount(ProfileSelect, {
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    // Wait for loadData promise ticks
    await new Promise((r) => setTimeout(r, 10))

    expect(wrapper.text()).toContain('Who is studying today?')
    expect(wrapper.text()).toContain('October Placement')
    expect(wrapper.text()).toContain('Kannada Basics')
    expect(wrapper.text()).toContain('17d left')
    expect(wrapper.text()).toContain('No target exam')
    expect(wrapper.text()).toContain('2 pending study tasks')
    expect(wrapper.text()).toContain('5 pending study tasks')
  })

  it('navigates to dashboard upon selecting a profile', async () => {
    const wrapper = mount(ProfileSelect, {
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    await new Promise((r) => setTimeout(r, 10))

    const cards = wrapper.findAll('.profile-picker-card:not(.add-profile-card)')
    expect(cards.length).toBe(2)

    await cards[1].trigger('click')
    await new Promise((r) => setTimeout(r, 10))

    expect(mockReplace).toHaveBeenCalledWith('/dashboard')
    expect(localStorage.getItem('studyloop_cached_active_profile_id')).toBe('prof-2')
    expect(sessionStorage.getItem('studyloop_profile_picker_shown')).toBe('true')
  })

  it('handles number keypress (1 or 2) to select profile', async () => {
    mount(ProfileSelect, {
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    await new Promise((r) => setTimeout(r, 10))

    window.dispatchEvent(new KeyboardEvent('keydown', { key: '2' }))
    await new Promise((r) => setTimeout(r, 10))

    expect(mockReplace).toHaveBeenCalledWith('/dashboard')
    expect(localStorage.getItem('studyloop_cached_active_profile_id')).toBe('prof-2')
  })

  it('navigates to settings on Manage Profiles click', async () => {
    const wrapper = mount(ProfileSelect, {
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    await new Promise((r) => setTimeout(r, 10))

    const manageBtn = wrapper.find('.manage-profiles-link')
    expect(manageBtn.exists()).toBe(true)
    await manageBtn.trigger('click')

    expect(mockPush).toHaveBeenCalledWith('/settings?category=profiles')
  })

  it('navigates to settings with action=add on Add Profile card click', async () => {
    const wrapper = mount(ProfileSelect, {
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    await new Promise((r) => setTimeout(r, 10))

    const addCard = wrapper.find('.add-profile-card')
    expect(addCard.exists()).toBe(true)
    await addCard.trigger('click')

    expect(mockPush).toHaveBeenCalledWith('/settings?category=profiles&action=add')
  })
})
