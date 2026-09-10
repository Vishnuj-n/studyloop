<template>
  <div v-if="pace && pace.has_deadline" class="telemetry-bar">
    <!-- Feasibility Status Interactive Pill -->
    <button
      type="button"
      class="telemetry-pill feasibility-pill"
      :class="`pill-${pace.feasibility_status || 'NO_DATA'}`"
      :title="`Click to view Study Pace & Routine for ${profileName}`"
      @click="$emit('open-pace-modal')"
    >
      <span class="status-dot"></span>
      <span class="pill-text">{{ feasibilityPillText }}</span>
      <svg class="pill-chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <polyline points="9 18 15 12 9 6"></polyline>
      </svg>
    </button>

    <!-- Progress Pill -->
    <div
      class="telemetry-pill session-pill"
      :class="{ 'pill-target-met': completedSessions >= targetSessions && targetSessions > 0 }"
      title="Daily reading session completion progress"
      @click="$emit('open-pace-modal')"
    >
      <span class="pill-text">
        <strong>{{ completedSessions }} / {{ targetSessions }}</strong> Reading Sessions Today
      </span>
    </div>

    <!-- Days Left Pill -->
    <div
      class="telemetry-pill deadline-pill"
      :class="{ warning: pace.days_remaining <= 3 }"
      :title="`Target Exam Deadline: ${pace.deadline || 'Set'}`"
      @click="$emit('open-pace-modal')"
    >
      <span class="pill-text">{{ formatDaysRemainingShort(pace.days_remaining) }}</span>
    </div>
  </div>
  <div v-else class="telemetry-bar">
    <!-- Fallback Session Counter Pill when no deadline set -->
    <div
      class="telemetry-pill session-pill"
      :class="{ 'pill-target-met': completedSessions >= targetSessions && targetSessions > 0 }"
      @click="$emit('open-pace-modal')"
    >
      <span class="pill-text">
        <strong>{{ completedSessions }} / {{ targetSessions }}</strong> Reading Sessions Today
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

defineEmits(['open-pace-modal'])

const feasibilityPillText = computed(() => {
  if (!props.pace) return 'Study Pace'
  const s = props.pace.feasibility_status
  if (s === 'AHEAD') {
    return `On track · Finish ${props.pace.days_gap}d early`
  }
  if (s === 'ON_TRACK') {
    return 'On track for deadline'
  }
  if (s === 'BEHIND') {
    return `Behind pace · ${Math.abs(props.pace.days_gap || 0)}d late`
  }
  return 'Estimating pace'
})

const targetSessions = computed(() => {
  if (props.pace && props.pace.required_daily_sessions) {
    const val = Math.ceil(props.pace.required_daily_sessions)
    return val > 0 ? val : 1
  }
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
  cursor: pointer;
}

.telemetry-pill:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.06);
}

/* Feasibility Pill */
.feasibility-pill {
  cursor: pointer;
}

.pill-chevron {
  opacity: 0.55;
  transition: transform 0.15s ease, opacity 0.15s ease;
  flex-shrink: 0;
  margin-left: 2px;
}

.feasibility-pill:hover .pill-chevron {
  opacity: 0.9;
  transform: translateX(1.5px);
}

.feasibility-pill .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.pill-AHEAD {
  background: rgba(39, 174, 96, 0.1);
  border-color: rgba(39, 174, 96, 0.35);
  color: #1b8744;
}

.pill-AHEAD .status-dot {
  background: #27ae60;
  box-shadow: 0 0 6px rgba(39, 174, 96, 0.4);
}

.pill-ON_TRACK {
  background: rgba(245, 158, 11, 0.1);
  border-color: rgba(245, 158, 11, 0.35);
  color: #b45309;
}

.pill-ON_TRACK .status-dot {
  background: #f59e0b;
}

.pill-BEHIND {
  background: rgba(235, 94, 85, 0.12);
  border-color: rgba(235, 94, 85, 0.35);
  color: #eb5e55;
  font-weight: 700;
}

.pill-BEHIND .status-dot {
  background: #eb5e55;
  box-shadow: 0 0 6px rgba(235, 94, 85, 0.4);
}

.pill-NO_DATA {
  background: var(--surface-container-low);
  border-color: var(--outline-variant);
  color: var(--muted-text);
}

.pill-NO_DATA .status-dot {
  background: var(--muted-text);
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

.pill-text {
  line-height: 1.2;
}
</style>
