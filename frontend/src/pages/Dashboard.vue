<template>
  <section class="page">
    <header class="topbar">
      <!-- Compact profile switcher component -->
      <ProfileSwitcher
        :profiles="profiles"
        :active-profile-id="userSettings.active_profile_id"
        @select-profile="selectProfile"
        @manage-profiles="goToProfileOverview"
      />

      <!-- Right-edge Minimal GitHub Icon Button -->
      <button type="button" class="topbar-github-icon" title="View GitHub Repository" @click="openGitHubRepo">
        <BaseIcon name="github" size="18" />
      </button>
    </header>

    <!-- Status Banners Component -->
    <DashboardBanners
      :show-git-hub-star-toast="showGitHubStarToast"
      :streak-saved-event="streakSavedEvent"
      :pending-ingestion-book="pendingIngestionBook"
      :is-cloud-account="Boolean(userSettings.classroom_code)"
      :skip-to-reading-active="userSettings.skip_to_reading_active"
      :has-socratic-rescue-task="hasSocraticRescueTask"
      :flashcard-notice="flashcardNotice"
      :flashcards-just-created="flashcardsJustCreated"
      :action-error="actionError"
      @star-github="handleStarGitHub"
      @dismiss-star-github="dismissGitHubStarToast"
      @dismiss-streak-saved="streakSavedEvent = null"
      @ingest-book="goToIngestBook"
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
          @open-pace-modal="showPaceModal = true"
        />

        <!-- Escape Hatch Quick Toggle Button -->
        <button
          class="escape-hatch-toggle"
          :class="{ active: userSettings.skip_to_reading_active }"
          @click="toggleEscapeHatch"
        >
          {{ userSettings.skip_to_reading_active ? 'Disable Escape Hatch' : 'Skip to Reading' }}
        </button>
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
          <ForecastChart
            :timeline-data="timelineData"
            :max-flashcards-limit="userSettings.max_flashcards_per_session"
          />
        </div>
      </div>
    </template>

    <!-- Study Pace & Schedule Modal -->
    <StudyPaceModal
      v-model="showPaceModal"
      :pace="activeProfilePace"
      :profile-name="activeProfileName"
      :study-slots-json="userSettings.study_slots_json"
      :study-start-time="userSettings.study_start_time"
      :study-end-time="userSettings.study_end_time"
    />
  </section>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  getDashboardOverview,
  updateUserSettings,
  getProfileDailyPace,
  retryFlashcardGeneration,
  getFlashcardDueTimeline,
  getGamificationState,
  openURLInBrowser,
} from '../services/appApi'
import { buildCalendarDays, MONTH_NAMES } from '../utils/dateFormat'

import BaseIcon from '../components/BaseIcon.vue'
import ProfileSwitcher from '../components/ProfileSwitcher.vue'
import DashboardBanners from '../components/DashboardBanners.vue'
import ReviewHeroCard from '../components/ReviewHeroCard.vue'
import FocusHeroCard from '../components/FocusHeroCard.vue'
import TaskCard from '../components/TaskCard.vue'
import OnboardingCard from '../components/OnboardingCard.vue'
import TelemetryWidget from '../components/TelemetryWidget.vue'
import StreakCalendar from '../components/StreakCalendar.vue'
import ForecastChart from '../components/ForecastChart.vue'
import StudyPaceModal from '../components/StudyPaceModal.vue'

function openGitHubRepo() {
  openURLInBrowser('https://github.com/Vishnuj-n/studyloop')
}

const router = useRouter()
const route = useRoute()

const showGitHubStarToast = ref(false)

async function checkGitHubStarToast() {
  if (localStorage.getItem('github_star_toast_dismissed') === 'true') {
    showGitHubStarToast.value = false
    return
  }
  try {
    const res = await getGamificationState()
    if (res && res.profile && res.profile.stats_json) {
      let stats = {}
      try {
        stats = JSON.parse(res.profile.stats_json)
      } catch (e) {
        console.error('Failed to parse gamification stats JSON', e)
      }
      const totalSessions = (stats.reading_sessions || 0) + (stats.quizzes_passed || 0) + (stats.flashcards_reviewed || 0)
      if (totalSessions >= 5) {
        showGitHubStarToast.value = true
      }
    }
  } catch (err) {
    console.error('Failed to check star toast eligibility', err)
  }
}

function handleStarGitHub() {
  localStorage.setItem('github_star_toast_dismissed', 'true')
  showGitHubStarToast.value = false
  openGitHubRepo()
}

function dismissGitHubStarToast() {
  localStorage.setItem('github_star_toast_dismissed', 'true')
  showGitHubStarToast.value = false
}

// --- Reactive State ---
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')
const actionError = ref('')
const flashcardNotice = ref('')
const streakSavedEvent = ref(null)
const tasks = ref([])
const hasActiveStudyContent = ref(false)
const dueReviewCards = ref(0)
const totalDueReviewCards = ref(0)
const flashcardsJustCreated = ref(0)

const profiles = ref([])
const userSettings = ref({
  max_flashcards_per_session: 30,
  study_start_time: '17:00',
  study_end_time: '18:00',
  study_slots_json: '[]',
  classroom_code: '',
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
const showPaceModal = ref(false)

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

// --- Calendar computeds ---
const now = ref(new Date())

const currentMonthLabel = computed(() => {
  return `${MONTH_NAMES[now.value.getMonth()]} ${now.value.getFullYear()}`
})

const calendarDays = computed(() => {
  return buildCalendarDays(now.value.getFullYear(), now.value.getMonth(), streakState.value?.active_dates || [])
})

// --- Task computeds ---
const reviewTask = computed(() => {
  return tasks.value.find((t) => (t.action_type || '').toLowerCase() === 'flashcard_review' || t.id === 'task-review-daily')
})

const nonReviewTasks = computed(() => {
  return tasks.value.filter((t) => (t.action_type || '').toLowerCase() !== 'flashcard_review' && t.id !== 'task-review-daily')
})

const isReviewHero = computed(() => {
  return (
    !!reviewTask.value && !userSettings.value.skip_to_reading_active
  )
})

const focusHeroTask = computed(() => {
  if (isReviewHero.value) return null
  return nonReviewTasks.value.length > 0 ? nonReviewTasks.value[0] : null
})

const queueTasks = computed(() => {
  if (isReviewHero.value) {
    return tasks.value.filter((t) => t.id !== reviewTask.value.id)
  }
  return nonReviewTasks.value.slice(1)
})

const activeProfileName = computed(() => {
  const p = profiles.value.find((pr) => pr.id === userSettings.value.active_profile_id)
  return p ? p.name : 'Unknown'
})

const hasSocraticRescueTask = computed(() => {
  return tasks.value.some((t) => (t.action_type || '').toLowerCase() === 'socratic_remedial')
})

// ponytail: pending ingestion notification state
const pendingIngestionBook = ref(null)

function goToIngestBook(notebookId) {
  router.push({ path: '/notebooks', query: { ingest: notebookId } })
}

const completedSessionsToday = computed(() => {
  if (streakState.value?.completed_reading_today !== undefined) {
    return Number(streakState.value.completed_reading_today) || 0
  }
  return Number(streakState.value?.completed_today) || 0
})

// --- Lifecycle ---
onMounted(async () => {
  const created = Number.parseInt(route.query.flashcardsCreated, 10)
  if (created > 0) {
    flashcardsJustCreated.value = created
    const newQuery = { ...route.query }
    delete newQuery.flashcardsCreated
    await router.replace({ query: newQuery })
  }

  await loadAgenda()
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

    await Promise.all([loadActiveProfilePace(), loadFlashcardTimeline(tzOffset), checkGitHubStarToast()])
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
      window.dispatchEvent(new Event('gamification-updated'))
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
async function selectProfile(newProfileID) {
  const oldProfileID = lastPersistedProfile.value
  try {
    refreshing.value = true
    actionError.value = ''
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
    refreshing.value = false
  }
}

function goToProfileOverview() {
  router.push({ path: '/settings', query: { category: 'profiles' } })
}

async function toggleEscapeHatch() {
  const previousSkipToReading = userSettings.value.skip_to_reading_active
  try {
    refreshing.value = true
    actionError.value = ''
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
    refreshing.value = false
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
      actionError.value = ''
      const count = res && typeof res.cards_scheduled === 'number' ? res.cards_scheduled : 0
      flashcardNotice.value = count > 0
        ? `Successfully generated ${count} flashcards for spaced repetition!`
        : 'Flashcards are ready and up to date!'
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
    query.autoGenerate = 'true'
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

.topbar-github-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--surface-container-low, rgba(255, 255, 255, 0.04));
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.08));
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.topbar-github-icon:hover {
  background: var(--surface-container-highest);
  color: var(--on-surface);
  border-color: var(--outline-variant);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
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
