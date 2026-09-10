<template>
  <div class="time-range-section">
    <div class="time-range-header">
      <div class="header-left">
        <label>Study Schedule</label>
        <span v-if="displayDuration" class="duration-badge">{{ displayDuration }}</span>
      </div>
      <button
        type="button"
        class="add-slot-btn"
        :disabled="disabled"
        @click="addSlot"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        <span>Add Schedule</span>
      </button>
    </div>

    <div class="slots-container">
      <div
        v-for="(slot, idx) in slots"
        :key="idx"
        class="slot-card"
      >
        <div class="slot-card-header">
          <div class="slot-title-group">
            <span class="slot-index-badge">#{{ idx + 1 }}</span>
            <input
              v-model="slot.name"
              type="text"
              class="slot-name-input"
              :placeholder="`Study Session ${idx + 1}`"
              :disabled="disabled"
              @input="onSlotChanged"
            />
          </div>
          <div class="slot-actions">
            <span v-if="computeSlotDuration(slot.start, slot.end)" class="slot-chip">
              {{ computeSlotDuration(slot.start, slot.end) }}
            </span>
            <button
              v-if="slots.length > 1"
              type="button"
              class="remove-slot-btn"
              :disabled="disabled"
              title="Remove schedule"
              aria-label="Remove schedule"
              @click="removeSlot(idx)"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>
        </div>

        <div class="time-range-container">
          <div class="time-input-group">
            <label :for="`study-start-time-${idx}`" class="time-label">Start</label>
            <div class="time-input-wrapper">
              <svg
                class="time-icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="2" x2="12" y2="4" />
                <line x1="22" y1="12" x2="20" y2="12" />
                <line x1="12" y1="20" x2="12" y2="22" />
                <line x1="2" y1="12" x2="4" y2="12" />
                <line x1="12" y1="4" x2="12" y2="8" />
                <line x1="12" y1="12" x2="12" y2="8" />
                <line x1="12" y1="12" x2="15.5" y2="14.5" />
                <circle cx="12" cy="12" r="1" fill="currentColor" />
              </svg>
              <input
                :id="`study-start-time-${idx}`"
                v-model="slot.start"
                type="time"
                class="time-input"
                :disabled="disabled"
                required
                @input="onSlotChanged"
              />
            </div>
          </div>

          <div class="time-connector">
            <svg viewBox="0 0 24 8" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M0 4 L20 4 M16 1 L20 4 L16 7" />
            </svg>
          </div>

          <div class="time-input-group">
            <label :for="`study-end-time-${idx}`" class="time-label">End</label>
            <div class="time-input-wrapper">
              <svg
                class="time-icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="2" x2="12" y2="4" />
                <line x1="22" y1="12" x2="20" y2="12" />
                <line x1="12" y1="20" x2="12" y2="22" />
                <line x1="2" y1="12" x2="4" y2="12" />
                <line x1="12" y1="4" x2="12" y2="8" />
                <line x1="12" y1="12" x2="12" y2="8" />
                <line x1="12" y1="12" x2="15.5" y2="14.5" />
                <circle cx="12" cy="12" r="1" fill="currentColor" />
              </svg>
              <input
                :id="`study-end-time-${idx}`"
                v-model="slot.end"
                type="time"
                class="time-input"
                :disabled="disabled"
                required
                @input="onSlotChanged"
              />
            </div>
          </div>
        </div>

        <div class="quick-durations">
          <button
            v-for="preset in durationPresets"
            :key="preset.label"
            type="button"
            class="duration-preset"
            :class="{ active: computeSlotDuration(slot.start, slot.end) === preset.label }"
            :disabled="disabled"
            @click="applyPresetToSlot(idx, preset)"
          >
            {{ preset.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  startValue: { type: String, default: '17:00' },
  endValue: { type: String, default: '18:00' },
  duration: { type: String, default: '' },
  slotsJson: { type: String, default: '[]' },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:startValue',
  'update:endValue',
  'update:slotsJson',
  'slots-changed',
  'apply-preset',
])

const durationPresets = [
  { label: '30 min', minutes: 30 },
  { label: '1 hour', minutes: 60 },
  { label: '1.5 hours', minutes: 90 },
  { label: '2 hours', minutes: 120 },
  { label: '3 hours', minutes: 180 },
]

const slots = ref([])

function parseSlotsFromProps() {
  let parsed = []
  if (props.slotsJson) {
    try {
      const arr = JSON.parse(props.slotsJson)
      if (Array.isArray(arr) && arr.length > 0) {
        parsed = arr.map((s, idx) => ({
          name: s.name || (idx === 0 ? 'Daily Study Session' : `Study Slot ${idx + 1}`),
          start: s.start || props.startValue || '17:00',
          end: s.end || props.endValue || '18:00',
        }))
      }
    } catch {
      parsed = []
    }
  }

  if (parsed.length === 0) {
    parsed = [
      {
        name: 'Daily Study Session',
        start: props.startValue || '17:00',
        end: props.endValue || '18:00',
      },
    ]
  }
  slots.value = parsed
}

watch(
  () => props.slotsJson,
  (newJson) => {
    if (newJson) {
      try {
        const arr = JSON.parse(newJson)
        if (Array.isArray(arr) && arr.length > 0) {
          // If content differs from current serialized state, sync it
          if (JSON.stringify(slots.value) !== JSON.stringify(arr)) {
            slots.value = arr.map((s, idx) => ({
              name: s.name || (idx === 0 ? 'Daily Study Session' : `Study Slot ${idx + 1}`),
              start: s.start || props.startValue || '17:00',
              end: s.end || props.endValue || '18:00',
            }))
          }
          return
        }
      } catch {
        // Ignore JSON parse errors
      }
    }
    if (slots.value.length === 0) {
      parseSlotsFromProps()
    }
  },
  { immediate: true }
)

function computeSlotDuration(start, end) {
  if (!start || !end) return ''
  const [sh, sm] = start.split(':').map(Number)
  const [eh, em] = end.split(':').map(Number)
  if (isNaN(sh) || isNaN(sm) || isNaN(eh) || isNaN(em)) return ''
  let diff = eh * 60 + em - (sh * 60 + sm)
  if (diff <= 0) return ''
  if (diff < 60) return `${diff} min`
  const hours = Math.floor(diff / 60)
  const mins = diff % 60
  if (mins === 0) return hours === 1 ? '1 hour' : `${hours} hours`
  if (mins === 30) return `${hours}.5 hours`
  return `${hours}h ${mins}m`
}

const displayDuration = computed(() => {
  if (slots.value.length === 0) return props.duration || ''
  let totalMinutes = 0
  for (const s of slots.value) {
    if (!s.start || !s.end) continue
    const [sh, sm] = s.start.split(':').map(Number)
    const [eh, em] = s.end.split(':').map(Number)
    if (isNaN(sh) || isNaN(sm) || isNaN(eh) || isNaN(em)) continue
    let diff = eh * 60 + em - (sh * 60 + sm)
    if (diff > 0) totalMinutes += diff
  }
  if (totalMinutes <= 0) return ''
  if (totalMinutes < 60) return `${totalMinutes} min total`
  const hours = Math.floor(totalMinutes / 60)
  const mins = totalMinutes % 60
  if (mins === 0) return `${hours === 1 ? '1 hour' : `${hours} hours`} total`
  return `${hours}h ${mins}m total`
})

function onSlotChanged() {
  const serialized = JSON.stringify(slots.value)
  emit('update:slotsJson', serialized)
  emit('slots-changed', slots.value)
  if (slots.value.length > 0) {
    emit('update:startValue', slots.value[0].start)
    emit('update:endValue', slots.value[0].end)
  }
}

function addSlot() {
  const last = slots.value[slots.value.length - 1]
  let newStart = '19:00'
  let newEnd = '20:00'
  if (last && last.end) {
    const [h, m] = last.end.split(':').map(Number)
    const nextStartMin = (h * 60 + m + 60) % 1440
    const nextEndMin = (nextStartMin + 60) % 1440
    newStart = `${String(Math.floor(nextStartMin / 60)).padStart(2, '0')}:${String(nextStartMin % 60).padStart(2, '0')}`
    newEnd = `${String(Math.floor(nextEndMin / 60)).padStart(2, '0')}:${String(nextEndMin % 60).padStart(2, '0')}`
  }

  slots.value.push({
    name: `Study Session ${slots.value.length + 1}`,
    start: newStart,
    end: newEnd,
  })
  onSlotChanged()
}

function removeSlot(index) {
  if (slots.value.length <= 1) return
  slots.value.splice(index, 1)
  onSlotChanged()
}

function applyPresetToSlot(index, preset) {
  const target = slots.value[index]
  if (!target || !target.start) return
  const [h, m] = target.start.split(':').map(Number)
  const endMinutes = h * 60 + m + preset.minutes
  const endH = Math.floor(endMinutes / 60) % 24
  const endM = endMinutes % 60
  target.end = `${String(endH).padStart(2, '0')}:${String(endM).padStart(2, '0')}`
  onSlotChanged()
}
</script>

<style scoped>
.time-range-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.time-range-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
}

.duration-badge {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
  padding: 3px 10px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
}

.add-slot-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
  cursor: pointer;
  transition: all 0.2s ease;
}

.add-slot-btn:hover:not(:disabled) {
  background: var(--primary);
  color: var(--on-primary);
  transform: translateY(-1px);
}

.add-slot-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.slots-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.slot-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: border-color 0.2s ease;
}

.slot-card:hover {
  border-color: color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
}

.slot-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.slot-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.slot-index-badge {
  font-size: 11px;
  font-weight: 700;
  color: var(--muted-text);
  background: var(--surface-container);
  padding: 2px 6px;
  border-radius: 6px;
}

.slot-name-input {
  font-size: 13px;
  font-weight: 600;
  padding: 4px 8px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  color: var(--on-surface);
  max-width: 220px;
}

.slot-name-input:focus {
  border-color: var(--outline-variant);
  background: var(--surface-container-lowest);
}

.slot-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slot-chip {
  font-size: 11px;
  font-weight: 600;
  color: var(--muted-text);
  background: var(--surface-container);
  padding: 2px 8px;
  border-radius: 6px;
}

.remove-slot-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: 1px solid var(--outline-variant);
  background: transparent;
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.2s ease;
}

.remove-slot-btn:hover:not(:disabled) {
  border-color: #ef4444;
  color: #ef4444;
  background: color-mix(in srgb, #ef4444 10%, transparent);
}

.duration-preset:hover:not(:disabled) {
  background: var(--surface-container);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
  color: var(--on-surface);
}

.duration-preset:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 480px) {
  .time-range-container {
    flex-direction: column;
    align-items: stretch;
  }

  .time-connector {
    transform: rotate(90deg);
    width: 100%;
    height: 24px;
  }

  .quick-durations {
    justify-content: center;
  }
}
</style>
