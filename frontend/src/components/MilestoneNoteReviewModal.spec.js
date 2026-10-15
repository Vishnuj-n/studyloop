import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import MilestoneNoteReviewModal from './MilestoneNoteReviewModal.vue'

const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

describe('MilestoneNoteReviewModal', () => {
  it('renders correctly when show is true', () => {
    const wrapper = mount(MilestoneNoteReviewModal, {
      props: {
        show: true,
        topics: [{ topic_id: 'top-1', title: 'Calculus Derivatives' }],
        notebookId: 'nb-123',
        taskId: 'task-456',
      },
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Prepare for Milestone Exam')
    expect(wrapper.text()).toContain('Calculus Derivatives')
    expect(wrapper.text()).toContain('Review Notes')
    expect(wrapper.text()).toContain('Start Exam')
  })

  it('routes to /notes with chapter parameters when Review Notes is clicked', async () => {
    mockPush.mockClear()
    const wrapper = mount(MilestoneNoteReviewModal, {
      props: {
        show: true,
        topics: [{ topic_id: 'top-1', title: 'Calculus Derivatives' }],
        notebookId: 'nb-123',
        taskId: 'task-456',
      },
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    const reviewBtn = wrapper.find('.review-note-btn')
    expect(reviewBtn.exists()).toBe(true)
    await reviewBtn.trigger('click')

    expect(mockPush).toHaveBeenCalledWith({
      path: '/notes',
      query: {
        notebookId: 'nb-123',
        topicId: 'top-1',
        from: 'quiz',
        taskId: 'task-456',
      },
    })
  })

  it('emits start-exam when Start Exam is clicked', async () => {
    const wrapper = mount(MilestoneNoteReviewModal, {
      props: {
        show: true,
        topics: [{ topic_id: 'top-1', title: 'Calculus Derivatives' }],
      },
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    const startBtn = wrapper.find('.start-exam-btn')
    await startBtn.trigger('click')
    expect(wrapper.emitted('start-exam')).toBeTruthy()
  })

  it('emits start-exam when close button is clicked', async () => {
    const wrapper = mount(MilestoneNoteReviewModal, {
      props: {
        show: true,
        topics: [{ topic_id: 'top-1', title: 'Calculus Derivatives' }],
      },
      global: {
        stubs: {
          BaseIcon: true,
        },
      },
    })

    const closeBtn = wrapper.find('.close-btn')
    await closeBtn.trigger('click')
    expect(wrapper.emitted('start-exam')).toBeTruthy()
  })
})
