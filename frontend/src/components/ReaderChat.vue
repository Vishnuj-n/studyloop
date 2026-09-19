<template>
  <aside
    id="reader-chat-panel"
    class="panel chat"
    :class="{ closed: chatCollapsed, 'rag-off': !ragEnabled }"
  >
    <div class="chat-head">
      <div class="chat-head-title-group">
        <h2>AI Chat</h2>
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
      </div>
      <div class="chat-head-actions">
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

    <template v-if="!chatCollapsed">
      <div v-if="!ragSettingsLoaded" class="rag-disabled-overlay">
        <h3>Loading settings...</h3>
      </div>
      <div v-else-if="ragSettingsError" class="rag-disabled-overlay">
        <div class="lock-icon"><BaseIcon name="alert-triangle" size="24" /></div>
        <h3>Settings Error</h3>
        <p>{{ ragSettingsError }}</p>
        <button class="primary" @click="$emit('retry-settings')">Retry</button>
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
    </template>
  </aside>
</template>

<script setup>
import { computed, inject, watch } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { logFrontendEvent } from '../services/appApi'

const props = defineProps({
  selectedTopicID: { type: String, default: '' },
  selectedTopicTitle: { type: String, default: '' },
  selectedNotebookID: { type: String, default: '' },
  selectedNotebookTitle: { type: String, default: '' },
  currentPage: { type: Number, required: true },
  topicStartPage: { type: Number, required: true },
  topicEndPage: { type: Number, required: true },
  ragEnabled: { type: Boolean, required: true },
  ragSettingsLoaded: { type: Boolean, required: true },
  ragSettingsError: { type: String, default: null },
})

defineEmits(['retry-settings'])

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
  gap: 0;
}

.chat.closed .chat-head { flex-direction: column; }
.chat.closed h2 {
  writing-mode: vertical-rl;
  text-orientation: mixed;
  transform: rotate(180deg);
  font-size: 14px;
  word-break: break-word;
}
.chat.closed .chat-head button.ghost {
  width: 100%;
  padding: 8px 4px;
  font-size: 11px;
  white-space: normal;
}
.chat.closed .chat-tutor-link { display: none; }

.chat-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.chat-head-title-group, .chat-head-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.clear-chat-btn, .chat-tutor-link {
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

.clear-chat-btn {
  color: var(--on-surface-variant);
}
.clear-chat-btn:hover {
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
.tutor-link-icon { opacity: 0.8; }

h2, h3 { margin: 0; font-family: 'Manrope', sans-serif; }
h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.01em; }
h3 { font-size: 16px; }

/* Scope & Context Bar */
.chat-meta-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  flex-shrink: 0;
}

.chat-context-pill {
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

.context-pill-icon { color: var(--primary); flex-shrink: 0; }
.context-pill-title {
  font-size: 11.5px;
  font-weight: 600;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.scope-compact-wrap { position: relative; display: inline-flex; align-items: center; flex-shrink: 0; }
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
.scope-compact-select:hover { border-color: var(--outline); color: var(--on-surface); }
.scope-compact-select:focus { border-color: var(--primary); outline: none; }
.scope-select-chevron { position: absolute; right: 6px; pointer-events: none; color: var(--muted-text); }

.visually-hidden { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); border: 0; }

/* Messages Area */
.messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 3px;
}

/* Empty State */
.empty-chat-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  text-align: center;
  padding: 16px 8px;
  gap: 16px;
  margin: auto 0;
}
.empty-hero { display: flex; flex-direction: column; align-items: center; gap: 4px; }
.empty-icon-badge {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
}
.empty-title { margin: 0; font-size: 13.5px; font-weight: 700; color: var(--on-surface); font-family: 'Manrope', sans-serif; }
.empty-desc { margin: 0; font-size: 11.5px; color: var(--muted-text); max-width: 230px; line-height: 1.4; }

.quick-chips-grid { display: flex; flex-direction: column; gap: 6px; width: 100%; max-width: 270px; }
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
.quick-chip-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.chip-icon { color: var(--primary); flex-shrink: 0; }
.chip-label { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

/* Chat Messages */
.msg { border-radius: 10px; padding: 9px 10px; display: grid; gap: 4px; }
.msg.user { background: color-mix(in srgb, var(--primary) 14%, var(--surface-container-lowest)); }
.msg.assistant { background: var(--surface-container-low); }
.msg .role { margin: 0; font-size: 11px; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted-text); font-weight: 700; }
.msg p { margin: 0; font-size: 13.5px; line-height: 1.5; }

.markdown-body { font-size: 13.5px; line-height: 1.6; }
.markdown-body :first-child { margin-top: 0; }
.markdown-body :last-child { margin-bottom: 0; }
.markdown-body p, .markdown-body ul, .markdown-body ol, .markdown-body pre, .markdown-body blockquote { margin: 0 0 8px; }
.markdown-body code { background: var(--surface-container-low); border-radius: 6px; padding: 1px 5px; font-size: 12px; }
.markdown-body pre { background: var(--surface-container-low); border-radius: 8px; padding: 8px; overflow-x: auto; }
.markdown-body pre code { background: transparent; padding: 0; }

/* Composer */
.composer { display: flex; flex-direction: column; gap: 4px; flex-shrink: 0; }
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
.composer-send-btn:hover:not(:disabled) { transform: scale(1.05); background: var(--primary-dim); }
.composer-send-btn:disabled { background: var(--surface-container-highest); color: var(--muted-text); opacity: 0.55; cursor: not-allowed; }
.send-svg { width: 14px; height: 14px; }
.composer-hint-row { display: flex; justify-content: flex-end; padding: 0 4px; }
.composer-hint-row span { font-size: 10px; color: var(--muted-text); }

.thinking-dot-loader { display: flex; gap: 2px; align-items: center; }
.thinking-dot-loader span {
  width: 4px;
  height: 4px;
  background-color: var(--on-primary);
  border-radius: 50%;
  animation: pulse 1.1s infinite ease-in-out;
}
.thinking-dot-loader span:nth-child(2) { animation-delay: 0.12s; }
.thinking-dot-loader span:nth-child(3) { animation-delay: 0.24s; }
@keyframes pulse { 0%, 80%, 100% { opacity: 0.32; } 40% { opacity: 1; } }

button { border: 0; border-radius: 8px; padding: 6px 10px; font-weight: 600; cursor: pointer; }
button:disabled { opacity: 0.55; cursor: not-allowed; }
.primary { color: var(--on-primary); background: linear-gradient(160deg, var(--primary), var(--primary-dim)); }
.ghost { color: var(--on-surface); background: var(--surface-container-low); font-size: 12px; }
.ghost:hover { background: var(--surface-container-highest); }
.error { color: #b42318; background: color-mix(in srgb, #b42318 12%, var(--surface-container-lowest)); border: 1px solid var(--outline-variant); border-radius: 10px; padding: 10px; font-size: 13px; }

/* RAG Disabled Overlay */
.panel.chat.rag-off { background: color-mix(in srgb, var(--surface-container-low) 90%, #000000); opacity: 0.85; }
.rag-disabled-overlay { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px 20px; text-align: center; height: calc(100% - 60px); }
.rag-disabled-overlay .lock-icon { font-size: 32px; margin-bottom: 16px; opacity: 0.7; }
.rag-disabled-overlay h3 { font-size: 16px; font-weight: 700; margin-bottom: 12px; color: var(--on-surface); }
.rag-disabled-overlay p { font-size: 13px; line-height: 1.5; color: var(--on-surface-variant); margin-bottom: 24px; }
.enable-rag-btn { display: inline-block; padding: 8px 16px; background: var(--primary); color: var(--on-primary); border-radius: 6px; font-size: 13px; font-weight: 600; text-decoration: none; }
.enable-rag-btn:hover { background: color-mix(in srgb, var(--primary) 85%, #000000); }
</style>
