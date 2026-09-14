<template>
  <aside
    id="reader-chat-panel"
    class="panel chat"
    :class="{ closed: chat.chatCollapsed.value, 'rag-off': !ragEnabled }"
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
          🧠 Socratic Tutor ↗
        </router-link>
      </div>
      <button
        class="ghost"
        :aria-expanded="!chat.chatCollapsed.value"
        aria-controls="reader-chat-panel"
        @click="chat.toggleChat"
      >
        {{ chat.chatCollapsed.value ? 'Expand' : 'Collapse' }}
      </button>
    </div>

    <template v-if="!chat.chatCollapsed.value">
      <div v-if="!ragSettingsLoaded" class="rag-disabled-overlay">
        <h3>Loading settings...</h3>
      </div>
      <div v-else-if="ragSettingsError" class="rag-disabled-overlay">
        <div class="lock-icon">⚠️</div>
        <h3>Settings Error</h3>
        <p>{{ ragSettingsError }}</p>
        <button class="primary" @click="$emit('retry-settings')">Retry</button>
      </div>
      <div v-else-if="!ragEnabled" class="rag-disabled-overlay">
        <div class="lock-icon">🔒</div>
        <h3>Local AI Retrieval Offline</h3>
        <p>Local semantic search and Q&A is currently disabled to save memory and CPU.</p>
        <router-link to="/settings" class="enable-rag-btn">Enable in Settings</router-link>
      </div>
      <template v-else>
        <p class="chat-context">
          Using topic <strong>{{ selectedTopicTitle || 'None' }}</strong>
          <span v-if="selectedNotebookTitle">from {{ selectedNotebookTitle }}</span>
        </p>

        <div class="scope-bar">
          <div class="scope-main">
            <label for="scope-select" class="scope-label">Retrieval Scope</label>
            <select id="scope-select" v-model="chat.chatScope.value" class="scope-select">
              <option value="entire_notebook">Entire Notebook</option>
              <option value="current_chapter">Current Chapter</option>
              <option value="current_page">Current Page</option>
            </select>
          </div>
          <p class="scope-helper">Broader scopes search more of your notebook.</p>
        </div>

        <div :ref="(el) => { if (chat && chat.messagesPane) chat.messagesPane.value = el }" class="messages">
          <article
            v-for="(msg, idx) in chat.chatMessages.value"
            :key="msg.id || idx"
            class="msg"
            :class="msg.role"
          >
            <p class="role">{{ msg.role === 'user' ? 'You' : 'Tutor' }}</p>
            <p v-if="msg.role === 'user'">{{ msg.text }}</p>
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div v-else class="markdown-body" v-html="chat.renderMarkdown(msg.text)"></div>
          </article>
        </div>

        <article v-if="chat.chatError.value" class="error">{{ chat.chatError.value }}</article>

        <form class="composer" @submit.prevent="sendChat">
          <div class="composer-box">
            <textarea
              v-model="chat.chatInput.value"
              class="composer-input"
              :disabled="chat.chatLoading.value || !selectedTopicID"
              placeholder="Ask about what you’re reading right now..."
              rows="1"
              @keydown.enter="handleEnterKey"
            ></textarea>
            <button
              type="submit"
              class="composer-send-btn"
              :disabled="chat.chatLoading.value || !chat.chatInput.value.trim() || !selectedTopicID"
              title="Send question"
            >
              <svg
                v-if="!chat.chatLoading.value"
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
import { inject, watch } from 'vue'
import { logFrontendEvent } from '../services/appApi'

const props = defineProps({
  selectedTopicID: {
    type: String,
    default: '',
  },
  selectedTopicTitle: {
    type: String,
    default: '',
  },
  selectedNotebookID: {
    type: String,
    default: '',
  },
  selectedNotebookTitle: {
    type: String,
    default: '',
  },
  currentPage: {
    type: Number,
    required: true,
  },
  topicStartPage: {
    type: Number,
    required: true,
  },
  topicEndPage: {
    type: Number,
    required: true,
  },
  ragEnabled: {
    type: Boolean,
    required: true,
  },
  ragSettingsLoaded: {
    type: Boolean,
    required: true,
  },
  ragSettingsError: {
    type: String,
    default: null,
  },
})

defineEmits(['retry-settings'])

const chat = inject('chat')

// Watch settings errors and RAG toggle status
watch(
  () => props.ragSettingsError,
  (newVal) => {
    if (newVal) {
      logFrontendEvent('error', 'ReaderChat', 'rag_settings_error', { error: newVal })
    }
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
    await chat.sendMessage({
      topicID: props.selectedTopicID,
      notebookID: props.selectedNotebookID,
      currentPage: props.currentPage,
      chapterStartPage: props.topicStartPage,
      chapterEndPage: props.topicEndPage,
    })
  } catch (err) {
    if (chat && chat.chatError) {
      chat.chatError.value = err.message || 'Failed to send message'
    }
  }
}

function handleEnterKey(event) {
  if (event.shiftKey || event.isComposing) return
  event.preventDefault()
  if (!chat.chatLoading.value && chat.chatInput.value.trim() && props.selectedTopicID) {
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
  box-sizing: border-box;
}

.chat.closed {
  padding: 10px 8px;
  gap: 0;
}

.chat.closed .chat-head {
  flex-direction: column;
}

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
  word-break: break-word;
}

.chat-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.chat-head-title-group {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}

.chat-tutor-link {
  font-size: 11px;
  font-weight: 600;
  text-decoration: none;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 2px 8px;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.chat-tutor-link:hover {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  border-color: var(--primary);
}

.chat.closed .chat-tutor-link {
  display: none;
}

h2 {
  margin: 0;
  font-size: 28px;
  font-family: 'Manrope', sans-serif;
}

h3 {
  margin: 0;
  font-size: 18px;
  font-family: 'Manrope', sans-serif;
}

.chat-context {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
  flex-shrink: 0;
}

.scope-bar {
  display: grid;
  gap: 6px;
  padding: 10px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-container-low) 86%, transparent);
  flex-shrink: 0;
}

.scope-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.scope-helper {
  margin: 0;
  font-size: 11px;
  color: var(--muted-text);
  line-height: 1.3;
}

.scope-label {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-text);
}

.scope-select {
  width: auto;
  min-width: 160px;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: 10px;
}

.messages {
  flex: 1;
  min-height: 140px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-right: 3px;
}

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
  font-size: 14px;
  line-height: 1.5;
}

.markdown-body {
  font-size: 14px;
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

/* Integrated Composer Input Box */
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
  border-radius: 16px;
  padding: 6px 10px;
  transition: all 0.2s ease;
  position: relative;
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
  font-size: 13.5px;
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

.composer-send-btn:active:not(:disabled) {
  transform: scale(0.95);
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
  display: inline-block;
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

select {
  width: 100%;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-lowest);
  color: var(--on-surface);
  border-radius: 10px;
  font: inherit;
  padding: 10px;
  outline: 0;
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

.ghost {
  color: var(--on-surface);
  background: var(--surface-container-low);
}

.error {
  color: #b42318;
  background: color-mix(in srgb, #b42318 12%, var(--surface-container-lowest));
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 10px;
  font-size: 13px;
}

/* RAG Disabled styles in Reader */
.panel.chat.rag-off {
  background: color-mix(in srgb, var(--surface-container-low) 90%, #000000);
  opacity: 0.85;
}

.rag-disabled-overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
  height: calc(100% - 60px);
}

.rag-disabled-overlay .lock-icon {
  font-size: 32px;
  margin-bottom: 16px;
  opacity: 0.7;
}

.rag-disabled-overlay h3 {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 12px;
  color: var(--on-surface);
}

.rag-disabled-overlay p {
  font-size: 13px;
  line-height: 1.5;
  color: var(--on-surface-variant);
  margin-bottom: 24px;
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
  transition: background 0.2s ease;
}

.enable-rag-btn:hover {
  background: color-mix(in srgb, var(--primary) 85%, #000000);
}
</style>
