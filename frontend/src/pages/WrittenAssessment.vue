<template>
  <StudyPageLayout eyebrow="Assessment" title="Written Assessment">
    <!-- Toolbar: notebook selector -->
    <template #toolbar>
      <div class="toolbar-field">
        <label class="field-label" for="wa-notebook-select">Notebook</label>
        <select
          id="wa-notebook-select"
          v-model="selectedNotebookID"
          class="ghost-select"
          :disabled="loading || scoring"
        >
          <option value="">— Select Notebook —</option>
          <option v-for="nb in notebooks" :key="nb.id" :value="nb.id">{{ nb.title }}</option>
        </select>
      </div>
    </template>

    <!-- ── COMPREHENSIVE TAB ───────────────────── -->
    <section class="tab-content">
      <!-- Config panel: idle / pre-question state -->
      <div v-if="!question" class="config-panel">
        <p class="config-panel__hint">
          Select a page range and generate a long-form question drawn from your notebook.
          <span v-if="selectedNotebook" class="range-hint">
            (Available: Pages {{ selectedNotebook.start_page || 1 }} – {{ selectedNotebook.end_page || selectedNotebook.page_count || 1 }})
          </span>
        </p>
        <div class="config-panel__row">
          <div class="number-field">
            <label class="field-label" for="wa-start">Start Page</label>
            <input
              id="wa-start"
              v-model.number="startPage"
              class="ghost-input"
              type="number"
              :min="selectedNotebook?.start_page || 1"
              :max="selectedNotebook?.end_page || selectedNotebook?.page_count || 99999"
              :disabled="loading"
            />
          </div>
          <div class="number-field">
            <label class="field-label" for="wa-end">End Page</label>
            <input
              id="wa-end"
              v-model.number="endPage"
              class="ghost-input"
              type="number"
              :min="selectedNotebook?.start_page || 1"
              :max="selectedNotebook?.end_page || selectedNotebook?.page_count || 99999"
              :disabled="loading"
            />
          </div>
          <button
            id="wa-generate-btn"
            class="primary-btn"
            :disabled="!canGenerate"
            @click="generate"
          >
            {{ loading ? 'Generating…' : 'Generate Question' }}
          </button>
        </div>

        <!-- Error state -->
        <article v-if="error" class="state-panel state-panel--error">
          <p class="state-text">{{ error }}</p>
          <button
            v-if="canGenerate"
            class="primary-btn retry-generation-btn"
            :disabled="loading"
            @click="generate"
          >
            Retry Generation
          </button>
        </article>
      </div>

      <!-- Active exam: question + answer -->
      <div v-if="question && !result" class="exam-area">
        <!-- Question card -->
        <article class="question-card">
          <p class="question-prompt">{{ question.prompt }}</p>
          <span class="source-badge"
            >Pages {{ question.sourcePageStart }}–{{ question.sourcePageEnd }}</span
          >
        </article>

        <!-- Answer textarea -->
        <div class="answer-field">
          <label class="field-label" for="wa-answer">Your Answer</label>
          <textarea
            id="wa-answer"
            v-model="userAnswer"
            class="ghost-textarea"
            rows="7"
            placeholder="Write or dictate your answer here…"
            :disabled="scoring"
          />
        </div>

        <!-- STT Tip -->
        <div class="stt-tip">
          <span class="stt-tip__icon">💡</span>
          <span class="stt-tip__text">
            Prefer speaking? Use OS dictation (<kbd class="kbd-badge">Win + H</kbd>) or offline tools like
            <a
              href="https://github.com/cjpais/Handy"
              target="_blank"
              rel="noopener noreferrer"
              class="stt-tip__link"
            >
              Handy
            </a>
            to dictate your explanation directly into the answer field.
          </span>
        </div>

        <!-- Actions row -->
        <div class="form-footer">
          <button id="wa-discard-btn" class="ghost-btn" :disabled="scoring" @click="reset">
            Discard
          </button>
          <button
            id="wa-submit-btn"
            class="primary-btn"
            :disabled="!userAnswer.trim() || scoring"
            @click="submitAnswer"
          >
            {{ scoring ? 'Scoring…' : 'Submit Answer' }}
          </button>
        </div>

        <!-- Inline error during submission -->
        <article v-if="error" class="state-panel state-panel--error">
          <p class="state-text">{{ error }}</p>
          <button
            class="primary-btn retry-generation-btn"
            :disabled="scoring || !userAnswer.trim()"
            @click="submitAnswer"
          >
            Retry Scoring
          </button>
        </article>
      </div>

      <!-- Result panel -->
      <article v-if="result" class="result-panel">
        <div class="result-panel__score-row">
          <div class="score-display" :class="scoreClass">
            <span class="score-num">{{ result.score }}</span>
            <span class="score-denom">/10</span>
          </div>
        </div>

        <p class="result-panel__feedback">{{ result.feedback }}</p>

        <div class="form-footer">
          <button id="wa-dashboard-btn" class="primary-btn" @click="goToDashboard">
            Back to Dashboard
          </button>
          <button id="wa-new-btn" class="ghost-btn" @click="reset">New Question</button>
        </div>
      </article>
    </section>
  </StudyPageLayout>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getNotebooks, generateComprehensiveExam, scoreShortAnswer } from '../services/appApi.js'
import StudyPageLayout from '../components/StudyPageLayout.vue'

const route = useRoute()
const router = useRouter()

const notebooks = ref([])
const selectedNotebookID = ref('')
const startPage = ref(1)
const endPage = ref(10)
const loading = ref(false)
const scoring = ref(false)
const error = ref('')
const question = ref(null)
const userAnswer = ref('')
const result = ref(null)

const selectedNotebook = computed(() =>
  notebooks.value.find((n) => n.id === selectedNotebookID.value)
)

watch(selectedNotebookID, (newID) => {
  if (!newID) return
  const nb = notebooks.value.find((n) => n.id === newID)
  if (nb) {
    const minP = nb.start_page || 1
    const maxP = nb.end_page || nb.page_count || minP
    startPage.value = minP
    endPage.value = Math.min(minP + 5, maxP)
  }
})

const canGenerate = computed(
  () =>
    selectedNotebookID.value &&
    startPage.value > 0 &&
    endPage.value >= startPage.value &&
    !loading.value
)

const scoreClass = computed(() => {
  const s = result.value?.score ?? 0
  if (s >= 8) return 'score--great'
  if (s >= 5) return 'score--ok'
  return 'score--low'
})

onMounted(async () => {
  try {
    const res = await getNotebooks()
    notebooks.value = Array.isArray(res) ? res.filter((n) => !n.error) : []

    // Read optional query parameters from router (e.g. from Quiz completion)
    const { notebookID, startPage: queryStart, endPage: queryEnd, autoGenerate } = route.query
    if (notebookID) {
      selectedNotebookID.value = String(notebookID)
    }
    if (queryStart && Number(queryStart) > 0) {
      startPage.value = Number(queryStart)
    }
    if (queryEnd && Number(queryEnd) >= startPage.value) {
      endPage.value = Number(queryEnd)
    }

    if (autoGenerate === 'true' && selectedNotebookID.value) {
      await router.replace({ query: { ...route.query, autoGenerate: undefined } })
      await generate()
    }
  } catch {
    error.value = 'Failed to load notebooks.'
  }
})

function goToDashboard() {
  router.push('/dashboard')
}

async function generate() {
  error.value = ''
  question.value = null
  result.value = null
  userAnswer.value = ''
  loading.value = true
  try {
    const res = await generateComprehensiveExam(
      selectedNotebookID.value,
      startPage.value,
      endPage.value
    )
    if (res.error) {
      error.value = res.error
      return
    }
    question.value = {
      questionId: res.questionID,
      prompt: res.prompt,
      topicId: res.topicID,
      notebookId: res.notebook_id,
      startPage: res.start_page,
      endPage: res.end_page,
      llmTier: res.llm_tier,
      sourcePageStart: res.source_page_start,
      sourcePageEnd: res.source_page_end,
    }
  } catch (e) {
    error.value = e?.message ?? 'Exam generation failed.'
  } finally {
    loading.value = false
  }
}

async function submitAnswer() {
  if (!question.value || !userAnswer.value.trim()) return
  if (!question.value.questionId) {
    error.value = 'Invalid question: missing question ID'
    return
  }
  scoring.value = true
  try {
    const res = await scoreShortAnswer(question.value.questionId, userAnswer.value.trim())
    if (res.error) {
      error.value = res.error
      return
    }
    result.value = {
      questionId: res.question_id,
      prompt: res.prompt,
      score: res.score,
      feedback: res.feedback,
    }
  } catch (e) {
    error.value = e?.message ?? 'Scoring failed.'
  } finally {
    scoring.value = false
  }
}

function reset() {
  question.value = null
  result.value = null
  userAnswer.value = ''
  error.value = ''
}
</script>

<style scoped>
/* ── Toolbar ──────────────────────────────────── */
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

/* ── Tab content ──────────────────────────────── */
.tab-content {
  display: grid;
  gap: 16px;
  animation: fadeIn 0.18s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
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

.range-hint {
  display: inline-block;
  margin-left: 0.5rem;
  color: var(--color-accent, #6366f1);
  font-weight: 500;
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

/* Ghost input: suggestion of a border, no hard box */
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
  padding: 40px 24px;
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

.retry-generation-btn {
  margin-top: 12px;
  min-width: 120px;
}

/* ── Exam area ────────────────────────────────── */
.exam-area {
  display: grid;
  gap: 16px;
}

/* Question card: lowest surface to "pop" */
.question-card {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 24px;
  display: grid;
  gap: 12px;
}

.question-prompt {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface);
  line-height: 1.6;
}

.source-badge {
  display: inline-flex;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--muted-text);
  background: var(--surface-container-low);
  border-radius: 999px;
  padding: 4px 12px;
  width: fit-content;
}

/* STT Tip banner */
.stt-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--surface-container-low);
  border-radius: 12px;
  padding: 10px 14px;
  font-size: 13px;
  color: var(--muted-text);
}

.stt-tip__icon {
  font-size: 16px;
  flex-shrink: 0;
}

.stt-tip__text {
  line-height: 1.5;
}

.stt-tip__link {
  color: var(--primary);
  text-decoration: underline;
  font-weight: 600;
}

.kbd-badge {
  display: inline-block;
  padding: 1px 6px;
  font-size: 11px;
  font-family: inherit;
  font-weight: 700;
  color: var(--on-surface);
  background: var(--surface-container-highest, rgba(255, 255, 255, 0.1));
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
}

/* Answer field */
.answer-field {
  display: grid;
  gap: 6px;
}

.ghost-textarea {
  width: 100%;
  padding: 14px 16px;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  font: inherit;
  font-size: 15px;
  color: var(--on-surface);
  resize: vertical;
  line-height: 1.6;
  box-sizing: border-box;
  transition: border-color 0.15s ease;
  min-height: 160px;
}

.ghost-textarea:focus {
  outline: none;
  border-color: var(--primary);
}

.ghost-textarea:disabled {
  opacity: 0.5;
}

/* ── Form footer ──────────────────────────────── */
.form-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

/* ── Ghost secondary button ───────────────────── */
.ghost-btn {
  padding: 11px 20px;
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  background: transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  color: var(--muted-text);
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    color 0.15s ease;
}

.ghost-btn:hover:not(:disabled) {
  border-color: #ef4444;
  color: #dc2626;
}

.ghost-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
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

/* ── Result panel ─────────────────────────────── */
.result-panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 32px 24px;
  display: grid;
  gap: 20px;
}

.result-panel__score-row {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.score-display {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.score-num {
  font-family: 'Manrope', sans-serif;
  font-size: 48px;
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1;
}

.score-denom {
  font-size: 18px;
  color: var(--muted-text);
  font-weight: 500;
}

/* Score tonal colors */
.score--great .score-num {
  color: #16a34a;
}
.score--ok .score-num {
  color: #ea580c;
}
.score--low .score-num {
  color: #dc2626;
}

.result-panel__feedback {
  margin: 0;
  font-size: 15px;
  color: var(--on-surface);
  line-height: 1.65;
  max-width: 72ch;
}

/* ── Responsive ───────────────────────────────── */
@media (max-width: 720px) {
  .config-panel__row {
    flex-direction: column;
    align-items: flex-start;
  }

  .ghost-textarea {
    font-size: 14px;
  }

  .score-num {
    font-size: 40px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .tab-content {
    animation: none;
  }
}
</style>
