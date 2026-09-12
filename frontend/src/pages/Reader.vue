<template>
  <section class="page">
    <header class="head">
      <p class="eyebrow">Reader</p>
      <h1>{{ reader.topicTitle.value }}</h1>
      <p class="meta">
        <span>{{ reader.sections.value.length }} sections</span>
        <span v-if="reader.selectedNotebookTitle.value"
          >Notebook: {{ reader.selectedNotebookTitle.value }}</span
        >
        <span v-if="isTaskFlow" class="task-badge">Task Mode</span>
        <span v-else class="browse-badge">Browse Mode</span>
      </p>
    </header>

    <!-- Browse Mode: Notebook/Topic Selection -->
    <article v-if="!isTaskFlow" class="panel controls">
      <NotebookTopicSelector
        v-model:notebook-id="reader.selectedNotebookID.value"
        v-model:topic-id="reader.selectedTopicID.value"
        :notebook-tree="reader.notebookTree.value"
        :disabled="reader.loadingTree.value || reader.loadingBundle.value"
        variant="fields"
        @change-notebook="onNotebookChange"
        @change-topic="reader.loadBundle"
      />
    </article>

    <article v-if="reader.globalError.value" class="panel error fatal-error">
      <h3>Reader Initialization Error</h3>
      <p>{{ reader.globalError.value }}</p>
      <div class="error-actions">
        <button class="secondary" @click="router.push('/dashboard')">Back to Dashboard</button>
        <button class="primary" @click="reloadPage()">Retry</button>
      </div>
    </article>

    <!-- Empty State: No Topic Chosen / Flashcards Notebook -->
    <div v-else-if="!isTaskFlow && !reader.selectedTopicID.value" class="empty">
      {{ reader.availableTopics.value.length === 0 && reader.selectedNotebookID.value ? 'This notebook has no reading chapters (e.g. Flashcards notebook).' : 'No chapter chosen.' }}
    </div>

    <div v-else class="layout" :class="{ collapsed: chat.chatCollapsed.value }">
      <article class="panel stage">
        <!-- Scroll Progress Bar -->
        <div class="scroll-progress-bar" aria-hidden="true">
          <div class="scroll-progress-fill" :style="{ width: `${progressPercentage}%` }"></div>
        </div>

        <div class="stage-head">
          <div class="stage-head-left">
            <span class="page-indicator"
              >Page {{ reader.currentPage.value }} / {{ reader.pageCount.value }}</span
            >
            <button
              v-if="isTaskFlow"
              class="primary"
              :disabled="!resolvedTaskID || reader.loadingBundle.value || completingSession"
              @click="completeSession"
            >
              {{ completingSession ? 'Completing Session...' : 'Complete Session' }}
            </button>
            <button
              class="secondary copy-session-btn"
              :disabled="reader.loadingBundle.value || reader.loadingText.value"
              title="Copy reading session text as Markdown"
              @click="copySessionContent"
            >
              {{ copiedSession ? 'Copied to Clipboard! ✓' : '📋 Copy Session' }}
            </button>
            <button
              v-if="isExtensionActive('audio_overview')"
              class="secondary audio-overview-btn"
              :class="{ active: showAudioOverview }"
              :disabled="reader.loadingBundle.value || reader.loadingText.value || !reader.selectedTopicID.value"
              :title="showAudioOverview ? 'Hide AI Audio' : 'Listen to AI Audio Overview'"
              @click="showAudioOverview = !showAudioOverview"
            >
              {{ showAudioOverview ? '🎧 AI Audio Active' : '🎧 AI Audio Overview' }}
            </button>
            <button
              v-if="isExtensionActive('text_simplifier')"
              class="secondary simplify-btn"
              :disabled="reader.loadingBundle.value || simplifying"
              title="Open intuitive AI simplified breakdown on a dedicated Markdown reading screen"
              @click="handleSimplify"
            >
              {{ simplifying ? '✨ Opening...' : '✨ Simplify' }}
            </button>
            <span v-if="copyError" class="copy-error-msg" style="color: #b42318; font-size: 12px; font-weight: 500;">
              {{ copyError }}
            </span>
          </div>
          <div v-if="isTaskFlow" class="stage-head-right">
            <span v-if="reader.hasNavigationBounds.value" class="reading-window-info">
              Reading Window: Pages {{ reader.navigationMinPage.value }}-{{
                reader.navigationMaxPage.value
              }}
            </span>
            <button
              class="secondary skip-session-btn"
              :disabled="!resolvedTaskID || reader.loadingBundle.value || completingSession"
              title="Skip this session — pages marked as read, no quiz generated"
              @click="skipSession"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="trash-icon"
              >
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                <line x1="10" y1="11" x2="10" y2="17"></line>
                <line x1="14" y1="11" x2="14" y2="17"></line>
              </svg>
            </button>
          </div>
        </div>

        <div v-if="reader.loadingBundle.value || reader.loadingText.value" class="empty">
          Loading document...
        </div>
        <YouTubeReader
          v-else-if="reader.isYouTube.value"
          :embed-url="reader.youtubeEmbedUrl.value"
          :cached-video-url="reader.cachedVideoUrl.value"
          :transcript-content="reader.textContent.value"
          :topic-title="reader.topicTitle.value"
          :start-page="reader.navigationMinPage.value || reader.topicStartPage.value || 1"
          :end-page="reader.navigationMaxPage.value || reader.topicEndPage.value || 1"
          :is-task-flow="isTaskFlow"
          :completing="completingSession"
          :disabled="!resolvedTaskID || reader.loadingBundle.value || completingSession"
          @complete="completeSession"
        />
        <MarkdownReader
          v-else-if="reader.isMarkdown.value"
          :content="reader.textContent.value"
          :topic-title="reader.topicTitle.value"
          :start-page="reader.navigationMinPage.value || reader.topicStartPage.value || 1"
          :end-page="reader.navigationMaxPage.value || reader.topicEndPage.value || 1"
          :is-task-flow="isTaskFlow"
          :completing="completingSession"
          :disabled="!resolvedTaskID || reader.loadingBundle.value || completingSession"
          @complete="completeSession"
        />
        <div v-else-if="!reader.pdfVisible.value" class="empty">
          Document not available for selected notebook/topic.
        </div>
        <PdfViewer
          v-else
          ref="pdfViewerRef"
          :source="reader.notebookUrl.value"
          :page-count="reader.pageCount.value"
          :initial-page="reader.currentPage.value"
          :zoom-scale="zoomScale"
          @update:current-page="reader.updateCurrentPage"
          @load-error="handlePDFLoadFailed"
          @rendered="onPdfRendered"
        />

        <!-- Right-edge PDF Controls -->
        <div
          v-if="reader.pdfVisible.value && !reader.loadingBundle.value"
          class="pdf-edge-controls"
        >
          <button
            class="edge-btn zoom-btn"
            :disabled="zoomScale <= 0.5"
            title="Zoom out"
            @click="zoomOut"
          >
            −
          </button>
          <span class="edge-zoom-val">{{ Math.round(zoomScale * 100) }}%</span>
          <button
            class="edge-btn zoom-btn"
            :disabled="zoomScale >= 2.5"
            title="Zoom in"
            @click="zoomIn"
          >
            +
          </button>
        </div>

        <p v-if="isTaskFlow && completionMessage" class="completion-message">
          {{ completionMessage }}
        </p>
        <p v-if="isTaskFlow && completionError" class="error">{{ completionError }}</p>
      </article>

      <ReaderChat
        v-if="ragEnabled && ragQueueStudy"
        :selected-topic-i-d="reader.selectedTopicID.value"
        :selected-topic-title="reader.selectedTopicTitle.value"
        :selected-notebook-i-d="reader.selectedNotebookID.value"
        :selected-notebook-title="reader.selectedNotebookTitle.value"
        :current-page="reader.currentPage.value"
        :topic-start-page="reader.topicStartPage.value"
        :topic-end-page="reader.topicEndPage.value"
        :rag-enabled="ragEnabled"
        :rag-settings-loaded="ragSettingsLoaded"
        :rag-settings-error="ragSettingsError"
        @retry-settings="retryGetUserSettings"
      />
      <div v-else-if="ragEnabled && !ragQueueStudy" class="chat-disabled">
        Chat is currently disabled in queue study mode.
      </div>
    </div>

    <!-- Floating Audio Overview Bar -->
    <AudioOverviewBar
      v-if="showAudioOverview && reader.selectedTopicID.value"
      :topic-id="reader.selectedTopicID.value"
      :notebook-id="reader.selectedNotebookID.value"
      :start-page="reader.navigationMinPage.value || reader.topicStartPage.value || 0"
      :end-page="reader.navigationMaxPage.value || reader.topicEndPage.value || 0"
      :topic-title="reader.topicTitle.value"
      @close="showAudioOverview = false"
    />

  </section>
</template>

<script setup>
import { computed, onMounted, provide, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  completeReading,
  getUserSettings,
  logFrontendEvent,
  trackAnalyticsEvent,
  getTopicSectionsContent,
  skipReadingTask,
} from '../services/appApi'
import { useReaderBase, cleanTopicTitle } from '../composables/useReaderBase'
import { useChat } from '../composables/useChat'
import { useToast } from '../composables/useToast'
import { useExtensions } from '../composables/useExtensions'
import { useDialog } from '../composables/useDialog'
import NotebookTopicSelector from '../components/NotebookTopicSelector.vue'
import ReaderChat from '../components/ReaderChat.vue'
import MarkdownReader from '../components/MarkdownReader.vue'
import YouTubeReader from '../components/YouTubeReader.vue'
import AudioOverviewBar from '../components/AudioOverviewBar.vue'
import PdfViewer from '../components/PdfViewer.vue'

const { isExtensionActive } = useExtensions()
const { confirm } = useDialog()
const showAudioOverview = ref(false)
const simplifying = ref(false)

async function handleSimplify() {
  if (simplifying.value) return
  simplifying.value = true
  try {
    const startPage =
      reader.navigationMinPage.value ||
      reader.topicStartPage.value ||
      reader.currentPage.value ||
      1
    const endPage =
      reader.navigationMaxPage.value ||
      reader.topicEndPage.value ||
      reader.pageCount.value ||
      startPage
    const bookTitle = reader.selectedNotebookTitle.value || 'Notebook'
    const rawTopic = reader.topicTitle.value || reader.selectedTopicTitle.value || 'Reading Session'
    const topicTitle = cleanTopicTitle(rawTopic)

    let sessionText = ''
    if (Array.isArray(reader.sections.value) && reader.sections.value.length > 0) {
      const rangeSections = reader.sections.value.filter(
        (s) => !s.page_num || (s.page_num >= startPage && s.page_num <= endPage)
      )
      sessionText = (rangeSections.length > 0 ? rangeSections : reader.sections.value)
        .map((s) => s.content || s.text || '')
        .filter(Boolean)
        .join('\n\n')
    }
    if (!sessionText) {
      sessionText = reader.textContent.value || ''
    }
    if (!sessionText.trim() && reader.selectedTopicID.value) {
      const bundle = await getTopicSectionsContent(
        reader.selectedTopicID.value,
        reader.selectedNotebookID.value
      )
      if (bundle?.content || bundle?.sections_content) {
        sessionText = bundle.content || bundle.sections_content
      }
    }

    if (!sessionText.trim()) {
      showError('No text available to simplify.')
      return
    }

    try {
      sessionStorage.setItem('simplify_session_data', JSON.stringify({
        bookTitle,
        topicTitle,
        startPage,
        endPage,
        text: sessionText.trim(),
        taskId: resolvedTaskID.value,
        topicId: reader.selectedTopicID.value,
        notebookId: reader.selectedNotebookID.value,
      }))
    } catch (e) {
      console.warn('[Reader] Failed to store simplify_session_data in sessionStorage:', e)
    }

    router.push({
      path: '/simplify',
      query: {
        taskId: resolvedTaskID.value || undefined,
        topicId: reader.selectedTopicID.value || undefined,
        notebookId: reader.selectedNotebookID.value || undefined,
        startPage: startPage || undefined,
        endPage: endPage || undefined,
      }
    })
  } catch (err) {
    showError(String(err?.message || err))
  } finally {
    simplifying.value = false
  }
}

const route = useRoute()
const router = useRouter()

// Get task ID from route (task flow only - manual flow deprecated)
const routeTaskID = computed(() => {
  const id = route.query.taskId || route.query.task_id
  return typeof id === 'string' ? id.trim() : ''
})

// Initialize composables
const reader = useReaderBase(routeTaskID)
const chat = useChat()
const { showError } = useToast()
provide('chat', chat)

// Local state for completion
const completingSession = ref(false)
const completionMessage = ref('')
const completionError = ref('')
const sessionTask = ref(null)
const ragEnabled = ref(false)
const ragQueueStudy = ref(true)
const analyticsEnabled = ref(false)
const anonymousUserID = ref('')
const ragSettingsLoaded = ref(false)
const ragSettingsError = ref(null)

const resolvedTaskID = computed(() => {
  return (
    sessionTask.value?.task_id ||
    sessionTask.value?.id ||
    routeTaskID.value ||
    route.query.taskId ||
    route.query.task_id ||
    ''
  )
})

const progressPercentage = computed(() => {
  const current = reader.currentPage.value || 1
  if (reader.hasNavigationBounds.value) {
    const min = reader.navigationMinPage.value || 1
    const max = reader.navigationMaxPage.value || 1
    if (max <= min) return 100
    if (current < min) return 0
    if (current > max) return 100
    return Math.min(100, Math.max(0, ((current - min) / (max - min)) * 100))
  } else {
    const total = reader.pageCount.value || 1
    return Math.min(100, Math.max(0, (current / total) * 100))
  }
})

watch([resolvedTaskID, () => reader.selectedTopicID.value], (next, prev) => {
  console.warn('[READER_STATE] task/topic changed', { previous: prev, next })
})

// PDF Viewer ref and zoom
const pdfViewerRef = ref(null)
const zoomScale = ref(0.7)

const isTaskFlow = computed(() => {
  // Once context is settled, read mode from the context object.
  // Fall back to route query during the initialization window (context not yet set).
  const settled = reader.readerContext.value
  if (settled) return settled.mode === 'task'
  return !!routeTaskID.value
})

// Trust-based completion: user decides when reading is complete.
// Page navigation is for UI only and does not gate completion.

// ─── RAG settings ────────────────────────────────────────────────────────────

async function loadRagSettings() {
  ragSettingsLoaded.value = false
  ragSettingsError.value = null
  try {
    const settings = await getUserSettings()
    ragEnabled.value = settings?.rag_enabled ?? false
    ragQueueStudy.value = settings?.rag_queue_study ?? true
    analyticsEnabled.value = settings?.analytics_enabled ?? false
    anonymousUserID.value = settings?.anonymous_user_id ?? ''
  } catch (err) {
    console.error('Failed to load settings in Reader:', err)
    ragSettingsError.value = err?.message || 'Failed to load settings'
  } finally {
    ragSettingsLoaded.value = true
  }
}

// ─── Entry-path resolvers ─────────────────────────────────────────────────────

async function resolveTaskContext(taskQuery) {
  const init = await reader.initializeSession(taskQuery)
  console.log('[Reader] resolveTaskContext result:', {
    initOk: !!init,
    bundle: init?.bundle ? {
      page_count: init.bundle.page_count,
      file_type: init.bundle.file_type,
      notebook_url: init.bundle.notebook_url,
      topic_title: init.bundle.topic_title,
      sections_count: init.bundle.sections?.length,
    } : 'NULL_BUNDLE',
    readerState: {
      notebookUrl: reader.notebookUrl.value,
      fileType: reader.fileType.value,
      pageCount: reader.pageCount.value,
      currentPage: reader.currentPage.value,
      isPdf: reader.isPdf.value,
      pdfVisible: reader.pdfVisible.value,
      isMarkdown: reader.isMarkdown.value,
      isYouTube: reader.isYouTube.value,
      navigationMin: reader.navigationMinPage.value,
      navigationMax: reader.navigationMaxPage.value,
    },
  })
  if (init) {
    sessionTask.value = init.task
  }
}

async function resolveBrowseContext() {
  console.log('[Reader] Browse mode — resolveBrowseContext')
  await reader.loadNotebookTree()
}

// ─── Mounted ──────────────────────────────────────────────────────────────────

// Initialize on mount
onMounted(async () => {
  await loadRagSettings()

  logFrontendEvent('info', 'ReaderInit', 'mounted', {
    query: route.query,
    routeTaskID: routeTaskID.value,
  })

  if (routeTaskID.value) {
    const taskQuery = {
      notebookId: route.query.notebookId || route.query.notebook_id,
      topicId: route.query.topicId || route.query.topic_id,
      startPage: Number.parseInt(route.query.startPage || route.query.start_page) || 0,
      endPage: Number.parseInt(route.query.endPage || route.query.end_page) || 0,
    }
    await resolveTaskContext(taskQuery)
  } else {
    await resolveBrowseContext()
  }
})

function reloadPage() {
  window.location.reload()
}

function zoomIn() {
  zoomScale.value = Math.min(2.5, Math.round((zoomScale.value + 0.1) * 100) / 100)
}

function zoomOut() {
  zoomScale.value = Math.max(0.5, Math.round((zoomScale.value - 0.1) * 100) / 100)
}

function onPdfRendered(pageNum) {
  // ponytail: no-op hook for future telemetry if needed
}

function handlePDFLoadFailed(err) {
  console.error('[Reader] PDF loading failed:', err)
  const errMsg =
    typeof err === 'string'
      ? err
      : err?.message || (err && JSON.stringify(err)) || 'Failed to load PDF document.'
  logFrontendEvent('error', 'ReaderPDF', 'pdf_load_failed', { error: errMsg })
}

async function retryGetUserSettings() {
  await loadRagSettings()
}

function onNotebookChange() {
  // Clear topic selection when notebook changes to prevent stale topic IDs
  reader.selectedTopicID.value = ''
  // Don't call loadBundle() here - let user select a topic first
}

async function completeSession() {
  if (completingSession.value || reader.loadingBundle.value || !resolvedTaskID.value) return

  completionError.value = ''
  completionMessage.value = ''
  completingSession.value = true

  try {
    const taskIDForCompletion = resolvedTaskID.value
    console.warn('[COMPLETE_SESSION] pre-completeReading ids', {
      routeQueryTaskId: route.query.taskId,
      routeQueryTask_id: route.query.task_id,
      routeTaskIDComputed: routeTaskID.value,
      resolvedTaskID: resolvedTaskID.value,
      actualArg: taskIDForCompletion,
    })
    const done = await completeReading(taskIDForCompletion)
    console.warn('[COMPLETE_SESSION] completeSession() completeReading response', done)
    if (done?.error) {
      completionError.value = done.error
      showError(done.error, 'Session Completion Failed')
      return
    }
    if (done?.rewards) {
      window.dispatchEvent(
        new CustomEvent('study-reward-earned', { detail: { rewards: done.rewards } })
      )
    }
    if (analyticsEnabled.value) {
      const fileHash = reader.readerContext.value?.notebookFileHash || ''
      trackAnalyticsEvent('reading_complete', fileHash, reader.currentPage.value, {
        task_id: taskIDForCompletion,
        anonymous_user_id: anonymousUserID.value,
      }).catch((err) => {
        console.error('[READER] trackAnalyticsEvent reading_complete failed:', err)
      })
    }
    const nextRoute = done?.quiz_task_id ? `/quiz?taskId=${done.quiz_task_id}` : '/dashboard'
    // Completion writes the follow-up quiz into the queue; navigation follows the existing route behavior.
    console.warn('[COMPLETE_SESSION] completeSession() before router.push', {
      nextRoute,
      quizTaskID: done?.quiz_task_id || null,
    })
    await router.push(nextRoute)
    console.warn('[COMPLETE_SESSION] completeSession() router.push resolved', { nextRoute })
  } catch (err) {
    console.error('[COMPLETE_SESSION] completeSession() catch', err)
    const errMsg = err?.message || 'Failed to complete session'
    completionError.value = errMsg
    showError(errMsg, 'Session Completion Failed')
  } finally {
    completingSession.value = false
  }
}

async function skipSession() {
  if (!resolvedTaskID.value) return
  const confirmed = await confirm({
    title: 'Skip this session?',
    message: 'These pages will be marked as read. No quiz will be generated for this session.',
    confirmText: 'Skip Session',
    cancelText: 'Keep Reading',
    type: 'warning',
  })
  if (!confirmed) return
  try {
    const res = await skipReadingTask(resolvedTaskID.value)
    if (res?.error) {
      showError(res.error, 'Skip Failed')
      return
    }
    await router.push('/dashboard')
  } catch (err) {
    showError(err?.message || 'Failed to skip session', 'Skip Failed')
  }
}

// ponytail: clean structured markdown clipboard export
const copiedSession = ref(false)
const copyError = ref('')

async function copySessionContent() {
  const startPage =
    reader.navigationMinPage.value ||
    reader.topicStartPage.value ||
    reader.currentPage.value ||
    1
  const endPage =
    reader.navigationMaxPage.value ||
    reader.topicEndPage.value ||
    reader.pageCount.value ||
    startPage
  const bookTitle = reader.selectedNotebookTitle.value || 'Notebook'
  const rawTopic = reader.topicTitle.value || reader.selectedTopicTitle.value || 'Reading Session'
  const topicTitle = cleanTopicTitle(rawTopic)

  let sessionText = ''
  if (Array.isArray(reader.sections.value) && reader.sections.value.length > 0) {
    const rangeSections = reader.sections.value.filter(
      (s) => !s.page_num || (s.page_num >= startPage && s.page_num <= endPage)
    )
    sessionText = (rangeSections.length > 0 ? rangeSections : reader.sections.value)
      .map((s) => s.content || s.text || '')
      .filter(Boolean)
      .join('\n\n')
  }
  if (!sessionText) {
    sessionText = reader.textContent.value || ''
  }

  const markdown = `# ${bookTitle}
## ${topicTitle} (Pages ${startPage}–${endPage})

${sessionText.trim()}`

  try {
    await navigator.clipboard.writeText(markdown)
    copiedSession.value = true
    copyError.value = ''
    setTimeout(() => {
      copiedSession.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy session content:', err)
    copiedSession.value = false
    copyError.value = 'Failed to copy session content'
  }
}
</script>

<style scoped>
.page {
  display: grid;
  gap: 14px;
}

.head {
  display: grid;
  gap: 6px;
}

.eyebrow {
  margin: 0;
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--muted-text);
  font-weight: 700;
}

h1 {
  margin: 0;
  font-size: 42px;
  font-family: 'Manrope', sans-serif;
  letter-spacing: -0.02em;
}

h3 {
  margin: 0;
  font-size: 18px;
  font-family: 'Manrope', sans-serif;
}

.meta {
  margin: 0;
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  color: var(--muted-text);
}

.task-badge {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  padding: 2px 8px;
  border-radius: 4px;
}

.browse-badge {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted-text);
  background: var(--surface-container-low);
  padding: 2px 8px;
  border-radius: 4px;
}

.panel {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 12px;
}

.controls {
  display: grid;
  grid-template-columns: repeat(2, minmax(220px, 360px));
  gap: 10px;
}

.layout {
  display: grid;
  grid-template-columns: 1.8fr 1fr;
  gap: 12px;
}

.layout.collapsed {
  grid-template-columns: 1fr 78px;
}

.stage {
  display: grid;
  gap: 10px;
  position: relative;
  min-width: 0;
}

.stage-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}

.stage-head-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--muted-text);
}

.skip-session-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 7px 9px;
  color: #ef4444;
  background: color-mix(in srgb, #ef4444 12%, var(--surface-container-low));
  border: 1px solid color-mix(in srgb, #ef4444 28%, transparent);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.skip-session-btn:hover:not(:disabled) {
  background: color-mix(in srgb, #ef4444 24%, var(--surface-container-low));
  color: #dc2626;
  border-color: color-mix(in srgb, #ef4444 45%, transparent);
  transform: translateY(-1px);
}

.skip-session-btn:active:not(:disabled) {
  transform: translateY(0);
}

.skip-session-btn .trash-icon {
  width: 16px;
  height: 16px;
  stroke: currentColor;
}

.stage-head-right {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--muted-text);
}

.reading-window-info {
  white-space: nowrap;
}

.page-indicator {
  font-weight: 600;
  white-space: nowrap;
}



.completion-message {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

.sections {
  display: grid;
  gap: 8px;
}

button {
  border: 0;
  border-radius: 10px;
  padding: 9px 12px;
  font-weight: 700;
  cursor: pointer;
}

button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.primary {
  color: var(--on-primary);
  background: linear-gradient(160deg, var(--primary), var(--primary-dim));
}

.secondary {
  color: var(--on-surface);
  background: var(--surface-container-low);
}

.audio-overview-btn.active {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 14%, var(--surface-container-low));
  border: 1px solid var(--outline-variant);
}

.error {
  color: #b42318;
  background: color-mix(in srgb, #b42318 12%, var(--surface-container-lowest));
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 10px;
  font-size: 13px;
}

.fatal-error {
  display: grid;
  gap: 12px;
  padding: 24px;
  text-align: center;
  border-style: solid;
  border-width: 2px;
  margin: 20px 0;
}

.fatal-error h3 {
  color: #b42318;
  margin: 0;
}

.fatal-error p {
  margin: 0;
  font-size: 15px;
  color: var(--on-surface);
}

.error-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-top: 8px;
}

.empty {
  color: var(--muted-text);
  background: var(--surface-container-low);
  border-radius: 10px;
  padding: 12px;
  font-size: 14px;
}




@media (max-width: 1180px) {
  .layout,
  .layout.collapsed {
    grid-template-columns: 1fr;
  }
}

/* Right-edge PDF Controls */
.pdf-edge-controls {
  position: absolute;
  top: 50%;
  right: 10px;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  background: color-mix(in srgb, var(--surface-bright) 72%, transparent);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
  padding: 10px 8px;
  border-radius: 20px;
  box-shadow: 0 4px 18px rgba(0, 0, 0, 0.1);
  border: 1px solid var(--outline-variant);
  z-index: 10;
  transition: opacity 0.25s ease;
}

.edge-btn {
  background: transparent;
  color: var(--on-surface);
  border: none;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition:
    background 0.18s ease,
    transform 0.12s ease;
  padding: 0;
  font-size: 15px;
  font-weight: 700;
  line-height: 1;
}

.edge-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--surface-container-low) 70%, transparent);
  transform: scale(1.08);
}

.edge-btn:active:not(:disabled) {
  transform: scale(0.94);
}

.edge-btn:disabled {
  opacity: 0.38;
  cursor: not-allowed;
}



.chat-disabled {
  color: var(--muted-text);
  background: var(--surface-container-low);
  border-radius: 10px;
  padding: 12px;
  font-size: 14px;
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
}

.scroll-progress-bar {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 3px;
  background: var(--surface-container-low, rgba(0, 0, 0, 0.05));
  z-index: 10;
  overflow: hidden;
  border-top-left-radius: 13px;
  border-top-right-radius: 13px;
}

.scroll-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-dim), var(--primary));
  transition: width 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.simplify-btn {
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
  color: var(--primary);
  font-weight: 600;
}

.simplify-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--primary) 22%, transparent);
}


</style>
