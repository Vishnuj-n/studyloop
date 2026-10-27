<template>
  <StudyPageLayout
    eyebrow="Assessment"
    :title="taskMeta?.task_type === 'MILESTONE_EXAM' ? 'Milestone Exam' : 'Quiz'"
    :subtitle="
      taskMeta
        ? taskMeta.task_type === 'MILESTONE_EXAM'
          ? 'Cumulative Notebook Assessment'
          : taskMeta.start_page || taskMeta.end_page
            ? `Pages ${taskMeta.start_page}–${taskMeta.end_page}`
            : 'Pages N/A'
        : ''
    "
  >
    <!-- Toolbar: notebook selector (manual mode only) -->
    <template v-if="!taskID && !generating && questions.length === 0" #toolbar>
      <div class="toolbar-field">
        <label class="field-label" for="quiz-notebook-select">Notebook</label>
        <select
          id="quiz-notebook-select"
          v-model="selectedNotebookID"
          class="ghost-select"
          :disabled="loading || generating"
        >
          <option value="">— Select Notebook —</option>
          <option v-for="nb in notebooks" :key="nb.id" :value="nb.id">{{ nb.title }}</option>
        </select>
      </div>
    </template>

    <!-- Loading -->
    <article v-if="loading" class="state-panel">
      <p class="state-text">Loading quiz…</p>
    </article>

    <!-- Error -->
    <article v-else-if="error" class="state-panel state-panel--error">
      <p class="state-text">{{ error }}</p>
      <button
        v-if="canRetryGeneration"
        class="primary-btn retry-generation-btn"
        @click="generateManualQuiz"
      >
        Retry
      </button>
    </article>

    <!-- Manual controls: page range + generate -->
    <div v-else-if="!taskID && !generating && questions.length === 0" class="config-panel">
      <p class="config-panel__hint">Enter the page range to generate quiz questions from.</p>
      <div class="config-panel__row">
        <div class="number-field">
          <label class="field-label" for="quiz-start">Start Page</label>
          <input
            id="quiz-start"
            v-model.number="startPage"
            class="ghost-input"
            type="number"
            min="1"
            :disabled="loading"
          />
        </div>
        <div class="number-field">
          <label class="field-label" for="quiz-end">End Page</label>
          <input
            id="quiz-end"
            v-model.number="endPage"
            class="ghost-input"
            type="number"
            min="1"
            :disabled="loading"
          />
        </div>
        <button
          id="quiz-generate-btn"
          class="primary-btn"
          :disabled="!canGenerateManual"
          @click="generateManualQuiz"
        >
          {{ generating ? 'Generating…' : 'Generate Quiz' }}
        </button>
      </div>
    </div>

    <!-- Generating state -->
    <article v-else-if="generating" class="state-panel">
      <p class="state-text">Generating questions…</p>
    </article>

    <!-- Result card -->
    <article v-else-if="submitted && result" class="result-panel">
      <div class="result-panel__badge" :class="result.passed ? 'badge--pass' : 'badge--fail'">
        {{ result.passed ? 'Passed' : 'Needs Reread' }}
      </div>
      <p class="result-panel__score">
        <span class="score-value">{{ result.score }}%</span>
        <span class="score-threshold">threshold {{ result.passing_score }}%</span>
      </p>
      <p v-if="result.feedback" class="result-panel__feedback">{{ result.feedback }}</p>

      <!-- Flashcard generation pending message (only when passed and flashcards pending) -->
      <div
        v-if="result.passed && result.flashcards_pending"
        class="result-panel__flashcard-success"
      >
        <p class="flashcard-success-title">Flashcards scheduled for generation.</p>
        <p class="flashcard-success-detail">
          Click Continue to generate flashcards for spaced repetition.
        </p>
        <p class="flashcard-success-hint">
          These cards will appear in your dashboard when they're due for review.
        </p>
      </div>

      <!-- Flashcard generation success message (only when generated during Continue) -->
      <div
        v-if="result.passed && !result.flashcards_pending && result.flashcards_generated > 0"
        class="result-panel__flashcard-success"
      >
        <p class="flashcard-success-title">Flashcards generated successfully.</p>
        <p class="flashcard-success-detail">
          {{ result.flashcards_generated }} review cards scheduled for spaced repetition.
        </p>
        <p class="flashcard-success-hint">
          These cards will appear in your dashboard when they're due for review.
        </p>
      </div>

      <!-- Flashcard generation/sync error -->
      <div
        v-if="result.passed && result.flashcards_generation_error"
        class="result-panel__network-warning"
      >
        <span class="warning-icon"><BaseIcon name="alert-triangle" size="18" /></span>
        <div class="warning-text">
          <p class="warning-title">Generation Warning / Network Error</p>
          <p class="warning-detail">
            Flashcard generation could not be completed automatically. A retry task has been added
            to your queue. You can also try retrying the generation now.
          </p>
          <p class="warning-message">
            Reason: {{ result.flashcards_generation_message || 'Unknown error' }}
          </p>
        </div>
      </div>

      <!-- Reread message (when failed) -->
      <p v-if="!result.passed && result.reread_task_id" class="result-panel__reread-message">
        Reread session added to your learning queue.
      </p>
      <p v-if="!result.passed && result.reread_task_id" class="result-panel__attempts">
        Attempt {{ result.reread_attempt_count }} of {{ result.max_reread_attempts }}
      </p>

      <!-- Socratic Rescue message (when rescue task created) -->
      <div
        v-if="!result.passed && result.socratic_task_id"
        class="result-panel__rescue-notice"
      >
        <p class="rescue-notice-title">Concept Rescue Active</p>
        <p class="rescue-notice-desc">
          Complete the Adaptive Tutor Concept Rescue session to review tricky areas before retrying your quiz.
        </p>
      </div>

      <!-- External help notice (when failed re-quiz) -->
      <div
        v-if="!result.passed && !result.reread_task_id && result.manual_review_recommended"
        class="result-panel__external-help-notice"
      >
        <p class="external-help-title">External Review Required</p>
        <p class="external-help-desc">
          This concept requires external review. Your next reading task has been unlocked so you
          don't fall behind. Please consult your notes or instructor for this page range.
        </p>
      </div>

      <!-- Questions Breakdown (Correct / Incorrect review) -->
      <section v-if="questions.length > 0" class="result-breakdown">
        <h3 class="breakdown-title">Question Breakdown</h3>
        <article
          v-for="(q, index) in questions"
          :key="q.id"
          class="breakdown-card"
          :class="isCorrect(q) ? 'breakdown-card--correct' : 'breakdown-card--incorrect'"
        >
          <div class="breakdown-header">
            <span class="question-num">{{ index + 1 }}</span>
            <span class="breakdown-status" style="display: inline-flex; align-items: center; gap: 4px;">
              <BaseIcon :name="isCorrect(q) ? 'check' : 'x'" size="13" />
              <span>{{ isCorrect(q) ? 'Correct' : 'Incorrect' }}</span>
            </span>
          </div>
          <p class="question-prompt">{{ q.prompt }}</p>
          <div class="breakdown-answers">
            <p class="answer-item">
              <span class="answer-label">Your Answer:</span>
              <span
                class="answer-val"
                :class="isCorrect(q) ? 'val--correct' : 'val--incorrect'"
              >
                {{ answers[q.id] || 'None' }}
              </span>
            </p>
            <p v-if="!isCorrect(q)" class="answer-item">
              <span class="answer-label">Correct Answer:</span>
              <span class="answer-val val--correct">{{ q.correct_answer }}</span>
            </p>
          </div>
        </article>
      </section>

      <!-- Feynman Reflection Feedback Card (when provided) -->
      <section v-if="result.reflection_feedback" class="feynman-feedback-card">
        <div class="feynman-feedback-header">
          <div class="feynman-badge">
            <BaseIcon name="sparkles" size="14" />
            <span>Feynman Synthesis Feedback</span>
          </div>
          <span
            class="feynman-calibration-badge"
            :class="getCalibrationClass(result.reflection_feedback.calibration)"
          >
            {{ result.reflection_feedback.calibration || 'Reflected' }}
          </span>
        </div>

        <div v-if="result.reflection_text" class="feynman-your-words">
          <span class="feynman-words-label">Your Explanation:</span>
          <p class="feynman-words-quote">&ldquo;{{ result.reflection_text }}&rdquo;</p>
        </div>

        <div v-if="result.reflection_feedback.strengths && result.reflection_feedback.strengths.length > 0" class="feynman-feedback-block block--strengths">
          <h4 class="feedback-heading">
            <BaseIcon name="check" size="14" />
            <span>What You Got Right</span>
          </h4>
          <ul class="feedback-list">
            <li v-for="(item, idx) in result.reflection_feedback.strengths" :key="idx">{{ item }}</li>
          </ul>
        </div>

        <div v-if="result.reflection_feedback.gaps && result.reflection_feedback.gaps.length > 0" class="feynman-feedback-block block--gaps">
          <h4 class="feedback-heading">
            <BaseIcon name="alert-triangle" size="14" />
            <span>Nuances / Gaps to Watch</span>
          </h4>
          <ul class="feedback-list">
            <li v-for="(item, idx) in result.reflection_feedback.gaps" :key="idx">{{ item }}</li>
          </ul>
        </div>

        <p v-if="result.reflection_feedback.summary" class="feynman-summary-text">
          {{ result.reflection_feedback.summary }}
        </p>
      </section>

      <!-- Viva Examiner Challenge Hero Banner on Pass -->
      <section v-if="result.passed && synthesisMode === 'viva'" class="viva-challenge-card">
        <div class="viva-challenge-card__content">
          <div class="viva-challenge-card__badge">
            <span style="display: inline-flex; align-items: center; gap: 4px;">
              <BaseIcon name="trophy" size="14" />
              <span>EXAMINER VIVA CHALLENGE</span>
            </span>
            <span class="viva-challenge-card__tag" style="display: inline-flex; align-items: center; gap: 4px;">
              <BaseIcon name="dice" size="13" />
              <span>Gamble for Loot</span>
            </span>
          </div>
          <h4 class="viva-challenge-card__title">Want to gamble for better loot?</h4>
          <p class="viva-challenge-card__desc">
            Explain the concept in your own words (Speak via STT or Type).
          </p>
          <div class="viva-challenge-card__perks">
            <span style="display: inline-flex; align-items: center; gap: 4px;">
              <BaseIcon name="sparkles" size="13" />
              <span>Bonus XP</span>
            </span>
            <span class="perk-bullet">&bull;</span>
            <span style="display: inline-flex; align-items: center; gap: 4px;">
              <BaseIcon name="gift" size="13" />
              <span>Boosted Rare-Chest Odds</span>
            </span>
          </div>
        </div>
        <button
          class="viva-challenge-btn"
          :disabled="generatingFlashcards"
          @click="handleGoToExaminer"
        >
          <BaseIcon name="file-text" size="14" />
          <span>TAKE THE VIVA CHALLENGE</span>
        </button>
      </section>

      <div class="result-panel__footer">
        <!-- Retry Sync Button -->
        <button
          v-if="result.passed && result.flashcards_generation_error"
          class="secondary-btn retry-btn"
          :disabled="generatingFlashcards"
          @click="retryFlashcardGeneration"
        >
          {{ generatingFlashcards ? 'Retrying...' : 'Retry Generation' }}
        </button>

        <!-- Restart Reading Session (final rescue failure only) -->
        <button
          v-if="!result.passed && result.manual_review_recommended"
          class="secondary-btn restart-reading-btn"
          :disabled="generatingFlashcards || restartingReading"
          @click="handleRestartReading"
        >
          <BaseIcon name="rotate-ccw" size="13" />
          <span>Restart Reading Session</span>
        </button>

        <button
          class="primary-btn continue-btn"
          :disabled="generatingFlashcards"
          @click="handleContinue"
        >
          <span v-if="generatingFlashcards">Generating flashcards for spaced repetition...</span>
          <span v-else-if="!result.passed && result.socratic_task_id">Start Concept Rescue →</span>
          <span v-else>Continue →</span>
        </button>
      </div>

      <div class="result-panel__analysis-action">
        <button
          type="button"
          class="analysis-link-btn"
          :disabled="generatingFlashcards"
          @click="handleGoToDetailedAnalysis"
        >
          <svg
            class="analysis-svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M18 20V10" />
            <path d="M12 20V4" />
            <path d="M6 20v-6" />
          </svg>
          <span>Detailed Performance Breakdown & Socratic Prompts</span>
          <svg
            class="analysis-arrow-svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M5 12h14" />
            <path d="M12 5l7 7-7 7" />
          </svg>
        </button>
      </div>
    </article>

    <!-- Empty state: no questions -->
    <article v-else-if="questions.length === 0 && !generating" class="state-panel">
      <p class="state-text">No quiz questions found for this task.</p>
      <button
        v-if="isMilestoneExam"
        class="primary-btn retry-generation-btn"
        :disabled="submitting"
        @click="submitMilestone"
      >
        {{ submitting ? 'Completing...' : 'Complete Milestone Exam' }}
      </button>
      <button
        v-else-if="taskID"
        class="primary-btn retry-generation-btn"
        :disabled="loading"
        @click="loadQuizTask"
      >
        Retry Loading Quiz
      </button>
    </article>

    <!-- Quiz form -->
    <form v-else class="quiz-form" @submit.prevent="submitQuiz">
      <article v-for="(q, index) in questions" :key="q.id" class="question-card">
        <p class="question-prompt">
          <span class="question-num">{{ index + 1 }}</span>
          {{ q.prompt }}
        </p>
        <div class="options-grid">
          <label
            v-for="option in q.options"
            :key="option"
            class="option-row"
            :class="{ 'option-row--selected': answers[q.id] === option }"
          >
            <input
              v-model="answers[q.id]"
              class="option-radio"
              type="radio"
              :name="q.id"
              :value="option"
              :disabled="submitting"
            />
            <span class="option-text">{{ option }}</span>
          </label>
        </div>
      </article>

      <!-- Optional Feynman Reflection Card -->
      <article v-if="synthesisMode === 'feynman'" class="question-card feynman-reflection-card">
        <div class="feynman-header">
          <span class="feynman-badge">
            <BaseIcon name="sparkles" size="14" />
            <span>Self-Explanation &bull; Optional</span>
          </span>
          <span class="feynman-hint">Explain in your own words to check your mental model</span>
        </div>
        <p class="question-prompt feynman-prompt">
          What was the core insight or mechanism of this section? (2&ndash;4 sentences)
        </p>
        <div class="reflection-input-shell">
          <textarea
            v-model="feynmanReflection"
            class="feynman-textarea"
            placeholder="In my own words, the key idea here is..."
            rows="3"
            :disabled="submitting"
          ></textarea>
        </div>
      </article>

      <div class="form-footer">
        <p v-if="!allAnswered" class="footer-hint">Answer all questions to submit.</p>
        <button
          id="quiz-submit-btn"
          class="primary-btn"
          type="submit"
          :disabled="submitting || !allAnswered"
        >
          {{ submitButtonLabel }}
        </button>
      </div>
    </form>
  </StudyPageLayout>

  <!-- Pre-Milestone Exam Retrieval Modal -->
  <MilestoneNoteReviewModal
    :show="showMilestonePreReview"
    :topics="milestoneReviewTopics"
    :notebook-id="taskMeta?.notebook_id || ''"
    :task-id="taskID"
    @start-exam="onMilestonePreReviewComplete"
    @close="onMilestonePreReviewComplete"
  />

  <!-- Restart Reading Session Confirmation Modal -->
  <Teleport to="body">
    <div
      v-if="showRestartReadingModal"
      class="restart-reading-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="restart-reading-title"
    >
      <div class="restart-reading-dialog">
        <div class="restart-reading-dialog__header">
          <BaseIcon name="alert-triangle" size="18" class="restart-reading-dialog__icon" />
          <h3 id="restart-reading-title" class="restart-reading-dialog__title">
            Restart Reading Session?
          </h3>
        </div>

        <div class="restart-reading-dialog__body">
          <p class="restart-reading-dialog__desc">
            This will reset your reading progress for this topic section back to the beginning.
          </p>
          <ul class="restart-reading-dialog__list">
            <li>Your reading task will be reset to <strong>Active</strong> from page {{ taskMeta?.start_page || '?' }}</li>
            <li>All pending quiz, tutor, and flashcard tasks for this section will be cancelled</li>
            <li>Quiz attempt records for this section will be cleared</li>
          </ul>
          <div class="restart-reading-dialog__warning">
            <BaseIcon name="alert-triangle" size="16" class="restart-reading-dialog__warning-icon" />
            <div>
              <strong>Be careful:</strong> any task for this section currently in Quiz stage or higher
              will also be reverted. This cannot be undone.
            </div>
          </div>
          <p v-if="restartReadingError" class="restart-reading-dialog__error">
            {{ restartReadingError }}
          </p>
        </div>

        <div class="restart-reading-dialog__footer">
          <button
            class="secondary-btn"
            :disabled="restartingReading"
            @click="showRestartReadingModal = false; restartReadingError = ''"
          >
            Cancel
          </button>
          <button
            class="danger-btn"
            :disabled="restartingReading"
            @click="confirmRestartReading"
          >
            <span v-if="restartingReading">Restarting…</span>
            <span v-else>Yes, Restart Reading</span>
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- External Help / Terminal Failure Move Forward Modal -->
  <Teleport to="body">
    <div
      v-if="showMoveForwardModal"
      class="restart-reading-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="move-forward-title"
    >
      <div class="restart-reading-dialog">
        <div class="restart-reading-dialog__header">
          <BaseIcon name="help-circle" size="18" class="move-forward-dialog__icon" />
          <h3 id="move-forward-title" class="restart-reading-dialog__title">
            How would you like to proceed?
          </h3>
        </div>

        <div class="restart-reading-dialog__body">
          <p class="restart-reading-dialog__desc">
            This topic section has been flagged for external review. Choose how you want to move forward:
          </p>
          <div class="move-forward-options">
            <div class="move-forward-card">
              <div class="move-forward-card__title">
                <BaseIcon name="layers" size="15" />
                <span>Create Flashcards & Move Forward</span>
              </div>
              <p class="move-forward-card__desc">
                Generates spaced repetition cards for pages {{ taskMeta?.start_page || '?' }}–{{ taskMeta?.end_page || '?' }} so you can retain and reinforce key concepts in daily FSRS reviews while advancing to the next topic.
              </p>
            </div>
            <div class="move-forward-card">
              <div class="move-forward-card__title">
                <BaseIcon name="skip-forward" size="15" />
                <span>Skip to Next Topic</span>
              </div>
              <p class="move-forward-card__desc">
                Leaves the topic flagged for external help and advances to your next queued reading task without adding any review cards.
              </p>
            </div>
          </div>
          <p v-if="moveForwardError" class="restart-reading-dialog__error">
            {{ moveForwardError }}
          </p>
        </div>

        <div class="restart-reading-dialog__footer move-forward-footer">
          <button
            class="secondary-btn skip-btn"
            :disabled="generatingFlashcards"
            @click="confirmSkipToNextTopic"
          >
            Skip Without Cards
          </button>
          <button
            class="primary-btn"
            :disabled="generatingFlashcards"
            @click="confirmCreateFlashcardsAndMove"
          >
            <span v-if="generatingFlashcards">Generating Flashcards…</span>
            <span v-else>Create Flashcards & Move</span>
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseIcon from '../components/BaseIcon.vue'
import {
  activateTask,
  getTask,
  submitQuizAttempt,
  getNotebooks,
  generateQuizForPageRange,
  generateFlashcardsForQuizTask,
  completeMilestoneExam,
  trackAnalyticsEvent,
  getUserSettings,
  revertTopicToReadingFromQuiz,
} from '../services/appApi'
import StudyPageLayout from '../components/StudyPageLayout.vue'
import MilestoneNoteReviewModal from '../components/MilestoneNoteReviewModal.vue'
import { cleanTopicTitle } from '../composables/useReaderBase'
import { playCorrectChime, playIncorrectThud } from '../utils/audioJuice'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const submitting = ref(false)
const submitted = ref(false)
const error = ref('')
const taskMeta = ref(null)
const questions = ref([])
const answers = ref({})
const analyticsEnabled = ref(false)
const anonymousUserID = ref('')
const result = ref(null)
const generatingFlashcards = ref(false)
const feynmanReflection = ref('')
const synthesisMode = ref('feynman')

function getCalibrationClass(calibration) {
  const c = String(calibration || '').toLowerCase()
  if (c.includes('solid')) return 'cal--solid'
  if (c.includes('partially')) return 'cal--partial'
  return 'cal--misaligned'
}

// Restart Reading Session modal state
const showRestartReadingModal = ref(false)
const restartingReading = ref(false)
const restartReadingError = ref('')

// Terminal Failure / External Help Move Forward modal state
const showMoveForwardModal = ref(false)
const moveForwardError = ref('')

const isMilestoneExam = ref(false)
const milestonePayload = ref(null)
const showMilestonePreReview = ref(false)
const milestoneReviewTopics = ref([])

// Manual generation state
const notebooks = ref([])
const selectedNotebookID = ref('')
const startPage = ref(1)
const endPage = ref(10)
const generating = ref(false)
const manualPassingScore = ref(70)

const taskID = computed(() => {
  if (typeof route.query.taskId === 'string' && route.query.taskId.trim() !== '') {
    return route.query.taskId.trim()
  }
  if (typeof route.query.task_id === 'string' && route.query.task_id.trim() !== '') {
    return route.query.task_id.trim()
  }
  return ''
})

const allAnswered = computed(() => {
  if (questions.value.length === 0) {
    return false
  }
  return questions.value.every(
    (q) => typeof answers.value[q.id] === 'string' && answers.value[q.id].trim() !== ''
  )
})

const submitButtonLabel = computed(() => {
  if (submitting.value) return 'Scoring…'
  if (feynmanReflection.value.trim()) return 'Submit Quiz with Reflection'
  return 'Submit Quiz'
})

const canGenerateManual = computed(
  () =>
    selectedNotebookID.value &&
    startPage.value > 0 &&
    endPage.value >= startPage.value &&
    !generating.value
)

const canRetryGeneration = computed(() => {
  return (
    !taskID.value &&
    selectedNotebookID.value &&
    startPage.value > 0 &&
    endPage.value >= startPage.value
  )
})

function isCorrect(q) {
  if (!q || !q.correct_answer) return false
  return (answers.value[q.id] || '').trim().toLowerCase() === q.correct_answer.trim().toLowerCase()
}

onMounted(async () => {
  await loadNotebooks()
  try {
    const settings = await getUserSettings()
    analyticsEnabled.value = settings?.analytics_enabled ?? false
    anonymousUserID.value = settings?.anonymous_user_id ?? ''
    synthesisMode.value = (settings?.quiz_synthesis_mode || 'feynman').toLowerCase()
  } catch (err) {
    console.error('Failed to load user settings in Quiz:', err)
  }
  if (taskID.value) {
    await loadQuizTask()
  } else {
    loading.value = false
  }
})

watch(taskID, (newID) => {
  if (newID) {
    loadQuizTask()
  } else {
    taskMeta.value = null
    questions.value = []
    submitted.value = false
    result.value = null
  }
})

async function loadNotebooks() {
  try {
    const res = await getNotebooks()
    notebooks.value = Array.isArray(res) ? res.filter((n) => !n.error) : []
  } catch {
    if (!taskID.value) {
      error.value = 'Failed to load notebooks.'
    }
  }
}

async function loadQuizTask() {
  loading.value = true
  error.value = ''
  try {
    const activate = await activateTask(taskID.value)
    if (activate?.error && activate.error !== 'ErrTaskNotPending') {
      error.value = activate.error
      return
    }

    const response = await getTask(taskID.value)
    if (response?.error) {
      error.value = response.error
      return
    }

    const task = response?.task
    if (!task || (task.task_type !== 'QUIZ' && task.task_type !== 'MILESTONE_EXAM')) {
      error.value = 'Task is not a quiz or milestone exam task.'
      return
    }

    taskMeta.value = task
    let payload = null
    if (typeof task.payload_json === 'string' && task.payload_json.trim() !== '') {
      try {
        payload = JSON.parse(task.payload_json)
      } catch {
        error.value = 'Quiz payload data is corrupted.'
        return
      }
    }

    questions.value = Array.isArray(payload?.questions) ? payload.questions : []
    if (task.task_type === 'MILESTONE_EXAM') {
      isMilestoneExam.value = true
      milestonePayload.value = payload

      // Extract unique topic IDs / covered topics from payload
      const topicList = []
      if (payload?.quizzes && typeof payload.quizzes === 'object') {
        for (const tid of Object.keys(payload.quizzes)) {
          const title = (task.topic_id === tid && (task.topic_title || task.title)) ? (task.topic_title || task.title) : cleanTopicTitle(tid)
          topicList.push({ topic_id: tid, title: title || 'Chapter Review' })
        }
      }
      if (topicList.length === 0 && task.topic_id) {
        const title = task.topic_title || task.title || cleanTopicTitle(task.topic_id) || 'Chapter Review'
        topicList.push({ topic_id: task.topic_id, title })
      }
      milestoneReviewTopics.value = topicList
      const skipReview = route.query.reviewed === '1' || route.query.skipReview === '1'
      showMilestonePreReview.value = !skipReview
    } else {
      isMilestoneExam.value = false
      milestonePayload.value = null
      showMilestonePreReview.value = false
      milestoneReviewTopics.value = []
    }
    answers.value = {}
    submitted.value = false
    result.value = null
  } catch (err) {
    error.value = err?.message || 'Failed to load quiz task.'
  } finally {
    loading.value = false
  }
}

function onMilestonePreReviewComplete() {
  showMilestonePreReview.value = false
}

async function submitMilestone() {
  if (!taskID.value) {
    error.value = 'Missing task ID for milestone exam.'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const res = await completeMilestoneExam(taskID.value)
    if (res && res.error) {
      error.value = res.error
    } else {
      if (res?.rewards) {
        window.dispatchEvent(
          new CustomEvent('study-reward-earned', { detail: { rewards: res.rewards } })
        )
      }
      router.push('/dashboard')
    }
  } catch (err) {
    error.value = err?.message || 'Failed to complete milestone exam.'
  } finally {
    submitting.value = false
  }
}

async function generateManualQuiz() {
  error.value = ''
  questions.value = []
  answers.value = {}
  submitted.value = false
  result.value = null
  generating.value = true
  try {
    const res = await generateQuizForPageRange(
      selectedNotebookID.value,
      startPage.value,
      endPage.value
    )
    if (res.error) {
      error.value = res.error
      return
    }
    questions.value = Array.isArray(res.questions) ? res.questions : []
    // Respect backend-supplied passing score when available; default to 70
    if (
      typeof res.passing_score === 'number' &&
      !isNaN(res.passing_score) &&
      res.passing_score > 0
    ) {
      manualPassingScore.value = Math.round(res.passing_score)
    } else {
      manualPassingScore.value = 70
    }
    if (questions.value.length === 0) {
      error.value = 'No questions generated. Try a different page range.'
    }
  } catch (e) {
    error.value = e?.message ?? 'Quiz generation failed.'
  } finally {
    generating.value = false
  }
}

const scrollToTop = () => document.querySelector('.content-shell')?.scrollTo({ top: 0, behavior: 'smooth' })

async function submitQuiz() {
  if (!allAnswered.value) {
    return
  }
  submitting.value = true
  error.value = ''

  if (!taskID.value) {
    // Stateless frontend scoring for manual quizzes (so they do not touch the study queue)
    try {
      let correctCount = 0
      questions.value.forEach((q) => {
        if (isCorrect(q)) {
          correctCount++
        }
      })
      const score = Math.round((correctCount / questions.value.length) * 100)
      result.value = {
        score,
        passed: score >= manualPassingScore.value,
        passing_score: manualPassingScore.value,
        correct_count: correctCount,
        total_count: questions.value.length,
        feedback:
          score >= manualPassingScore.value
            ? 'Strong work. You can move forward.'
            : 'Review the missed concepts and retry the material.',
      }
      submitted.value = true
      scrollToTop()
    } catch (err) {
      error.value = err?.message || 'Failed to grade manual quiz.'
    } finally {
      submitting.value = false
    }
    return
  }

  try {
    const payload = questions.value.map((q) => ({
      question_id: q.id,
      selected: answers.value[q.id],
    }))
    const response = await submitQuizAttempt(taskID.value, payload, feynmanReflection.value.trim())
    if (response?.error) {
      error.value = response.error
      return
    }
    result.value = response?.result || null
    if (!result.value) {
      error.value = response?.error || 'Quiz submission returned no result.'
      submitted.value = false
      return
    }
    submitted.value = true
    scrollToTop()

    if (result.value?.passed) {
      playCorrectChime()
      if (response?.rewards) {
        window.dispatchEvent(
          new CustomEvent('study-reward-earned', { detail: { rewards: response.rewards } })
        )
      }
    } else {
      playIncorrectThud()
    }

    if (analyticsEnabled.value) {
      const notebook = notebooks.value.find((n) => n.id === taskMeta.value?.notebook_id)
      const fileHash = notebook?.file_hash || ''
      const currentQuizResult = response?.result
      void trackAnalyticsEvent('quiz_complete', fileHash, taskMeta.value?.start_page || 0, {
        task_id: taskID.value,
        score: currentQuizResult?.score || 0,
        passed: currentQuizResult?.passed || false,
        correct_count: currentQuizResult?.correct_count || 0,
        total_count: currentQuizResult?.total_count || 0,
        anonymous_user_id: anonymousUserID.value,
      }).catch(() => {})
    }
  } catch (err) {
    error.value = err?.message || 'Failed to submit quiz.'
  } finally {
    submitting.value = false
  }
}

async function retryFlashcardGeneration() {
  if (result.value) {
    result.value.flashcards_generation_error = false
    result.value.flashcards_generation_message = ''
    result.value.flashcards_pending = true
    await handleContinue()
  }
}

async function handleContinue() {
  // If quiz passed with flashcards pending, generate flashcards first
  if (result.value?.passed && result.value?.flashcards_pending) {
    generatingFlashcards.value = true
    error.value = ''
    result.value.flashcards_generation_error = false
    result.value.flashcards_generation_message = ''
    try {
      const genResult = await generateFlashcardsForQuizTask(result.value.task_id)
      if (genResult?.error) {
        console.warn('[FLASHCARD_PIPELINE] Flashcard generation failed:', genResult.error)
        result.value.flashcards_generation_error = true
        result.value.flashcards_generation_message = genResult.error
        result.value.flashcards_pending = false
        return
      } else {
        if (genResult?.rewards) {
          window.dispatchEvent(
            new CustomEvent('study-reward-earned', { detail: { rewards: genResult.rewards } })
          )
        }
        // Update result with generated counts for dashboard banner
        result.value.flashcards_generated = genResult?.cards_scheduled || 0
        result.value.flashcards_pending = false
      }
    } catch (err) {
      console.warn('[FLASHCARD_PIPELINE] Flashcard generation error:', err)
      result.value.flashcards_generation_error = true
      result.value.flashcards_generation_message = err?.message || 'Flashcard generation failed.'
      result.value.flashcards_pending = false
      return
    } finally {
      generatingFlashcards.value = false
    }
  }

  // Route to Socratic Rescue if rescue task is pending
  if (!result.value?.passed && result.value?.socratic_task_id) {
    router.push({
      path: '/socratic-rescue',
      query: { taskId: result.value.socratic_task_id },
    })
    return
  }

  // If failed re-quiz reached terminal manual review (external help required),
  // prompt user with clean choices: Create flashcards & move forward OR skip to next topic
  if (!result.value?.passed && !result.value?.reread_task_id && result.value?.manual_review_recommended) {
    moveForwardError.value = ''
    showMoveForwardModal.value = true
    return
  }

  // Route to dashboard after quiz completion
  const query = {}
  if (result.value?.passed && result.value?.flashcards_generated > 0) {
    query.flashcardsCreated = result.value.flashcards_generated
  } else if (!result.value?.passed && result.value?.reread_task_id) {
    query.highlight = result.value.reread_task_id
  }
  router.push({ path: '/dashboard', query })
}

function confirmSkipToNextTopic() {
  showMoveForwardModal.value = false
  moveForwardError.value = ''
  router.push({ path: '/dashboard' })
}

async function confirmCreateFlashcardsAndMove() {
  const quizTaskId = result.value?.task_id || taskID.value || ''
  if (!quizTaskId) {
    moveForwardError.value = 'Quiz task ID not found. Cannot generate flashcards.'
    return
  }

  generatingFlashcards.value = true
  moveForwardError.value = ''
  try {
    const genResult = await generateFlashcardsForQuizTask(quizTaskId)
    if (genResult?.error) {
      console.warn('[FLASHCARD_PIPELINE] Terminal move-forward flashcard generation failed:', genResult.error)
      moveForwardError.value = genResult.error
      return
    }
    if (genResult?.rewards) {
      window.dispatchEvent(
        new CustomEvent('study-reward-earned', { detail: { rewards: genResult.rewards } })
      )
    }
    showMoveForwardModal.value = false
    const cardsScheduled = genResult?.cards_scheduled || 0
    const query = {}
    if (cardsScheduled > 0) {
      query.flashcardsCreated = cardsScheduled
    }
    router.push({ path: '/dashboard', query })
  } catch (err) {
    console.warn('[FLASHCARD_PIPELINE] Error generating flashcards on move forward:', err)
    moveForwardError.value = err?.message || 'Failed to generate flashcards.'
  } finally {
    generatingFlashcards.value = false
  }
}

async function handleGoToExaminer() {
  const nbID = taskMeta.value?.notebook_id || selectedNotebookID.value || ''
  const startP = taskMeta.value?.start_page || startPage.value || 1
  const endP = taskMeta.value?.end_page || endPage.value || startP

  const quizTaskId = result.value?.task_id || taskID.value || ''
  const hasPendingFlashcards = Boolean(result.value?.passed && result.value?.flashcards_pending)

  const query = {
    notebookID: nbID,
    startPage: String(startP),
    endPage: String(endP),
    autoGenerate: 'true',
  }
  if (quizTaskId) {
    query.quizTaskId = quizTaskId
  }
  if (hasPendingFlashcards) {
    query.flashcardsPending = 'true'
  }

  router.push({
    path: '/examiner',
    query,
  })
}

function handleGoToDetailedAnalysis() {
  const nbID = taskMeta.value?.notebook_id || selectedNotebookID.value || ''
  const isMilestone = taskMeta.value?.task_type === 'MILESTONE_EXAM'

  // Cluster grouping logic
  const clusterMap = new Map()
  for (const q of questions.value) {
    const topicId = q.topic_id || taskMeta.value?.topic_id || 'default'
    const startP = q.source_page_start || taskMeta.value?.start_page || startPage.value || 1
    const rawEndP =
      q.source_page_end ||
      taskMeta.value?.end_page ||
      (!taskID.value ? endPage.value : startP) ||
      startP
    const endP = Math.max(startP, rawEndP)
    const key = `${topicId}-${startP}-${endP}`

    if (!clusterMap.has(key)) {
      clusterMap.set(key, {
        topicId,
        topicTitle: q.source_heading || taskMeta.value?.topic_title || '',
        startPage: startP,
        endPage: endP,
        questions: [],
      })
    }
    clusterMap.get(key).questions.push(q)
  }

  const clusters = []
  for (const item of clusterMap.values()) {
    const failed = []
    let correct = 0
    for (const q of item.questions) {
      if (isCorrect(q)) {
        correct++
      } else {
        failed.push({
          id: q.id,
          prompt: q.prompt,
          options: q.options || [],
          userAnswer: answers.value[q.id] || '',
          correctAnswer: q.correct_answer,
          sourcePageStart: q.source_page_start,
          sourceSnippet: q.source_snippet,
        })
      }
    }
    const scorePct =
      item.questions.length > 0 ? Math.round((correct / item.questions.length) * 100) : 0
    clusters.push({
      topicId: item.topicId,
      topicTitle: item.topicTitle,
      startPage: item.startPage,
      endPage: item.endPage,
      totalQuestions: item.questions.length,
      correctCount: correct,
      scorePercent: scorePct,
      passed: scorePct >= (result.value?.passing_score || 70),
      failedQuestions: failed,
    })
  }

  const analysisPayload = {
    taskId: result.value?.task_id || taskID.value || '',
    notebookId: nbID,
    isMilestone,
    overallScore: result.value?.score || 0,
    overallPassed: !!result.value?.passed,
    flashcardsPending: !!(result.value?.passed && result.value?.flashcards_pending),
    clusters,
  }

  try {
    sessionStorage.setItem('studyloop_quiz_analysis', JSON.stringify(analysisPayload))
  } catch (e) {
    console.warn('Failed to save quiz analysis to sessionStorage:', e)
    error.value = 'Failed to save quiz analysis data. Storage is unavailable.'
    return
  }

  router.push({
    path: '/quiz-analysis',
    query: {
      taskId: analysisPayload.taskId || undefined,
    },
  })
}

function handleRestartReading() {
  restartReadingError.value = ''
  showRestartReadingModal.value = true
}

async function confirmRestartReading() {
  const quizTaskId = result.value?.task_id || taskID.value || ''
  if (!quizTaskId) {
    restartReadingError.value = 'Quiz task ID not found. Cannot restart reading session.'
    return
  }

  restartingReading.value = true
  restartReadingError.value = ''
  try {
    const res = await revertTopicToReadingFromQuiz(quizTaskId)
    if (res?.error) {
      restartReadingError.value = res.error
      return
    }
    const readingTaskId = res?.reading_task_id || ''
    showRestartReadingModal.value = false
    if (readingTaskId) {
      router.push(`/reader?taskId=${readingTaskId}`)
    } else {
      router.push('/dashboard')
    }
  } catch (err) {
    restartReadingError.value = err?.message || 'Failed to restart reading session.'
  } finally {
    restartingReading.value = false
  }
}
</script>

<style scoped>
/* ── Toolbar controls ─────────────────────────── */
.toolbar-field {
  display: grid;
  gap: 4px;
}

.field-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--muted-text);
}

/* Ghost select: suggestion of a border, no hard box */
.ghost-select {
  appearance: none;
  width: 100%;
  padding: 8px 32px 8px 12px;
  background: var(--surface-container-lowest)
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' fill='none'%3E%3Cpath d='M1 1l4 4 4-4' stroke='%2364707d' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E")
    no-repeat right 12px center;
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  font: inherit;
  font-size: 14px;
  color: var(--on-surface);
  cursor: pointer;
  transition: border-color 0.15s ease;
  max-width: 220px;
}

.ghost-select:focus {
  outline: none;
  border-color: var(--primary);
}

.ghost-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Config panel ─────────────────────────────── */
.config-panel {
  background: var(--surface-container-low);
  border-radius: 16px;
  padding: 24px;
  display: grid;
  gap: 16px;
}

.config-panel__hint {
  margin: 0;
  font-size: 14px;
  color: var(--muted-text);
  line-height: 1.5;
}

.config-panel__row {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  flex-wrap: wrap;
}

.number-field {
  display: grid;
  gap: 4px;
}

/* Ghost input: 1px outline-variant hint, no heavy box */
.ghost-input {
  width: 96px;
  padding: 8px 12px;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  font: inherit;
  font-size: 14px;
  color: var(--on-surface);
  transition: border-color 0.15s ease;
}

.ghost-input:focus {
  outline: none;
  border-color: var(--primary);
}

.ghost-input:disabled {
  opacity: 0.5;
}

/* ── State panels ─────────────────────────────── */
.state-panel {
  background: var(--surface-container-low);
  border-radius: 16px;
  padding: 48px 24px;
  text-align: center;
}

.state-panel--error .state-text {
  color: #b42318;
}

.state-text {
  margin: 0;
  font-size: 15px;
  color: var(--muted-text);
}

/* ── Result panel ─────────────────────────────── */
.result-panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 32px 24px;
  display: grid;
  gap: 12px;
  justify-items: start;
}

.result-panel__badge {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  padding: 4px 12px;
  border-radius: 999px;
}

.badge--pass {
  background: color-mix(in srgb, #16a34a 12%, var(--surface-container-low));
  color: #16a34a;
}

.badge--fail {
  background: color-mix(in srgb, #b42318 12%, var(--surface-container-low));
  color: #b42318;
}

.result-panel__score {
  margin: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.score-value {
  font-family: 'Manrope', sans-serif;
  font-size: 40px;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--on-surface);
  line-height: 1;
}

.score-threshold {
  font-size: 13px;
  color: var(--muted-text);
}

.result-panel__feedback {
  margin: 0;
  font-size: 15px;
  color: var(--on-surface);
  line-height: 1.6;
  max-width: 60ch;
}

.result-panel__attempts {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

/* Flashcard success message styles */
.result-panel__flashcard-success {
  background: color-mix(in srgb, #16a34a 8%, var(--surface-container-low));
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 16px 20px;
  margin: 8px 0;
}

.flashcard-success-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  color: #16a34a;
}

.flashcard-success-detail {
  margin: 0 0 4px;
  font-size: 14px;
  color: var(--on-surface);
}

.flashcard-success-hint {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
  font-style: italic;
}

.result-panel__reread-message {
  margin: 8px 0 0;
  font-size: 14px;
  color: var(--on-surface);
  font-weight: 500;
}

/* ── Quiz form ────────────────────────────────── */
.quiz-form {
  display: grid;
  gap: 12px;
}

.question-card {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 24px;
  display: grid;
  gap: 16px;
  transition: background 0.15s ease;
}

.question-prompt {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface);
  line-height: 1.5;
  display: flex;
  gap: 10px;
}

.question-num {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--muted-text);
  padding-top: 3px;
  min-width: 1.5ch;
}

.options-grid {
  display: grid;
  gap: 8px;
}

.option-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--surface-container-low);
  cursor: pointer;
  transition: background 0.12s ease;
}

.option-row:hover {
  background: color-mix(in srgb, var(--primary) 6%, var(--surface-container-low));
}

.option-row--selected {
  background: color-mix(in srgb, var(--primary) 10%, var(--surface-container-lowest));
}

.option-radio {
  margin-top: 2px;
  flex-shrink: 0;
  accent-color: var(--primary);
}

.option-text {
  font-size: 14px;
  color: var(--on-surface);
  line-height: 1.5;
}

/* ── Form footer ──────────────────────────────── */
.form-footer {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-top: 8px;
  flex-wrap: wrap;
}

.footer-hint {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

/* ── Primary CTA ──────────────────────────────── */
.primary-btn {
  border: 0;
  border-radius: 12px;
  padding: 11px 24px;
  color: var(--on-primary);
  font-family: inherit;
  font-size: 14px;
  font-weight: 700;
  background: linear-gradient(15deg, var(--primary-dim), var(--primary));
  cursor: pointer;
  transition:
    transform 0.14s ease,
    filter 0.14s ease;
  white-space: nowrap;
}

.primary-btn:hover:not(:disabled) {
  filter: brightness(1.08);
}

.primary-btn:active:not(:disabled) {
  transform: scale(0.96);
}

.primary-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.result-panel__network-warning {
  display: flex;
  gap: 12px;
  background: rgba(230, 126, 34, 0.1);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 8px;
  width: 100%;
}

.warning-icon {
  font-size: 20px;
  color: #d35400;
  flex-shrink: 0;
}

.warning-title {
  margin: 0 0 4px;
  font-size: 14px;
  font-weight: 700;
  color: #d35400;
}

.warning-detail,
.warning-message {
  margin: 0;
  font-size: 13px;
  line-height: 1.4;
  color: var(--on-surface);
}

.warning-message {
  margin-top: 6px;
  font-family: monospace;
  font-size: 11.5px;
  background: var(--surface-container-low);
  padding: 4px 8px;
  border-radius: 6px;
}

.result-panel__rescue-notice {
  background: color-mix(in srgb, var(--primary, #3b82f6) 10%, transparent);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 8px;
  width: 100%;
}

.rescue-notice-title {
  margin: 0 0 4px;
  font-size: 14px;
  font-weight: 700;
  color: var(--primary, #3b82f6);
}

.rescue-notice-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.4;
  color: var(--on-surface);
}

.result-panel__external-help-notice {
  background: rgba(192, 41, 43, 0.1);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 8px;
  width: 100%;
}

.external-help-title {
  margin: 0 0 4px;
  font-size: 14px;
  font-weight: 700;
  color: #c0392b;
}

.external-help-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.4;
  color: var(--on-surface);
}

.result-panel__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
  width: 100%;
}

.secondary-btn {
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 11px 20px;
  color: var(--on-surface);
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  background: var(--surface-container-low);
  cursor: pointer;
  transition: background 0.14s ease, transform 0.14s ease;
}

.secondary-btn:hover:not(:disabled) {
  background: var(--surface-container-high);
}

.retry-generation-btn {
  margin-top: 12px;
  min-width: 120px;
}

/* ── Result Breakdown ─────────────────────────── */
.result-breakdown {
  display: grid;
  gap: 16px;
  margin-top: 24px;
  border-top: 1px solid var(--outline-variant);
  padding-top: 20px;
  text-align: left;
  width: 100%;
}

.breakdown-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface);
}

.breakdown-card {
  background: var(--surface-container-high);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid var(--outline-variant);
  display: grid;
  gap: 10px;
}

.breakdown-card--correct {
  border-left: 4px solid #2ecc71;
}

.breakdown-card--incorrect {
  border-left: 4px solid #e74c3c;
}

.breakdown-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.breakdown-status {
  font-size: 12px;
  font-weight: 700;
}

.breakdown-card--correct .breakdown-status {
  color: #2ecc71;
}

.breakdown-card--incorrect .breakdown-status {
  color: #e74c3c;
}

.breakdown-answers {
  display: grid;
  gap: 6px;
  font-size: 13px;
}

.answer-item {
  margin: 0;
  display: flex;
  gap: 8px;
  align-items: baseline;
}

.answer-label {
  font-weight: 600;
  color: var(--muted-text);
}

.answer-val {
  font-weight: 500;
}

.val--correct {
  color: #2ecc71;
}

.val--incorrect {
  color: #e74c3c;
}

/* Voice Viva Challenge Hero Card */
.viva-challenge-card {
  width: 100%;
  margin: 20px 0 8px;
  padding: 20px 24px;
  background: linear-gradient(135deg, rgba(243, 156, 18, 0.08) 0%, rgba(211, 84, 0, 0.12) 100%);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  box-shadow: 0 4px 16px rgba(243, 156, 18, 0.08);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  box-sizing: border-box;
}

.viva-challenge-card__content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  text-align: left;
}

.viva-challenge-card__badge {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.05em;
  color: #f39c12;
}

.viva-challenge-card__tag {
  background: rgba(243, 156, 18, 0.2);
  padding: 2px 10px;
  border-radius: 20px;
  font-size: 10px;
}

.viva-challenge-card__title {
  margin: 2px 0 0;
  font-size: 17px;
  font-weight: 700;
  color: var(--on-surface);
}

.viva-challenge-card__desc {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
  line-height: 1.4;
}

.viva-challenge-card__perks {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 12px;
  font-weight: 600;
  color: #f1c40f;
  margin-top: 2px;
}

.viva-challenge-btn {
  flex-shrink: 0;
  padding: 12px 20px;
  background: linear-gradient(135deg, #f39c12 0%, #d35400 100%);
  color: #ffffff;
  font-weight: 700;
  font-size: 13px;
  letter-spacing: 0.04em;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  box-shadow: 0 3px 10px rgba(211, 84, 0, 0.3);
  transition: all 0.16s ease;
  white-space: nowrap;
}

.viva-challenge-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 5px 15px rgba(211, 84, 0, 0.45);
}

@media (max-width: 768px) {
  .viva-challenge-card {
    flex-direction: column;
    align-items: stretch;
  }
  .viva-challenge-btn {
    width: 100%;
  }
}

.result-panel__analysis-action {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}

.analysis-link-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 8px 16px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.analysis-link-btn:hover:not(:disabled) {
  color: var(--on-surface);
  border-color: var(--muted-text);
  background: var(--surface-container-low);
}

.analysis-link-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.analysis-svg,
.analysis-arrow-svg {
  flex-shrink: 0;
}

/* ── Restart Reading Button ───────────────────── */
.restart-reading-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 18px;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--on-surface);
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.restart-reading-btn:hover:not(:disabled) {
  border-color: var(--error, #dc2626);
  color: var(--error, #dc2626);
  background: color-mix(in srgb, var(--error, #dc2626) 6%, transparent);
}

.restart-reading-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ── Restart Reading Modal ────────────────────── */
.restart-reading-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(2px);
  padding: 16px;
}

.restart-reading-dialog {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  width: 100%;
  max-width: 460px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  overflow: hidden;
}

.restart-reading-dialog__header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 24px 0;
}

.restart-reading-dialog__icon {
  color: var(--error, #dc2626);
  flex-shrink: 0;
}

.restart-reading-dialog__title {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  color: var(--on-surface);
}

.restart-reading-dialog__body {
  padding: 16px 24px;
}

.restart-reading-dialog__desc {
  margin: 0 0 12px;
  font-size: 0.875rem;
  color: var(--on-surface);
  line-height: 1.5;
}

.restart-reading-dialog__list {
  margin: 0 0 14px;
  padding-left: 20px;
  font-size: 0.8125rem;
  color: var(--muted-text);
  line-height: 1.7;
}

.restart-reading-dialog__warning {
  margin: 0;
  padding: 10px 14px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--error, #dc2626) 8%, transparent);
  border: 1px solid var(--outline-variant);
  font-size: 0.8125rem;
  color: var(--on-surface);
  line-height: 1.5;
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.restart-reading-dialog__warning-icon {
  flex-shrink: 0;
  margin-top: 1px;
  color: var(--error, #dc2626);
}

.restart-reading-dialog__error {
  margin: 10px 0 0;
  padding: 8px 12px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--error, #dc2626) 10%, transparent);
  border: 1px solid var(--outline-variant);
  font-size: 0.8125rem;
  color: var(--error, #dc2626);
}

.restart-reading-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--outline-variant);
}

.danger-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  font-size: 0.875rem;
  font-weight: 600;
  color: #fff;
  background: var(--error, #dc2626);
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.danger-btn:hover:not(:disabled) {
  filter: brightness(1.1);
}

.danger-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* ── Move Forward Modal Styles ─────────────────── */
.move-forward-dialog__icon {
  color: var(--primary, #3b82f6);
  flex-shrink: 0;
}

.move-forward-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
  margin-bottom: 8px;
}

.move-forward-card {
  padding: 14px 16px;
  border-radius: 10px;
  background: var(--surface-container-high, rgba(255, 255, 255, 0.04));
  border: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.move-forward-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--on-surface);
}

.move-forward-card__desc {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--muted-text);
  line-height: 1.45;
}

.move-forward-footer {
  gap: 12px;
}

.skip-btn {
  background: var(--surface-container-highest, rgba(255, 255, 255, 0.08));
  border: none;
  color: var(--on-surface);
}

.skip-btn:hover:not(:disabled) {
  filter: brightness(1.1);
}

/* Feynman Reflection Styles - Blends seamlessly into normal cards */
.feynman-reflection-card {
  border: none;
  background: var(--surface-container-lowest);
  box-shadow: none;
}

.feynman-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 2px;
}

.feynman-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--muted-text);
}

.feynman-hint {
  font-size: 12px;
  color: var(--muted-text);
}

.feynman-prompt {
  font-size: 15px;
  font-weight: 500;
  color: var(--on-surface);
  line-height: 1.5;
}

.reflection-input-shell {
  margin-top: 6px;
}

.feynman-textarea {
  width: 100%;
  box-sizing: border-box;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-family: inherit;
  font-size: 14px;
  line-height: 1.5;
  resize: vertical;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.feynman-textarea:focus {
  outline: none;
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
  background: color-mix(in srgb, var(--primary) 3%, var(--surface-container-low));
}

/* Feynman Feedback Card */
.feynman-feedback-card {
  margin: 20px 0;
  padding: 18px 20px;
  border-radius: 8px;
  background: var(--bg-surface-elevated, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.1));
}

.feynman-feedback-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.feynman-calibration-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.cal--solid {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.cal--partial {
  background: rgba(234, 179, 8, 0.15);
  color: #eab308;
  border: 1px solid rgba(234, 179, 8, 0.3);
}

.cal--misaligned {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.feynman-your-words {
  margin-bottom: 14px;
  padding: 10px 12px;
  border-radius: 6px;
  background: var(--bg-surface, rgba(0, 0, 0, 0.15));
  border-left: 2px solid var(--border-subtle, rgba(255, 255, 255, 0.2));
}

.feynman-words-label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted, #888);
  margin-bottom: 4px;
}

.feynman-words-quote {
  font-size: 13.5px;
  font-style: italic;
  color: var(--text-normal, #ddd);
  margin: 0;
  line-height: 1.45;
}

.feynman-feedback-block {
  margin-bottom: 12px;
  padding: 10px 14px;
  border-radius: 6px;
}

.block--strengths {
  background: rgba(34, 197, 94, 0.06);
  border: 1px solid rgba(34, 197, 94, 0.15);
}

.block--gaps {
  background: rgba(234, 179, 8, 0.06);
  border: 1px solid rgba(234, 179, 8, 0.15);
}

.feedback-heading {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  font-weight: 700;
  margin: 0 0 6px 0;
}

.block--strengths .feedback-heading {
  color: #22c55e;
}

.block--gaps .feedback-heading {
  color: #eab308;
}

.feedback-list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.45;
  color: var(--text-normal, #ddd);
}

.feynman-summary-text {
  font-size: 13px;
  color: var(--text-muted, #aaa);
  margin: 8px 0 0 0;
  line-height: 1.4;
}
</style>
