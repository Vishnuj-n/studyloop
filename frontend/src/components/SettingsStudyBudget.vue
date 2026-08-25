<template>
  <article class="panel form-grid">
    <h2>Study Budget & Routine</h2>

    <div class="form-group">
      <label for="max-flashcards">Max Flashcards per Session</label>
      <input
        id="max-flashcards"
        v-model.number="settings.max_flashcards_per_session"
        type="number"
        min="5"
        max="200"
        step="5"
        :disabled="disabled"
        required
      />
      <p class="hint">Caps the number of FSRS reviews active in any single study session.</p>
    </div>

    <div class="form-group">
      <label for="target-session-words">Target Reading Session Words</label>
      <input
        id="target-session-words"
        v-model.number="settings.target_session_words"
        type="number"
        min="1000"
        max="20000"
        step="500"
        :disabled="disabled"
        required
      />
      <p class="hint">
        Target word count per reading session (3,000 words ≈ 15 minutes of standard reading).
      </p>
      <p v-if="hasTokenWarning" class="warning-hint">
        ⚠️ {{ settings.target_session_words }} words (~{{ Math.round(settings.target_session_words * 1.3) }} tokens) may exceed your Max Input Tokens limit ({{ maxInputTokens }} tokens). Content may be truncated during quizzes.
      </p>
    </div>

    <TimeRangeInput
      :start-value="settings.study_start_time"
      :end-value="settings.study_end_time"
      :duration="studyDuration"
      :disabled="disabled"
      @update:start-value="settings.study_start_time = $event"
      @update:end-value="settings.study_end_time = $event"
      @apply-preset="applyDurationPreset"
    />

    <!-- Calendar Sync & Offline Reminders -->
    <div class="calendar-sync-card">
      <div class="calendar-header">
        <div>
          <h3>📅 Calendar Routine Sync</h3>
          <p class="hint">
            Export your daily study routine to your calendar. Your phone and laptop will remind you every day with alarms even when StudyLoop is closed.
          </p>
        </div>
        <button
          type="button"
          class="test-chime-btn"
          title="Play sample in-app chime"
          @click="playStudyChime"
        >
          🔔 Test In-App Chime
        </button>
      </div>

      <div class="form-group custom-url-group">
        <label for="custom-study-url">Custom Study Portal / Web Link (Optional)</label>
        <input
          id="custom-study-url"
          v-model="customStudyUrl"
          type="url"
          placeholder="e.g. https://my-school.edu/dashboard or your custom notes URL"
          class="custom-url-input"
        />
        <p class="hint">
          This link will be included in the reminder description on your phone & desktop calendar.
        </p>
      </div>

      <div class="calendar-actions">
        <button
          type="button"
          class="calendar-btn google-btn"
          @click="openGoogle"
        >
          📅 Add to Google Calendar
        </button>
        <button
          type="button"
          class="calendar-btn outlook-btn"
          @click="openOutlook"
        >
          📧 Add to Outlook Web
        </button>
        <button
          type="button"
          class="calendar-btn ics-btn"
          @click="downloadICS"
        >
          🍎 Download .ics (Apple / Windows)
        </button>
      </div>
    </div>

    <SettingsToggle
      v-model="settings.reminders_enabled"
      :disabled="disabled"
      title="Enable In-App Study Chimes & Banners"
      hint="Play an audio chime and show banners when daily study time starts and ends while StudyLoop is open."
    />

    <SettingsToggle
      v-model="settings.analytics_enabled"
      :disabled="disabled"
      title="Help improve the app by sharing anonymous usage data"
      hint="Telemetry events are anonymized. No personal information is ever collected."
    />

    <SettingsToggle
      v-model="settings.skip_to_reading_active"
      :disabled="disabled"
      title='Enable "Skip to Reading" (Escape Hatch)'
      hint="Temporarily deprioritizes review backlogs, letting you read new material first. FSRS records remain safe."
    />
  </article>
</template>

<script setup>
import { computed, ref } from 'vue'
import SettingsToggle from './SettingsToggle.vue'
import TimeRangeInput from './TimeRangeInput.vue'
import {
  playStudyChime,
  getGoogleCalendarUrl,
  getOutlookCalendarUrl,
  downloadRoutineICS,
} from '../services/calendarService'

const props = defineProps({
  settings: { type: Object, required: true },
  studyDuration: { type: String, default: '' },
  maxInputTokens: { type: Number, default: 4000 },
  disabled: { type: Boolean, default: false },
})

const customStudyUrl = ref('')

const hasTokenWarning = computed(() => {
  const words = Number(props.settings?.target_session_words) || 0
  const maxTokens = Number(props.maxInputTokens) || 4000
  return words * 1.3 > maxTokens
})

const emit = defineEmits(['apply-duration-preset'])

function applyDurationPreset(preset) {
  emit('apply-duration-preset', preset)
}

function openGoogle() {
  const url = getGoogleCalendarUrl(
    props.settings?.study_start_time,
    props.settings?.study_end_time,
    customStudyUrl.value
  )
  window.open(url, '_blank')
}

function openOutlook() {
  const url = getOutlookCalendarUrl(
    props.settings?.study_start_time,
    props.settings?.study_end_time,
    customStudyUrl.value
  )
  window.open(url, '_blank')
}

function downloadICS() {
  downloadRoutineICS(
    props.settings?.study_start_time,
    props.settings?.study_end_time,
    customStudyUrl.value
  )
}
</script>

<style scoped>
label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
}

input[type='number'],
input[type='url'] {
  border: 1px solid color-mix(in srgb, var(--outline-variant) 20%, transparent);
  border-radius: 12px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  padding: 12px 14px;
  font-size: 14px;
  font-family: inherit;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

input[type='number']:focus,
input[type='url']:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 15%, transparent);
  outline: none;
}

.hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--muted-text);
  line-height: 1.4;
}

.warning-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--warning, #f59e0b);
  line-height: 1.4;
  font-weight: 500;
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

h2 {
  font-size: 20px;
  margin: 0 0 16px;
  font-weight: 700;
}

.panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid color-mix(in srgb, var(--outline-variant) 20%, transparent);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

/* Calendar Sync Card */
.calendar-sync-card {
  background: var(--surface-container-low);
  border: 1px solid color-mix(in srgb, var(--outline-variant) 25%, transparent);
  border-radius: 14px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.calendar-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}

.calendar-header h3 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface);
}

.test-chime-btn {
  padding: 8px 14px;
  font-size: 13px;
  font-weight: 600;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--outline-variant) 30%, transparent);
  background: var(--surface-container);
  color: var(--on-surface);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.test-chime-btn:hover {
  background: var(--surface-container-high);
  border-color: var(--primary);
  transform: translateY(-1px);
}

.custom-url-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.custom-url-input {
  width: 100%;
  box-sizing: border-box;
}

.calendar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.calendar-btn {
  flex: 1;
  min-width: 180px;
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  border-radius: 10px;
  border: 1px solid color-mix(in srgb, var(--outline-variant) 30%, transparent);
  background: var(--surface-container-highest, #eceef4);
  color: var(--on-surface);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.2s ease;
}

.calendar-btn:hover {
  border-color: var(--primary);
  background: var(--primary-container, #dbe2f9);
  color: var(--on-primary-container, #131b2e);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--primary) 15%, transparent);
}
</style>
