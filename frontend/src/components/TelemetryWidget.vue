<template>
  <div v-if="pace && pace.has_deadline" class="telemetry-bar">
    <!-- Progress Pill -->
    <div
      class="telemetry-pill session-pill"
      :class="{ 'pill-target-met': completedSessions >= targetSessions && targetSessions > 0 }"
      title="Daily session completion progress"
    >
      <span class="pill-icon">🎯</span>
      <span class="pill-text">
        <strong>{{ completedSessions }} / {{ targetSessions }}</strong> Sessions Today
      </span>
    </div>

    <!-- Days Left Pill -->
    <div
      class="telemetry-pill deadline-pill"
      :class="{ warning: pace.days_remaining <= 3 }"
      :title="`Target Exam Deadline: ${pace.deadline || 'Set'}`"
    >
      <span class="pill-icon">⏳</span>
      <span class="pill-text">{{ formatDaysRemainingShort(pace.days_remaining) }}</span>
    </div>

    <!-- Daily Pace Pill -->
    <div
      class="telemetry-pill pace-pill"
      :title="`Pace: ${pace.daily_pace} words/day (${pace.remaining_words || 0} remaining words)`"
    >
      <span class="pill-icon">⚡</span>
      <span class="pill-text">{{ pace.daily_pace }} w/d</span>
      <span v-if="pace.pace_label" class="pace-sublabel">({{ pace.pace_label }})</span>
    </div>
  </div>
  <div v-else class="telemetry-bar">
    <!-- Fallback Session Counter Pill when no deadline set -->
    <div
      class="telemetry-pill session-pill"
      :class="{ 'pill-target-met': completedSessions >= targetSessions && targetSessions > 0 }"
    >
      <span class="pill-icon">🎯</span>
      <span class="pill-text">
        <strong>{{ completedSessions }} / {{ targetSessions }}</strong> Sessions Today
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  pace: { type: Object, default: null },
  profileName: { type: String, default: 'Unknown' },
  completedSessions: { type: Number, default: 0 },
})

const targetSessions = computed(() => {
  if (props.pace && props.pace.sessions_per_day) {
    const val = Math.ceil(props.pace.sessions_per_day)
    return val > 0 ? val : 2
  }
  return 2
})

function formatDaysRemainingShort(days) {
  if (days === undefined || days === null) return ''
  if (days === 0) return 'Deadline today!'
  if (days < 0) return 'Passed'
  return `${days}d left`
}
</script>

<style scoped>
.telemetry-bar {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.telemetry-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 9999px;
  font-size: 13px;
  font-weight: 600;
  background: var(--surface-container-low, #f4f4f6);
  border: 1px solid var(--outline-variant, #e0e0e0);
  color: var(--on-surface, #1e1e1e);
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  white-space: nowrap;
}

.session-pill {
  background: color-mix(in srgb, var(--primary) 8%, var(--surface-container-lowest, #fff));
  border-color: color-mix(in srgb, var(--primary) 25%, transparent);
  color: var(--primary);
}

.session-pill.pill-target-met {
  background: rgba(39, 174, 96, 0.12);
  border-color: rgba(39, 174, 96, 0.35);
  color: #219653;
}

.deadline-pill {
  color: var(--on-surface-variant, #555);
}

.deadline-pill.warning {
  background: rgba(235, 94, 85, 0.12);
  border-color: rgba(235, 94, 85, 0.3);
  color: #eb5e55;
  font-weight: 700;
}

.pace-pill {
  color: var(--on-surface-variant, #555);
}

.pace-sublabel {
  font-size: 11px;
  opacity: 0.8;
  font-weight: 500;
}

.pill-icon {
  font-size: 14px;
  line-height: 1;
}

.pill-text {
  line-height: 1.2;
}
</style>
