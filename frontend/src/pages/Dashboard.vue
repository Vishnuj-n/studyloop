<template>
  <section class="page">
    <header class="topbar">
      <!-- Compact profile switcher: keep profile context visible without adding dashboard cards. -->
      <div class="profile-selector-container" @click.stop>
        <span class="profile-context-label">Profile</span>
        <button
          id="active-profile-select"
          type="button"
          class="profile-switcher-trigger"
          :aria-expanded="profileMenuOpen"
          aria-haspopup="listbox"
          @click="profileMenuOpen = !profileMenuOpen"
        >
          <span class="profile-trigger-status" aria-hidden="true"></span>
          <span class="profile-trigger-name">{{ activeProfileName }}</span>
          <span class="profile-trigger-chevron" aria-hidden="true">⌄</span>
        </button>

        <div v-if="profileMenuOpen" class="profile-switcher-menu" role="listbox" aria-label="Switch profile">
          <button
            v-for="p in profiles"
            :key="p.id"
            type="button"
            class="profile-menu-item"
            :class="{ active: p.id === userSettings.active_profile_id }"
            role="option"
            :aria-selected="p.id === userSettings.active_profile_id"
            @click="selectProfile(p.id)"
          >
            <span class="profile-menu-dot" aria-hidden="true"></span>
            <span class="profile-menu-copy">
              <strong>{{ p.name }}</strong>
              <small>{{ profileDeadlineLabel(p) }}</small>
            </span>
            <span v-if="p.id === userSettings.active_profile_id" class="profile-menu-current">Active</span>
          </button>

          <div class="profile-menu-divider" aria-hidden="true"></div>
          <button type="button" class="profile-menu-action" @click="goToProfileSettings">
            <span aria-hidden="true">⚙</span>
            Manage profiles
          </button>
          <button type="button" class="profile-menu-action" @click="goToProfileOverview">
            <span aria-hidden="true">◈</span>
            View all profiles
          </button>
        </div>
      </div>
    </header>

    <!-- Status Banners -->
    <StatusBanner
      v-if="streakSavedEvent"
      variant="info"
      icon="🛡️"
      title="Your streak was saved!"
      :subtitle="`We used 1 Streak Freeze to protect your ${streakSavedEvent.streak_length}-day streak yesterday. You have ${streakSavedEvent.freezes_remaining} freeze(s) remaining.`"
      action-label="Dismiss"
      @action="streakSavedEvent = null"
    />
    <StatusBanner
      v-if="pendingIngestionBook"
      variant="warning"
      icon="⚡"
      :title="pendingIngestionBannerTitle"
      :subtitle="`${pendingIngestionBook.title} is ready for chapter extraction and ingestion.`"
      action-label="✨ Ingest Book"
      @action="goToIngestBook(pendingIngestionBook.id)"
    />
    <StatusBanner
      v-if="userSettings.skip_to_reading_active"
      variant="info"
      icon="⚡"
      title='"Skip to Reading" Escape Hatch Active'
      subtitle="Review tasks have been pushed to the background so you can focus on reading new chapters."
    />
    <StatusBanner
      v-if="hasSocraticRescueTask"
      variant="rescue"
      icon="🛡"
      title="Concept Rescue Active"
      subtitle="Your study queue is locked because you failed the quiz twice on this topic. You must complete the Socratic tutor rescue session to unblock your timeline."
    />
    <StatusBanner
      v-if="flashcardNotice"
      variant="success"
      icon="🎉"
      title="Flashcards Ready"
      :subtitle="flashcardNotice"
    />
    <StatusBanner
      v-if="flashcardsJustCreated"
      variant="success"
      icon="✓"
      :title="'Flashcards generated successfully!'"
      :subtitle="flashcardsJustCreated + ' cards scheduled for spaced repetition.'"
    />
    <StatusBanner
      v-if="actionError"
      variant="error"
      icon="⚠"
      title="Error starting task"
      :subtitle="actionError"
    />

    <!-- Top Context Bar (Study Queue + Pacing Telemetry + Actions) -->
    <article class="status-strip">
      <div class="status-title-group">
        <p class="eyebrow">Study Queue</p>
        <h1>Today's Tasks</h1>
      </div>

      <div class="header-actions">
        <!-- Horizontal Pacing Telemetry with 0/2 Reading Session Counter -->
        <TelemetryWidget
          :pace="activeProfilePace"
          :profile-name="activeProfileName"
          :completed-sessions="completedSessionsToday"
        />

        <!-- Escape Hatch Quick Toggle Button -->
        <button
          class="escape-hatch-toggle"
          :class="{ active: userSettings.skip_to_reading_active }"
          @click="toggleEscapeHatch"
        >
          {{ userSettings.skip_to_reading_active ? 'Disable Escape Hatch' : 'Skip to Reading' }}
        </button>

        <div v-if="dueReviewCards > 0" class="review-stats">
          <p class="review-count">{{ dueReviewCards }} cards due for review</p>
          <p class="review-hint">Spaced repetition strengthens long-term retention</p>
        </div>
      </div>
    </article>

    <template v-if="loading">
      <article class="card state-card">
        <h2>Loading study workspace...</h2>
        <p class="muted">Querying SQLite database & syncing with cloud.</p>
      </article>
    </template>

    <template v-else-if="error">
      <article class="card state-card error-card">
        <h2>Agenda unavailable</h2>
        <p class="muted">{{ error }}</p>
      </article>
    </template>

    <template v-else>
      <!-- Dashboard Layout Grid -->
      <div class="dashboard-grid">
        <!-- Main Panel (Focus Hero Task & Vertical Queue) -->
        <div class="dashboard-main">
          <div v-if="tasks.length > 0" class="tasks-container">
            <!-- TIER 1: FOCUS HERO CARD (Action) -->
            <section class="focus-hero-section">
              <ReviewHeroCard
                v-if="isReviewHero"
                :task="reviewTask"
                :due-review-cards="dueReviewCards"
                :total-due-review-cards="totalDueReviewCards"
                @start="startTask"
              />
              <FocusHeroCard
                v-else-if="focusHeroTask"
                :task="focusHeroTask"
                :is-syncing="isSyncing"
                @start="startTask"
              />
            </section>

            <!-- TIER 2: UP NEXT IN QUEUE (Visibility & Pipeline) -->
            <section v-if="queueTasks.length > 0" class="up-next-section">
              <h2 class="queue-section-title">Up Next in Queue</h2>
              <div class="vertical-queue-stack">
                <TaskCard
                  v-for="(task, idx) in queueTasks"
                  :key="task.id"
                  :task="task"
                  :queue-index="idx + 2"
                  :is-syncing="isSyncing"
                  @start="startTask"
                />
              </div>
            </section>
          </div>

          <div v-else-if="hasActiveStudyContent" class="card state-card victory-card">
            <h2>Tasks Complete!</h2>
            <p class="muted">You've completed all tasks for today. Great work!</p>
          </div>

          <OnboardingCard v-else @go-to-notebooks="goToNotebooks" />
        </div>

        <!-- Sidebar Panel (Streak Calendar & Forecast Chart) -->
        <div class="dashboard-sidebar">
          <StreakCalendar
            :streak-state="streakState"
            :streak-error="streakError"
            :calendar-days="calendarDays"
            :month-label="currentMonthLabel"
          />
          <ForecastChart :timeline-data="timelineData" :max-flashcards-limit="maxFlashcardsLimit" />
        </div>
      </div>
    </template>
  </section>
</template>

<script setup>
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  getDashboardOverview,
  updateUserSettings,
  getProfileDailyPace,
  retryFlashcardGeneration,
  getFlashcardDueTimeline,
} from '../services/appApi'
import { buildCalendarDays, MONTH_NAMES } from '../utils/dateFormat'

import StatusBanner from '../components/StatusBanner.vue'
import ReviewHeroCard from '../components/ReviewHeroCard.vue'
import FocusHeroCard from '../components/FocusHeroCard.vue'
import TaskCard from '../components/TaskCard.vue'
import OnboardingCard from '../components/OnboardingCard.vue'
import TelemetryWidget from '../components/TelemetryWidget.vue'
import StreakCalendar from '../components/StreakCalendar.vue'
import ForecastChart from '../components/ForecastChart.vue'


const router = useRouter()
const route = useRoute()

// --- Reactive State ---
const loading = ref(true)
const error = ref('')
const actionError = ref('')
const flashcardNotice = ref('')
const streakSavedEvent = ref(null)
const tasks = ref([])
const hasActiveStudyContent = ref(false)
const dueReviewCards = ref(0)
const totalDueReviewCards = ref(0)

const profiles = ref([])
const userSettings = ref({
  max_flashcards_per_session: 30,
  study_start_time: '17:00',
  study_end_time: '18:00',
  reminders_enabled: true,
  active_profile_id: '',
  skip_to_reading_active: false,
  cloud_sync_url: '',
  cloud_api_token: '',
  theme: '',
  rag_enabled: false,
  rag_notebook_chapter: true,
  rag_entire_notebook: true,
  rag_queue_study: true,
  default_remedial_strategy: 'FAST',
})
const activeProfilePace = ref(null)
const lastPersistedProfile = ref('')

const timelineData = ref([])

const streakState = ref({
  current_streak: 0,
  longest_streak: 0,
  active_dates: [],
  today_completed: false,
  completed_today: 0,
})
const streakError = ref('')
const isSyncing = ref(false)
const profileMenuOpen = ref(false)

// --- Calendar computeds ---
const currentDate = new Date()
const currentYear = currentDate.getFullYear()
const currentMonth = currentDate.getMonth()

const currentMonthLabel = computed(() => {
  return `${MONTH_NAMES[currentMonth]} ${currentYear}`
})

const calendarDays = computed(() => {
  return buildCalendarDays(currentYear, currentMonth, streakState.value?.active_dates || [])
})

// --- Task computeds ---
const maxFlashcardsLimit = computed(() => {
  return userSettings.value.max_flashcards_per_session || 30
})

const reviewTask = computed(() => {
  return tasks.value.find((t) => t.id === 'task-review-daily')
})

const nonReviewTasks = computed(() => {
  return tasks.value.filter((t) => t.id !== 'task-review-daily')
})

const isReviewHero = computed(() => {
  return (
    !!reviewTask.value && !userSettings.value.skip_to_reading_active && dueReviewCards.value > 0
  )
})

const focusHeroTask = computed(() => {
  if (isReviewHero.value) return null
  return nonReviewTasks.value.length > 0 ? nonReviewTasks.value[0] : null
})

const queueTasks = computed(() => {
  if (isReviewHero.value) {
    return nonReviewTasks.value
  }
  return nonReviewTasks.value.slice(1)
})

const flashcardsJustCreated = computed(() => {
  const created = Number.parseInt(route.query.flashcardsCreated, 10)
  return isNaN(created) || created <= 0 ? 0 : created
})

const activeProfileName = computed(() => {
  const p = profiles.value.find((pr) => pr.id === userSettings.value.active_profile_id)
  return p ? p.name : 'Unknown'
})

function profileDeadlineLabel(profile) {
  const deadlineAt = Number(profile?.deadline_at)
  if (!Number.isFinite(deadlineAt) || deadlineAt <= 0) return 'No deadline set'

  const daysLeft = Math.ceil((deadlineAt * 1000 - Date.now()) / 86400000)
  if (daysLeft < 0) return 'Deadline passed'
  if (daysLeft === 0) return 'Due today'
  return `${daysLeft}d left`
}

const hasSocraticRescueTask = computed(() => {
  return tasks.value.some((t) => t.action_type === 'socratic_remedial')
})

// ponytail: pending ingestion notification state
const pendingIngestionBook = ref(null)

const pendingIngestionBannerTitle = computed(() => {
  const isCloud = Boolean(userSettings.value?.classroom_code)
  return isCloud ? 'New Assignment — Ingestion Needed' : 'New Book — Ingestion Needed'
})

function goToIngestBook(notebookId) {
  router.push({ path: '/notebooks', query: { ingest: notebookId } })
}

const completedSessionsToday = computed(() => {
  return Number(streakState.value?.completed_today) || 0
})

// --- Lifecycle ---
onMounted(async () => {
  window.addEventListener('click', closeProfileMenu)
  if (flashcardsJustCreated.value > 0) {
    const newQuery = { ...route.query }
    delete newQuery.flashcardsCreated
    await router.replace({ query: newQuery })
  }
  await loadAgenda()
})

onUnmounted(() => {
  window.removeEventListener('click', closeProfileMenu)
})

// --- Data Fetching ---
async function loadAgenda() {
  try {
    loading.value = true
    error.value = ''
    actionError.value = ''

    const tzOffset = new Date().getTimezoneOffset()
    const overview = await getDashboardOverview(tzOffset)

    if (!applyDashboardOverview(overview)) {
      return
    }

    await Promise.all([loadActiveProfilePace(), loadFlashcardTimeline(tzOffset)])
  } catch (err) {
    error.value = err.message || 'Failed to load tasks'
  } finally {
    loading.value = false
  }
}

function applyDashboardOverview(overview) {
  if (overview.settings?.error) {
    error.value = overview.settings.error
    return false
  }
  if (overview.settings) {
    userSettings.value = overview.settings
    lastPersistedProfile.value = overview.settings.active_profile_id || ''
  }

  if (overview.profiles?.error) {
    error.value = overview.profiles.error
    return false
  }
  if (overview.profiles) {
    profiles.value = overview.profiles.profiles || []
  }

  if (overview.today_plan?.error) {
    error.value = overview.today_plan.error
    return false
  }
  if (overview.today_plan) {
    const response = overview.today_plan
    tasks.value = response.tasks || []
    dueReviewCards.value = response.due_review_cards || 0
    totalDueReviewCards.value = response.total_due_review_cards || 0
    const activeNotebookCount = response.active_notebook_count || 0
    hasActiveStudyContent.value = tasks.value.length > 0 || activeNotebookCount > 0
  }

  if (overview.streak_state?.error) {
    streakError.value = overview.streak_state.error
  } else if (overview.streak_state) {
    streakState.value = overview.streak_state
    streakError.value = ''
    if (overview.streak_state.streak_saved_event) {
      streakSavedEvent.value = overview.streak_state.streak_saved_event
    }
  }

  if (overview.pending_notebook_error) {
    actionError.value = overview.pending_notebook_error
  }
  pendingIngestionBook.value = overview.pending_notebook || null

  return true
}

async function loadActiveProfilePace() {
  const profileId = userSettings.value.active_profile_id
  const knownProfile = profiles.value.find((pr) => pr.id === profileId)
  if (!profileId || !knownProfile) {
    activeProfilePace.value = null
    return
  }

  try {
    const pace = await getProfileDailyPace(profileId)
    activeProfilePace.value = pace.error ? null : pace
  } catch (err) {
    console.error('Failed to get profile daily pace', err)
    activeProfilePace.value = null
  }
}

async function loadFlashcardTimeline(tzOffset) {
  try {
    const timelineRes = await getFlashcardDueTimeline(tzOffset)
    timelineData.value = timelineRes && !timelineRes.error ? timelineRes.timeline || [] : []
  } catch (err) {
    console.error('Failed to get flashcard due timeline', err)
    timelineData.value = []
  }
}

// --- User Actions ---
function closeProfileMenu() {
  profileMenuOpen.value = false
}

async function selectProfile(newProfileID) {
  profileMenuOpen.value = false
  const oldProfileID = lastPersistedProfile.value
  try {
    loading.value = true
    const res = await updateUserSettings({
      ...userSettings.value,
      active_profile_id: newProfileID,
    })
    if (res && res.error) {
      userSettings.value.active_profile_id = oldProfileID
      actionError.value = res.error
      return
    }
    lastPersistedProfile.value = newProfileID
    window.dispatchEvent(new CustomEvent('settings-updated'))
    await loadAgenda()
  } catch (err) {
    userSettings.value.active_profile_id = oldProfileID
    actionError.value = 'Failed to switch active profile'
  } finally {
    loading.value = false
  }
}

function goToProfileSettings() {
  profileMenuOpen.value = false
  router.push('/settings')
}

function goToProfileOverview() {
  profileMenuOpen.value = false
  router.push({ path: '/settings', query: { category: 'profiles' } })
}

async function toggleEscapeHatch() {
  const previousSkipToReading = userSettings.value.skip_to_reading_active
  try {
    loading.value = true
    userSettings.value.skip_to_reading_active = !userSettings.value.skip_to_reading_active
    const res = await updateUserSettings(userSettings.value)
    if (res && res.error) {
      userSettings.value.skip_to_reading_active = previousSkipToReading
      actionError.value = res.error
      return
    }
    window.dispatchEvent(new CustomEvent('settings-updated'))
    await loadAgenda()
  } catch (err) {
    userSettings.value.skip_to_reading_active = previousSkipToReading
    actionError.value = 'Failed to toggle escape hatch'
  } finally {
    loading.value = false
  }
}

async function runFlashcardSyncInline(task) {
  try {
    isSyncing.value = true
    actionError.value = ''
    flashcardNotice.value = ''
    const res = await retryFlashcardGeneration(task.id)
    if (res && res.error) {
      actionError.value = `Flashcard Generation Failed: ${res.error}. Please check your connection.`
    } else {
      const count = res && typeof res.cards_scheduled === 'number' ? res.cards_scheduled : 0
      flashcardNotice.value = count > 0
        ? `🎉 Successfully generated ${count} flashcards for spaced repetition!`
        : '✨ Flashcards are ready and up to date!'
      await loadAgenda()
    }
  } catch (err) {
    actionError.value = `Flashcard Generation Error: ${err.message || err}`
  } finally {
    isSyncing.value = false
  }
}

function startTask(task) {
  let routePath = '/dashboard'
  const query = {
    topicId: task.topic_id,
    notebookId: task.notebook_id,
    startPage: task.start_page,
    endPage: task.end_page,
    taskId: task.id,
  }

  const action = (task.action_type || '').toLowerCase()
  if (action === 'reading') {
    routePath = '/reader'
  } else if (action === 'flashcard_review') {
    routePath = '/flashcards'
  } else if (action === 'quiz' || action === 'milestone_exam') {
    routePath = '/quiz'
  } else if (action === 'examiner' || action === 'written') {
    routePath = '/examiner'
  } else if (action === 'reread') {
    routePath = '/reader'
  } else if (action === 'socratic_remedial') {
    routePath = '/socratic-rescue'
  } else if (action === 'flashcard_generate') {
    runFlashcardSyncInline(task)
    return
  } else {
    actionError.value = `Unknown task action type: ${task.action_type}`
    return
  }

  router.push({ path: routePath, query })
}

function goToNotebooks() {
  router.push('/notebooks')
}


</script>

<style scoped>
.page {
  display: grid;
  gap: 20px;
  font-family: 'Inter', sans-serif;
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--outline-variant, #e0e0e0);
}

.profile-selector-container {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
}

.profile-context-label {
  font-size: 13px;
  font-weight: 700;
  color: var(--muted-text, #666);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.profile-switcher-trigger {
  min-width: 190px;
  display: inline-flex;
  align-items: center;
  gap: 9px;
  border: 1px solid var(--outline-variant, #e0e0e0);
  border-radius: 12px;
  background: var(--surface-container-low, #f8f9fa);
  color: var(--on-surface, #1e1e1e);
  padding: 9px 12px;
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.profile-switcher-trigger:hover,
.profile-switcher-trigger:focus-visible {
  border-color: var(--primary);
  background-color: var(--surface-container-highest);
}

.profile-switcher-trigger:focus-visible,
.profile-menu-item:focus-visible,
.profile-menu-action:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 20%, transparent);
}

.profile-trigger-status,
.profile-menu-dot {
  flex: 0 0 auto;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--muted-text, #64707d);
}

.profile-trigger-status {
  background: var(--primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 16%, transparent);
}

.profile-trigger-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-trigger-chevron {
  margin-left: auto;
  color: var(--muted-text, #64707d);
  font-size: 18px;
  line-height: 1;
}

.profile-switcher-menu {
  position: absolute;
  top: calc(100% + 8px);
  left: 66px;
  z-index: 20;
  width: 280px;
  padding: 7px;
  border: 1px solid var(--outline-variant, #e0e0e0);
  border-radius: 14px;
  background: var(--surface-container-lowest, #ffffff);
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.12);
}

.profile-menu-item,
.profile-menu-action {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--on-surface, #1e1e1e);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.profile-menu-item {
  padding: 10px;
}

.profile-menu-item + .profile-menu-item {
  margin-top: 4px;
}

.profile-menu-item:hover,
.profile-menu-item.active {
  background: color-mix(in srgb, var(--primary) 10%, transparent);
}

.profile-menu-item.active .profile-menu-dot {
  background: var(--primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 16%, transparent);
}

.profile-menu-copy {
  min-width: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.profile-menu-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.profile-menu-copy small {
  color: var(--muted-text, #64707d);
  font-size: 11px;
}

.profile-menu-current {
  color: var(--primary);
  font-size: 10px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.profile-menu-divider {
  height: 1px;
  margin: 5px 3px;
  background: var(--outline-variant, #e0e0e0);
}

.profile-menu-action {
  padding: 9px 10px;
  color: var(--muted-text, #64707d);
  font-size: 12px;
  font-weight: 700;
}

.profile-menu-action:hover {
  background: var(--surface-container-high, rgba(0, 0, 0, 0.04));
  color: var(--on-surface, #1e1e1e);
}

.status-strip {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 4px 0 8px;
  flex-wrap: wrap;
}

.status-title-group .eyebrow {
  margin: 0 0 2px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--primary);
}

.status-strip h1 {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 38px;
  letter-spacing: -0.03em;
  line-height: 1.1;
  color: var(--on-surface);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.escape-hatch-toggle {
  background: var(--surface-container-low);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  border-radius: 9999px;
  padding: 7px 16px;
  font-weight: 700;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  white-space: nowrap;
}

.escape-hatch-toggle:hover {
  border-color: var(--primary);
}

.escape-hatch-toggle.active {
  background: linear-gradient(135deg, #e67e22, #d35400);
  border-color: #d35400;
  color: white;
  box-shadow: 0 0 12px rgba(230, 126, 34, 0.3);
}

.review-stats {
  text-align: right;
}

.review-count {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--primary);
  font-family: 'Manrope', sans-serif;
}

.review-hint {
  margin: 2px 0 0;
  font-size: 11px;
  color: var(--muted-text, #777);
}

.card {
  background: var(--surface-container-lowest, #ffffff);
  border: 1px solid var(--outline-variant, #e0e0e0);
  border-radius: 16px;
}

.state-card {
  padding: 40px;
  text-align: center;
}

.state-card h2 {
  margin: 0 0 8px;
  font-size: 24px;
}

.muted {
  color: var(--muted-text, #666);
}

.error-card h2 {
  color: #eb5e55;
}

.tasks-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.focus-hero-section {
  display: flex;
  flex-direction: column;
}

.up-next-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.queue-section-title {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 18px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--on-surface, #1e1e1e);
}

.vertical-queue-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* Dashboard Two-Column Layout Grid */
.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 24px;
}

@media (min-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr 310px;
    align-items: start;
  }
}

.dashboard-main {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.dashboard-sidebar {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

</style>
