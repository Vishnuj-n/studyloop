import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import Reader from './Reader.vue'
import * as appApi from '../services/appApi'

// Mock JSDOM missing browser features
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.IntersectionObserver = class IntersectionObserver {
  constructor(callback) {
    this.callback = callback
  }
  observe(el) {
    if (typeof this.callback === 'function') {
      this.callback([{ target: el, isIntersecting: true }])
    }
  }
  unobserve() {}
  disconnect() {}
}
window.HTMLElement.prototype.scrollIntoView = vi.fn()

const routeQuery = ref({})

// Mock services/appApi
vi.mock('../services/appApi', () => ({
  completeReading: vi.fn(),
  getUserSettings: vi.fn(),
  logFrontendEvent: vi.fn(),
  getNotebookTopicTree: vi.fn(),
  getReaderTopicBundle: vi.fn(),
  initializeReadingSession: vi.fn(),
  askReaderAI: vi.fn(),
}))

// Mock VuePdfEmbed since we cannot load PDF canvas in JSDOM
vi.mock('vue-pdf-embed', () => ({
  default: {
    name: 'VuePdfEmbed',
    template: '<div class="mock-pdf-embed">Mock PDF Content</div>',
  },
}))

// Mock vue-router hooks
vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery.value,
  }),
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

describe('Reader.vue Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routeQuery.value = { taskId: 'task-read-456', notebookId: 'nb-1', topicId: 'topic-1' }

    // Mock settings
    appApi.getUserSettings.mockResolvedValue({
      rag_enabled: true,
    })

    // Mock reading session init to return simple task context satisfying composable validation
    appApi.initializeReadingSession.mockResolvedValue({
      ok: true,
      task: {
        id: 'task-read-456',
        task_type: 'READING',
        notebook_id: 'nb-1',
        topic_id: 'topic-1',
        start_page: 1,
        end_page: 5,
      },
      page_bounds: {
        start_page: 1,
        end_page: 5,
        current_page: 1,
      },
      navigation: {
        some_state: {},
      },
      bundle: {
        topic_title: 'Intro to AI',
        notebook_url: 'http://localhost/test.pdf',
        file_type: 'pdf',
        page_count: 5,
        topic_start_page: 1,
        topic_end_page: 5,
        sections: [],
      },
    })

    // Mock topic tree and bundle
    appApi.getNotebookTopicTree.mockResolvedValue([])
    appApi.getReaderTopicBundle.mockResolvedValue({
      topic_id: 'topic-1',
      title: 'Intro to AI',
      start_page: 1,
      end_page: 5,
    })
  })

  it('initializes reading session and displays PDF viewer placeholder', async () => {
    const wrapper = mount(Reader)
    await flushPromises()

    expect(appApi.initializeReadingSession).toHaveBeenCalledWith(
      'task-read-456',
      'nb-1',
      'topic-1',
      0,
      0
    )
    expect(wrapper.find('.mock-pdf-embed').exists()).toBe(true)
    expect(wrapper.find('button.primary').text()).toBe('Complete Session')
  })

  it('completes reading task when Complete Session is clicked', async () => {
    appApi.completeReading.mockResolvedValue({
      error: null,
      quiz_task_id: 'quiz-next-789',
    })

    const wrapper = mount(Reader)
    await flushPromises()

    const completeBtn = wrapper.find('button.primary')
    await completeBtn.trigger('click')
    await flushPromises()

    expect(appApi.completeReading).toHaveBeenCalledWith('task-read-456')
  })

  it('copies session content as formatted markdown when Copy Session is clicked', async () => {
    const writeTextMock = vi.fn().mockResolvedValue()
    Object.assign(navigator, {
      clipboard: {
        writeText: writeTextMock,
      },
    })

    const wrapper = mount(Reader)
    await flushPromises()

    const copyBtn = wrapper.find('.copy-session-btn')
    expect(copyBtn.exists()).toBe(true)
    expect(copyBtn.text()).toContain('Copy Session')

    await copyBtn.trigger('click')
    await flushPromises()

    expect(writeTextMock).toHaveBeenCalledTimes(1)
    const copiedText = writeTextMock.mock.calls[0][0]
    expect(copiedText).toContain('# Notebook')
    expect(copiedText).toContain('## Intro to AI (Pages 1–5)')
  })

  it('formats raw topic IDs into human readable titles in the header', async () => {
    routeQuery.value = { taskId: 'task-raw-slug', notebookId: 'nb-1', topicId: 'nb-92c8f059-78e2-440c-81e8-62d5032d4330-ch-01-cn-final-revision-sh' }
    appApi.initializeReadingSession.mockResolvedValueOnce({
      ok: true,
      task: { id: 'task-raw-slug', notebook_id: 'nb-1', topic_id: 'nb-92c8f059-78e2-440c-81e8-62d5032d4330-ch-01-cn-final-revision-sh', topic_title: 'nb-92c8f059-78e2-440c-81e8-62d5032d4330-ch-01-cn-final-revision-sh' },
      page_bounds: { start_page: 1, end_page: 1, current_page: 1 },
      navigation_bounds: { start_page: 1, end_page: 1, current_page: 1 },
      navigation: { mode: 'task', current_page: 1, min_page: 1, max_page: 1 },
      bundle: {
        topic_id: 'nb-92c8f059-78e2-440c-81e8-62d5032d4330-ch-01-cn-final-revision-sh',
        topic_title: 'nb-92c8f059-78e2-440c-81e8-62d5032d4330-ch-01-cn-final-revision-sh',
        file_type: 'md',
        sections: [{ id: 's1', heading: 'Section 1', content: 'Sample markdown', page_num: 1 }],
      },
    })

    const wrapper = mount(Reader)
    await flushPromises()

    expect(wrapper.find('h1').text()).toBe('Chapter 1: Cn Final Revision Sh')
  })

  it('renders continuous markdown document with boundary banners and completes session when clicking proceed button', async () => {
    const pushMock = vi.fn()
    vi.mocked(appApi.completeReading).mockResolvedValue({
      error: null,
      quiz_task_id: 'quiz-flow-test-123',
    })

    routeQuery.value = {
      taskId: 'task-md-flow-1',
      notebookId: 'nb-notes',
      topicId: 'nb-notes-ch-01-cn',
    }

    appApi.initializeReadingSession.mockResolvedValueOnce({
      ok: true,
      task: {
        id: 'task-md-flow-1',
        task_type: 'READING',
        notebook_id: 'nb-notes',
        topic_id: 'nb-notes-ch-01-cn',
        start_page: 1,
        end_page: 1,
      },
      page_bounds: { start_page: 1, end_page: 1, current_page: 1 },
      navigation: { mode: 'task', current_page: 1, min_page: 1, max_page: 1 },
      bundle: {
        topic_id: 'nb-notes-ch-01-cn',
        topic_title: 'Chapter 1: Computer Networks',
        file_type: 'md',
        page_count: 1,
        topic_start_page: 1,
        topic_end_page: 1,
        raw_content: '# Course Notes\n\n## 1. OSI Model\n\nApplication layer text\n\n| Layer | Job |\n|---|---|\n| App | Web |',
        sections: [
          {
            id: 'c1',
            heading: 'OSI Model',
            content: '## 1. OSI Model\n\nApplication layer text',
            page_num: 1,
          },
        ],
      },
    })

    const wrapper = mount(Reader, {
      global: {
        mocks: {
          $router: { push: pushMock },
        },
      },
    })
    await flushPromises()

    // 1. Verify markdown viewport is rendered with full continuous content & tables
    const viewport = wrapper.find('.markdown-viewport')
    expect(viewport.exists()).toBe(true)
    expect(viewport.text()).toContain('OSI Model')
    expect(wrapper.find('.top-boundary').exists()).toBe(true)
    expect(wrapper.find('.bottom-boundary').exists()).toBe(true)

    // 2. Find and click the prominent bottom boundary Proceed to Quiz button
    const proceedBtn = wrapper.find('.proceed-button')
    expect(proceedBtn.exists()).toBe(true)
    expect(proceedBtn.text()).toContain('Complete & Proceed to Quiz')

    await proceedBtn.trigger('click')
    await flushPromises()

    // 3. Verify completion call was dispatched with the correct task ID
    expect(appApi.completeReading).toHaveBeenCalledWith('task-md-flow-1')
  })
})
