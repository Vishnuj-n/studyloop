<template>
  <div v-if="loops && loops.length > 0" class="zeigarnik-container">
    <div v-for="loop in loops" :key="loop.topic_id" class="zeigarnik-card">
      <div class="zeigarnik-ring-wrapper">
        <svg class="progress-ring" width="48" height="48">
          <circle
            class="progress-ring-bg"
            stroke="#334155"
            stroke-width="4"
            fill="transparent"
            r="20"
            cx="24"
            cy="24"
          />
          <circle
            class="progress-ring-circle"
            stroke="#3b82f6"
            stroke-width="4"
            fill="transparent"
            r="20"
            cx="24"
            cy="24"
            :stroke-dasharray="circumference"
            :stroke-dashoffset="getOffset(loop.completion_ratio)"
          />
        </svg>
        <span class="progress-text">{{ Math.round(loop.completion_ratio * 100) }}%</span>
      </div>

      <div class="zeigarnik-content">
        <span class="notebook-title-badge">{{ loop.notebook_title }}</span>
        <h4 class="topic-title">{{ loop.topic_title }}</h4>
        <p class="tension-message">{{ loop.message }}</p>
      </div>

      <button class="resume-btn" type="button" @click="$emit('resume', loop)">
        Resume →
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  loops: {
    type: Array,
    default: () => [],
  },
})

defineEmits(['resume'])

const radius = 20
const circumference = 2 * Math.PI * radius

function getOffset(ratio) {
  const clamped = Math.max(0, Math.min(1, ratio || 0))
  return circumference - clamped * circumference
}
</script>

<style scoped>
.zeigarnik-container {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.zeigarnik-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--bg-card, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-left: 4px solid #3b82f6;
  border-radius: 10px;
  padding: 0.85rem 1.25rem;
}

.zeigarnik-ring-wrapper {
  position: relative;
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.progress-ring {
  transform: rotate(-90deg);
}

.progress-ring-circle {
  transition: stroke-dashoffset 0.35s ease;
  stroke-linecap: round;
}

.progress-text {
  position: absolute;
  font-size: 0.68rem;
  font-weight: 700;
  color: var(--text-primary, #f8fafc);
}

.zeigarnik-content {
  flex: 1;
  min-width: 0;
}

.notebook-title-badge {
  font-size: 0.7rem;
  font-weight: 600;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  display: block;
}

.topic-title {
  margin: 0.15rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary, #f8fafc);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tension-message {
  margin: 0;
  font-size: 0.82rem;
  color: #fbbf24;
}

.resume-btn {
  background: #3b82f6;
  color: #fff;
  border: none;
  font-size: 0.85rem;
  font-weight: 600;
  padding: 0.45rem 0.85rem;
  border-radius: 6px;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s ease;
}

.resume-btn:hover {
  background: #2563eb;
}
</style>
