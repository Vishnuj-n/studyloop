<template>
  <article class="card focus-hero-card" :class="heroVariantClass">
    <div class="hero-top-row">
      <div class="hero-type-badge">
        <span class="hero-icon">{{ actionIcon }}</span>
        <span class="hero-type-label">{{ formatTaskType(task.action_type) }}</span>
        <span v-if="task.estimate_minutes > 0" class="hero-estimate">
          • {{ task.estimate_minutes }} min
        </span>
      </div>
      <span class="hero-step-tag">Up Next #1</span>
    </div>

    <div class="hero-main-content">
      <h2 class="hero-title">
        <span v-if="showReadingPrefix" class="title-prefix">Read: </span>
        {{ cleanTitle }}
      </h2>
      <p class="hero-meta">
        {{ formattedMeta }}
      </p>
    </div>

    <div class="hero-action-row">
      <button
        type="button"
        class="hero-cta-btn"
        :class="{ 'is-syncing': task.action_type === 'flashcard_generate' && isSyncing }"
        :disabled="task.action_type === 'flashcard_generate' && isSyncing"
        @click="$emit('start', task)"
      >
        <span v-if="task.action_type === 'flashcard_generate' && isSyncing">Generating Flashcards...</span>
        <span v-else-if="task.action_type === 'flashcard_generate'">⚡ Generate Flashcards</span>
        <span v-else-if="isReadingTask">▶ Start Reading</span>
        <span v-else-if="task.action_type === 'quiz' || task.action_type === 'milestone_exam'">▶ Start Quiz</span>
        <span v-else-if="task.action_type === 'socratic_remedial'">🛡️ Start Rescue Session</span>
        <span v-else>▶ Start Task</span>
      </button>
    </div>
  </article>
</template>

<script setup>
import { computed } from 'vue'
import { formatTaskType } from '../utils/dateFormat'

const props = defineProps({
  task: { type: Object, required: true },
  isSyncing: { type: Boolean, default: false },
})

defineEmits(['start'])

const isReadingTask = computed(() => {
  const t = (props.task.action_type || '').toLowerCase()
  return t === 'reading' || t === 'reread' || t === 'start_reading'
})

const cleanTitle = computed(() => {
  let title = (props.task.title || 'Study Session').trim()
  if (isReadingTask.value) {
    title = title.replace(/^read:\s*/i, '')
  }
  return title || 'Study Session'
})

const showReadingPrefix = computed(() => {
  return isReadingTask.value
})

const actionIcon = computed(() => {
  const t = (props.task.action_type || '').toLowerCase()
  if (t === 'reading' || t === 'reread' || t === 'start_reading') return '📖'
  if (t === 'quiz' || t === 'milestone_exam') return '📝'
  if (t === 'flashcard_review') return '🗂️'
  if (t === 'flashcard_generate') return '⚡'
  if (t === 'socratic_remedial') return '🛡️'
  return '📌'
})

const heroVariantClass = computed(() => {
  const t = (props.task.action_type || '').toLowerCase()
  if (t === 'socratic_remedial') return 'hero-rescue'
  if (t === 'flashcard_generate') return 'hero-sync'
  return 'hero-default'
})

const formattedMeta = computed(() => {
  if (props.task.meta) return props.task.meta
  if (
    props.task.start_page !== undefined &&
    props.task.start_page !== null &&
    props.task.end_page !== undefined &&
    props.task.end_page !== null
  ) {
    return `Pages ${props.task.start_page}–${props.task.end_page}`
  }
  return 'Ready for study'
})
</script>

<style scoped>
.focus-hero-card {
  background: var(--surface-container-lowest, #ffffff);
  border: none;
  border-radius: 16px;
  padding: 28px 32px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  box-shadow: 0 4px 20px rgba(45, 51, 56, 0.04);
  position: relative;
  overflow: hidden;
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s;
}

.focus-hero-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 28px rgba(45, 51, 56, 0.08);
}

.hero-top-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.hero-type-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--primary, #005bc1);
  background: var(--surface-container, #ebeef2);
  padding: 6px 12px;
  border-radius: 8px;
}

.hero-icon {
  font-size: 14px;
}

.hero-estimate {
  font-weight: 600;
  opacity: 0.85;
}

.hero-step-tag {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.05em;
  color: var(--muted-text, #64707d);
  background: var(--surface-container-low, #f2f4f7);
  border: none;
  padding: 4px 10px;
  border-radius: 9999px;
  text-transform: uppercase;
}

.hero-main-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.hero-title {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--on-surface, #2d3338);
}

.title-prefix {
  font-weight: 600;
  color: var(--muted-text, #64707d);
}

.hero-meta {
  margin: 0;
  font-size: 14px;
  color: var(--muted-text, #64707d);
  font-weight: 400;
}

.hero-action-row {
  display: flex;
  align-items: center;
  padding-top: 4px;
}

.hero-cta-btn {
  background: linear-gradient(15deg, var(--primary, #005bc1) 0%, var(--primary-dim, #004faa) 100%);
  color: var(--on-primary, #ffffff);
  border: none;
  border-radius: 0.75rem;
  padding: 12px 24px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(45, 51, 56, 0.08);
  transition: transform 0.15s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.15s;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.hero-cta-btn:hover:not(:disabled) {
  opacity: 0.94;
  transform: translateY(-1px);
}

.hero-cta-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.hero-cta-btn.is-syncing {
  opacity: 0.7;
  cursor: wait;
}

/* Rescue & Sync Variants adhere to tonal styling without harsh borders */
.hero-rescue .hero-type-badge {
  color: #9f403d;
  background: var(--surface-container, #ebeef2);
}

.hero-rescue .hero-cta-btn {
  background: linear-gradient(15deg, #9f403d 0%, #7a2c2a 100%);
}

.hero-sync .hero-type-badge {
  color: #9f403d;
  background: var(--surface-container, #ebeef2);
}

.hero-sync .hero-cta-btn {
  background: linear-gradient(15deg, #9f403d 0%, #7a2c2a 100%);
}
</style>
