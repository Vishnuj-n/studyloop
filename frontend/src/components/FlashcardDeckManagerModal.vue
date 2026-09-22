<template>
  <Teleport to="body">
    <Transition name="deck-modal-fade">
      <div v-if="show" class="deck-modal-overlay" @click.self="close">
        <div class="deck-modal-container" role="dialog" aria-modal="true">
          <!-- Modal Header -->
          <div class="deck-modal-header">
            <div class="header-titles">
              <div class="header-eyebrow">
                <BaseIcon name="layers" size="14" />
                <span>Retention Console</span>
              </div>
              <h2 class="header-title">Flashcard Decks & FSRS Metrics</h2>
            </div>
            <div class="header-actions">
              <button
                type="button"
                class="icon-action-btn"
                title="Refresh decks and metrics"
                :disabled="loading"
                @click="loadOverview"
              >
                <BaseIcon name="refresh" size="16" :class="{ 'spin-anim': loading }" />
              </button>
              <button type="button" class="modal-close-btn" @click="close">
                <BaseIcon name="x" size="18" />
              </button>
            </div>
          </div>

          <!-- Modal Body -->
          <div class="deck-modal-body">
            <!-- Top Metric Cards -->
            <div class="metrics-grid">
              <div class="metric-card">
                <div class="metric-label">
                  <BaseIcon name="layers" size="13" />
                  <span>Total Cards</span>
                </div>
                <div class="metric-value">{{ metrics.total_cards || 0 }}</div>
                <div class="metric-sub">Across all notebooks</div>
              </div>

              <div class="metric-card due-highlight">
                <div class="metric-label">
                  <BaseIcon name="clock" size="13" />
                  <span>Due for Review</span>
                </div>
                <div class="metric-value text-amber">{{ metrics.due_today || 0 }}</div>
                <div class="metric-sub">Ready in queue</div>
              </div>

              <div class="metric-card">
                <div class="metric-label">
                  <BaseIcon name="pause" size="13" />
                  <span>Paused / Suspended</span>
                </div>
                <div class="metric-value text-muted">{{ metrics.suspended_cards || 0 }}</div>
                <div class="metric-sub">Excluded from study</div>
              </div>

              <div class="metric-card mature-highlight">
                <div class="metric-label">
                  <BaseIcon name="star" size="13" />
                  <span>Retention Mastery</span>
                </div>
                <div class="metric-value text-primary">{{ masteryPercent }}%</div>
                <div class="metric-sub">{{ metrics.mature_cards || 0 }} mature cards</div>
              </div>
            </div>

            <!-- FSRS Memory Stability Meter -->
            <div class="stability-section">
              <div class="stability-header">
                <span class="stability-title">FSRS Memory Maturity Distribution</span>
                <span class="stability-legend">
                  <span class="legend-chip new"><span class="dot"></span>New ({{ metrics.new_cards || 0 }})</span>
                  <span class="legend-chip learning"><span class="dot"></span>Learning ({{ metrics.learning_cards || 0 }})</span>
                  <span class="legend-chip young"><span class="dot"></span>Young ({{ metrics.young_cards || 0 }})</span>
                  <span class="legend-chip mature"><span class="dot"></span>Mature ({{ metrics.mature_cards || 0 }})</span>
                </span>
              </div>
              <div class="stability-bar-track">
                <div
                  v-if="pctNew > 0"
                  class="bar-slice bar-new"
                  :style="{ width: `${pctNew}%` }"
                  :title="`New: ${metrics.new_cards || 0} (${pctNew}%)`"
                ></div>
                <div
                  v-if="pctLearning > 0"
                  class="bar-slice bar-learning"
                  :style="{ width: `${pctLearning}%` }"
                  :title="`Learning: ${metrics.learning_cards || 0} (${pctLearning}%)`"
                ></div>
                <div
                  v-if="pctYoung > 0"
                  class="bar-slice bar-young"
                  :style="{ width: `${pctYoung}%` }"
                  :title="`Young: ${metrics.young_cards || 0} (${pctYoung}%)`"
                ></div>
                <div
                  v-if="pctMature > 0"
                  class="bar-slice bar-mature"
                  :style="{ width: `${pctMature}%` }"
                  :title="`Mature: ${metrics.mature_cards || 0} (${pctMature}%)`"
                ></div>
              </div>
            </div>

            <!-- Filter & Search Controls -->
            <div class="filter-toolbar">
              <div class="search-box">
                <BaseIcon name="search" size="15" class="search-icon" />
                <input
                  v-model="searchQuery"
                  type="text"
                  placeholder="Search cards by question or answer..."
                  class="search-input"
                />
                <button
                  v-if="searchQuery"
                  class="search-clear-btn"
                  type="button"
                  @click="searchQuery = ''"
                >
                  <BaseIcon name="x" size="13" />
                </button>
              </div>

              <div class="filter-chips">
                <button
                  type="button"
                  :class="['filter-chip', { active: statusFilter === 'all' }]"
                  @click="statusFilter = 'all'"
                >
                  All ({{ totalFilteredCards }})
                </button>
                <button
                  type="button"
                  :class="['filter-chip', { active: statusFilter === 'due' }]"
                  @click="statusFilter = 'due'"
                >
                  Due ({{ metrics.due_today || 0 }})
                </button>
                <button
                  type="button"
                  :class="['filter-chip', { active: statusFilter === 'active' }]"
                  @click="statusFilter = 'active'"
                >
                  Active ({{ metrics.active_cards || 0 }})
                </button>
                <button
                  type="button"
                  :class="['filter-chip', { active: statusFilter === 'suspended' }]"
                  @click="statusFilter = 'suspended'"
                >
                  Paused ({{ metrics.suspended_cards || 0 }})
                </button>
              </div>
            </div>

            <!-- Loading & Empty States -->
            <div v-if="loading && notebooks.length === 0" class="state-container">
              <div class="loading-spinner"></div>
              <p>Loading flashcard decks...</p>
            </div>

            <div v-else-if="filteredNotebooks.length === 0" class="state-container empty-state">
              <BaseIcon name="layers" size="36" class="empty-icon" />
              <h3>No Flashcards Found</h3>
              <p v-if="searchQuery || statusFilter !== 'all'">
                Try adjusting your search query or filter selection.
              </p>
              <p v-else>
                Upload textbooks or generate flashcards to start building your retention deck.
              </p>
            </div>

            <!-- Notebooks Accordion Tree -->
            <div v-else class="decks-list">
              <div
                v-for="nb in filteredNotebooks"
                :key="nb.notebook_id || 'standalone'"
                class="notebook-deck-card"
              >
                <!-- Notebook Header -->
                <div class="notebook-deck-header" @click="toggleNotebookExpand(nb.notebook_id)">
                  <div class="deck-title-group">
                    <button type="button" class="chevron-btn" aria-label="Toggle notebook expansion">
                      <BaseIcon
                        :name="isExpanded(nb.notebook_id) ? 'chevron-down' : 'chevron-right'"
                        size="16"
                      />
                    </button>
                    <BaseIcon name="book" size="16" class="nb-icon" />
                    <span class="nb-title">{{ nb.notebook_title || 'Standalone Flashcards' }}</span>
                    <span class="count-badge">{{ nb.visibleCardCount }} cards</span>
                    <span v-if="nb.dueCards > 0" class="due-badge">{{ nb.dueCards }} due</span>
                  </div>

                  <!-- Bulk Notebook Actions -->
                  <div class="deck-header-actions" @click.stop>
                    <button
                      type="button"
                      :class="['action-toggle-btn', nb.is_all_suspended ? 'resume-mode' : 'pause-mode']"
                      :disabled="pendingNotebookIds.has(nb.notebook_id || 'standalone')"
                      @click="handleToggleNotebook(nb)"
                    >
                      <BaseIcon :name="nb.is_all_suspended ? 'play' : 'pause'" size="13" />
                      <span>{{ nb.is_all_suspended ? 'Resume Notebook' : 'Pause Notebook' }}</span>
                    </button>
                  </div>
                </div>

                <!-- Topics & Cards Container -->
                <div v-if="isExpanded(nb.notebook_id)" class="notebook-deck-body">
                  <div
                    v-for="topic in nb.filteredTopics"
                    :key="topic.topic_id"
                    class="topic-group"
                  >
                    <div class="topic-group-header">
                      <BaseIcon name="book" size="13" class="topic-icon" />
                      <span class="topic-title">{{ topic.topic_title }}</span>
                      <span class="topic-count">{{ topic.cards.length }} cards</span>
                    </div>

                    <div class="cards-grid">
                      <div
                        v-for="card in topic.cards"
                        :key="card.id"
                        :class="['card-item-row', { 'is-suspended': card.suspended }]"
                      >
                        <!-- Card Content Snippets -->
                        <div class="card-item-content">
                          <div class="card-qa-row">
                            <span class="qa-tag front-tag">Q</span>
                            <span class="qa-text front-text">{{ card.prompt }}</span>
                          </div>
                          <div class="card-qa-row back-row">
                            <span class="qa-tag back-tag">A</span>
                            <span class="qa-text back-text">{{ card.answer }}</span>
                          </div>

                          <!-- Card Meta Badges -->
                          <div class="card-meta-chips">
                            <span v-if="card.suspended" class="meta-chip chip-paused">
                              <span class="dot"></span> Paused
                            </span>
                            <span v-else-if="isCardDue(card)" class="meta-chip chip-due">
                              <span class="dot"></span> Due Now
                            </span>
                            <span v-else class="meta-chip chip-interval">
                              Next: {{ formatDueDate(card.due_at) }}
                            </span>

                            <span class="meta-chip chip-stat">
                              Interval: {{ formatInterval(card.stability) }}
                            </span>
                            <span class="meta-chip chip-stat">
                              Reps: {{ card.reps || 0 }}
                            </span>
                          </div>
                        </div>

                        <!-- Card Action Buttons -->
                        <div class="card-item-actions">
                          <button
                            type="button"
                            :class="['card-action-btn', card.suspended ? 'resume-btn' : 'suspend-btn']"
                            :title="card.suspended ? 'Resume card in queue' : 'Pause / Suspend card'"
                            :disabled="pendingCardIds.has(card.id)"
                            @click="handleToggleCard(card)"
                          >
                            <BaseIcon :name="card.suspended ? 'play' : 'pause'" size="13" />
                            <span>{{ card.suspended ? 'Resume' : 'Pause' }}</span>
                          </button>

                          <button
                            type="button"
                            class="card-delete-btn"
                            title="Delete flashcard permanently"
                            :disabled="pendingCardIds.has(card.id)"
                            @click="confirmDeleteCard(card)"
                          >
                            <BaseIcon name="trash" size="14" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import {
  getFlashcardsDeckOverview,
  toggleCardSuspension,
  toggleNotebookCardsSuspension,
  deleteFlashcard,
} from '../services/appApi.js'
import { useDialog } from '../composables/useDialog'
import { useToast } from '../composables/useToast'

defineOptions({
  name: 'FlashcardDeckManagerModal',
})

const props = defineProps({
  show: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close', 'updated'])

const { showConfirm } = useDialog()
const { showSuccess, showError } = useToast()

const loading = ref(false)
const pendingCardIds = ref(new Set())
const pendingNotebookIds = ref(new Set())
const searchQuery = ref('')
const statusFilter = ref('all') // 'all' | 'due' | 'active' | 'suspended'
const expandedNotebooks = ref(new Set())

const metrics = ref({
  total_cards: 0,
  due_today: 0,
  active_cards: 0,
  suspended_cards: 0,
  new_cards: 0,
  learning_cards: 0,
  young_cards: 0,
  mature_cards: 0,
})

const notebooks = ref([])

function close() {
  emit('close')
}

// Unified due date checker
function isCardDue(card, nowSec = Math.floor(Date.now() / 1000)) {
  if (!card || card.suspended) return false
  return (card.due_at || 0) <= nowSec
}

// FSRS Breakdown Percentages
const masteryPercent = computed(() => {
  if (!metrics.value.total_cards || metrics.value.total_cards === 0) return 0
  return Math.round(((metrics.value.mature_cards || 0) / metrics.value.total_cards) * 100)
})

const pctNew = computed(() => {
  if (!metrics.value.total_cards) return 0
  return Math.round(((metrics.value.new_cards || 0) / metrics.value.total_cards) * 100)
})

const pctLearning = computed(() => {
  if (!metrics.value.total_cards) return 0
  return Math.round(((metrics.value.learning_cards || 0) / metrics.value.total_cards) * 100)
})

const pctYoung = computed(() => {
  if (!metrics.value.total_cards) return 0
  return Math.round(((metrics.value.young_cards || 0) / metrics.value.total_cards) * 100)
})

const pctMature = computed(() => {
  if (!metrics.value.total_cards) return 0
  return Math.round(((metrics.value.mature_cards || 0) / metrics.value.total_cards) * 100)
})

function formatDueDate(dueAt) {
  if (!dueAt || dueAt === 0) return 'New'
  const nowUnix = Math.floor(Date.now() / 1000)
  const diffSec = dueAt - nowUnix
  if (diffSec <= 0) return 'Due now'
  const diffDays = Math.ceil(diffSec / (24 * 3600))
  if (diffDays === 1) return 'Tomorrow'
  return `In ${diffDays}d`
}

function formatInterval(stability) {
  if (!stability || stability <= 0) return '--'
  if (stability < 1) return `${Math.round(stability * 24)}h`
  if (stability < 30) return `${Math.round(stability)}d`
  return `${(stability / 30).toFixed(1)}mo`
}

function isExpanded(nbID) {
  return expandedNotebooks.value.has(nbID || 'standalone')
}

function toggleNotebookExpand(nbID) {
  const key = nbID || 'standalone'
  const nextSet = new Set(expandedNotebooks.value)
  if (nextSet.has(key)) {
    nextSet.delete(key)
  } else {
    nextSet.add(key)
  }
  expandedNotebooks.value = nextSet
}

// Filtered notebook hierarchy
const filteredNotebooks = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  const filter = statusFilter.value
  const nowUnix = Math.floor(Date.now() / 1000)

  return notebooks.value
    .map((nb) => {
      const filteredTopics = nb.topics
        .map((top) => {
          const matchedCards = top.cards.filter((c) => {
            // Status filter
            if (filter === 'due' && !isCardDue(c, nowUnix)) return false
            if (filter === 'active' && c.suspended) return false
            if (filter === 'suspended' && !c.suspended) return false

            // Search query
            if (query) {
              const textMatch =
                c.prompt.toLowerCase().includes(query) ||
                c.answer.toLowerCase().includes(query) ||
                (c.notebook_title && c.notebook_title.toLowerCase().includes(query)) ||
                (c.topic_title && c.topic_title.toLowerCase().includes(query))
              if (!textMatch) return false
            }

            return true
          })

          return {
            ...top,
            cards: matchedCards,
          }
        })
        .filter((top) => top.cards.length > 0)

      const visibleCardCount = filteredTopics.reduce((acc, t) => acc + t.cards.length, 0)
      const dueCards = filteredTopics.reduce(
        (acc, t) => acc + t.cards.filter((c) => isCardDue(c, nowUnix)).length,
        0
      )

      return {
        ...nb,
        filteredTopics,
        visibleCardCount,
        dueCards,
      }
    })
    .filter((nb) => nb.visibleCardCount > 0)
})

const totalFilteredCards = computed(() => {
  return filteredNotebooks.value.reduce((acc, nb) => acc + nb.visibleCardCount, 0)
})

async function loadOverview() {
  if (loading.value) return
  loading.value = true
  try {
    const res = await getFlashcardsDeckOverview()
    if (res && res.metrics) {
      metrics.value = res.metrics
      notebooks.value = res.notebooks || []
      // Auto-expand first 2 notebooks
      if (expandedNotebooks.value.size === 0 && notebooks.value.length > 0) {
        const initialSet = new Set()
        notebooks.value.slice(0, 2).forEach((nb) => {
          initialSet.add(nb.notebook_id || 'standalone')
        })
        expandedNotebooks.value = initialSet
      }
    }
  } catch (err) {
    showError(err?.message || 'Failed to load flashcard decks')
  } finally {
    loading.value = false
  }
}

async function handleToggleCard(card) {
  if (pendingCardIds.value.has(card.id)) return
  pendingCardIds.value = new Set(pendingCardIds.value).add(card.id)

  const newSuspended = !card.suspended
  try {
    const res = await toggleCardSuspension(card.id, newSuspended)
    if (res && res.ok) {
      card.suspended = newSuspended
      showSuccess(newSuspended ? 'Card paused' : 'Card resumed')
      // Refetch for reliable metrics consistency
      await loadOverview()
      emit('updated')
    }
  } catch (err) {
    showError(err?.message || 'Failed to toggle card state')
  } finally {
    const nextSet = new Set(pendingCardIds.value)
    nextSet.delete(card.id)
    pendingCardIds.value = nextSet
  }
}

async function handleToggleNotebook(nb) {
  const nbKey = nb.notebook_id || 'standalone'
  if (pendingNotebookIds.value.has(nbKey)) return

  const newSuspended = !nb.is_all_suspended
  const confirmed = await showConfirm({
    title: newSuspended ? 'Pause All Cards in Notebook?' : 'Resume All Cards in Notebook?',
    message: newSuspended
      ? `This will pause all flashcards in "${nb.notebook_title}" and exclude them from your daily review queue.`
      : `This will resume all flashcards in "${nb.notebook_title}" for retention review.`,
    confirmText: newSuspended ? 'Pause Notebook' : 'Resume Notebook',
    type: newSuspended ? 'warning' : 'info',
  })
  if (!confirmed) return

  pendingNotebookIds.value = new Set(pendingNotebookIds.value).add(nbKey)
  try {
    const res = await toggleNotebookCardsSuspension(nb.notebook_id, newSuspended)
    if (res && res.ok) {
      showSuccess(newSuspended ? 'Notebook cards paused' : 'Notebook cards resumed')
      await loadOverview()
      emit('updated')
    }
  } catch (err) {
    showError(err?.message || 'Failed to toggle notebook suspension')
  } finally {
    const nextSet = new Set(pendingNotebookIds.value)
    nextSet.delete(nbKey)
    pendingNotebookIds.value = nextSet
  }
}

async function confirmDeleteCard(card) {
  if (pendingCardIds.value.has(card.id)) return

  const confirmed = await showConfirm({
    title: 'Delete Flashcard?',
    message: `Are you sure you want to delete this flashcard? This action cannot be undone.\n\n"${card.prompt.slice(0, 80)}${card.prompt.length > 80 ? '...' : ''}"`,
    confirmText: 'Delete Card',
    type: 'danger',
  })
  if (!confirmed) return

  pendingCardIds.value = new Set(pendingCardIds.value).add(card.id)
  try {
    const res = await deleteFlashcard(card.id)
    if (res && res.ok) {
      showSuccess('Flashcard deleted')
      await loadOverview()
      emit('updated')
    }
  } catch (err) {
    showError(err?.message || 'Failed to delete card')
  } finally {
    const nextSet = new Set(pendingCardIds.value)
    nextSet.delete(card.id)
    pendingCardIds.value = nextSet
  }
}

// Only trigger load on show change (prevents double load)
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      loadOverview()
    }
  },
  { immediate: true }
)

function handleGlobalKeydown(e) {
  if (!props.show) return
  if (e.key === 'Escape') {
    close()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<style scoped>
.deck-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 15, 0.78);
  backdrop-filter: blur(8px);
  padding: 24px;
}

.deck-modal-container {
  width: 100%;
  max-width: 980px;
  height: 90vh;
  max-height: 840px;
  background: var(--surface-container-lowest, #141617);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  box-shadow: 0 24px 48px -12px rgba(0, 0, 0, 0.45);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: var(--on-surface);
}

/* Header */
.deck-modal-header {
  padding: 18px 24px;
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--surface-container-low, #1d2021);
}

.header-titles {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.header-eyebrow {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--primary);
}

.header-title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--on-surface);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.icon-action-btn,
.modal-close-btn {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.icon-action-btn:hover,
.modal-close-btn:hover {
  background: var(--surface-container);
  color: var(--on-surface);
  border-color: var(--outline-variant);
}

.icon-action-btn:active,
.modal-close-btn:active {
  transform: scale(0.95);
}

.spin-anim {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Body */
.deck-modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.metric-card {
  padding: 14px 16px;
  background: var(--surface-container, #282828);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-card.due-highlight {
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.05);
}

.metric-card.mature-highlight {
  border-color: rgba(0, 91, 193, 0.3);
  background: rgba(0, 91, 193, 0.05);
}

.metric-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--muted-text);
}

.metric-value {
  font-size: 1.6rem;
  font-weight: 700;
  line-height: 1.1;
  color: var(--on-surface);
}

.metric-sub {
  font-size: 11px;
  color: var(--muted-text);
}

.text-amber {
  color: #f59e0b;
}

.text-primary {
  color: var(--primary);
}

.text-muted {
  color: var(--muted-text);
}

/* Stability Meter */
.stability-section {
  padding: 14px 16px;
  background: var(--surface-container-low, #1d2021);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.stability-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.stability-title {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted-text);
}

.stability-legend {
  display: flex;
  align-items: center;
  gap: 12px;
}

.legend-chip {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 600;
  color: var(--muted-text);
}

.legend-chip .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.legend-chip.new .dot { background: #94a3b8; }
.legend-chip.learning .dot { background: #f59e0b; }
.legend-chip.young .dot { background: #3b82f6; }
.legend-chip.mature .dot { background: #10b981; }

.stability-bar-track {
  height: 10px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 6px;
  overflow: hidden;
  display: flex;
}

.bar-slice {
  height: 100%;
  transition: width 0.3s ease;
}

.bar-new { background: #94a3b8; }
.bar-learning { background: #f59e0b; }
.bar-young { background: #3b82f6; }
.bar-mature { background: #10b981; }

/* Filter Toolbar */
.filter-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.search-box {
  flex: 1;
  min-width: 260px;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--muted-text);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 9px 34px 9px 34px;
  background: var(--surface-container, #282828);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  color: var(--on-surface);
  font-size: 13px;
  outline: none;
  transition: all 0.15s ease;
}

.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(0, 91, 193, 0.15);
}

.search-clear-btn {
  position: absolute;
  right: 10px;
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  padding: 2px;
}

.filter-chips {
  display: flex;
  gap: 6px;
}

.filter-chip {
  padding: 7px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  background: var(--surface-container, #282828);
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-chip:hover {
  color: var(--on-surface);
}

.filter-chip.active {
  background: var(--primary);
  color: #ffffff;
  border-color: var(--primary);
}

/* States */
.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  color: var(--muted-text);
  gap: 12px;
  text-align: center;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.empty-icon {
  opacity: 0.4;
  margin-bottom: 4px;
}

.empty-state h3 {
  margin: 0;
  color: var(--on-surface);
}

.empty-state p {
  margin: 0;
  font-size: 13px;
}

/* Decks List */
.decks-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.notebook-deck-card {
  background: var(--surface-container-low, #1d2021);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  overflow: hidden;
}

.notebook-deck-header {
  padding: 14px 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  background: var(--surface-container, #282828);
  user-select: none;
  transition: background 0.15s ease;
}

.notebook-deck-header:hover {
  background: var(--surface-container-highest, #3c3836);
}

.deck-title-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.chevron-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 0;
}

.nb-icon {
  color: var(--primary);
}

.nb-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface);
}

.count-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 12px;
  background: var(--surface-container-lowest, #141617);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.due-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 12px;
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.action-toggle-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--outline-variant);
  transition: all 0.15s ease;
}

.action-toggle-btn.pause-mode {
  background: transparent;
  color: var(--muted-text);
}

.action-toggle-btn.pause-mode:hover {
  background: rgba(239, 68, 68, 0.1);
  color: #f87171;
  border-color: rgba(239, 68, 68, 0.3);
}

.action-toggle-btn.resume-mode {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

.action-toggle-btn.resume-mode:hover {
  background: rgba(16, 185, 129, 0.25);
}

.action-toggle-btn:active {
  transform: scale(0.95);
}

/* Notebook Deck Body */
.notebook-deck-body {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-top: 1px solid var(--outline-variant);
}

.topic-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.topic-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
}

.topic-icon {
  color: var(--muted-text);
}

.topic-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--muted-text);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.topic-count {
  font-size: 11px;
  color: var(--muted-text);
  opacity: 0.7;
}

.cards-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.card-item-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  background: var(--surface-container-lowest, #141617);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  transition: all 0.15s ease;
}

.card-item-row:hover {
  border-color: rgba(255, 255, 255, 0.15);
}

.card-item-row.is-suspended {
  opacity: 0.65;
  background: rgba(0, 0, 0, 0.15);
}

.card-item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.card-qa-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.qa-tag {
  font-size: 10px;
  font-weight: 800;
  padding: 1px 5px;
  border-radius: 4px;
  flex-shrink: 0;
}

.front-tag {
  background: rgba(0, 91, 193, 0.15);
  color: var(--primary);
}

.back-tag {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.qa-text {
  font-size: 13px;
  line-height: 1.4;
  word-break: break-word;
}

.front-text {
  font-weight: 600;
  color: var(--on-surface);
}

.back-text {
  font-weight: 400;
  color: var(--muted-text);
}

.card-meta-chips {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 2px;
}

.meta-chip {
  font-size: 11px;
  padding: 2px 7px;
  border-radius: 6px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
}

.meta-chip .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.chip-paused {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}
.chip-paused .dot { background: #f59e0b; }

.chip-due {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
}
.chip-due .dot { background: #f87171; }

.chip-interval {
  background: var(--surface-container, #282828);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.chip-stat {
  background: var(--surface-container, #282828);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.card-item-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.card-action-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--outline-variant);
  transition: all 0.15s ease;
}

.card-action-btn.suspend-btn {
  background: transparent;
  color: var(--muted-text);
}

.card-action-btn.suspend-btn:hover {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border-color: rgba(245, 158, 11, 0.3);
}

.card-action-btn.resume-btn {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

.card-action-btn.resume-btn:hover {
  background: rgba(16, 185, 129, 0.25);
}

.card-delete-btn {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.card-delete-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
  border-color: rgba(239, 68, 68, 0.3);
}

.card-action-btn:active,
.card-delete-btn:active {
  transform: scale(0.95);
}

/* Modal Transition */
.deck-modal-fade-enter-active,
.deck-modal-fade-leave-active {
  transition: opacity 0.2s ease;
}

.deck-modal-fade-enter-from,
.deck-modal-fade-leave-to {
  opacity: 0;
}

.deck-modal-fade-enter-active .deck-modal-container,
.deck-modal-fade-leave-active .deck-modal-container {
  transition: transform 0.2s ease;
}

.deck-modal-fade-enter-from .deck-modal-container {
  transform: scale(0.96) translateY(8px);
}

.deck-modal-fade-leave-to .deck-modal-container {
  transform: scale(0.98) translateY(4px);
}

@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .deck-modal-overlay {
    padding: 12px;
  }
}
</style>
