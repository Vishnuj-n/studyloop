<template>
  <Teleport to="body">
    <div v-if="show" class="modal-backdrop recap-modal-backdrop" @click.self="handleDismiss">
      <div class="recap-modal-card" role="dialog" aria-modal="true" aria-labelledby="recap-title">
        <header class="recap-modal-header">
          <div class="header-badge-row">
            <span class="recap-chip">
              <BaseIcon name="book-open" size="14" />
              <span>Session Memory Recall</span>
            </span>
            <span v-if="previousSlotRange" class="page-range-chip">
              Pages {{ previousSlotRange }}
            </span>
          </div>
          <h2 id="recap-title" class="recap-title">
            {{ topicTitle ? `${topicTitle}: Previous Session Notes` : 'Previous Session Notes' }}
          </h2>
          <p class="recap-desc">
            Quickly review your key takeaways and notes from the last reading session before diving in.
          </p>
        </header>

        <div class="recap-modal-body">
          <div v-if="loading" class="recap-loading">
            <div class="spinner"></div>
            <span>Loading previous session notes...</span>
          </div>

          <div v-else-if="noteContent" class="recap-content-box">
            <div
              class="shared-markdown-content recap-markdown"
              v-html="renderedMarkdown"
            ></div>
          </div>

          <div v-else class="recap-empty-box">
            <p>No notes available for the previous session.</p>
          </div>
        </div>

        <footer class="recap-modal-footer">
          <label class="dont-show-again-label">
            <input
              v-model="dontShowAgain"
              type="checkbox"
              class="recap-checkbox"
            />
            <span>Don't show recap automatically</span>
          </label>
          <div class="footer-actions">
            <button
              type="button"
              class="primary continue-btn"
              @click="handleDismiss"
            >
              <span>Continue to Reading ({{ startPage ? `Page ${startPage}` : 'Next' }}) →</span>
            </button>
          </div>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, ref } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { renderMarkdown } from '../services/markdown'

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
  topicTitle: {
    type: String,
    default: '',
  },
  startPage: {
    type: Number,
    default: 0,
  },
  previousSlot: {
    type: Object,
    default: () => null,
  },
  loading: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close', 'dismiss'])

const dontShowAgain = ref(false)

const previousSlotRange = computed(() => {
  if (!props.previousSlot) return ''
  const start = props.previousSlot.start_page
  const end = props.previousSlot.end_page
  if (start && end) return `${start}–${end}`
  if (start) return `${start}`
  return ''
})

const noteContent = computed(() => {
  return props.previousSlot?.content || ''
})

const renderedMarkdown = computed(() => {
  if (!noteContent.value) return ''
  return renderMarkdown(noteContent.value, {
    noteRelativeFolder: props.previousSlot?.relative_folder_path,
  })
})

function handleDismiss() {
  emit('dismiss', { dontShowAgain: dontShowAgain.value })
}
</script>

<style scoped>
.recap-modal-backdrop {
  position: fixed;
  inset: 0;
  background-color: rgba(15, 17, 19, 0.75);
  backdrop-filter: blur(8px);
  z-index: 10005;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  animation: fadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: scale(0.98);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.recap-modal-card {
  background: var(--surface-container-low, #232526);
  border-radius: 1.25rem;
  width: 100%;
  max-width: 820px;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.28), 0 0 0 1px var(--outline-variant, rgba(255, 255, 255, 0.08));
  overflow: hidden;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}

.recap-modal-header {
  padding: 1.5rem 1.75rem 1.25rem 1.75rem;
  background: var(--surface-container, #282828);
}

.header-badge-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.6rem;
}

.recap-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 0.25rem 0.65rem;
  border-radius: 999px;
  background: var(--surface-container-highest, #3c3836);
  color: var(--primary, #d79921);
}

.page-range-chip {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.6rem;
  border-radius: 999px;
  background: var(--surface-container-highest, #3c3836);
  color: var(--muted-text, #a89984);
}

.recap-title {
  margin: 0 0 0.35rem 0;
  font-family: 'Manrope', sans-serif;
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.015em;
  color: var(--on-surface, #ebdbb2);
  line-height: 1.35;
}

.recap-desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--muted-text, #a89984);
  line-height: 1.45;
}

.recap-modal-body {
  padding: 1.5rem 1.75rem;
  overflow-y: auto;
  flex: 1;
  min-height: 160px;
  background: var(--surface-container-lowest, #141617);
}

.recap-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 2.5rem 0;
  color: var(--muted-text, #a89984);
  font-size: 0.875rem;
}

.spinner {
  width: 22px;
  height: 22px;
  border: 2px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
  border-top-color: var(--primary, #d79921);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.recap-content-box {
  line-height: 1.65;
  color: var(--on-surface, #ebdbb2);
}

.recap-markdown :deep(h1),
.recap-markdown :deep(h2),
.recap-markdown :deep(h3) {
  font-family: 'Manrope', sans-serif;
  color: var(--primary, #d79921);
  margin-top: 1.25rem;
  margin-bottom: 0.5rem;
  font-size: 1rem;
  font-weight: 700;
}

.recap-markdown :deep(h1:first-child),
.recap-markdown :deep(h2:first-child),
.recap-markdown :deep(h3:first-child) {
  margin-top: 0;
}

.recap-markdown :deep(p) {
  margin: 0.6rem 0;
  font-size: 0.9375rem;
  color: var(--on-surface, #ebdbb2);
}

.recap-markdown :deep(ul),
.recap-markdown :deep(ol) {
  margin: 0.6rem 0;
  padding-left: 1.25rem;
}

.recap-markdown :deep(li) {
  margin-bottom: 0.4rem;
  font-size: 0.9375rem;
  color: var(--on-surface-variant, #d5c4a1);
}

.recap-markdown :deep(strong) {
  color: var(--on-surface, #ebdbb2);
  font-weight: 600;
}

.recap-empty-box {
  text-align: center;
  padding: 2rem 0;
  color: var(--muted-text, #a89984);
  font-size: 0.875rem;
}

.recap-modal-footer {
  padding: 1.15rem 1.75rem;
  background: var(--surface-container, #282828);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.dont-show-again-label {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--muted-text, #a89984);
  cursor: pointer;
  user-select: none;
}

.dont-show-again-label:hover {
  color: var(--on-surface, #ebdbb2);
}

.recap-checkbox {
  cursor: pointer;
  accent-color: var(--primary, #d79921);
  width: 14px;
  height: 14px;
}

.footer-actions {
  display: flex;
  align-items: center;
}

.continue-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-family: 'Manrope', sans-serif;
  font-size: 0.875rem;
  font-weight: 600;
  padding: 0.65rem 1.25rem;
  border-radius: 0.75rem; /* DESIGN.md xl button radius */
  background: linear-gradient(15deg, var(--primary, #005bc1), var(--primary-dim, #004faa));
  color: var(--on-primary, #ffffff);
  border: none;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  transition: opacity 0.15s ease, transform 0.1s ease;
}

.continue-btn:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}

.continue-btn:active {
  transform: translateY(0) scale(0.98);
}
</style>
