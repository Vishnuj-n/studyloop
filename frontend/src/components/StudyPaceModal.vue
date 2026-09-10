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
        <button type="button" class="icon-close-btn" aria-label="Close" @click="close">✕</button>
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
              <span class="metric-value">~{{ pace?.remaining_sessions ?? 0 }} sessions</span>
              <span class="metric-subtext">1 session ≈ {{ formatNumber(pace?.target_session_words || 3000) }} words</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Current Velocity</span>
              <span class="metric-value">{{ pace?.current_daily_sessions ? `${pace.current_daily_sessions} sess/day` : '—' }}</span>
              <span class="metric-subtext">Last 7 days average</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Target Pace</span>
              <span class="metric-value">{{ pace?.required_daily_sessions ? `${pace.required_daily_sessions} sess/day` : '—' }}</span>
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

        <!-- Daily Study Times (Alarms & Calendar) -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">DAILY STUDY WINDOWS</span>
            <span class="total-duration-tag">{{ totalScheduledTime }} total</span>
          </div>

          <div class="slots-list">
            <div v-for="(slot, idx) in localSlots" :key="idx" class="slot-row">
              <input
                v-model="slot.name"
                type="text"
                placeholder="Session label"
                class="slot-name-input"
                @change="persistSlots"
              />
              <div class="slot-times-group">
                <input v-model="slot.start" type="time" class="slot-time-input" required @change="persistSlots" />
                <span class="time-arrow">→</span>
                <input v-model="slot.end" type="time" class="slot-time-input" required @change="persistSlots" />
              </div>
              <span class="slot-duration">{{ computeSlotDuration(slot.start, slot.end) }}</span>
              <button type="button" class="slot-delete-btn" title="Remove" @click="removeSlot(idx)">✕</button>
            </div>
          </div>

          <button v-if="localSlots.length < 4" type="button" class="add-slot-btn" @click="addSlot">
            + Add Study Window
          </button>

          <div class="templates-strip">
            <span class="templates-label">Presets:</span>
            <button type="button" class="template-pill" @click="applyTemplate('morning_evening')">Morning + Evening</button>
            <button type="button" class="template-pill" @click="applyTemplate('split_shift')">College Split</button>
            <button type="button" class="template-pill" @click="applyTemplate('deep_work')">Power Hour</button>
          </div>
        </section>

        <!-- Calendar Export -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">CALENDAR ROUTINE SYNC</span>
          </div>
          <div class="calendar-btn-row">
            <button type="button" class="cal-btn ics-btn" @click="downloadICS">Download .ics</button>
            <button type="button" class="cal-btn" @click="openGoogle">Google Calendar</button>
            <button type="button" class="cal-btn" @click="openOutlook">Outlook Web</button>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import {
  downloadRoutineICS,
  getGoogleCalendarUrl,
  getOutlookCalendarUrl,
} from '../services/calendarService'
import { openURLInBrowser } from '../services/appApi'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  pace: { type: Object, default: null },
  profileName: { type: String, default: 'Current Profile' },
  studySlotsJson: { type: String, default: '[]' },
  studyStartTime: { type: String, default: '17:00' },
  studyEndTime: { type: String, default: '18:00' },
})

const emit = defineEmits(['update:modelValue', 'save-slots'])
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
    return `Your reading pace matches the required ${props.pace?.required_daily_sessions || 1} sessions/day to complete on schedule.`
  }
  if (s === 'BEHIND') {
    const needed = props.pace?.extra_sessions_needed || 0.5
    return `Add ~${needed} session/day (or extend your daily study time by 20–30 mins) to finish before your exam.`
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
  return h > 0 && remM > 0 ? `${h}h ${remM}m` : h > 0 ? `${h}h` : `${remM}m`
})

function computeSlotDuration(start, end) {
  if (!start || !end) return ''
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  let m = eh * 60 + em - (sh * 60 + sm)
  if (m < 0) m += 1440
  return m >= 60 ? `${(m / 60).toFixed(1).replace('.0', '')} hr` : `${m} min`
}

function addSlot() {
  if (localSlots.value.length >= 4) return
  localSlots.value.push({
    name: `Study Slot ${localSlots.value.length + 1}`,
    start: '19:00',
    end: '20:00',
  })
  persistSlots()
}

function removeSlot(index) {
  if (localSlots.value.length <= 1) {
    localSlots.value[0] = { name: 'Daily Study Session', start: '17:00', end: '18:00' }
  } else {
    localSlots.value.splice(index, 1)
  }
  persistSlots()
}

function applyTemplate(type) {
  if (type === 'morning_evening') {
    localSlots.value = [
      { name: 'Morning Focus', start: '07:30', end: '08:30' },
      { name: 'Evening Review', start: '20:00', end: '21:00' },
    ]
  } else if (type === 'split_shift') {
    localSlots.value = [
      { name: 'Pre-Class Reading', start: '07:00', end: '08:00' },
      { name: 'Night Practice', start: '21:00', end: '22:15' },
    ]
  } else if (type === 'deep_work') {
    localSlots.value = [{ name: 'Deep Study Window', start: '18:00', end: '19:30' }]
  }
  persistSlots()
}

function persistSlots() {
  const jsonStr = JSON.stringify(localSlots.value)
  const first = localSlots.value[0] || { start: '17:00', end: '18:00' }
  emit('save-slots', {
    study_slots_json: jsonStr,
    study_start_time: first.start,
    study_end_time: first.end,
  })
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

async function openExternal(url) {
  try {
    await openURLInBrowser(url)
  } catch {
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}

function downloadICS() {
  downloadRoutineICS(localSlots.value)
}

function openGoogle() {
  const first = localSlots.value[0] || { start: '17:00', end: '18:00' }
  openExternal(getGoogleCalendarUrl(first.start, first.end))
}

function openOutlook() {
  const first = localSlots.value[0] || { start: '17:00', end: '18:00' }
  openExternal(getOutlookCalendarUrl(first.start, first.end))
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
.slot-row { display: flex; align-items: center; gap: 8px; padding: 8px 10px; border-radius: 8px; border: 1px solid var(--outline-variant); background: var(--surface-container-highest); }
.slot-name-input { flex: 1; font-size: 13px; font-weight: 600; border: none; background: transparent; color: var(--on-surface); }
.slot-name-input:focus { outline: none; }
.slot-times-group { display: flex; align-items: center; gap: 4px; }
.slot-time-input { border: 1px solid var(--outline-variant); border-radius: 6px; background: var(--surface-container-lowest); color: var(--on-surface); padding: 4px 6px; font-size: 12px; font-family: inherit; }
.time-arrow { color: var(--muted-text); font-size: 11px; }
.slot-duration { font-size: 11px; font-weight: 600; color: var(--muted-text); min-width: 44px; text-align: right; }
.slot-delete-btn { background: none; border: none; color: var(--muted-text); cursor: pointer; padding: 3px 6px; border-radius: 4px; }
.slot-delete-btn:hover { color: #eb5e55; }
.add-slot-btn { background: none; border: 1px dashed var(--outline-variant); color: var(--primary); font-size: 12px; font-weight: 600; padding: 8px; border-radius: 8px; cursor: pointer; width: 100%; }
.add-slot-btn:hover { border-color: var(--primary); }
.templates-strip { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-top: 2px; }
.templates-label { font-size: 11px; color: var(--muted-text); font-weight: 600; }
.template-pill { font-size: 11px; font-weight: 600; padding: 3px 8px; border-radius: 9999px; border: 1px solid var(--outline-variant); background: var(--surface-container-low); color: var(--on-surface); cursor: pointer; }
.template-pill:hover { border-color: var(--primary); color: var(--primary); }
.calendar-btn-row { display: flex; gap: 8px; flex-wrap: wrap; }
.cal-btn { flex: 1; min-width: 120px; padding: 8px 12px; font-size: 12px; font-weight: 600; border-radius: 8px; border: 1px solid var(--outline-variant); cursor: pointer; background: var(--surface-container-highest); color: var(--on-surface); }
.cal-btn.ics-btn { border-color: color-mix(in srgb, var(--primary) 30%, transparent); background: color-mix(in srgb, var(--primary) 8%, var(--surface-container-highest)); color: var(--primary); }
.cal-btn:hover { border-color: var(--primary); }
</style>
