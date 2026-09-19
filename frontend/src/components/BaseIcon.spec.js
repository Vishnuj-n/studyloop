import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseIcon from './BaseIcon.vue'

describe('BaseIcon', () => {
  it('renders known icon properly with default size and attributes', () => {
    const wrapper = mount(BaseIcon, {
      props: {
        name: 'settings',
      },
    })
    const svg = wrapper.find('svg')
    expect(svg.exists()).toBe(true)
    expect(svg.attributes('width')).toBe('16')
    expect(svg.attributes('height')).toBe('16')
    expect(svg.attributes('viewBox')).toBe('0 0 24 24')
    expect(svg.attributes('aria-hidden')).toBe('true')
    expect(svg.html()).toContain('<circle cx="12" cy="12" r="3"></circle>')
  })

  it('renders custom size and accessibility label', () => {
    const wrapper = mount(BaseIcon, {
      props: {
        name: 'wrench',
        size: 24,
        ariaLabel: 'Repair tool',
        customClass: 'custom-spin',
      },
    })
    const svg = wrapper.find('svg')
    expect(svg.attributes('width')).toBe('24')
    expect(svg.attributes('height')).toBe('24')
    expect(svg.attributes('aria-label')).toBe('Repair tool')
    expect(svg.attributes('role')).toBe('img')
    expect(svg.classes()).toContain('custom-spin')
  })

  it('falls back gracefully on unknown icon name', () => {
    const wrapper = mount(BaseIcon, {
      props: {
        name: 'non-existent-icon',
      },
    })
    const svg = wrapper.find('svg')
    expect(svg.exists()).toBe(true)
    expect(svg.html()).toContain('<line')
  })
})
