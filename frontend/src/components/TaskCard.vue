<template>
  <article class="card task-card" :class="{ 'queue-row': queueIndex !== undefined }">
    <div class="task-left-section">
      <!-- Step Badge for Up Next queue -->
      <span v-if="queueIndex !== undefined" class="queue-step-badge">#{{ queueIndex }}</span>

      <div class="task-content">
        <div class="task-header">
          <span class="task-type" :class="task.action_type ? task.action_type.toLowerCase() : ''">
            {{ formatTaskType(task.action_type) }}
          </span>
          <span
            v-if="task.action_type !== 'flashcard_generate' && task.estimate_minutes > 0"
            class="task-estimate"
          >
            • {{ task.estimate_minutes }} min
          </span>
        </div>

        <div class="task-title-group">
          <h3 v-if="isReadingTask">
            <span class="title-action-prefix">Read: </span>{{ cleanTitle }}
          </h3>
          <h3 v-else>{{ cleanTitle }}</h3>

          <span v-if="taskMetaText" class="task-meta-inline"> ({{ taskMetaText }}) </span>
        </div>
      </div>
    </div>

    <div class="task-action-wrapper">
      <button
        type="button"
        class="primary-btn"
        :class="{ 'sync-btn': task.action_type === 'flashcard_generate' }"
        :aria-label="'Start task ' + (task.title || task.id)"
        :disabled="task.action_type === 'flashcard_generate' && isSyncing"
        @click="$emit('start', task)"
      >
        <span v-if="task.action_type === 'flashcard_generate' && isSyncing">Generating...</span>
        <span v-else-if="task.action_type === 'flashcard_generate'">Generate</span>
        <span v-else-if="isReadingTask">Start</span>
        <span v-else>Start</span>
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
  queueIndex: { type: Number, default: undefined },
})

defineEmits(['start'])

const isReadingTask = computed(() => {
  const t = (props.task.action_type || '').toLowerCase()
  return t === 'reading' || t === 'reread' || t === 'start_reading'
})

const cleanTitle = computed(() => {
  let title = (props.task.title || '').trim()
  if (isReadingTask.value) {
    title = title.replace(/^read:\s*/i, '')
  }
  return title || 'Untitled Task'
})

const taskMetaText = computed(() => {
  if (props.task.meta) return props.task.meta
  if (
    props.task.start_page !== undefined &&
    props.task.start_page !== null &&
    props.task.end_page !== undefined &&
    props.task.end_page !== null
  ) {
    return `Pages ${props.task.start_page}–${props.task.end_page}`
  }
  return ''
})
</script>

<style scoped>
.card {
  background: var(--surface-container-lowest, #ffffff);
  border: none;
  border-radius: 12px;
}

.task-card {
  padding: 16px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  box-shadow: 0 2px 10px rgba(45, 51, 56, 0.03);
  transition:
    transform 0.2s cubic-bezier(0.16, 1, 0.3, 1),
    background-color 0.2s,
    box-shadow 0.2s;
}

.task-card:hover {
  transform: translateY(-1px);
  background-color: var(--surface-container-highest, #f8fafb);
  box-shadow: 0 4px 16px rgba(45, 51, 56, 0.06);
}

.task-left-section {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
  min-width: 0;
}

.queue-step-badge {
  font-family: 'Manrope', sans-serif;
  font-size: 13px;
  font-weight: 700;
  color: var(--muted-text, #64707d);
  background: var(--surface-container-low, #f2f4f7);
  border: none;
  padding: 4px 10px;
  border-radius: 6px;
  flex-shrink: 0;
}

.task-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-type {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--primary, #005bc1);
  background: var(--surface-container, #ebeef2);
  padding: 3px 8px;
  border-radius: 4px;
}

.task-estimate {
  font-size: 12px;
  color: var(--muted-text, #64707d);
  font-weight: 500;
}

.task-title-group {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
}

.task-card h3 {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.3;
  color: var(--on-surface, #2d3338);
}

.title-action-prefix {
  font-weight: 500;
  color: var(--muted-text, #64707d);
}

.task-meta-inline {
  font-size: 13px;
  color: var(--muted-text, #64707d);
}

.task-action-wrapper {
  flex-shrink: 0;
}

.primary-btn {
  background: linear-gradient(15deg, var(--primary, #005bc1) 0%, var(--primary-dim, #004faa) 100%);
  color: var(--on-primary, #ffffff);
  border: none;
  border-radius: 0.75rem;
  padding: 8px 18px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(45, 51, 56, 0.06);
  transition:
    opacity 0.15s,
    transform 0.15s;
  white-space: nowrap;
}

.primary-btn:hover:not(:disabled) {
  opacity: 0.92;
  transform: translateY(-1px);
}

.primary-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.task-type.flashcard_generate {
  color: #9f403d;
  background: var(--surface-container, #ebeef2);
}

.sync-btn {
  background: linear-gradient(15deg, #9f403d 0%, #7a2c2a 100%);
}
</style>
