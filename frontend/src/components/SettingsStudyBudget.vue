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
      <label for="max-active-notebooks">Max Active Notebooks</label>
      <input
        id="max-active-notebooks"
        v-model.number="settings.max_active_notebooks"
        type="number"
        min="0"
        max="50"
        step="1"
        :disabled="disabled"
        required
      />
      <p class="hint">Maximum active textbooks/decks per profile simultaneously (default 4, set 0 for unlimited).</p>
    </div>

    <div class="settings-row-pair">
      <div class="form-group field-half">
        <label for="quiz-question-count">Questions per Quiz</label>
        <input
          id="quiz-question-count"
          v-model.number="settings.quiz_question_count"
          type="number"
          min="3"
          max="15"
          step="1"
          :disabled="disabled"
          required
        />
        <p class="hint">Target number of questions generated per quiz attempt (3–15, default 8).</p>
      </div>

      <div class="form-group field-half">
        <label for="quiz-passing-score">Passing Score (%)</label>
        <select
          id="quiz-passing-score"
          v-model.number="settings.quiz_passing_score"
          :disabled="disabled"
          class="setting-select"
        >
          <option :value="60">60% (Lenient)</option>
          <option :value="70">70% (Standard)</option>
          <option :value="80">80% (Strict)</option>
          <option :value="90">90% (Mastery)</option>
        </select>
        <p class="hint">Minimum score required to master a topic without remedial review.</p>
      </div>
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
        Warning: {{ settings.target_session_words }} words (~{{ Math.round(settings.target_session_words * 1.3) }} tokens) may exceed your Max Input Tokens limit ({{ maxInputTokens }} tokens). Content may be truncated during quizzes.
      </p>
    </div>

    <div class="form-group">
      <label for="min-session-words">Minimum Reading Session Words</label>
      <input
        id="min-session-words"
        v-model.number="settings.min_session_words"
        type="number"
        min="0"
        :max="settings.target_session_words || 20000"
        step="500"
        :disabled="disabled"
      />
      <p class="hint">
        Minimum word threshold before ending a slice (set to 0 for automatic default).
      </p>
    </div>

    <TimeRangeInput
      :start-value="settings.study_start_time"
      :end-value="settings.study_end_time"
      :slots-json="settings.study_slots_json"
      :duration="studyDuration"
      :disabled="disabled"
      @update:start-value="settings.study_start_time = $event"
      @update:end-value="settings.study_end_time = $event"
      @update:slots-json="onUpdateSlotsJson"
      @slots-changed="onSlotsChanged"
      @apply-preset="applyDurationPreset"
    />

    <!-- Calendar Sync -->
    <div class="calendar-sync-card">
      <div class="calendar-header">
        <div>
          <h3>Calendar Routine Sync</h3>
          <p class="hint">
            Export your daily study routine to your calendar. The .ics file imports all study sessions at once. You can also add individual sessions directly to Google or Outlook.
          </p>
        </div>
        <button
          type="button"
          class="calendar-btn ics-btn main-ics-btn"
          @click="downloadICS"
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="7 10 12 15 17 10" />
            <line x1="12" y1="15" x2="12" y2="3" />
          </svg>
          <span>Download All Sessions (.ics)</span>
        </button>
      </div>

      <!-- Multi-slot Web Calendar Direct Links -->
      <div class="slots-calendar-list">
        <div
          v-for="(slot, idx) in getActiveSlots()"
          :key="idx"
          class="slot-cal-row"
        >
          <div class="slot-cal-info">
            <span class="slot-cal-name">{{ slot.name || `Study Session ${idx + 1}` }}</span>
            <span class="slot-cal-time">{{ slot.start }} – {{ slot.end }}</span>
          </div>
          <div class="slot-cal-actions">
            <button
              type="button"
              class="cal-mini-link-btn"
              title="Add this session to Google Calendar"
              @click="openGoogleForSlot(slot)"
            >
              + Google Calendar
            </button>
            <button
              type="button"
              class="cal-mini-link-btn"
              title="Add this session to Outlook Web"
              @click="openOutlookForSlot(slot)"
            >
              + Outlook Web
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- In-App Notifications Section -->
    <div class="notification-card">
      <div class="notification-header">
        <div>
          <h3>In-App Notifications</h3>
          <p class="hint">
            Play an audio chime and show banner alerts when daily study time starts and ends while StudyLoop is open.
          </p>
        </div>
        <button
          type="button"
          class="test-chime-btn"
          title="Play sample in-app chime"
          @click="playStudyChime"
        >
          Test Chime
        </button>
      </div>

      <SettingsToggle
        v-model="settings.reminders_enabled"
        :disabled="disabled"
        title="Enable Study Time Chimes & Banners"
        hint="Notify when your scheduled study session begins and concludes."
      />

      <SettingsToggle
        v-model="soundEnabled"
        :disabled="disabled"
        title="Interactive Sound Effects"
        hint="Play procedural audio chimes and feedback when answering quizzes, reviewing cards, and unlocking mystery chests."
      />

      <SettingsToggle
        v-model="settings.show_reward_notifications"
        :disabled="disabled"
        title="Show study reward notifications"
        hint="Show the non-blocking XP and mystery chest confirmation after a study session. Rewards remain available in the vault when disabled."
      />
    </div>

    <!-- Pomodoro & Focus Music Section -->
    <div class="notification-card">
      <div class="notification-header">
        <div>
          <h3>Pomodoro &amp; Focus Audio</h3>
          <p class="hint">
            Run a lightweight background focus timer and loop study music or lo-fi folders during study sessions.
          </p>
        </div>
      </div>

      <SettingsToggle
        v-model="pomoSettings.enabled"
        :disabled="disabled"
        title="Enable Pomodoro Focus Timer Widget"
        hint="Display the compact Pomodoro timer in your sidebar navigation."
        @update:model-value="onSavePomoSettings"
      />

      <div v-if="pomoSettings.enabled" class="pomo-settings-grid">
        <div class="settings-row-pair">
          <div class="form-group field-half">
            <label for="pomo-work-min">Focus Duration (Minutes)</label>
            <input
              id="pomo-work-min"
              v-model.number="workMinutes"
              type="number"
              min="1"
              max="120"
              step="1"
              :disabled="disabled"
            />
          </div>
          <div class="form-group field-half">
            <label for="pomo-break-min">Break Duration (Minutes)</label>
            <input
              id="pomo-break-min"
              v-model.number="breakMinutes"
              type="number"
              min="0"
              max="60"
              step="1"
              :disabled="disabled"
            />
          </div>
        </div>

        <div class="form-group">
          <label>Focus Study Music (MP3 File or Lo-Fi Folder)</label>
          <div class="music-path-row">
            <input
              v-model="pomoMusicPath"
              type="text"
              placeholder="No audio track configured"
              readonly
              class="music-path-input"
            />
            <button
              type="button"
              class="action-mini-btn"
              :disabled="disabled"
              title="Select single MP3 audio track"
              @click="handlePickMusicFile"
            >
              <span>Pick File</span>
            </button>
            <button
              type="button"
              class="action-mini-btn"
              :disabled="disabled"
              title="Select folder with MP3 tracks (shuffle lo-fi)"
              @click="handlePickMusicFolder"
            >
              <span>Pick Folder</span>
              <span v-if="!isPro" class="pro-tag">#PRO</span>
            </button>
            <button
              v-if="pomoMusicPath"
              type="button"
              class="action-mini-btn"
              :class="{ 'active-mode-btn': pomoIsShuffle }"
              :disabled="disabled"
              :title="pomoIsShuffle ? 'Shuffle is ON (Click to switch to loop mode)' : 'Shuffle is OFF (Click to turn Shuffle ON)'"
              @click="handleToggleShuffle"
            >
              <span>Shuffle: {{ pomoIsShuffle ? 'ON' : 'OFF' }}</span>
              <span v-if="!isPro && !pomoIsShuffle" class="pro-tag">#PRO</span>
            </button>
            <button
              v-if="pomoMusicPath"
              type="button"
              class="action-mini-btn clear-btn"
              :disabled="disabled"
              title="Clear audio"
              aria-label="Clear audio"
              @click="handleClearMusic"
            >
              <span>Clear</span>
            </button>
          </div>
          <p class="hint">
            {{ pomoIsShuffle ? 'Shuffling all tracks in selected folder.' : 'Looping audio continuously during focus.' }}
          </p>
        </div>

        <div class="form-group">
          <label for="pomo-volume">Music Volume ({{ pomoSettings.defaultVolume ?? 70 }}%)</label>
          <input
            id="pomo-volume"
            v-model.number="pomoSettings.defaultVolume"
            type="range"
            min="0"
            max="100"
            step="5"
            :disabled="disabled"
            class="volume-slider"
            :style="{
              background: `linear-gradient(to right, var(--primary) 0%, var(--primary) ${pomoSettings.defaultVolume ?? 70}%, var(--surface-container-highest) ${pomoSettings.defaultVolume ?? 70}%, var(--surface-container-highest) 100%)`
            }"
            @change="onSavePomoSettings"
          />
        </div>
      </div>
    </div>

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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import SettingsToggle from './SettingsToggle.vue'
import TimeRangeInput from './TimeRangeInput.vue'
import { isAudioMuted, setAudioMuted } from '../utils/audioJuice'
import {
  playStudyChime,
  getGoogleCalendarUrl,
  getOutlookCalendarUrl,
  downloadRoutineICS,
} from '../services/calendarService'
import { openURLInBrowser } from '../services/appApi'
import {
  getPomodoroSettings,
  savePomodoroSettings,
  loadPomodoroProfiles,
  savePomodoroProfile,
  pickPomodoroMusicFile,
  pickPomodoroMusicFolder,
} from '../services/pomodoroApi'
import { useClerkAuth } from '../services/clerkAuth'

const { isPro, openBilling } = useClerkAuth()

const props = defineProps({
  settings: { type: Object, required: true },
  studyDuration: { type: String, default: '' },
  maxInputTokens: { type: Number, default: 4000 },
  disabled: { type: Boolean, default: false },
})

const soundEnabled = ref(!isAudioMuted())

watch(soundEnabled, (enabled) => {
  setAudioMuted(!enabled)
})

// Pomodoro state
const pomoSettings = ref({
  enabled: true,
  defaultVolume: 70,
  autoStartAudio: true,
})
const pomoProfile = ref({
  id: 'default',
  name: 'Standard Focus',
  durationSec: 25 * 60,
  breakDurationSec: 5 * 60,
  musicPath: '',
  shuffle: false,
  isDefault: true,
})

const workMinutes = ref(25)
const breakMinutes = ref(5)

const pomoMusicPath = computed(() => pomoProfile.value.musicPath || '')
const pomoIsShuffle = computed(() => !!pomoProfile.value.shuffle)

let isInternalUpdate = false

async function loadPomoData() {
  try {
    const s = await getPomodoroSettings()
    if (s) {
      pomoSettings.value = { ...pomoSettings.value, ...s }
      if (typeof s.enabled === 'undefined') {
        pomoSettings.value.enabled = true
      }
    }
    const profiles = await loadPomodoroProfiles()
    if (Array.isArray(profiles) && profiles.length > 0) {
      const def = profiles.find((p) => p.isDefault) || profiles[0]
      pomoProfile.value = def
      isInternalUpdate = true
      workMinutes.value = Math.round((def.durationSec || 25 * 60) / 60)
      breakMinutes.value = Math.round((def.breakDurationSec ?? 5 * 60) / 60)
      setTimeout(() => {
        isInternalUpdate = false
      }, 50)
    }
  } catch (err) {
    console.warn('[POMODORO_SETTINGS] Error loading config:', err)
  }
}

async function onSavePomoSettings() {
  try {
    await savePomodoroSettings({
      ...pomoSettings.value,
      enabled: Boolean(pomoSettings.value.enabled),
      defaultVolume: Number(pomoSettings.value.defaultVolume) || 70,
    })
    window.dispatchEvent(new CustomEvent('pomodoro-settings-updated'))
  } catch (err) {
    console.error('[POMODORO_SETTINGS] Error saving settings:', err)
  }
}

let pomoSaveTimer = null

watch([workMinutes, breakMinutes], ([newWork, newBreak]) => {
  if (isInternalUpdate) return

  if (pomoSaveTimer) clearTimeout(pomoSaveTimer)
  pomoSaveTimer = setTimeout(async () => {
    const w = typeof newWork === 'number' && newWork > 0 ? newWork : 25
    const b = typeof newBreak === 'number' && newBreak >= 0 ? newBreak : 0

    pomoProfile.value.durationSec = w * 60
    pomoProfile.value.breakDurationSec = b * 60

    try {
      await savePomodoroProfile(pomoProfile.value)
      window.dispatchEvent(new CustomEvent('pomodoro-settings-updated'))
    } catch (err) {
      console.error('[POMODORO_SETTINGS] Error saving profile:', err)
    }
  }, 350)
})

async function onSavePomoProfile() {
  if (pomoSaveTimer) {
    clearTimeout(pomoSaveTimer)
    pomoSaveTimer = null
  }
  try {
    await savePomodoroProfile(pomoProfile.value)
    window.dispatchEvent(new CustomEvent('pomodoro-settings-updated'))
  } catch (err) {
    console.error('[POMODORO_SETTINGS] Error saving profile:', err)
  }
}

async function handlePickMusicFile() {
  try {
    const path = await pickPomodoroMusicFile()
    if (path) {
      pomoProfile.value.musicPath = path
      pomoProfile.value.shuffle = false
      await onSavePomoProfile()
    }
  } catch (err) {
    console.warn('[POMODORO_SETTINGS] Error picking music file:', err)
  }
}

async function handlePickMusicFolder() {
  if (!isPro.value) {
    openBilling()
    return
  }
  try {
    const path = await pickPomodoroMusicFolder()
    if (path) {
      pomoProfile.value.musicPath = path
      pomoProfile.value.shuffle = true
      await onSavePomoProfile()
    }
  } catch (err) {
    console.warn('[POMODORO_SETTINGS] Error picking music folder:', err)
  }
}

async function handleToggleShuffle() {
  if (!pomoProfile.value.shuffle && !isPro.value) {
    openBilling()
    return
  }
  pomoProfile.value.shuffle = !pomoProfile.value.shuffle
  await onSavePomoProfile()
}

async function handleClearMusic() {
  pomoProfile.value.musicPath = ''
  pomoProfile.value.shuffle = false
  await onSavePomoProfile()
}

onMounted(() => {
  loadPomoData()
  window.addEventListener('profile-switched', loadPomoData)
})

onUnmounted(() => {
  window.removeEventListener('profile-switched', loadPomoData)
})

const hasTokenWarning = computed(() => {
  const words = Number(props.settings?.target_session_words) || 0
  const maxTokens = Number(props.maxInputTokens) || 4000
  return words * 1.3 > maxTokens
})

const emit = defineEmits(['apply-duration-preset'])

function applyDurationPreset(preset) {
  emit('apply-duration-preset', preset)
}

async function openExternalLink(url) {
  try {
    await openURLInBrowser(url)
  } catch {
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}

const localSlots = ref([])

function onSlotsChanged(newSlots) {
  localSlots.value = Array.isArray(newSlots) ? JSON.parse(JSON.stringify(newSlots)) : []
}

function onUpdateSlotsJson(json) {
  props.settings.study_slots_json = json
  try {
    const parsed = JSON.parse(json)
    if (Array.isArray(parsed)) {
      localSlots.value = parsed
    }
  } catch {
    // Ignore
  }
}

function getActiveSlots() {
  if (localSlots.value && localSlots.value.length > 0) {
    return localSlots.value
  }
  if (props.settings?.study_slots_json) {
    try {
      const parsed = JSON.parse(props.settings.study_slots_json)
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed
      }
    } catch {
      // Fallback
    }
  }
  return [
    {
      name: 'Daily Study Session',
      start: props.settings?.study_start_time || '17:00',
      end: props.settings?.study_end_time || '18:00',
    },
  ]
}

function openGoogleForSlot(slot) {
  if (!slot) return
  const url = getGoogleCalendarUrl(slot.start || '17:00', slot.end || '18:00')
  openExternalLink(url)
}

function openOutlookForSlot(slot) {
  if (!slot) return
  const url = getOutlookCalendarUrl(slot.start || '17:00', slot.end || '18:00', slot.name || 'Study Session')
  openExternalLink(url)
}

function openGoogle() {
  const slots = getActiveSlots()
  const first = slots[0] || { start: '17:00', end: '18:00' }
  openGoogleForSlot(first)
}

function openOutlook() {
  const slots = getActiveSlots()
  const first = slots[0] || { start: '17:00', end: '18:00' }
  openOutlookForSlot(first)
}

function downloadICS() {
  const slots = getActiveSlots()
  downloadRoutineICS(slots)
}
</script>

<style scoped>
.pomo-settings-grid {
  display: grid;
  gap: 14px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--outline-variant);
}

.music-path-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.music-path-input {
  flex: 1;
}

.action-mini-btn {
  padding: 10px 14px;
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s ease;
}

.action-mini-btn:hover:not(:disabled) {
  background: var(--surface-container-highest);
  color: var(--primary);
  border-color: var(--primary);
}

.action-mini-btn.active-mode-btn {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border-color: var(--primary);
  color: var(--primary);
}

.action-mini-btn.clear-btn {
  padding: 10px 12px;
  color: #ef4444;
}

.action-mini-btn.clear-btn:hover:not(:disabled) {
  border-color: #ef4444;
}

.pro-tag {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 28%, transparent);
  margin-left: 6px;
  letter-spacing: 0.03em;
  display: inline-block;
  vertical-align: middle;
  line-height: 1.2;
}

label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
}

input[type='text'],
input[type='number'],
input[type='url'],
select {
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  padding: 12px 14px;
  font-size: 14px;
  font-family: inherit;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
  width: 100%;
  box-sizing: border-box;
}

input[type='text']:focus,
input[type='number']:focus,
input[type='url']:focus,
select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 15%, transparent);
  outline: none;
}

.volume-slider {
  appearance: none;
  -webkit-appearance: none;
  width: 100%;
  height: 6px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  border-radius: 9999px;
  outline: none;
  cursor: pointer;
  padding: 0;
  margin: 10px 0 6px;
  transition: background 0.2s ease, border-color 0.2s ease;
}

.volume-slider:focus-visible {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 20%, transparent);
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary);
  border: 2px solid var(--surface-container-lowest);
  box-shadow: 0 2px 6px color-mix(in srgb, var(--on-surface) 25%, transparent);
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.volume-slider::-webkit-slider-thumb:hover {
  transform: scale(1.15);
  box-shadow: 0 0 0 6px color-mix(in srgb, var(--primary) 18%, transparent);
}

.volume-slider::-webkit-slider-thumb:active {
  transform: scale(1.05);
  box-shadow: 0 0 0 8px color-mix(in srgb, var(--primary) 28%, transparent);
}

.volume-slider::-moz-range-track {
  height: 6px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  border-radius: 9999px;
}

.volume-slider::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--primary);
  border: 2px solid var(--surface-container-lowest);
  box-shadow: 0 2px 6px color-mix(in srgb, var(--on-surface) 25%, transparent);
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.volume-slider::-moz-range-thumb:hover {
  transform: scale(1.15);
  box-shadow: 0 0 0 6px color-mix(in srgb, var(--primary) 18%, transparent);
}

.volume-slider::-moz-range-thumb:active {
  transform: scale(1.05);
  box-shadow: 0 0 0 8px color-mix(in srgb, var(--primary) 28%, transparent);
}

.volume-slider:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.settings-row-pair {
  display: flex;
  gap: 16px;
  width: 100%;
}

.field-half {
  flex: 1;
  display: flex;
  flex-direction: column;
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
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

/* Calendar Sync Card */
.calendar-sync-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
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

/* Notification Card */
.notification-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}

.notification-header h3 {
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
  border: 1px solid var(--outline-variant);
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

.calendar-btn {
  flex: 1;
  min-width: 180px;
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
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

.main-ics-btn {
  background: var(--primary);
  color: var(--on-primary);
  border-color: var(--primary);
}

.main-ics-btn:hover {
  background: color-mix(in srgb, var(--primary) 85%, black);
  color: var(--on-primary);
  border-color: var(--primary);
}

.slots-calendar-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 4px;
}

.slot-cal-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  flex-wrap: wrap;
}

.slot-cal-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.slot-cal-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}

.slot-cal-time {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted-text);
  background: var(--surface-container-high);
  padding: 2px 8px;
  border-radius: 6px;
}

.slot-cal-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cal-mini-link-btn {
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface);
  cursor: pointer;
  transition: all 0.2s ease;
}

.cal-mini-link-btn:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--surface-container-highest);
}
</style>
