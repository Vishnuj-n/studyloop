<template>
  <div v-if="show" class="modal-backdrop milestone-modal-backdrop">
    <div class="milestone-modal-card">
      <header class="milestone-modal-header">
        <div class="header-badge-row">
          <span class="exam-chip">
            <BaseIcon name="trophy" size="14" />
            <span>Pre-Milestone Exam Review</span>
          </span>
          <span class="step-chip">Topic {{ currentIndex + 1 }} of {{ reviewTopics.length }}</span>
        </div>
        <h2 class="exam-title">{{ currentTopic?.title || 'Comprehensive Concept Check' }}</h2>
        <p class="exam-desc">Active retrieval before assessment significantly improves long-term recall.</p>
      </header>

      <div class="milestone-modal-body">
        <!-- Step 1: Self-Reconstruction -->
        <section class="retrieval-prompt-section">
          <div class="prompt-badge-row">
            <span class="step-num-badge">Step 1</span>
            <span class="prompt-title">Active Recall Challenge</span>
          </div>
          <p class="retrieval-question">
            In 1–2 sentences, explain the core idea of <strong>{{ currentTopic?.title }}</strong> to yourself before looking at the notes.
          </p>

          <textarea
            v-model="userExplanation"
            class="shared-textarea"
            placeholder="Type your brief self-explanation or mental outline here..."
            rows="3"
          ></textarea>
        </section>

        <!-- Step 2: Unfold Study Note Verification -->
        <section class="note-reveal-section">
          <div class="prompt-badge-row">
            <span class="step-num-badge">Step 2</span>
            <span class="prompt-title">Verify & Reinforce Mental Model</span>
          </div>

          <div v-if="!noteRevealed" class="reveal-action-box">
            <button
              type="button"
              class="primary-btn reveal-btn"
              @click="revealNote"
            >
              <BaseIcon name="eye" size="16" />
              <span>Reveal Study Note</span>
            </button>
            <span class="reveal-hint">Click when you are ready to check your mental reconstruction against the key notes.</span>
          </div>

          <div v-else class="revealed-note-box">
            <div v-if="noteLoading" class="note-loading-spinner">
              <span>Loading topic notes...</span>
            </div>
            <div v-else-if="currentNoteContent" class="shared-markdown-content" v-html="renderedNote"></div>
            <div v-else class="no-note-prompt">
              <p>No study note has been recorded for this topic yet.</p>
              <button
                type="button"
                class="secondary-btn inline-generate-btn"
                :disabled="generatingNote"
                @click="generateNoteForTopic"
              >
                <BaseIcon name="sparkles" size="14" />
                <span>{{ generatingNote ? 'Generating...' : 'Generate Study Note with AI' }}</span>
              </button>
            </div>
          </div>
        </section>
      </div>

      <footer class="milestone-modal-footer">
        <button
          type="button"
          class="secondary-modal-btn"
          @click="skipPreReview"
        >
          Skip to Exam Directly
        </button>

        <button
          v-if="currentIndex < reviewTopics.length - 1"
          type="button"
          class="primary-btn"
          @click="nextTopic"
        >
          <span>Next Topic ({{ currentIndex + 2 }}/{{ reviewTopics.length }}) →</span>
        </button>

        <button
          v-else
          type="button"
          class="primary-btn"
          @click="proceedToExam"
        >
          <BaseIcon name="zap" size="16" />
          <span>Ready for Exam →</span>
        </button>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import BaseIcon from './BaseIcon.vue'
import {
  getTopicStudyNote,
  generateTopicStudyNote,
  markTopicReviewed,
} from '../services/appApi'
import { renderMarkdown } from '../services/markdown'

const props = defineProps({
  show: { type: Boolean, default: false },
  topics: { type: Array, default: () => [] },
  notebookId: { type: String, default: '' },
})

const emit = defineEmits(['start-exam', 'close'])

const currentIndex = ref(0)
const userExplanation = ref('')
const noteRevealed = ref(false)
const noteLoading = ref(false)
const generatingNote = ref(false)
const currentNoteContent = ref('')

const reviewTopics = computed(() => {
  return Array.isArray(props.topics) && props.topics.length > 0
    ? props.topics
    : [{ topic_id: '', title: 'Notebook Concepts' }]
})

const currentTopic = computed(() => {
  return reviewTopics.value[currentIndex.value] || null
})

const renderedNote = computed(() => {
  return renderMarkdown(currentNoteContent.value || '')
})

watch(
  () => [props.show, currentIndex.value],
  async () => {
    if (props.show && currentTopic.value?.topic_id) {
      userExplanation.value = ''
      noteRevealed.value = false
      currentNoteContent.value = ''
      await fetchNote()
    }
  },
  { immediate: true }
)

async function fetchNote() {
  const tid = currentTopic.value?.topic_id
  if (!tid) return
  noteLoading.value = true
  try {
    const res = await getTopicStudyNote(tid)
    if (res && res.note && res.note.content) {
      currentNoteContent.value = res.note.content
    } else {
      currentNoteContent.value = ''
    }
  } catch (err) {
    console.warn('[PRE_EXAM_NOTE] Error fetching note:', err)
    currentNoteContent.value = ''
  } finally {
    noteLoading.value = false
  }
}

async function revealNote() {
  noteRevealed.value = true
  if (currentTopic.value?.topic_id) {
    // Mark as reviewed in background
    markTopicReviewed(currentTopic.value.topic_id).catch(() => {})
  }
}

async function generateNoteForTopic() {
  const tid = currentTopic.value?.topic_id
  if (!tid) return
  generatingNote.value = true
  try {
    const res = await generateTopicStudyNote(tid, props.notebookId || '')
    if (res && res.note) {
      currentNoteContent.value = res.note.content
    }
  } catch (err) {
    console.error('[PRE_EXAM_NOTE] Generation error:', err)
  } finally {
    generatingNote.value = false
  }
}

async function nextTopic() {
  if (currentTopic.value?.topic_id) {
    markTopicReviewed(currentTopic.value.topic_id).catch(() => {})
  }
  if (currentIndex.value < reviewTopics.value.length - 1) {
    currentIndex.value++
  }
}

function skipPreReview() {
  emit('start-exam')
}

function proceedToExam() {
  if (currentTopic.value?.topic_id) {
    markTopicReviewed(currentTopic.value.topic_id).catch(() => {})
  }
  emit('start-exam')
}
</script>

<style scoped>
.milestone-modal-backdrop {
  backdrop-filter: blur(6px);
  z-index: 2000;
  padding: 24px;
}

.milestone-modal-card {
  width: 100%;
  max-width: 680px;
  max-height: 90vh;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  animation: modalFadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.milestone-modal-header {
  padding: 24px 28px 16px;
  border-bottom: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
}

.header-badge-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.exam-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--primary);
}

.exam-title {
  margin: 0 0 4px;
  font-size: 20px;
  font-weight: 700;
  color: var(--on-surface);
}

.exam-desc {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

.milestone-modal-body {
  padding: 24px 28px;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.prompt-badge-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.prompt-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}

.retrieval-question {
  margin: 0 0 10px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--on-surface);
}

.reveal-action-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 24px;
  background: var(--surface-container-low);
  border: 1px dashed var(--outline-variant);
  border-radius: 10px;
}

.reveal-btn {
  font-size: 14px;
  font-weight: 700;
  padding: 10px 20px;
}

.reveal-hint {
  font-size: 12px;
  color: var(--muted-text);
  text-align: center;
}

.revealed-note-box {
  padding: 16px 20px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  max-height: 240px;
  overflow-y: auto;
  font-size: 13px;
}

.note-loading-spinner,
.no-note-prompt {
  text-align: center;
  padding: 16px;
  color: var(--muted-text);
  font-size: 13px;
}

.inline-generate-btn {
  font-size: 12px;
  margin-top: 8px;
}

.milestone-modal-footer {
  padding: 16px 28px;
  border-top: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.secondary-modal-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 6px;
}

.secondary-modal-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}
</style>
