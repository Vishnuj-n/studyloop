import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import PetSanctuaryView from './PetSanctuaryView.vue'
import * as appApi from '../services/appApi'

vi.mock('../services/appApi', () => ({
  unlockCosmeticItem: vi.fn(),
}))

describe('PetSanctuaryView.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('renders companions with Mochi free by default and Buster locked', async () => {
    const wrapper = mount(PetSanctuaryView, {
      props: {
        coins: 500,
      },
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Pet Sanctuary & Companions')
    expect(wrapper.text()).toContain('Mochi')
    expect(wrapper.text()).toContain('Buster')
    expect(wrapper.text()).toContain('Ignis')

    // Find dog select card
    const cards = wrapper.findAll('.pet-select-card')
    const dogCard = cards.find((c) => c.text().includes('Buster'))
    expect(dogCard.classes()).toContain('locked')
  })

  it('allows unlocking Buster when user has sufficient coins', async () => {
    appApi.unlockCosmeticItem.mockResolvedValue({ success: true })

    const wrapper = mount(PetSanctuaryView, {
      props: {
        coins: 1500,
      },
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    await flushPromises()

    // Select Buster
    const cards = wrapper.findAll('.pet-select-card')
    const dogCard = cards.find((c) => c.text().includes('Buster'))
    await dogCard.trigger('click')
    await flushPromises()

    // Click Adopt button
    const adoptBtn = wrapper.find('.primary-adopt-btn')
    expect(adoptBtn.exists()).toBe(true)
    await adoptBtn.trigger('click')
    await flushPromises()

    expect(appApi.unlockCosmeticItem).toHaveBeenCalledWith('pet:dog', 1000)
    expect(wrapper.emitted('coins-updated')).toBeTruthy()
  })
})
