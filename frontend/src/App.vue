<script setup>
import Sidebar from './components/Sidebar.vue'
import { useRoute } from 'vue-router'
import { onMounted, onUnmounted, ref } from 'vue'
import ConfirmModal from './components/ConfirmModal.vue'
import MysteryChestModal from './components/MysteryChestModal.vue'
import RewardToast from './components/RewardToast.vue'
import AppToast from './components/AppToast.vue'
import {
  getUserSettings,
  updateUserSettings,
  getTodayPlan,
  checkForUpdates,
  openRepoURL,
} from './services/appApi'
import { useToast } from './composables/useToast'
import { playStudyChime } from './services/calendarService'
import { EventsOn } from '../wailsjs/runtime/runtime'

const { toast, hideToast, showNotice, showError } = useToast()

const route = useRoute()

const banner = ref({
  show: false,
  type: 'start',
  title: '',
  desc: '',
  unfinishedCount: 0,
})

let schedulerTimeout = null

function handleGlobalIngestionProgress(payload) {
  if (!payload || typeof payload !== 'object') return
  if (payload.status === 'draft_ready') {
    showNotice(
      '✨ Deep Structured extraction complete! Click the book card or dashboard banner to review your chapter syllabus.',
      'Extraction Complete'
    )
  } else if (payload.status === 'failed') {
    showError(
      `Extraction failed: ${payload.message || 'Error parsing PDF'}`,
      'Extraction Error'
    )
  }
}

// Helper to parse "HH:MM" into a Date object on a given base date
function parseTime(timeStr, baseDate) {
  const [h, m] = timeStr.split(':').map(Number)
  const d = new Date(baseDate)
  d.setHours(h, m, 0, 0)
  return d
}

// Calculate delay in ms and event type for next closest start or end time
function getNextEventTimeout(startTimeStr, endTimeStr) {
  if (!startTimeStr || !endTimeStr) return null
  const now = new Date()
  const events = []

  // Start time event
  const startToday = parseTime(startTimeStr, now)
  if (startToday > now) {
    events.push({ type: 'start', time: startToday })
  } else {
    const startTomorrow = parseTime(startTimeStr, new Date(now.getTime() + 86400000))
    events.push({ type: 'start', time: startTomorrow })
  }

  // End time event
  const endToday = parseTime(endTimeStr, now)
  if (endToday > now) {
    events.push({ type: 'end', time: endToday })
  } else {
    const endTomorrow = parseTime(endTimeStr, new Date(now.getTime() + 86400000))
    events.push({ type: 'end', time: endTomorrow })
  }

  // Sort events to find the closest upcoming one
  events.sort((a, b) => a.time - b.time)
  const next = events[0]
  return {
    type: next.type,
    delay: next.time.getTime() - now.getTime(),
  }
}

// Fire audio chime and in-app banner based on event type
async function fireEvent(type) {
  playStudyChime()

  if (type === 'start') {
    banner.value = {
      show: true,
      type: 'start',
      title: 'Study Time Started!',
      desc: 'Your study window has started. Time to work on your queue!',
      unfinishedCount: 0,
    }
  } else if (type === 'end') {
    let unfinishedCount = 0
    try {
      const plan = await getTodayPlan()
      if (plan && plan.tasks) {
        unfinishedCount = plan.tasks.length
      }
    } catch (err) {
      console.error('Failed to check tasks at end time:', err)
    }

    if (unfinishedCount > 0) {
      banner.value = {
        show: true,
        type: 'end',
        title: 'Study Time is Up!',
        desc: `You still have ${unfinishedCount} unfinished study tasks remaining today.`,
        unfinishedCount,
      }
    } else {
      banner.value = {
        show: true,
        type: 'end',
        title: 'Study Time is Up!',
        desc: 'Great job! You finished all your study tasks for today.',
        unfinishedCount: 0,
      }
    }
  }
}

// Fetch settings and schedule next setTimeout
async function syncScheduler() {
  if (schedulerTimeout) {
    clearTimeout(schedulerTimeout)
    schedulerTimeout = null
  }

  try {
    const settings = await getUserSettings()
    if (!settings || settings.error) return

    // Apply theme
    if (settings.theme) {
      document.documentElement.setAttribute('data-theme', settings.theme)
      localStorage.setItem('app-theme', settings.theme)
    }

    if (!settings.reminders_enabled) return

    const next = getNextEventTimeout(settings.study_start_time, settings.study_end_time)
    if (!next) return

    // Schedule next timeout
    schedulerTimeout = setTimeout(async () => {
      await fireEvent(next.type)
      syncScheduler() // Queue up the next event
    }, next.delay)
  } catch (err) {
    console.error('Scheduler sync failed:', err)
  }
}

// Extend study window by X minutes
async function extendStudyWindow(minutes) {
  try {
    const settings = await getUserSettings()
    if (!settings || settings.error) return

    const [h, m] = settings.study_end_time.split(':').map(Number)
    let newMins = h * 60 + m + minutes
    if (newMins >= 1440) {
      newMins -= 1440 // Midnight wrap-around
    }

    const newH = Math.floor(newMins / 60)
      .toString()
      .padStart(2, '0')
    const newM = (newMins % 60).toString().padStart(2, '0')
    const newEndTimeStr = `${newH}:${newM}`

    const updated = { ...settings, study_end_time: newEndTimeStr }
    const res = await updateUserSettings(updated)

    if (res.error) {
      console.error('Failed to extend study window:', res.error)
      return
    }

    window.dispatchEvent(new CustomEvent('settings-updated'))
    banner.value.show = false
  } catch (err) {
    console.error('Extend study window failed:', err)
  }
}

function closeBanner() {
  banner.value.show = false
}

const showUpdateModal = ref(false)
const currentVersion = ref('')
const latestVersion = ref('')

const pendingReward = ref(null)
const activeChest = ref(null)
const activeChestTitle = ref('')

function handleStudyReward(event) {
  const rewards = event?.detail?.rewards
  if (!rewards) return

  pendingReward.value = rewards
  window.dispatchEvent(new Event('gamification-updated'))
}

function openRewardChest(box) {
  activeChestTitle.value = pendingReward.value?.new_title_unlocked || ''
  pendingReward.value = null
  activeChest.value = box
}

function dismissRewardToast() {
  pendingReward.value = null
}

function closeActiveChest() {
  activeChest.value = null
  activeChestTitle.value = ''
}

async function checkAppUpdates() {
  try {
    const res = await checkForUpdates()
    if (res && res.update_available) {
      currentVersion.value = (res.current_version || '').replace(/^v+/, '')
      latestVersion.value = (res.latest_version || '').replace(/^v+/, '')
      showUpdateModal.value = true
    }
  } catch (err) {
    console.error('Failed to check for updates on startup:', err)
  }
}

function goToUpdatePage() {
  showUpdateModal.value = false
  openRepoURL()
}

let cancelIngestionListener = null

onMounted(() => {
  syncScheduler()
  window.addEventListener('settings-updated', syncScheduler)
  checkAppUpdates()
  cancelIngestionListener = EventsOn('ingestion-progress', handleGlobalIngestionProgress)
  window.addEventListener('study-reward-earned', handleStudyReward)
})

onUnmounted(() => {
  if (schedulerTimeout) {
    clearTimeout(schedulerTimeout)
    schedulerTimeout = null
  }
  window.removeEventListener('settings-updated', syncScheduler)
  window.removeEventListener('study-reward-earned', handleStudyReward)
  if (cancelIngestionListener) cancelIngestionListener()
})
</script>

<template>
  <div class="app-shell">
    <Sidebar v-if="route.path !== '/onboarding'" />

    <main class="content-shell">
      <!-- Update Alert Modal -->
      <div v-if="showUpdateModal" class="update-modal-overlay">
        <div class="update-modal">
          <div class="update-modal-header">
            <span class="warning-icon">🚀</span>
            <h2>Update Available</h2>
          </div>
          <div class="update-modal-body">
            <p class="update-msg">A new update for Studyloop is ready!</p>
            <div class="version-badge-container">
              <span class="version-badge current">v{{ currentVersion }}</span>
              <span class="version-arrow">→</span>
              <span class="version-badge latest">v{{ latestVersion }}</span>
            </div>
            <p class="warning-text">
              Click "Get Update" to go to the releases repository and download the latest version.
            </p>
          </div>
          <div class="update-modal-footer">
            <button class="modal-btn secondary" @click="showUpdateModal = false">
              Remind Me Later
            </button>
            <button class="modal-btn primary" @click="goToUpdatePage">Get Update</button>
          </div>
        </div>
      </div>

      <!-- Global study reminder banner -->
      <div v-if="banner.show" class="study-alert-banner">
        <div class="banner-content">
          <div class="banner-icon-badge">
            <svg
              v-if="banner.type === 'start'"
              xmlns="http://www.w3.org/2000/svg"
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
            <svg
              v-else
              xmlns="http://www.w3.org/2000/svg"
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M5 22h14"></path>
              <path d="M5 2h14"></path>
              <path d="M17 22v-4.172a2 2 0 0 0-.586-1.414L12 12l-4.414 4.414A2 2 0 0 0 7 17.828V22"></path>
              <path d="M7 2v4.172a2 2 0 0 0 .586 1.414L12 12l4.414-4.414A2 2 0 0 0 17 6.172V2"></path>
            </svg>
          </div>
          <div class="banner-text">
            <strong class="banner-title">{{ banner.title }}</strong>
            <p class="banner-desc">{{ banner.desc }}</p>
          </div>
        </div>
        <div class="banner-actions">
          <template v-if="banner.type === 'end' && banner.unfinishedCount > 0">
            <button class="banner-btn primary" @click="extendStudyWindow(15)">+15 mins</button>
            <button class="banner-btn primary" @click="extendStudyWindow(30)">+30 mins</button>
          </template>
          <button class="banner-btn secondary" @click="closeBanner">Dismiss</button>
        </div>
      </div>

      <RouterView />
      <ConfirmModal />

      <RewardToast
        v-if="pendingReward"
        :rewards="pendingReward"
        @open-chest="openRewardChest"
        @dismiss="dismissRewardToast"
      />
      <MysteryChestModal
        v-if="activeChest"
        :box="activeChest"
        :new-title="activeChestTitle"
        @close="closeActiveChest"
      />

      <!-- Global Toaster -->
      <AppToast />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  width: 100%;
  height: 100vh;
  display: flex;
  background: var(--background);
  overflow: hidden;
}

.content-shell {
  flex: 1;
  min-height: 0;
  padding: 16px 20px;
  overflow-y: auto;
  position: relative;
}

.content-shell::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.content-shell::-webkit-scrollbar-track {
  background: transparent;
}

.content-shell::-webkit-scrollbar-thumb {
  background: var(--surface-container, rgba(0, 0, 0, 0.15));
  border-radius: 99px;
}

.content-shell::-webkit-scrollbar-thumb:hover {
  background: var(--outline-variant, rgba(0, 0, 0, 0.12));
}

/* ── Global study reminder banner ── */
.study-alert-banner {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  box-shadow: 0 20px 40px rgba(45, 51, 56, 0.06);
  backdrop-filter: blur(12px);
  border-radius: 16px;
  padding: 12px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  width: 90%;
  max-width: 580px;
  animation: slideDown 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

@keyframes slideDown {
  from {
    transform: translate(-50%, -30px);
    opacity: 0;
  }
  to {
    transform: translate(-50%, 0);
    opacity: 1;
  }
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 14px;
}

.banner-icon-badge {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--surface-container);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.banner-text {
  display: flex;
  flex-direction: column;
}

.banner-title {
  font-family: 'Manrope', sans-serif;
  color: var(--on-surface);
  font-size: 0.95rem;
  font-weight: 600;
}

.banner-desc {
  font-family: 'Inter', sans-serif;
  color: var(--muted-text);
  font-size: 0.85rem;
  margin: 2px 0 0 0;
}

.banner-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.banner-btn {
  border: none;
  border-radius: 0.75rem;
  padding: 7px 14px;
  font-family: 'Inter', sans-serif;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.banner-btn.primary {
  background: linear-gradient(15deg, var(--primary) 0%, var(--primary-dim) 100%);
  color: var(--on-primary);
}

.banner-btn.primary:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}

.banner-btn.secondary {
  background: var(--surface-container-highest);
  color: var(--primary);
}

.banner-btn.secondary:hover {
  background: var(--surface-container);
}

@media (max-width: 960px) {
  .app-shell {
    flex-direction: column;
  }

  .content-shell {
    padding: 16px;
  }
}

/* ── Update Alert Modal ── */
.update-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(15, 23, 42, 0.75);
  backdrop-filter: blur(8px);
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.update-modal {
  background: var(--surface-container-lowest, #0f172a);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  padding: 32px;
  max-width: 440px;
  width: 100%;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  gap: 20px;
  animation: scaleUp 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes scaleUp {
  from {
    transform: scale(0.95);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}

.update-modal-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.warning-icon {
  font-size: 28px;
}

.update-modal-header h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--on-surface, #ffffff);
}

.update-modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: var(--on-surface, #f8fafc);
}

.update-msg {
  margin: 0;
  font-size: 15px;
}

.version-badge-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--surface-container-low, rgba(255, 255, 255, 0.03));
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--outline-variant);
}

.version-badge {
  font-family: monospace;
  font-size: 14px;
  font-weight: bold;
  padding: 4px 10px;
  border-radius: 6px;
}

.version-badge.current {
  background: rgba(239, 68, 68, 0.1);
  color: #f87171;
  border: 1px solid var(--outline-variant);
}

.version-badge.latest {
  background: rgba(16, 185, 129, 0.1);
  color: #34d399;
  border: 1px solid var(--outline-variant);
}

.version-arrow {
  color: var(--muted-text, #94a3b8);
  font-weight: bold;
}

.warning-text {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text, #94a3b8);
  line-height: 1.5;
}

.update-modal-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.modal-btn {
  border: none;
  border-radius: 10px;
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.modal-btn.primary {
  background: var(--primary, #4f46e5);
  color: var(--on-primary, #ffffff);
}

.modal-btn.primary:hover {
  background: var(--primary-hover, #4338ca);
}

.modal-btn.secondary {
  background: transparent;
  color: var(--muted-text, #94a3b8);
  border: 1px solid var(--outline-variant);
}

.modal-btn.secondary:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--on-surface, #ffffff);
}
</style>
