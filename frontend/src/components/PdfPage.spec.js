import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PdfPage from './PdfPage.vue'

// Stub vue-pdf-embed
vi.mock('vue-pdf-embed', () => ({
  default: {
    name: 'VuePdfEmbed',
    props: ['source', 'page', 'width', 'textLayer', 'annotationLayer'],
    emits: ['rendered', 'loading-failed', 'rendering-failed', 'internal-link-clicked'],
    template: `
      <div class="vue-pdf-embed-stub">
        <button class="trigger-internal-link" @click="$emit('internal-link-clicked', 15)">Jump 15</button>
        <button class="trigger-internal-link-str" @click="$emit('internal-link-clicked', '42')">Jump 42</button>
        <button class="trigger-invalid-link" @click="$emit('internal-link-clicked', -1)">Invalid Jump</button>
        <div class="annotationLayer">
          <section class="linkAnnotation">
            <a href="https://example.com" class="external-link">External Link</a>
          </section>
        </div>
      </div>
    `,
  },
}))

describe('PdfPage.vue', () => {
  it('forwards internal-link-clicked from vue-pdf-embed as navigate event with integer targetPage', async () => {
    const wrapper = mount(PdfPage, {
      props: {
        source: 'blob:test-pdf',
        pageNum: 3,
        width: 800,
      },
    })

    const jumpBtn = wrapper.find('.trigger-internal-link')
    await jumpBtn.trigger('click')

    expect(wrapper.emitted('navigate')).toBeTruthy()
    expect(wrapper.emitted('navigate')[0]).toEqual([{ fromPage: 3, targetPage: 15 }])
  })

  it('parses string target page numbers into integers', async () => {
    const wrapper = mount(PdfPage, {
      props: {
        source: 'blob:test-pdf',
        pageNum: 1,
        width: 800,
      },
    })

    const jumpStrBtn = wrapper.find('.trigger-internal-link-str')
    await jumpStrBtn.trigger('click')

    expect(wrapper.emitted('navigate')).toBeTruthy()
    expect(wrapper.emitted('navigate')[0]).toEqual([{ fromPage: 1, targetPage: 42 }])
  })

  it('ignores invalid or non-positive target pages', async () => {
    const wrapper = mount(PdfPage, {
      props: {
        source: 'blob:test-pdf',
        pageNum: 5,
        width: 800,
      },
    })

    const invalidBtn = wrapper.find('.trigger-invalid-link')
    await invalidBtn.trigger('click')

    expect(wrapper.emitted('navigate')).toBeFalsy()
  })

  it('configures external links with target="_blank" and rel="noopener noreferrer"', async () => {
    const wrapper = mount(PdfPage, {
      props: {
        source: 'blob:test-pdf',
        pageNum: 1,
        width: 800,
      },
    })

    const extLink = wrapper.find('a.external-link')
    await extLink.trigger('click')

    expect(extLink.attributes('target')).toBe('_blank')
    expect(extLink.attributes('rel')).toBe('noopener noreferrer')
  })
})
