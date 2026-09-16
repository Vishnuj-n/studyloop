<template>
  <StudyPageLayout
    eyebrow="Diagnostic"
    :title="isMilestone ? 'Milestone Exam Analysis' : 'Quiz Performance Breakdown'"
    :subtitle="
      clusters.length > 1
        ? `${clusters.length} chapters / sessions evaluated`
        : clusters[0]
          ? `Pages ${clusters[0].startPage}–${clusters[0].endPage}`
          : 'Assessment Overview'
    "
  >
    <template #toolbar>
      <div class="analysis-toolbar">
        <button
          type="button"
          class="ghost-action-btn"
          :disabled="processingExit"
          @click="handleExit(true)"
        >
          <svg
            class="action-svg"
            width="15"
            height="15"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M19 12H5" />
            <path d="M12 19l-7-7 7-7" />
          </svg>
          <span>Return to Dashboard</span>
        </button>
      </div>
    </template>

    <!-- Top Assessment Metrics Panel -->
    <section class="diagnostic-summary">
      <div class="summary-metric">
        <span class="metric-label">Overall Score</span>
        <div class="metric-value-row">
          <span class="metric-value" :class="overallPassed ? 'val--pass' : 'val--fail'">
            {{ overallScore }}%
          </span>
          <span class="metric-badge" :class="overallPassed ? 'badge--pass' : 'badge--fail'">
            {{ overallPassed ? 'Passed' : 'Needs Review' }}
          </span>
        </div>
      </div>

      <div class="summary-metric">
        <span class="metric-label">Questions Evaluated</span>
        <div class="metric-value-row">
          <span class="metric-value">{{ totalCorrect }} / {{ totalQuestions }}</span>
          <span class="metric-subtext">correct</span>
        </div>
      </div>

      <div class="summary-metric">
        <span class="metric-label">Concept Chapters</span>
        <div class="metric-value-row">
          <span class="metric-value">{{ masteredClustersCount }} / {{ clusters.length }}</span>
          <span class="metric-subtext">mastered</span>
        </div>
      </div>
    </section>

    <!-- Diagnostic Filter Tabs (when multiple clusters) -->
    <div v-if="clusters.length > 1" class="cluster-filter-nav">
      <button
        type="button"
        class="filter-chip"
        :class="{ 'filter-chip--active': activeFilter === 'all' }"
        @click="activeFilter = 'all'"
      >
        All Chapters ({{ clusters.length }})
      </button>
      <button
        type="button"
        class="filter-chip"
        :class="{ 'filter-chip--active': activeFilter === 'weak' }"
        @click="activeFilter = 'weak'"
      >
        Needs Attention ({{ weakClusters.length }})
      </button>
      <button
        type="button"
        class="filter-chip"
        :class="{ 'filter-chip--active': activeFilter === 'strong' }"
        @click="activeFilter = 'strong'"
      >
        Mastered ({{ masteredClusters.length }})
      </button>
    </div>

    <!-- Cluster Performance List -->
    <div class="clusters-container">
      <article
        v-for="cluster in visibleClusters"
        :key="cluster.topicId + cluster.startPage"
        class="cluster-card"
        :class="cluster.passed ? 'cluster-card--pass' : 'cluster-card--fail'"
      >
        <!-- Cluster Header -->
        <header class="cluster-header">
          <div class="cluster-title-wrap">
            <h3 class="cluster-title">
              {{ cluster.topicTitle || 'Concept Module' }}
            </h3>
            <span class="cluster-pages-tag">
              Pages {{ cluster.startPage }}–{{ cluster.endPage }}
            </span>
          </div>

          <div class="cluster-score-pill" :class="cluster.passed ? 'pill--pass' : 'pill--fail'">
            <span class="score-fraction"
              >{{ cluster.correctCount }}/{{ cluster.totalQuestions }} Correct</span
            >
            <span class="score-percent">({{ cluster.scorePercent }}%)</span>
          </div>
        </header>

        <!-- Mastered State Banner (No Mistakes) -->
        <div v-if="cluster.failedQuestions.length === 0" class="cluster-mastered-note">
          <svg
            class="note-svg note-svg--success"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
            <polyline points="22 4 12 14.01 9 11.01" />
          </svg>
          <span>All questions answered correctly for this section.</span>
        </div>

        <!-- Weakness / Remedial Section -->
        <div v-else class="cluster-weakness-section">
          <div class="weakness-summary-bar">
            <div class="weakness-summary-text">
              <span class="weakness-count">
                {{ cluster.failedQuestions.length }} question{{
                  cluster.failedQuestions.length > 1 ? 's' : ''
                }}
                missed
              </span>
              <span class="weakness-hint">— Recommended active-recall remediation available</span>
            </div>

            <div class="cluster-actions">
              <button
                type="button"
                class="cluster-action-btn"
                :class="{ 'btn--copied': copiedClusterId === getClusterKey(cluster) }"
                @click="copySocraticPrompt(cluster)"
              >
                <svg
                  v-if="copiedClusterId === getClusterKey(cluster)"
                  class="btn-svg"
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
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                <svg
                  v-else
                  class="btn-svg"
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
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
                <span>{{
                  copiedClusterId === getClusterKey(cluster)
                    ? 'Prompt Copied'
                    : 'Copy Socratic Prompt'
                }}</span>
              </button>

              <button
                type="button"
                class="cluster-action-btn"
                @click="openInTutor(cluster)"
              >
                <svg
                  class="btn-svg"
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
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                </svg>
                <span>Launch Socratic Tutor</span>
              </button>

              <button
                type="button"
                class="cluster-action-btn cluster-action-btn--secondary"
                @click="openInReader(cluster)"
              >
                <svg
                  class="btn-svg"
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
                  <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
                  <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
                </svg>
                <span>Open Reader (p. {{ cluster.startPage }})</span>
              </button>
            </div>
          </div>

          <!-- Failed Questions Detail List -->
          <div class="failed-questions-list">
            <div
              v-for="(fq, fqIdx) in cluster.failedQuestions"
              :key="fq.id || fqIdx"
              class="failed-question-item"
            >
              <div class="failed-question-header">
                <span class="question-index-tag">Question {{ fqIdx + 1 }}</span>
                <span v-if="fq.sourcePageStart" class="question-page-tag">
                  Page {{ fq.sourcePageStart }}
                </span>
              </div>

              <p class="failed-question-prompt">{{ fq.prompt }}</p>

              <div class="answers-comparison-grid">
                <div class="answer-box answer-box--user">
                  <span class="answer-box-label">Your Response</span>
                  <p class="answer-box-text">{{ fq.userAnswer || 'No answer selected' }}</p>
                </div>
                <div class="answer-box answer-box--correct">
                  <span class="answer-box-label">Expected Answer</span>
                  <p class="answer-box-text">{{ fq.correctAnswer }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </article>
    </div>

    <!-- Bottom Actions Panel -->
    <footer class="analysis-footer">
      <div class="footer-status-text">
        <span v-if="flashcardsPending" class="status-indicator-dot" />
        <span v-if="flashcardsPending">
          Review flashcards will be scheduled when returning to Dashboard.
        </span>
      </div>

      <div class="footer-button-group">
        <button
          v-if="flashcardsPending"
          type="button"
          class="ghost-btn"
          :disabled="processingExit"
          @click="handleExit(false)"
        >
          Skip Card Generation
        </button>
        <button
          type="button"
          class="primary-btn complete-btn"
          :disabled="processingExit"
          @click="handleExit(true)"
        >
          <span v-if="processingExit">Scheduling flashcards…</span>
          <span v-else>Done & Return to Dashboard →</span>
        </button>
      </div>
    </footer>
  </StudyPageLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import StudyPageLayout from '../components/StudyPageLayout.vue'
import { generateFlashcardsForQuizTask, getTask } from '../services/appApi'

const route = useRoute()
const router = useRouter()

const clusters = ref([])
const taskID = ref('')
const notebookID = ref('')
const isMilestone = ref(false)
const overallScore = ref(0)
const overallPassed = ref(false)
const flashcardsPending = ref(false)
const processingExit = ref(false)
const activeFilter = ref('all')
const copiedClusterId = ref('')

onMounted(async () => {
  // Load analysis payload from sessionStorage or query
  const storedData = sessionStorage.getItem('studyloop_quiz_analysis')
  if (storedData) {
    try {
      const parsed = JSON.parse(storedData)
      clusters.value = Array.isArray(parsed.clusters) ? parsed.clusters : []
      taskID.value = parsed.taskId || ''
      notebookID.value = parsed.notebookId || ''
      isMilestone.value = !!parsed.isMilestone
      overallScore.value = parsed.overallScore || 0
      overallPassed.value = !!parsed.overallPassed
      flashcardsPending.value = !!parsed.flashcardsPending
    } catch (e) {
      console.warn('Failed to parse quiz analysis data from sessionStorage:', e)
    }
  }

  // Fallback: If no clusters in sessionStorage, attempt to load task from route query
  if (clusters.value.length === 0 && route.query.taskId) {
    taskID.value = String(route.query.taskId)
    try {
      const taskData = await getTask(taskID.value)
      if (taskData) {
        notebookID.value = taskData.notebook_id || ''
        isMilestone.value = taskData.task_type === 'MILESTONE_EXAM'
      }
    } catch (err) {
      console.warn('Failed to fallback load task context:', err)
    }
  }
})

const totalQuestions = computed(() => {
  return clusters.value.reduce((sum, c) => sum + (c.totalQuestions || 0), 0)
})

const totalCorrect = computed(() => {
  return clusters.value.reduce((sum, c) => sum + (c.correctCount || 0), 0)
})

const masteredClusters = computed(() => {
  return clusters.value.filter((c) => c.passed)
})

const weakClusters = computed(() => {
  return clusters.value.filter((c) => !c.passed || c.failedQuestions?.length > 0)
})

const masteredClustersCount = computed(() => {
  return masteredClusters.value.length
})

const visibleClusters = computed(() => {
  if (activeFilter.value === 'weak') {
    return weakClusters.value
  }
  if (activeFilter.value === 'strong') {
    return masteredClusters.value
  }
  return clusters.value
})

function getClusterKey(cluster) {
  return `${cluster.topicId}-${cluster.startPage}-${cluster.endPage}`
}

function buildSocraticPromptText(cluster) {
  const chapterContext = cluster.topicTitle
    ? `Chapter "${cluster.topicTitle}" (pages ${cluster.startPage}–${cluster.endPage})`
    : `pages ${cluster.startPage}–${cluster.endPage}`

  const questionsList = (cluster.failedQuestions || [])
    .map((fq, idx) => {
      return `${idx + 1}. Concept: "${fq.prompt}"\n   My Answer: "${fq.userAnswer || 'Unknown'}"\n   Expected: "${fq.correctAnswer}"`
    })
    .join('\n\n')

  return `You are an expert Socratic tutor. I am studying ${chapterContext}.\n\nDuring my recent assessment, I missed the following concept questions:\n\n${questionsList}\n\nDo not simply recite the answers. Guide me with focused, thought-provoking Socratic questions to help me identify where my understanding broke down and reconstruct the underlying principles.`
}

async function copySocraticPrompt(cluster) {
  const promptText = buildSocraticPromptText(cluster)
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(promptText)
    } else {
      const el = document.createElement('textarea')
      el.value = promptText
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    }
    const key = getClusterKey(cluster)
    copiedClusterId.value = key
    setTimeout(() => {
      if (copiedClusterId.value === key) {
        copiedClusterId.value = ''
      }
    }, 2500)
  } catch (err) {
    console.warn('Failed to copy prompt to clipboard:', err)
  }
}

function openInReader(cluster) {
  router.push({
    path: '/reader',
    query: {
      notebook_id: notebookID.value || undefined,
      topic_id: cluster.topicId || undefined,
      page: cluster.startPage || undefined,
    },
  })
}

function openInTutor(cluster) {
  const promptText = buildSocraticPromptText(cluster)
  router.push({
    path: '/tutor',
    query: {
      notebookId: notebookID.value || undefined,
      topicId: cluster.topicId || undefined,
      prompt: promptText,
    },
  })
}

async function handleExit(shouldGenerateFlashcards) {
  if (processingExit.value) return
  processingExit.value = true

  const query = {}

  if (shouldGenerateFlashcards && flashcardsPending.value && taskID.value) {
    try {
      const genResult = await generateFlashcardsForQuizTask(taskID.value)
      if (genResult?.rewards) {
        window.dispatchEvent(
          new CustomEvent('study-reward-earned', { detail: { rewards: genResult.rewards } })
        )
      }
      if (genResult?.cards_scheduled > 0) {
        query.flashcardsCreated = genResult.cards_scheduled
      }
    } catch (err) {
      console.warn('Deferred flashcard generation failed on exit from analysis:', err)
    }
  }

  // Clear analysis session storage to prevent stale reuse
  sessionStorage.removeItem('studyloop_quiz_analysis')
  router.push({ path: '/dashboard', query })
}
</script>

<style scoped>
.analysis-toolbar {
  display: flex;
  align-items: center;
}

.ghost-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.ghost-action-btn:hover:not(:disabled) {
  color: var(--on-surface);
  border-color: var(--muted-text);
  background: var(--surface-container-low);
}

.ghost-action-btn:active:not(:disabled) {
  transform: scale(0.97);
}

.diagnostic-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.summary-metric {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 16px 20px;
}

.metric-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted-text);
  margin-bottom: 8px;
}

.metric-value-row {
  display: flex;
  align-items: baseline;
  gap: 10px;
}

.metric-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--on-surface);
}

.val--pass {
  color: #10b981;
}

.val--fail {
  color: #f59e0b;
}

.metric-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
}

.badge--pass {
  background: rgba(16, 185, 129, 0.12);
  color: #10b981;
}

.badge--fail {
  background: rgba(245, 158, 11, 0.12);
  color: #f59e0b;
}

.metric-subtext {
  font-size: 0.8125rem;
  color: var(--muted-text);
}

.cluster-filter-nav {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
}

.filter-chip {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  padding: 6px 14px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-chip:hover {
  color: var(--on-surface);
}

.filter-chip--active {
  background: var(--primary);
  border-color: var(--primary);
  color: var(--on-primary);
}

.clusters-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-bottom: 32px;
}

.cluster-card {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 20px;
  transition: border-color 0.15s ease;
}

.cluster-card--pass {
  border-left: 4px solid #10b981;
}

.cluster-card--fail {
  border-left: 4px solid #f59e0b;
}

.cluster-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--outline-variant);
}

.cluster-title {
  margin: 0 0 6px 0;
  font-size: 1.0625rem;
  font-weight: 600;
  color: var(--on-surface);
}

.cluster-pages-tag {
  display: inline-block;
  font-size: 0.75rem;
  font-weight: 500;
  background: var(--surface-container-low);
  color: var(--muted-text);
  padding: 2px 8px;
  border-radius: 4px;
}

.cluster-score-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 600;
}

.pill--pass {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.pill--fail {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.cluster-mastered-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding: 12px 16px;
  background: rgba(16, 185, 129, 0.05);
  border-radius: 6px;
  font-size: 0.875rem;
  color: #10b981;
}

.weakness-summary-bar {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
  margin-bottom: 16px;
}

.weakness-count {
  font-size: 0.875rem;
  font-weight: 600;
  color: #f59e0b;
}

.weakness-hint {
  font-size: 0.8125rem;
  color: var(--muted-text);
}

.cluster-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.cluster-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--on-surface);
  cursor: pointer;
  transition: all 0.15s ease;
}

.cluster-action-btn:hover {
  background: var(--surface-container);
  border-color: var(--muted-text);
}

.cluster-action-btn:active {
  transform: scale(0.97);
}

.cluster-action-btn--secondary {
  color: var(--muted-text);
}

.btn--copied {
  background: rgba(16, 185, 129, 0.12);
  border-color: #10b981;
  color: #10b981;
}

.failed-questions-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 12px;
}

.failed-question-item {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 14px 16px;
}

.failed-question-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.question-index-tag {
  font-size: 0.6875rem;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--muted-text);
}

.question-page-tag {
  font-size: 0.75rem;
  color: var(--muted-text);
}

.failed-question-prompt {
  margin: 0 0 12px 0;
  font-size: 0.875rem;
  line-height: 1.45;
  color: var(--on-surface);
}

.answers-comparison-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.answer-box {
  padding: 8px 12px;
  border-radius: 4px;
}

.answer-box--user {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.answer-box--correct {
  background: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.answer-box-label {
  display: block;
  font-size: 0.6875rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 2px;
}

.answer-box--user .answer-box-label {
  color: #ef4444;
}

.answer-box--correct .answer-box-label {
  color: #10b981;
}

.answer-box-text {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--on-surface);
}

.analysis-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 20px;
  border-top: 1px solid var(--outline-variant);
  margin-top: 24px;
}

.footer-status-text {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  color: var(--muted-text);
}

.status-indicator-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary);
}

.footer-button-group {
  display: flex;
  gap: 12px;
}

.ghost-btn {
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  padding: 8px 16px;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.ghost-btn:hover:not(:disabled) {
  color: var(--on-surface);
  border-color: var(--muted-text);
}

.complete-btn {
  padding: 8px 20px;
  font-size: 0.875rem;
  font-weight: 600;
  border-radius: 6px;
}
</style>
