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
        <!-- Status Banner -->
        <div class="status-banner" :class="`status-${pace?.feasibility_status || 'NO_DATA'}`">
          <span class="status-dot-icon"></span>
          <div>
            <h3 class="status-headline">{{ statusHeadline }}</h3>
            <p class="status-explanation">{{ statusExplanation }}</p>
          </div>
        </div>

        <!-- Section A: The Workload vs Routine Capacity -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">CAPACITY STATUS</span>
            <span v-if="pace?.deadline" class="deadline-tag">
              Target: {{ formatDate(pace.deadline) }} ({{ pace.days_remaining }}d left)
            </span>
          </div>

          <div class="metrics-grid">
            <div class="metric-item">
              <span class="metric-label">Remaining Workload</span>
              <span class="metric-value">{{ pace?.remaining_sessions || 0 }} sessions</span>
              <span class="metric-subtext">~{{ formatNumber(pace?.remaining_words || 0) }} words total</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Scheduled Windows</span>
              <span class="metric-value">{{ pace?.remaining_windows || 0 }} study windows</span>
              <span class="metric-subtext">{{ isSelfPaced ? 'Self-paced (no deadline)' : 'Before exam target date' }}</span>
            </div>
          </div>

          <div class="balance-card" :class="balanceClass">
            <div class="balance-title">Routine Balance</div>
            <div class="balance-metric">
              <template v-if="isSelfPaced">
                Self-paced learning — progress through the queue at your own routine velocity
              </template>
              <template v-else-if="isDeadlinePassed">
                Target date has passed with {{ pace?.remaining_sessions || 0 }} sessions remaining
              </template>
              <template v-else-if="uncoveredCount > 0">
                ~{{ uncoveredCount }} {{ uncoveredCount === 1 ? 'session' : 'sessions' }} uncovered by current routine
              </template>
              <template v-else>
                All {{ pace?.remaining_sessions || 0 }} sessions covered by scheduled routine
              </template>
            </div>
          </div>
        </section>

        <!-- Section B: Historical Reality Check -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">HISTORICAL PACE REALITY CHECK</span>
          </div>

          <div class="metrics-grid">
            <div class="metric-item">
              <span class="metric-label">Recent Rhythm</span>
              <span class="metric-value">
                {{ pace?.sessions_past_7_days ? `~${pace.sessions_past_7_days} sessions / week` : '—' }}
              </span>
              <span class="metric-subtext">Completed in past 7 days</span>
            </div>

            <div class="metric-item">
              <span class="metric-label">Projected Finish</span>
              <span class="metric-value">
                {{ pace?.projected_finish ? formatDate(pace.projected_finish) : 'Need 2+ sessions' }}
              </span>
              <span class="metric-subtext">{{ projectedFinishSubtext }}</span>
            </div>
          </div>

          <p class="historical-disclaimer">
            Estimated based on your recent weekly pace, not a guaranteed completion date.
          </p>
        </section>

        <!-- Section C: Actionable Recovery Options -->
        <section class="drawer-section">
          <div class="section-label-row">
            <span class="section-badge">ADJUST STUDY PLAN</span>
          </div>

          <div class="action-options-card">
            <!-- Action 1: Manage Profiles & Notebooks in Settings -->
            <button type="button" class="plan-action-btn" @click="goToManageProfiles">
              <div class="btn-info">
                <strong>Manage Profiles &amp; Notebooks</strong>
                <span>Configure profile target dates, study scope, and linked textbooks</span>
              </div>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="9 18 15 12 9 6"></polyline>
              </svg>
            </button>

            <!-- Action 2: Self-Paced Mode Switch Toggle -->
            <div class="toggle-card">
              <div class="toggle-card-content">
                <div class="toggle-card-text">
                  <strong>Self-Paced Mode</strong>
                  <span>
                    {{ isSelfPaced
                      ? 'No exam target set. Toggle off to restore previous deadline.'
                      : 'Remove deadline pressure while keeping all study queue progress.'
                    }}
                  </span>
                </div>
                <button
                  type="button"
                  role="switch"
                  :aria-checked="isSelfPaced"
                  class="switch-toggle-btn"
                  :class="{ 'is-checked': isSelfPaced }"
                  :disabled="updatingDeadline"
                  title="Toggle Self-Paced Mode"
                  @click="toggleSelfPaced"
                >
                  <span class="switch-slider-track">
                    <span class="switch-slider-thumb"></span>
                  </span>
                </button>
              </div>

              <div v-if="actionError" class="action-error-msg">
                {{ actionError }}
              </div>
            </div>
          </div>
        </section>

        <!-- Daily Study Times (Read-Only Display) -->
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
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { updateProfile } from '../services/appApi'
import { useDialog } from '../composables/useDialog'

const router = useRouter()
const { confirm: showConfirm } = useDialog()

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  pace: { type: Object, default: null },
  profileName: { type: String, default: 'Current Profile' },
  profileId: { type: String, default: '' },
  studySlotsJson: { type: String, default: '[]' },
  studyStartTime: { type: String, default: '17:00' },
  studyEndTime: { type: String, default: '18:00' },
})

const emit = defineEmits(['update:modelValue', 'pace-updated'])
const localSlots = ref([])
const updatingDeadline = ref(false)
const actionError = ref('')

// Cache previous deadline in localStorage so turning Self-Paced mode OFF can cleanly revert
const cachedDeadlineKey = computed(() => `studyloop_last_deadline_${props.profileId}`)

const isSelfPaced = computed(() => {
  return !props.pace?.has_deadline || !props.pace?.deadline
})

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
    if (open) {
      parseSlots()
      actionError.value = ''
      if (props.pace?.deadline) {
        localStorage.setItem(cachedDeadlineKey.value, props.pace.deadline)
      }
    }
  },
  { immediate: true }
)

function goToManageProfiles() {
  close()
  router.push({ path: '/settings', query: { category: 'profiles' } })
}

async function toggleSelfPaced() {
  if (!props.profileId || updatingDeadline.value) return

  if (!isSelfPaced.value) {
    // Switching ON Self-Paced (Removing Deadline)
    const confirmed = await showConfirm({
      title: 'Switch to Self-Paced Mode?',
      message: 'This will remove your target exam date and deadline urgency from your study queue. All your notebook progress and queue tasks are preserved.',
      confirmText: 'Confirm Self-Paced',
      cancelText: 'Keep Deadline',
      type: 'warning',
    })
    if (!confirmed) return

    // Save current deadline before clearing
    if (props.pace?.deadline) {
      localStorage.setItem(cachedDeadlineKey.value, props.pace.deadline)
    }

    updatingDeadline.value = true
    actionError.value = ''
    try {
      const res = await updateProfile(props.profileId, props.profileName, '')
      if (res && res.error) {
        actionError.value = res.error
        return
      }
      emit('pace-updated')
    } catch (err) {
      console.error('Failed to switch to self paced:', err)
      actionError.value = err.message || 'Failed to switch to self paced'
    } finally {
      updatingDeadline.value = false
    }
  } else {
    // Switching OFF Self-Paced (Reverting / Restoring Deadline)
    const cached = localStorage.getItem(cachedDeadlineKey.value)
    let targetDate = cached
    if (!targetDate) {
      // Default to 30 days from now if no cached deadline found
      const d = new Date()
      d.setDate(d.getDate() + 30)
      targetDate = d.toISOString().split('T')[0]
    }

    const confirmed = await showConfirm({
      title: 'Restore Exam Target Date?',
      message: `Re-enable exam deadline scheduling with target date: ${formatDate(targetDate)}?`,
      confirmText: 'Restore Target',
      cancelText: 'Stay Self-Paced',
      type: 'info',
    })
    if (!confirmed) return

    updatingDeadline.value = true
    actionError.value = ''
    try {
      const res = await updateProfile(props.profileId, props.profileName, targetDate)
      if (res && res.error) {
        actionError.value = res.error
        return
      }
      emit('pace-updated')
    } catch (err) {
      console.error('Failed to restore deadline:', err)
      actionError.value = err.message || 'Failed to restore deadline'
    } finally {
      updatingDeadline.value = false
    }
  }
}

const isDeadlinePassed = computed(() => props.pace?.feasibility_status === 'DEADLINE_PASSED')
const uncoveredCount = computed(() => props.pace?.uncovered_sessions || 0)

const projectedFinishSubtext = computed(() => {
  if (!props.pace?.projected_finish) return 'At recent weekly rhythm'
  if (!props.pace?.deadline || isSelfPaced.value) return 'At recent weekly rhythm'
  try {
    const d1 = new Date(props.pace.deadline).setHours(0, 0, 0, 0)
    const d2 = new Date(props.pace.projected_finish).setHours(0, 0, 0, 0)
    const diffDays = Math.round((d1 - d2) / 86400000)
    if (diffDays > 0) {
      return `~${diffDays}d buffer before deadline`
    }
    if (diffDays < 0) {
      return `~${Math.abs(diffDays)}d after deadline`
    }
    return 'On target date'
  } catch {
    return 'At recent weekly rhythm'
  }
})

const balanceClass = computed(() => {
  if (isSelfPaced.value) return 'balance-neutral'
  if (isDeadlinePassed.value) return 'balance-danger'
  if (uncoveredCount.value > 0) return 'balance-warning'
  return 'balance-success'
})

const statusHeadline = computed(() => {
  if (isSelfPaced.value) return 'Self-Paced Profile'
  const s = props.pace?.feasibility_status
  if (s === 'DEADLINE_PASSED') return 'Target date has passed'
  if (s === 'UNCOVERED_WORKLOAD') {
    return `${uncoveredCount.value} ${uncoveredCount.value === 1 ? 'session' : 'sessions'} uncovered by routine`
  }
  if (s === 'AT_RISK') return 'Pace at risk — historical rhythm is behind'
  if (s === 'ON_TRACK') return 'On track to meet target date'
  return 'Schedule capacity on track'
})

const statusExplanation = computed(() => {
  if (isSelfPaced.value) {
    return 'No target date set. You can study without deadline urgency, or enable a target in Manage Profiles.'
  }
  const s = props.pace?.feasibility_status
  if (s === 'DEADLINE_PASSED') {
    return 'Your target exam date has arrived with remaining sessions. You can adjust your plan or switch to self-paced.'
  }
  if (s === 'UNCOVERED_WORKLOAD') {
    const rem = props.pace?.remaining_sessions || 0
    const win = props.pace?.remaining_windows || 0
    return `${rem} sessions remaining, but only ${win} study windows scheduled before deadline, leaving ~${uncoveredCount.value} sessions uncovered.`
  }
  if (s === 'AT_RISK') {
    return `Your schedule has sufficient windows, but your recent pace (~${props.pace?.sessions_past_7_days || 0} sess/week) is lower than required.`
  }
  if (s === 'ON_TRACK') {
    return 'Your current workload fits your scheduled routine and matches your recent study rhythm.'
  }
  return 'Your scheduled routine has enough windows to cover all remaining sessions. Complete 2+ sessions to calculate your projected finish date.'
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
  if (h > 0 && remM > 0) return `${h}h ${remM}m`
  if (h > 0) return `${h}h`
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
.status-ON_TRACK { background: rgba(39, 174, 96, 0.08); border-color: rgba(39, 174, 96, 0.3); }
.status-UNCOVERED_WORKLOAD, .status-AT_RISK, .status-BEHIND { background: rgba(245, 158, 11, 0.08); border-color: rgba(245, 158, 11, 0.3); }
.status-DEADLINE_PASSED { background: rgba(235, 94, 85, 0.08); border-color: rgba(235, 94, 85, 0.3); }
.status-NO_DATA { background: var(--surface-container-low); }

.status-dot-icon { width: 10px; height: 10px; border-radius: 50%; margin-top: 5px; flex-shrink: 0; background: currentColor; }
.status-ON_TRACK .status-dot-icon { background: #27ae60; }
.status-UNCOVERED_WORKLOAD .status-dot-icon, .status-AT_RISK .status-dot-icon, .status-BEHIND .status-dot-icon { background: #f59e0b; }
.status-DEADLINE_PASSED .status-dot-icon { background: #eb5e55; }
.status-NO_DATA .status-dot-icon { background: var(--muted-text); }
.status-headline { margin: 0 0 3px; font-size: 14px; font-weight: 700; }
.status-explanation { margin: 0; font-size: 13px; line-height: 1.4; color: var(--muted-text); }

.metrics-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.metric-item { display: flex; flex-direction: column; padding: 12px; border-radius: 10px; border: 1px solid var(--outline-variant); background: var(--surface-container-highest); }
.metric-label { font-size: 11px; font-weight: 600; color: var(--muted-text); margin-bottom: 3px; }
.metric-value { font-size: 15px; font-weight: 700; color: var(--on-surface); }
.metric-subtext { font-size: 11px; color: var(--muted-text); margin-top: 3px; }

.balance-card { padding: 12px 14px; border-radius: 10px; border: 1px solid var(--outline-variant); display: flex; flex-direction: column; gap: 4px; }
.balance-title { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em; }
.balance-metric { font-size: 14px; font-weight: 600; }
.balance-success { background: rgba(39, 174, 96, 0.06); border-color: rgba(39, 174, 96, 0.25); color: #27ae60; }
.balance-warning { background: rgba(245, 158, 11, 0.08); border-color: rgba(245, 158, 11, 0.3); color: #b45309; }
.balance-danger { background: rgba(235, 94, 85, 0.08); border-color: rgba(235, 94, 85, 0.3); color: #c0392b; }
.balance-neutral { background: var(--surface-container-low); border-color: var(--outline-variant); color: var(--muted-text); }

.historical-disclaimer { margin: 4px 0 0; font-size: 12px; font-style: italic; color: var(--muted-text); }

.action-options-card { display: flex; flex-direction: column; gap: 10px; }
.plan-action-btn {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-highest);
  color: var(--on-surface);
  cursor: pointer;
  text-align: left;
  transition: all 0.15s ease;
}
.plan-action-btn:hover { border-color: var(--primary); background: color-mix(in srgb, var(--primary) 5%, var(--surface-container-highest)); }
.btn-info { display: flex; flex-direction: column; gap: 2px; }
.btn-info strong { font-size: 13px; font-weight: 700; }
.btn-info span { font-size: 12px; color: var(--muted-text); }

/* Toggle Card */
.toggle-card {
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-highest);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.toggle-card-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.toggle-card-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.toggle-card-text strong {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}
.toggle-card-text span {
  font-size: 12px;
  color: var(--muted-text);
}

/* Switch Toggle Button */
.switch-toggle-btn {
  background: transparent;
  border: none;
  padding: 0;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  user-select: none;
  outline: none;
  flex-shrink: 0;
}
.switch-toggle-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.switch-slider-track {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  background-color: var(--surface-container-highest, #3a3a3c);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  transition: all 0.2s ease;
}
.switch-slider-thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  height: 14px;
  width: 14px;
  background-color: #fff;
  border-radius: 50%;
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.15);
}
.switch-toggle-btn.is-checked .switch-slider-track {
  background-color: var(--primary);
  border-color: var(--primary);
}
.switch-toggle-btn.is-checked .switch-slider-thumb {
  transform: translateX(16px);
}

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
.slot-display-info { display: flex; flex-direction: column; gap: 2px; }
.slot-display-name { font-size: 13px; font-weight: 700; color: var(--on-surface); }
.slot-display-time { font-size: 12px; color: var(--muted-text); font-variant-numeric: tabular-nums; }
.slot-display-badge {
  font-size: 11px;
  font-weight: 700;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container-highest));
  padding: 3px 8px;
  border-radius: 6px;
}

.action-error-msg {
  padding: 8px 12px;
  background: color-mix(in srgb, var(--error, #e53935) 15%, transparent);
  border: 1px solid var(--error, #e53935);
  color: var(--error, #e53935);
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
}
</style>
