<template>
  <div v-if="show" class="milestone-modal-backdrop" role="dialog" aria-modal="true" @click.self="proceedToExam">
    <div class="milestone-modal-card">
      <button
        type="button"
        class="close-btn"
        aria-label="Close"
        @click="proceedToExam"
      >
        <BaseIcon name="x" size="16" />
      </button>

      <header class="milestone-modal-header">
        <div class="exam-chip">
          <BaseIcon name="award" size="14" />
          <span>Pre-Milestone Review</span>
        </div>
        <h2 class="exam-title">Prepare for Milestone Exam</h2>
        <p class="exam-desc">
          Review your chapter notes before beginning the assessment, or jump straight into the exam.
        </p>
      </header>

      <div class="milestone-modal-body">
        <div class="chapters-list">
          <div
            v-for="topic in reviewTopics"
            :key="topic.topic_id || topic.title"
            class="chapter-card"
          >
            <div class="chapter-info">
              <div class="chapter-icon-box">
                <BaseIcon name="book-open" size="18" />
              </div>
              <div class="chapter-text">
                <span class="chapter-title">{{ topic.title || 'Chapter Notes' }}</span>
                <span class="chapter-hint">Study notes & topic summaries</span>
              </div>
            </div>
            <button
              type="button"
              class="review-note-btn"
              title="Open notes for this chapter"
              @click="goToChapterNotes(topic)"
            >
              <span>Review Notes</span>
              <BaseIcon name="arrow-right" size="14" />
            </button>
          </div>
        </div>
      </div>

      <footer class="milestone-modal-footer">
        <button
          type="button"
          class="secondary-modal-btn"
          @click="proceedToExam"
        >
          Skip Review
        </button>

        <button
          type="button"
          class="primary-btn start-exam-btn"
          @click="proceedToExam"
        >
          <BaseIcon name="zap" size="15" />
          <span>Start Exam</span>
        </button>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  topics: { type: Array, default: () => [] },
  notebookId: { type: String, default: '' },
  taskId: { type: String, default: '' },
})

const emit = defineEmits(['start-exam', 'close'])

const router = useRouter()

const reviewTopics = computed(() => {
  return Array.isArray(props.topics) && props.topics.length > 0
    ? props.topics
    : [{ topic_id: '', title: 'Notebook Concepts' }]
})

function goToChapterNotes(topic) {
  router.push({
    path: '/notes',
    query: {
      notebookId: props.notebookId || undefined,
      topicId: topic?.topic_id || undefined,
      from: 'quiz',
      taskId: props.taskId || undefined,
    },
  })
}

function proceedToExam() {
  emit('start-exam')
}
</script>

<style scoped>
.milestone-modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 100000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 14, 23, 0.75);
  backdrop-filter: blur(8px);
  padding: 16px;
  animation: modalFadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes modalFadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.milestone-modal-card {
  position: relative;
  width: 100%;
  max-width: 480px;
  background: var(--surface-container-lowest, #12141c);
  border: 1px solid var(--outline-variant);
  border-radius: 18px;
  padding: 24px;
  box-shadow: 0 20px 40px -10px rgba(0, 0, 0, 0.5), 0 0 25px rgba(99, 102, 241, 0.08);
  display: flex;
  flex-direction: column;
  gap: 18px;
  color: var(--on-surface);
  animation: cardScaleUp 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes cardScaleUp {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(8px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.close-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low, rgba(255, 255, 255, 0.04));
  color: var(--muted-text);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.15s ease;
}

.close-btn:hover {
  background: var(--surface-container, rgba(255, 255, 255, 0.1));
  color: var(--on-surface);
  border-color: var(--primary);
}

.milestone-modal-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-right: 28px;
}

.exam-chip {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 14%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
  color: var(--primary);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.exam-title {
  margin: 2px 0 0;
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--on-surface);
  line-height: 1.3;
}

.exam-desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--muted-text);
  line-height: 1.5;
}

.milestone-modal-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.chapters-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chapter-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  background: var(--surface-container-low, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  transition: all 0.15s ease;
}

.chapter-card:hover {
  background: var(--surface-container, rgba(255, 255, 255, 0.06));
  border-color: color-mix(in srgb, var(--primary) 40%, var(--outline-variant));
}

.chapter-info {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.chapter-icon-box {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 22%, transparent);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.chapter-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.chapter-title {
  font-size: 0.925rem;
  font-weight: 600;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chapter-hint {
  font-size: 0.775rem;
  color: var(--muted-text);
}

.review-note-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
  border-radius: 8px;
  color: var(--primary);
  font-size: 0.825rem;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.review-note-btn:hover {
  background: var(--primary);
  color: var(--on-primary, #ffffff);
  border-color: var(--primary);
  transform: translateX(2px);
}

.milestone-modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 8px;
}

.secondary-modal-btn {
  background: transparent;
  border: 1px solid transparent;
  color: var(--muted-text);
  font-size: 0.875rem;
  font-weight: 600;
  padding: 8px 14px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.secondary-modal-btn:hover {
  background: var(--surface-container-low, rgba(255, 255, 255, 0.05));
  color: var(--on-surface);
  border-color: var(--outline-variant);
}

.start-exam-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 20px;
  border-radius: 10px;
  font-size: 0.9rem;
  font-weight: 700;
  cursor: pointer;
}
</style>
