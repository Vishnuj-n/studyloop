import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PetAvatar from './PetAvatar.vue'

describe('PetAvatar.vue', () => {
  it('renders Cat (Mochi) model with default props', () => {
    const wrapper = mount(PetAvatar, {
      props: {
        petId: 'cat',
      },
    })
    expect(wrapper.find('.pet-model-cat').exists()).toBe(true)
    expect(wrapper.find('.pet-model-dog').exists()).toBe(false)
    expect(wrapper.find('.pet-model-fox').exists()).toBe(false)
  })

  it('renders Dog (Buster) and Fox (Rusty) models properly', () => {
    const dogWrapper = mount(PetAvatar, {
      props: {
        petId: 'dog',
      },
    })
    expect(dogWrapper.find('.pet-model-dog').exists()).toBe(true)

    const foxWrapper = mount(PetAvatar, {
      props: {
        petId: 'fox',
      },
    })
    expect(foxWrapper.find('.pet-model-fox').exists()).toBe(true)
  })

  it('applies custom skin colors', () => {
    const customSkin = {
      id: 'void',
      primaryColor: '#1E293B',
      secondaryColor: '#475569',
      accentColor: '#0F172A',
    }
    const wrapper = mount(PetAvatar, {
      props: {
        petId: 'cat',
        skin: customSkin,
      },
    })
    const body = wrapper.find('.pet-body')
    expect(body.attributes('fill')).toBe('#1E293B')
  })

  it('renders coffee mug and sleep/blink actions', () => {
    const coffeeWrapper = mount(PetAvatar, {
      props: {
        petId: 'dog',
        action: 'coffee',
      },
    })
    expect(coffeeWrapper.find('.pet-coffee-mug').exists()).toBe(true)

    const sleepWrapper = mount(PetAvatar, {
      props: {
        petId: 'dog',
        action: 'sleep',
      },
    })
    expect(sleepWrapper.find('.eyes-sleeping').exists()).toBe(true)
  })
})
