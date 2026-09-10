<template>
  <div v-if="modelValue" class="modal-backdrop" @click.self="close">
    <div class="pace-drawer">
      <div class="drawer-header">
        <div>
          <h2 class="drawer-title">Study Pace &amp; Routine</h2>
          <p class="drawer-subtitle">
            Pacing forecast for <strong>{{ profileName }}</strong>
          </p>
        </div>
        <button
          type="button"
          class="icon-close-btn"
          aria-label="Close"
          @click="close"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>

      <div class="drawer-body">
        <!-- Feasibility & Workload Section -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">EXAM FEASIBILITY</span>
            <span v-if="pace?.deadline" class="deadline-tag">
              Target: {{ formatDate(pace.deadline) }} ({{ pace.days_remaining }}d left)
            </span>
          </div>

          <div class="status-banner" :class="`status-${pace?.feasibility_status || 'NO_DATA'}`">
            <span class="status-dot-icon"></span>
            <div>
              <h3 class="status-headline">{{ statusHeadline }}</h3>
              <p class="status-explanation">{{ statusExplanation }}</p>
            </div>
          </div>

          <div class="metrics-grid">
            <div class="metric-item">
              <span class="metric-label">Remaining Workload</span>
              <span class="metric-value">~{{ pace?.remaining_sessions ? Math.ceil(pace.remaining_sessions) : 0 }} sessions</span>
              <span class="metric-subtext">1 session ≈ {{ formatNumber(pace?.target_session_words || 3000) }} words</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Current Velocity</span>
              <span class="metric-value">{{ pace?.current_daily_sessions ? `${pace.current_daily_sessions} sess/day` : '—' }}</span>
              <span class="metric-subtext">Last 7 days average</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Target Pace</span>
              <span class="metric-value">{{ pace?.required_daily_sessions ? `${Math.ceil(pace.required_daily_sessions)} sess/day` : '—' }}</span>
              <span class="metric-subtext">Needed for exam deadline</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Forecasted Finish</span>
              <span class="metric-value">{{ pace?.projected_finish ? formatDate(pace.projected_finish) : 'Need 2+ sessions' }}</span>
              <span v-if="pace?.days_gap !== undefined" class="metric-subtext">
                {{ pace.days_gap >= 0 ? `${pace.days_gap} days buffer` : `${-pace.days_gap} days late` }}
              </span>
            </div>
          </div>
        </section>

        <!-- Daily Study Times (Read-Only Display + Link to Settings) -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">DAILY STUDY WINDOWS</span>
            <span class="total-duration-tag">{{ totalScheduledTime }} total</span>
          </div>

          <div class="slots-list">
            <div v-for="(slot, idx) in localSlots" :key="idx" class="slot-display-row">
              <div class="slot-display-info">
                <span class="slot-display-name">{{ slot.name || 'Study Session' }}</span>
                <span class="slot-display-time">{{ slot.start }} – {{ slot.end }}</span>
              </div>
              <span class="slot-display-badge">{{ computeSlotDuration(slot.start, slot.end) }}</span>
            </div>
          </div>

          <button type="button" class="edit-settings-btn" @click="goToSettingsRoutine">
            <span>⚙ Manage Schedule &amp; Calendar Sync in Settings</span>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </button>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  pace: { type: Object, default: null },
  profileName: { type: String, default: 'Current Profile' },
  studySlotsJson: { type: String, default: '[]' },
  studyStartTime: { type: String, default: '17:00' },
  studyEndTime: { type: String, default: '18:00' },
})

const emit = defineEmits(['update:modelValue'])
const localSlots = ref([])

function parseSlots() {
  if (props.studySlotsJson) {
    try {
      const parsed = JSON.parse(props.studySlotsJson)
      if (Array.isArray(parsed) && parsed.length > 0) {
        localSlots.value = JSON.parse(JSON.stringify(parsed))
        return
      }
    } catch {
      // Fallback
    }
  }
  // Backward compatibility fallback to single study window
  localSlots.value = [
    {
      name: 'Daily Study Session',
      start: props.studyStartTime || '17:00',
      end: props.studyEndTime || '18:00',
    },
  ]
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) parseSlots()
  },
  { immediate: true }
)

function goToSettingsRoutine() {
  close()
  router.push({ path: '/settings', query: { category: 'study' } })
}

const statusHeadline = computed(() => {
  const s = props.pace?.feasibility_status
  if (s === 'AHEAD') return `On track — finishing ${props.pace.days_gap} days early`
  if (s === 'ON_TRACK') return 'On track to meet exam deadline'
  if (s === 'BEHIND') return `Behind pace — projected ${Math.abs(props.pace?.days_gap || 0)} days late`
  return 'Estimating your study velocity'
})

const statusExplanation = computed(() => {
  const s = props.pace?.feasibility_status
  if (s === 'AHEAD') {
    return `At your current velocity of ${props.pace?.current_daily_sessions || 1} sessions/day, you'll comfortably finish on ${formatDate(props.pace?.projected_finish)}.`
  }
  if (s === 'ON_TRACK') {
    return `Your reading pace matches the required ${Math.ceil(props.pace?.required_daily_sessions || 1)} sessions/day to complete on schedule.`
  }
  if (s === 'BEHIND') {
    const needed = Math.ceil(props.pace?.extra_sessions_needed || 1)
    const sessText = needed === 1 ? '1 session/day' : `${needed} sessions/day`
    if (needed >= 2) {
      return `Add ~${sessText} (or consider spacing out your target exam date) to complete your remaining curriculum on time.`
    }
    return `Add ~${sessText} (or extend your daily study window) to finish before your exam.`
  }
  return 'Complete 2 to 3 study sessions so StudyLoop can estimate your personalized reading velocity.'
})

const totalScheduledTime = computed(() => {
  let totalMinutes = 0
  for (const slot of localSlots.value) {
    if (!slot.start || !slot.end) continue
    const [sh, sm] = slot.start.split(':').map(Number)
    const [eh, em] = slot.end.split(':').map(Number)
    let m = eh * 60 + em - (sh * 60 + sm)
    if (m < 0) m += 1440
    totalMinutes += m
  }
  if (totalMinutes === 0) return '0 min'
  const h = Math.floor(totalMinutes / 60)
  const remM = totalMinutes % 60
  if (h > 0 && remM > 0) {
    return `${h}h ${remM}m`
  }
  if (h > 0) {
    return `${h}h`
  }
  return `${remM}m`
})

function computeSlotDuration(start, end) {
  if (!start || !end) return ''
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  let m = eh * 60 + em - (sh * 60 + sm)
  if (m < 0) m += 1440
  return m >= 60 ? `${(m / 60).toFixed(1).replace('.0', '')} hr` : `${m} min`
}

function close() {
  emit('update:modelValue', false)
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  try {
    return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
  } catch {
    return dateStr
  }
}

function formatNumber(num) {
  return num ? Number(num).toLocaleString() : '0'
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(4px);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}
.pace-drawer {
  width: 100%;
  max-width: 500px;
  height: 100%;
  background: var(--surface-container-lowest, #fff);
  color: var(--on-surface, #2d3338);
  box-shadow: -8px 0 32px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.drawer-header {
  padding: 24px 28px 16px;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  border-bottom: 1px solid var(--outline-variant);
}
.drawer-title { margin: 0; font-size: 20px; font-weight: 700; }
.drawer-subtitle { margin: 4px 0 0; font-size: 13px; color: var(--muted-text); }
.icon-close-btn { background: none; border: none; font-size: 16px; color: var(--muted-text); cursor: pointer; padding: 6px; border-radius: 6px; }
.drawer-body { padding: 24px 28px; display: flex; flex-direction: column; gap: 24px; }
.drawer-section { display: flex; flex-direction: column; gap: 10px; }
.section-label-row { display: flex; justify-content: space-between; align-items: center; }
.section-badge { font-size: 11px; font-weight: 700; letter-spacing: 0.06em; color: var(--muted-text); text-transform: uppercase; }
.deadline-tag, .total-duration-tag { font-size: 12px; font-weight: 600; color: var(--primary); }
.status-banner { display: flex; gap: 12px; padding: 14px 16px; border-radius: 12px; border: 1px solid var(--outline-variant); }
.status-AHEAD { background: rgba(39, 174, 96, 0.08); border-color: rgba(39, 174, 96, 0.3); }
.status-ON_TRACK { background: rgba(245, 158, 11, 0.08); border-color: rgba(245, 158, 11, 0.3); }
.status-BEHIND { background: rgba(235, 94, 85, 0.08); border-color: rgba(235, 94, 85, 0.3); }
.status-NO_DATA { background: var(--surface-container-low); }
.status-dot-icon { width: 10px; height: 10px; border-radius: 50%; margin-top: 5px; flex-shrink: 0; background: currentColor; }
.status-AHEAD .status-dot-icon { background: #27ae60; }
.status-ON_TRACK .status-dot-icon { background: #f59e0b; }
.status-BEHIND .status-dot-icon { background: #eb5e55; }
.status-NO_DATA .status-dot-icon { background: var(--muted-text); }
.status-headline { margin: 0 0 3px; font-size: 14px; font-weight: 700; }
.status-explanation { margin: 0; font-size: 13px; line-height: 1.4; color: var(--muted-text); }
.metrics-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.metric-item { display: flex; flex-direction: column; padding: 12px; border-radius: 10px; border: 1px solid var(--outline-variant); background: var(--surface-container-highest); }
.metric-label { font-size: 11px; font-weight: 600; color: var(--muted-text); margin-bottom: 3px; }
.metric-value { font-size: 15px; font-weight: 700; color: var(--on-surface); }
.metric-subtext { font-size: 11px; color: var(--muted-text); margin-top: 3px; }
.slots-list { display: flex; flex-direction: column; gap: 8px; }
.slot-display-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-highest);
}
.slot-display-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.slot-display-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}
.slot-display-time {
  font-size: 12px;
  color: var(--muted-text);
  font-variant-numeric: tabular-nums;
}
.slot-display-badge {
  font-size: 11px;
  font-weight: 700;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container-highest));
  padding: 3px 8px;
  border-radius: 6px;
}
.edit-settings-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1px dashed var(--outline-variant);
  background: transparent;
  color: var(--primary);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
  margin-top: 4px;
}
.edit-settings-btn:hover {
  background: color-mix(in srgb, var(--primary) 8%, transparent);
  border-color: var(--primary);
}
</style>
