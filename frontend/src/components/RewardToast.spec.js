import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import RewardToast from './RewardToast.vue'

vi.mock('../services/appApi', () => ({
  getUserSettings: vi.fn().mockResolvedValue({ show_reward_notifications: true }),
}))

describe('RewardToast.vue', () => {
  it('shows lightning bolt ⚡ and hides chest tier tag when rewards do not have a chest drop', async () => {
    const wrapper = mount(RewardToast, {
      props: {
        rewards: {
          xp_earned: 20,
          coins_earned: 5,
        },
      },
      global: {
        stubs: {
          Teleport: true,
        },
      },
    })

    await new Promise((r) => setTimeout(r, 0))

    expect(wrapper.find('.chest-badge-icon').text()).toBe('⚡')
    expect(wrapper.find('.tier-tag').exists()).toBe(false)
    expect(wrapper.text()).toContain('+20 XP')
    expect(wrapper.text()).toContain('+5 Coins')
    expect(wrapper.find('.toast-actions').exists()).toBe(false)
  })

  it('shows chest emoji and tier badge when a mystery chest is dropped', async () => {
    const wrapper = mount(RewardToast, {
      props: {
        rewards: {
          xp_earned: 50,
          coins_earned: 10,
          loot_box: {
            id: 'box-1',
            box_tier: 'GOLD',
          },
        },
      },
      global: {
        stubs: {
          Teleport: true,
        },
      },
    })

    await new Promise((r) => setTimeout(r, 0))

    expect(wrapper.find('.chest-badge-icon').text()).toBe('🎁')
    expect(wrapper.find('.tier-tag').exists()).toBe(true)
    expect(wrapper.find('.tier-tag').text()).toBe('GOLD CHEST')
    expect(wrapper.find('.toast-actions').exists()).toBe(true)
  })
})
