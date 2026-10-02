import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CompressionBadge from './CompressionBadge.vue'

describe('CompressionBadge.vue', () => {
  it('does not render when is_compressed is false', () => {
    const wrapper = mount(CompressionBadge, {
      props: {
        stats: {
          is_compressed: false,
          raw_tokens: 1000,
          compressed_tokens: 1000,
          tokens_saved: 0,
          saved_percentage: 0,
        },
      },
    })
    expect(wrapper.find('.compression-badge-wrapper').exists()).toBe(false)
  })

  it('renders pill button and toggle popover when is_compressed is true', async () => {
    const wrapper = mount(CompressionBadge, {
      props: {
        stats: {
          is_compressed: true,
          raw_tokens: 1250,
          compressed_tokens: 987,
          tokens_saved: 263,
          saved_percentage: 21.04,
        },
      },
    })

    expect(wrapper.find('.compression-pill-btn').exists()).toBe(true)
    expect(wrapper.text()).toContain('Compressed')
    expect(wrapper.text()).toContain('-21%')

    // Popover is initially closed
    expect(wrapper.find('.compression-popover-card').exists()).toBe(false)

    // Click to open popover
    await wrapper.find('.compression-pill-btn').trigger('click')
    expect(wrapper.find('.compression-popover-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('1,250 tokens')
    expect(wrapper.text()).toContain('987 tokens')
    expect(wrapper.text()).toContain('263 tokens pruned')
    expect(wrapper.text()).toContain('21% reduced')

    // Close button
    await wrapper.find('.popover-close-btn').trigger('click')
    expect(wrapper.find('.compression-popover-card').exists()).toBe(false)
  })
})
