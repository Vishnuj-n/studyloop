<template>
  <aside
    id="reader-chat-panel"
    class="panel chat"
    :class="{ closed: chatCollapsed, 'rag-off': !ragEnabled && currentTab === 'chat' }"
  >
    <!-- Top Companion Header / Tab Switcher -->
    <div class="chat-head">
      <div v-if="!chatCollapsed" class="companion-tabs-group">
        <button
          type="button"
          class="companion-tab-btn"
          :class="{ active: currentTab === 'chat' }"
          title="AI Reading Assistant"
          @click="setTab('chat')"
        >
          <BaseIcon name="message-square" size="13" />
          <span>AI Chat</span>
        </button>
        <button
          type="button"
          class="companion-tab-btn"
          :class="{ active: currentTab === 'note' }"
          title="Chapter Study Note"
          @click="setTab('note')"
        >
          <BaseIcon name="file-text" size="13" />
          <span>Study Note</span>
        </button>
      </div>

      <div v-else class="collapsed-head-label">
        <span class="collapsed-title-text">{{ currentTab === 'note' ? 'Study Note' : 'AI Chat' }}</span>
      </div>

      <div class="chat-head-actions">
        <!-- Actions for Chat Tab -->
        <template v-if="!chatCollapsed && currentTab === 'chat'">
          <router-link
            v-if="selectedNotebookID"
            :to="{
              path: '/tutor',
              query: {
                notebookId: selectedNotebookID,
                topicId: selectedTopicID || '',
              },
            }"
            class="chat-tutor-link"
            title="Open full-screen Socratic Tutor for this topic"
          >
            <span>Socratic Tutor</span>
            <BaseIcon name="external-link" size="11" custom-class="tutor-link-icon" />
          </router-link>
          <button
            v-if="chatMessages && chatMessages.length > 0"
            type="button"
            class="ghost clear-chat-btn"
            title="Clear conversation"
            @click="clearChat"
          >
            <BaseIcon name="refresh" size="12" />
            <span>Clear</span>
          </button>
        </template>

        <!-- Actions for Study Note Tab -->
        <template v-if="!chatCollapsed && currentTab === 'note'">
          <button
            v-if="currentNoteContent"
            type="button"
            class="ghost kb-link-btn"
            title="Open in Knowledge Base"
            @click="goToNotesPage"
          >
            <span>Knowledge Base</span>
            <BaseIcon name="external-link" size="11" />
          </button>
        </template>

        <button
          class="ghost collapse-btn"
          :aria-expanded="!chatCollapsed"
          aria-controls="reader-chat-panel"
          @click="toggleChat"
        >
          {{ chatCollapsed ? 'Expand' : 'Collapse' }}
        </button>
      </div>
    </div>

    <!-- Collapsed State Vertical Pill Clickers -->
    <div v-if="chatCollapsed" class="collapsed-tab-strip">
      <button
        type="button"
        class="collapsed-strip-btn"
        :class="{ active: currentTab === 'chat' }"
        title="Open AI Chat"
        @click="expandToTab('chat')"
      >
        <BaseIcon name="message-square" size="15" />
      </button>
      <button
        type="button"
        class="collapsed-strip-btn"
        :class="{ active: currentTab === 'note' }"
        title="Open Study Note"
        @click="expandToTab('note')"
      >
        <BaseIcon name="file-text" size="15" />
      </button>
    </div>

    <!-- Expanded Body -->
    <template v-if="!chatCollapsed">
      <!-- ================= TAB 1: AI CHAT ================= -->
      <div v-show="currentTab === 'chat'" class="tab-pane-container chat-tab-pane">
        <div v-if="!ragSettingsLoaded" class="rag-disabled-overlay">
          <h3>Loading settings...</h3>
        </div>
        <div v-else-if="ragSettingsError" class="rag-disabled-overlay">
          <div class="lock-icon"><BaseIcon name="alert-triangle" size="24" /></div>
          <h3>Settings Error</h3>
          <p>{{ ragSettingsError }}</p>
          <button class="primary" @click="$emit('retry-settings')">Retry</button>
        </div>
        <div v-else-if="ragQueueStudy === false" class="rag-disabled-overlay">
          <div class="lock-icon"><BaseIcon name="lock" size="24" /></div>
          <h3>Chat Disabled</h3>
          <p>AI Chat is disabled during queue study mode to keep focus on reading.</p>
          <button class="secondary tab-redirect-btn" @click="setTab('note')">View Chapter Note →</button>
        </div>
        <div v-else-if="!ragEnabled" class="rag-disabled-overlay">
          <div class="lock-icon"><BaseIcon name="lock" size="24" /></div>
          <h3>Local AI Retrieval Offline</h3>
          <p>Local semantic search and Q&A is currently disabled to save memory and CPU.</p>
          <router-link to="/settings" class="enable-rag-btn">Enable in Settings</router-link>
        </div>
        <template v-else>
          <!-- Compact Context & Retrieval Scope Bar -->
          <div class="chat-meta-bar">
            <div v-if="displayContextTitle" class="chat-context-pill" :title="contextTooltip">
              <BaseIcon name="book" size="12" custom-class="context-pill-icon" />
              <span class="context-pill-title">{{ displayContextTitle }}</span>
            </div>

            <div class="scope-compact-wrap" title="Select retrieval scope">
              <label for="scope-select" class="visually-hidden">Retrieval Scope</label>
              <select id="scope-select" v-model="chatScope" class="scope-compact-select">
                <option value="entire_notebook">Entire Notebook</option>
                <option value="current_chapter">Current Chapter</option>
                <option value="current_page">Current Page</option>
              </select>
              <BaseIcon name="chevron-down" size="11" custom-class="scope-select-chevron" />
            </div>
          </div>

          <!-- Messages Area / Empty Starter State -->
          <div :ref="setMessagesPaneRef" class="messages">
            <div v-if="!chatMessages || chatMessages.length === 0" class="empty-chat-state">
              <div class="empty-hero">
                <div class="empty-icon-badge">
                  <BaseIcon name="sparkles" size="16" />
                </div>
                <p class="empty-title">Reading Assistant</p>
                <p class="empty-desc">Ask anything about your reading or choose a prompt:</p>
              </div>

              <div class="quick-chips-grid">
                <button
                  v-for="chip in quickPrompts"
                  :key="chip.id"
                  type="button"
                  class="quick-chip-btn"
                  :disabled="chatLoading || !selectedTopicID"
                  @click="triggerQuickPrompt(chip.text)"
                >
                  <BaseIcon :name="chip.icon" size="13" custom-class="chip-icon" />
                  <span class="chip-label">{{ chip.label }}</span>
                </button>
              </div>
            </div>

            <article
              v-for="(msg, idx) in chatMessages"
              :key="msg.id || idx"
              class="msg"
              :class="msg.role"
            >
              <p class="role">{{ msg.role === 'user' ? 'You' : 'Tutor' }}</p>
              <p v-if="msg.role === 'user'">{{ msg.text }}</p>
              <!-- eslint-disable-next-line vue/no-v-html -->
              <div v-else class="markdown-body" v-html="renderMarkdown(msg.text)"></div>
            </article>
          </div>

          <article v-if="chatError" class="error">{{ chatError }}</article>

          <!-- Input Composer -->
          <form class="composer" @submit.prevent="sendChat">
            <div class="composer-box">
              <textarea
                v-model="chatInput"
                class="composer-input"
                :disabled="chatLoading || !selectedTopicID"
                placeholder="Ask about what you’re reading right now..."
                rows="1"
                @keydown.enter="handleEnterKey"
              ></textarea>
              <button
                type="submit"
                class="composer-send-btn"
                :disabled="chatLoading || !chatInput.trim() || !selectedTopicID"
                title="Send question"
              >
                <svg
                  v-if="!chatLoading"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  class="send-svg"
                >
                  <path
                    d="M3.478 2.404a.75.75 0 0 0-.926.941l2.432 7.905H13.5a.75.75 0 0 1 0 1.5H4.984l-2.432 7.905a.75.75 0 0 0 .926.94 60.519 60.519 0 0 0 18.445-8.986.75.75 0 0 0 0-1.218A60.517 60.517 0 0 0 3.478 2.404Z"
                  />
                </svg>
                <span v-else class="thinking-dot-loader">
                  <span></span><span></span><span></span>
                </span>
              </button>
            </div>
            <div class="composer-hint-row">
              <span>Enter to send, Shift+Enter for line break</span>
            </div>
          </form>
        </template>
      </div>

      <!-- ================= TAB 2: STUDY NOTE ================= -->
      <div v-show="currentTab === 'note'" class="tab-pane-container note-tab-pane">
        <!-- Subheader Chapter Label -->
        <div class="note-meta-bar">
          <div class="note-chapter-pill" :title="displayContextTitle">
            <BaseIcon name="book" size="12" custom-class="context-pill-icon" />
            <span class="note-chapter-title">{{ selectedTopicTitle || 'Chapter Study Note' }}</span>
          </div>
          <span v-if="noteLastSaved" class="note-saved-status">Saved</span>
        </div>

        <!-- Note Scroll Body -->
        <div class="note-content-area">
          <div v-if="noteLoading" class="note-state-box">
            <div class="generating-spinner"></div>
            <span>Loading note...</span>
          </div>
          <div v-else-if="noteGenerating" class="note-state-box generating-pulse">
            <div class="generating-spinner"></div>
            <p class="generating-title">Generating Study Note...</p>
            <span class="generating-sub">Extracting key concepts, formulas & takeaways</span>
          </div>
          <div v-else-if="noteEditing" class="note-edit-box">
            <textarea
              v-model="noteEditContent"
              class="reader-note-textarea"
              placeholder="Write chapter summary note (Markdown supported)..."
              rows="16"
            ></textarea>
          </div>
          <div v-else-if="currentNoteContent" class="note-view-box">
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div class="note-markdown" v-html="renderedNoteMarkdown"></div>
          </div>
          <div v-else class="note-empty-box">
            <div class="empty-icon-badge">
              <BaseIcon name="file-text" size="20" />
            </div>
            <p class="empty-title">No Study Note Yet</p>
            <p class="empty-desc">Generate structured AI notes for this chapter or create your own summary.</p>
            <button
              type="button"
              class="primary generate-note-hero-btn"
              :disabled="noteGenerating || !selectedTopicID"
              @click="generateNote"
            >
              <BaseIcon name="sparkles" size="14" />
              <span>Generate Study Note</span>
            </button>
          </div>
        </div>

        <!-- Note Footer Action Bar -->
        <div v-if="!noteLoading" class="note-footer-bar">
          <template v-if="noteEditing">
            <button type="button" class="secondary" :disabled="noteSaving" @click="cancelEdit">Cancel</button>
            <button type="button" class="primary" :disabled="noteSaving" @click="saveNote">
              {{ noteSaving ? 'Saving...' : 'Save Note' }}
            </button>
          </template>
          <template v-else-if="currentNoteContent">
            <button
              type="button"
              class="secondary edit-note-btn"
              :disabled="noteGenerating || noteSaving"
              @click="startEdit"
            >
              <BaseIcon name="edit" size="13" />
              <span>Edit</span>
            </button>
            <button
              type="button"
              class="primary regen-note-btn"
              :disabled="noteGenerating || noteSaving"
              @click="generateNote"
            >
              <BaseIcon name="sparkles" size="13" />
              <span>{{ noteGenerating ? 'Generating...' : 'Regenerate' }}</span>
            </button>
            <button
              type="button"
              class="secondary kb-bottom-btn"
              title="Open full page editor in Knowledge Base"
              @click="goToNotesPage"
            >
              <BaseIcon name="external-link" size="12" />
              <span>Notes Page</span>
            </button>
          </template>
        </div>
      </div>
    </template>
  </aside>
</template>

<script setup>
import { computed, inject, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import BaseIcon from './BaseIcon.vue'
import {
  logFrontendEvent,
  getTopicStudyNote,
  generateTopicStudyNote,
  updateTopicStudyNote,
} from '../services/appApi'

const props = defineProps({
  modelValue: { type: String, default: 'chat' },
  activeTab: { type: String, default: 'chat' },
  selectedTopicID: { type: String, default: '' },
  selectedTopicTitle: { type: String, default: '' },
  selectedNotebookID: { type: String, default: '' },
  selectedNotebookTitle: { type: String, default: '' },
  currentPage: { type: Number, required: true },
  topicStartPage: { type: Number, required: true },
  topicEndPage: { type: Number, required: true },
  ragEnabled: { type: Boolean, required: true },
  ragQueueStudy: { type: Boolean, default: true },
  ragSettingsLoaded: { type: Boolean, required: true },
  ragSettingsError: { type: String, default: null },
})

const emit = defineEmits(['retry-settings', 'update:modelValue', 'update:activeTab', 'tab-changed'])

const router = useRouter()

const chat = inject('chat')
const {
  chatCollapsed,
  chatMessages,
  chatInput,
  chatLoading,
  chatError,
  chatScope,
  toggleChat,
  clearChat,
  sendMessage,
  renderMarkdown,
  setMessagesPaneRef,
  setInput,
  setError,
} = chat

// Tab Management
const internalTab = ref(props.activeTab || props.modelValue || 'chat')
const currentTab = computed({
  get() {
    return props.activeTab || props.modelValue || internalTab.value
  },
  set(val) {
    internalTab.value = val
    emit('update:modelValue', val)
    emit('update:activeTab', val)
    emit('tab-changed', val)
  },
})

function setTab(tab) {
  currentTab.value = tab
  if (tab === 'note') {
    void ensureNoteLoaded()
  }
}

function expandToTab(tab) {
  if (chatCollapsed.value) {
    toggleChat()
  }
  setTab(tab)
}

// Study Note State
const noteLoading = ref(false)
const noteGenerating = ref(false)
const noteSaving = ref(false)
const noteEditing = ref(false)
const noteLastSaved = ref(false)
const noteEditContent = ref('')
const currentNoteContent = ref('')
const loadedTopicID = ref('')

const renderedNoteMarkdown = computed(() => {
  return renderMarkdown(currentNoteContent.value || '')
})

async function fetchTopicNote(topicId) {
  if (!topicId) {
    currentNoteContent.value = ''
    loadedTopicID.value = ''
    return
  }
  noteLoading.value = true
  noteEditing.value = false
  try {
    const res = await getTopicStudyNote(topicId)
    if (res && res.note && res.note.content) {
      currentNoteContent.value = res.note.content
    } else {
      currentNoteContent.value = ''
    }
    loadedTopicID.value = topicId
  } catch (err) {
    console.warn('[ReaderChat Note] Failed to fetch note:', err)
    currentNoteContent.value = ''
    loadedTopicID.value = topicId
  } finally {
    noteLoading.value = false
  }
}

async function ensureNoteLoaded() {
  const tid = props.selectedTopicID
  if (tid && loadedTopicID.value !== tid) {
    await fetchTopicNote(tid)
  }
}

function startEdit() {
  noteEditContent.value = currentNoteContent.value || ''
  noteEditing.value = true
}

function cancelEdit() {
  noteEditing.value = false
  noteEditContent.value = ''
}

async function saveNote() {
  const tid = props.selectedTopicID
  if (!tid) return
  noteSaving.value = true
  try {
    const res = await updateTopicStudyNote(tid, noteEditContent.value)
    if (res && res.note) {
      currentNoteContent.value = res.note.content
    } else {
      currentNoteContent.value = noteEditContent.value
    }
    noteEditing.value = false
    noteLastSaved.value = true
    setTimeout(() => {
      noteLastSaved.value = false
    }, 2500)
    logFrontendEvent('info', 'ReaderChat', 'study_note_saved', { topicID: tid })
  } catch (err) {
    console.error('[ReaderChat Note] Failed to save note:', err)
  } finally {
    noteSaving.value = false
  }
}

async function generateNote() {
  const tid = props.selectedTopicID
  const nbid = props.selectedNotebookID
  if (!tid) return
  noteGenerating.value = true
  noteEditing.value = false
  try {
    const res = await generateTopicStudyNote(tid, nbid)
    if (res && res.note) {
      currentNoteContent.value = res.note.content
    }
    loadedTopicID.value = tid
    logFrontendEvent('info', 'ReaderChat', 'study_note_generated', { topicID: tid })
  } catch (err) {
    console.error('[ReaderChat Note] Failed to generate note:', err)
  } finally {
    noteGenerating.value = false
  }
}

function goToNotesPage() {
  const tid = props.selectedTopicID
  const nbid = props.selectedNotebookID
  router.push({
    path: '/notes',
    query: { topicId: tid, notebookId: nbid },
  })
}

// Watchers
watch(
  () => props.selectedTopicID,
  (newTid) => {
    if (currentTab.value === 'note' && newTid) {
      void fetchTopicNote(newTid)
    } else {
      loadedTopicID.value = ''
    }
  }
)

watch(
  () => props.activeTab,
  (newTab) => {
    if (newTab && newTab !== internalTab.value) {
      internalTab.value = newTab
      if (newTab === 'note') {
        void ensureNoteLoaded()
      }
    }
  }
)

onMounted(() => {
  if (currentTab.value === 'note' && props.selectedTopicID) {
    void fetchTopicNote(props.selectedTopicID)
  }
})

const displayContextTitle = computed(() => props.selectedTopicTitle || props.selectedNotebookTitle || '')

const contextTooltip = computed(() => {
  if (props.selectedTopicTitle && props.selectedNotebookTitle) {
    return `${props.selectedTopicTitle} from ${props.selectedNotebookTitle}`
  }
  return displayContextTitle.value || ''
})

const quickPrompts = [
  {
    id: 'summary',
    label: 'Summarize Key Points',
    text: 'Summarize the core concepts and key points of this section concisely.',
    icon: 'file-text',
  },
  {
    id: 'explain',
    label: 'Explain Simply',
    text: 'Explain the main ideas in this topic using a simple, intuitive analogy.',
    icon: 'sparkles',
  },
  {
    id: 'quiz',
    label: 'Test My Knowledge',
    text: 'Ask me a conceptual quiz question to test my understanding of this section.',
    icon: 'zap',
  },
  {
    id: 'terms',
    label: 'Key Terminology',
    text: 'List the most important terms and definitions introduced in this reading.',
    icon: 'cards',
  },
]

watch(
  () => props.ragSettingsError,
  (newVal) => {
    if (newVal) logFrontendEvent('error', 'ReaderChat', 'rag_settings_error', { error: newVal })
  }
)

watch(
  () => props.ragEnabled,
  (newVal) => {
    logFrontendEvent('info', 'ReaderChat', 'rag_status_changed', { enabled: newVal })
  }
)

async function sendChat() {
  try {
    await sendMessage({
      topicID: props.selectedTopicID,
      notebookID: props.selectedNotebookID,
      currentPage: props.currentPage,
      chapterStartPage: props.topicStartPage,
      chapterEndPage: props.topicEndPage,
    })
  } catch (err) {
    setError(err?.message || 'Failed to send message')
  }
}

async function triggerQuickPrompt(text) {
  if (chatLoading.value || !props.selectedTopicID) return
  setInput(text)
  await sendChat()
}

function handleEnterKey(event) {
  if (event.shiftKey || event.isComposing) return
  event.preventDefault()
  if (!chatLoading.value && chatInput.value.trim() && props.selectedTopicID) {
    void sendChat()
  }
}
</script>

<style scoped>
.panel {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 12px;
}

.chat {
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  min-height: 0;
  box-sizing: border-box;
}

.chat.closed {
  padding: 10px 8px;
  gap: 8px;
}

.chat.closed .chat-head {
  flex-direction: column;
  gap: 8px;
}

.collapsed-head-label {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px 0;
}

.collapsed-title-text {
  writing-mode: vertical-rl;
  text-orientation: mixed;
  transform: rotate(180deg);
  font-size: 13px;
  font-weight: 700;
  color: var(--muted-text);
  word-break: break-word;
}

.chat.closed .chat-head button.ghost {
  width: 100%;
  padding: 8px 4px;
  font-size: 11px;
  white-space: normal;
}

.chat.closed .chat-tutor-link,
.chat.closed .kb-link-btn {
  display: none;
}

.collapsed-tab-strip {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: center;
  margin-top: 12px;
}

.collapsed-strip-btn {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s ease;
}

.collapsed-strip-btn:hover {
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container-low));
  color: var(--primary);
  border-color: var(--primary);
}

.collapsed-strip-btn.active {
  background: var(--primary);
  color: var(--on-primary);
  border-color: var(--primary);
}

.chat-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

/* Companion Top Tab Switcher */
.companion-tabs-group {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--surface-container-low);
  padding: 3px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
}

.companion-tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 7px;
  border: none;
  background: transparent;
  color: var(--muted-text);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.18s ease;
}

.companion-tab-btn:hover:not(.active) {
  color: var(--on-surface);
  background: color-mix(in srgb, var(--surface-container-highest) 50%, transparent);
}

.companion-tab-btn.active {
  background: var(--surface-container-lowest);
  color: var(--primary);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.chat-head-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.clear-chat-btn,
.chat-tutor-link,
.kb-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 6px;
  padding: 3px 7px;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.clear-chat-btn,
.kb-link-btn {
  color: var(--on-surface-variant);
}

.clear-chat-btn:hover,
.kb-link-btn:hover {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container-low));
}

.chat-tutor-link {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  border: 1px solid var(--outline-variant);
  text-decoration: none;
}

.chat-tutor-link:hover {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  border-color: var(--primary);
}

.tutor-link-icon {
  opacity: 0.8;
}

/* Tab Panes */
.tab-pane-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  height: 100%;
}

/* Scope & Context Bar */
.chat-meta-bar,
.note-meta-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  flex-shrink: 0;
}

.chat-context-pill,
.note-chapter-pill {
  display: flex;
  align-items: center;
  gap: 5px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 4px 7px;
  min-width: 0;
  flex: 1;
}

.context-pill-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.context-pill-title,
.note-chapter-title {
  font-size: 11.5px;
  font-weight: 600;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-saved-status {
  font-size: 10.5px;
  font-weight: 700;
  color: #10b981;
  background: color-mix(in srgb, #10b981 12%, transparent);
  padding: 2px 6px;
  border-radius: 4px;
}

.scope-compact-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}

.scope-compact-select {
  appearance: none;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 4px 22px 4px 8px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--on-surface-variant);
  cursor: pointer;
}

.scope-compact-select:hover {
  border-color: var(--outline);
  color: var(--on-surface);
}

.scope-compact-select:focus {
  border-color: var(--primary);
  outline: none;
}

.scope-select-chevron {
  position: absolute;
  right: 6px;
  pointer-events: none;
  color: var(--muted-text);
}

.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  border: 0;
}

/* Messages & Note Content Area */
.messages,
.note-content-area {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 3px;
}

/* Empty State */
.empty-chat-state,
.note-empty-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: 20px 12px;
  gap: 12px;
  margin: auto 0;
}

.empty-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.empty-icon-badge {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 2px;
}

.empty-title {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface);
  font-family: 'Manrope', sans-serif;
}

.empty-desc {
  margin: 0;
  font-size: 12px;
  color: var(--muted-text);
  max-width: 240px;
  line-height: 1.45;
}

.quick-chips-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  max-width: 270px;
}

.quick-chip-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 11.5px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
}

.quick-chip-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container-low));
  border-color: var(--primary);
  color: var(--primary);
}

.quick-chip-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.chip-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.chip-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Chat Messages */
.msg {
  border-radius: 10px;
  padding: 9px 10px;
  display: grid;
  gap: 4px;
}

.msg.user {
  background: color-mix(in srgb, var(--primary) 14%, var(--surface-container-lowest));
}

.msg.assistant {
  background: var(--surface-container-low);
}

.msg .role {
  margin: 0;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted-text);
  font-weight: 700;
}

.msg p {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.5;
}

.markdown-body {
  font-size: 13.5px;
  line-height: 1.6;
}

.markdown-body :first-child {
  margin-top: 0;
}

.markdown-body :last-child {
  margin-bottom: 0;
}

.markdown-body p,
.markdown-body ul,
.markdown-body ol,
.markdown-body pre,
.markdown-body blockquote {
  margin: 0 0 8px;
}

.markdown-body code {
  background: var(--surface-container-low);
  border-radius: 6px;
  padding: 1px 5px;
  font-size: 12px;
}

.markdown-body pre {
  background: var(--surface-container-low);
  border-radius: 8px;
  padding: 8px;
  overflow-x: auto;
}

.markdown-body pre code {
  background: transparent;
  padding: 0;
}

/* Study Note Panel Styles */
.note-state-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  text-align: center;
  color: var(--muted-text);
  gap: 12px;
  margin: auto 0;
}

.generating-spinner {
  width: 28px;
  height: 28px;
  border: 3px solid color-mix(in srgb, var(--primary) 20%, transparent);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.85s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.generating-pulse {
  animation: pulse-border 2s ease-in-out infinite;
}

@keyframes pulse-border {
  0%, 100% { opacity: 0.9; }
  50% { opacity: 1; }
}

.generating-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface);
  margin: 0;
}

.generating-sub {
  font-size: 11.5px;
  color: var(--muted-text);
}

.generate-note-hero-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 700;
  border-radius: 8px;
}

.note-view-box {
  padding: 4px 2px;
}

.note-markdown {
  font-size: 13.5px;
  line-height: 1.65;
  color: var(--on-surface);
}

.note-markdown :deep(h1),
.note-markdown :deep(h2),
.note-markdown :deep(h3) {
  margin: 14px 0 6px;
  font-family: 'Manrope', sans-serif;
  color: var(--on-surface);
  font-weight: 700;
}

.note-markdown :deep(h1) { font-size: 18px; }
.note-markdown :deep(h2) { font-size: 15px; color: var(--primary); }
.note-markdown :deep(h3) { font-size: 13.5px; }

.note-markdown :deep(p) {
  margin: 0.5em 0;
}

.note-markdown :deep(ul),
.note-markdown :deep(ol) {
  padding-left: 1.3em;
  margin: 0.5em 0;
}

.note-markdown :deep(li) {
  margin-bottom: 4px;
}

.note-markdown :deep(strong) {
  color: var(--primary);
  font-weight: 700;
}

.reader-note-textarea {
  width: 100%;
  padding: 10px 12px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.55;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface);
  resize: vertical;
  outline: none;
  box-sizing: border-box;
}

.reader-note-textarea:focus {
  border-color: var(--primary);
}

.note-footer-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--outline-variant);
  flex-shrink: 0;
}

.note-footer-bar button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 11px;
  font-size: 12px;
}

.tab-redirect-btn {
  margin-top: 8px;
}

/* Composer */
.composer {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
}

.composer-box {
  display: flex;
  align-items: flex-end;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 6px 10px;
  position: relative;
  transition: all 0.2s ease;
}

.composer-box:focus-within {
  border-color: var(--primary);
  background: var(--surface-container-lowest);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
}

.composer-input {
  flex: 1;
  border: none;
  background: transparent;
  padding: 4px 36px 4px 4px;
  color: var(--on-surface);
  font-family: inherit;
  font-size: 13px;
  line-height: 1.45;
  outline: none;
  resize: none;
  min-height: 24px;
  max-height: 90px;
}

.composer-send-btn {
  position: absolute;
  right: 6px;
  bottom: 6px;
  border: none;
  border-radius: 50%;
  width: 28px;
  height: 28px;
  padding: 0;
  background: var(--primary);
  color: var(--on-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.composer-send-btn:hover:not(:disabled) {
  transform: scale(1.05);
  background: var(--primary-dim);
}

.composer-send-btn:disabled {
  background: var(--surface-container-highest);
  color: var(--muted-text);
  opacity: 0.55;
  cursor: not-allowed;
}

.send-svg {
  width: 14px;
  height: 14px;
}

.composer-hint-row {
  display: flex;
  justify-content: flex-end;
  padding: 0 4px;
}

.composer-hint-row span {
  font-size: 10px;
  color: var(--muted-text);
}

.thinking-dot-loader {
  display: flex;
  gap: 2px;
  align-items: center;
}

.thinking-dot-loader span {
  width: 4px;
  height: 4px;
  background-color: var(--on-primary);
  border-radius: 50%;
  animation: pulse 1.1s infinite ease-in-out;
}

.thinking-dot-loader span:nth-child(2) {
  animation-delay: 0.12s;
}

.thinking-dot-loader span:nth-child(3) {
  animation-delay: 0.24s;
}

@keyframes pulse {
  0%, 80%, 100% {
    opacity: 0.32;
  }
  40% {
    opacity: 1;
  }
}

button {
  border: 0;
  border-radius: 8px;
  padding: 6px 10px;
  font-weight: 600;
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

.secondary:hover:not(:disabled) {
  background: var(--surface-container-high);
}

.ghost {
  color: var(--on-surface);
  background: var(--surface-container-low);
  font-size: 12px;
}

.ghost:hover {
  background: var(--surface-container-highest);
}

.error {
  color: #b42318;
  background: color-mix(in srgb, #b42318 12%, var(--surface-container-lowest));
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 10px;
  font-size: 13px;
}

/* RAG Disabled Overlay */
.panel.chat.rag-off {
  background: color-mix(in srgb, var(--surface-container-low) 90%, #000000);
  opacity: 0.95;
}

.rag-disabled-overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 30px 16px;
  text-align: center;
  height: calc(100% - 40px);
}

.rag-disabled-overlay .lock-icon {
  font-size: 32px;
  margin-bottom: 14px;
  opacity: 0.7;
}

.rag-disabled-overlay h3 {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 8px;
  color: var(--on-surface);
}

.rag-disabled-overlay p {
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--on-surface-variant);
  margin-bottom: 16px;
  max-width: 240px;
}

.enable-rag-btn {
  display: inline-block;
  padding: 8px 16px;
  background: var(--primary);
  color: var(--on-primary);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
}

.enable-rag-btn:hover {
  background: color-mix(in srgb, var(--primary) 85%, #000000);
}
</style>
