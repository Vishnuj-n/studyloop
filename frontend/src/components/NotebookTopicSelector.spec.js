import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import NotebookTopicSelector from './NotebookTopicSelector.vue'

describe('NotebookTopicSelector.vue', () => {
  const mockTree = [
    {
      notebook_id: 'nb-1',
      title: 'Deep Learning Book',
      topics: [
        {
          topic_id: 'top-3',
          title: 'Chapter 1: 3 Introduction To Ne (Pages 42 to 67)',
          start_page: 42,
          end_page: 67,
        },
        {
          topic_id: 'top-4',
          title: '4 introduction to neural learning: gradient descent (Pages 68 to 99)',
          start_page: 68,
          end_page: 99,
        },
        {
          topic_id: 'top-5',
          title: '5 learning multiple weights at a time (Pages 100 to 119)',
          start_page: 100,
          end_page: 119,
        },
      ],
    },
  ]

  it('disables topic dropdown and displays prompt when no notebook is selected', () => {
    const wrapper = mount(NotebookTopicSelector, {
      props: {
        notebookId: '',
        topicId: '',
        notebookTree: mockTree,
        variant: 'pills',
      },
    })

    const topicSelect = wrapper.find('#topic-select')
    expect(topicSelect.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Choose notebook first')
  })

  it('orders topics chronologically by page numbers instead of regex title bias', () => {
    const wrapper = mount(NotebookTopicSelector, {
      props: {
        notebookId: 'nb-1',
        topicId: 'top-3',
        notebookTree: mockTree,
        variant: 'pills',
      },
    })

    const options = wrapper.findAll('#topic-select option')
    // Option 0: disabled "Choose topic" placeholder
    // Options 1, 2, 3: sorted topics
    expect(options[1].text()).toContain('(Pages 42 to 67)')
    expect(options[2].text()).toContain('(Pages 68 to 99)')
    expect(options[3].text()).toContain('(Pages 100 to 119)')
  })

  it('supports allowEntireNotebook option for Socratic/RAG scope', () => {
    const wrapper = mount(NotebookTopicSelector, {
      props: {
        notebookId: 'nb-1',
        topicId: '',
        notebookTree: mockTree,
        allowEntireNotebook: true,
        variant: 'pills',
      },
    })

    expect(wrapper.text()).toContain('Entire book (No topic filter)')
  })

  it('emits selection changes and resets topic when notebook changes', async () => {
    const wrapper = mount(NotebookTopicSelector, {
      props: {
        notebookId: '',
        topicId: '',
        notebookTree: mockTree,
        variant: 'fields',
      },
    })

    const selects = wrapper.findAll('select')
    await selects[0].setValue('nb-1')

    expect(wrapper.emitted('update:notebookId')).toBeTruthy()
    expect(wrapper.emitted('update:notebookId')[0]).toEqual(['nb-1'])
    expect(wrapper.emitted('update:topicId')).toBeTruthy()
    expect(wrapper.emitted('update:topicId')[0]).toEqual([''])
  })
})
