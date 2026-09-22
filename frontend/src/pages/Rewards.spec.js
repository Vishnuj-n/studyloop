import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import Rewards from './Rewards.vue'
import * as appApi from '../services/appApi'

vi.mock('../services/appApi', () => ({
  getGamificationState: vi.fn(),
  buyStreakFreeze: vi.fn(),
  getUserSettings: vi.fn(),
  updateUserSettings: vi.fn(),
  getGamificationStore: vi.fn(),
  unlockCosmeticItem: vi.fn(),
}))

const mockSettings = {
  max_flashcards_per_session: 30,
  study_start_time: '17:00',
  study_end_time: '18:00',
  reminders_enabled: true,
  active_profile_id: 'profile-123',
  theme: 'dark-gruvbox',
}

const mockGamificationState = {
  profile: {
    level: 2,
    total_xp: 500,
    coins: 100,
    current_title: 'The Apprentice I',
    next_title: 'The Scholar I',
    next_title_xp: 1500,
    streak_freezes_owned: 1,
    last_freeze_purchased_at: 0,
  },
  pending_chests: [],
}

describe('Rewards.vue Theme Handling', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
    appApi.getGamificationState.mockResolvedValue(mockGamificationState)
    appApi.getUserSettings.mockResolvedValue(mockSettings)
    appApi.updateUserSettings.mockResolvedValue({ success: true })
  })

  it('persists full merged user settings to the backend when theme changes', async () => {
    const wrapper = mount(Rewards, {
      global: {
        stubs: {
          StudyPageLayout: { template: '<div><slot /></div>' },
          RewardsShopModal: {
            name: 'RewardsShopModal',
            props: ['activeTheme'],
            template: '<div class="shop-stub"><button class="change-theme-btn" @click="$emit(\'theme-changed\', \'dark-emerald\')">Equip</button></div>',
          },
          MysteryChestModal: true,
          StreakFreezeModal: true,
          GamificationIcon: true,
        },
      },
    })

    await flushPromises()

    // Trigger theme change from shop component
    const shopStub = wrapper.findComponent({ name: 'RewardsShopModal' })
    await shopStub.find('.change-theme-btn').trigger('click')
    await flushPromises()

    // Verify updateUserSettings was called with full merged settings object containing all existing fields
    expect(appApi.updateUserSettings).toHaveBeenCalledTimes(1)
    expect(appApi.updateUserSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        max_flashcards_per_session: 30,
        study_start_time: '17:00',
        active_profile_id: 'profile-123',
        theme: 'dark-emerald',
      })
    )

    // Verify localStorage and DOM were updated
    expect(localStorage.getItem('app-theme')).toBe('dark-emerald')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark-emerald')
  })
})
