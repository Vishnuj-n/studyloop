<template>
  <Teleport to="body">
    <Transition name="deck-modal-fade">
      <div v-if="show" class="deck-modal-overlay" @click.self="close">
        <div class="deck-modal-container" role="dialog" aria-modal="true" aria-labelledby="fc-deck-manager-title">
          <!-- Modal Header -->
          <div class="deck-modal-header">
            <div class="header-titles">
              <div class="header-eyebrow">
                <BaseIcon name="layers" size="13" />
                <span>Retention Console</span>
              </div>
              <h2 id="fc-deck-manager-title" class="header-title">Flashcard Library & FSRS Metrics</h2>
            </div>
            <div class="header-actions">
              <button
                type="button"
                class="copy-all-btn"
                aria-label="Copy all filtered flashcards as Markdown"
                :title="copiedAll ? 'Copied to Clipboard!' : 'Copy all filtered flashcards as Markdown'"
                :disabled="loading || filteredNotebooks.length === 0"
                @click="copyAllFlashcards"
              >
                <BaseIcon :name="copiedAll ? 'check' : 'copy'" size="13" />
                <span>{{ copiedAll ? 'Copied!' : 'Copy All Cards' }}</span>
              </button>
              <button
                type="button"
                class="icon-action-btn"
                aria-label="Refresh decks and metrics"
                title="Refresh decks and metrics"
                :disabled="loading"
                @click="loadOverview"
              >
                <BaseIcon name="refresh" size="15" :class="{ 'spin-anim': loading }" />
              </button>
              <button
                type="button"
                class="modal-close-btn"
                aria-label="Close retention modal"
                title="Close"
                @click="close"
              >
                <BaseIcon name="x" size="16" />
              </button>
            </div>
          </div>

          <!-- Modal Body -->
          <div class="deck-modal-body">
            <!-- Compact Executive Summary Strip -->
            <div class="executive-summary-strip">
              <div class="stats-row">
                <div class="stat-pill">
                  <span class="stat-label">Total Cards</span>
                  <span class="stat-val">{{ metrics.total_cards || 0 }}</span>
                </div>
                <div class="stat-pill due-pill">
                  <span class="stat-label">Due Today</span>
                  <span class="stat-val text-amber">{{ metrics.due_today || 0 }}</span>
                </div>
                <div class="stat-pill">
                  <span class="stat-label">Paused</span>
                  <span class="stat-val text-muted">{{ metrics.suspended_cards || 0 }}</span>
                </div>
                <div class="stat-pill mastery-pill">
                  <span class="stat-label">Mastery Rate</span>
                  <span class="stat-val text-emerald">{{ masteryPercent }}%</span>
                  <span class="stat-sub">({{ metrics.mature_cards || 0 }} mature)</span>
                </div>
              </div>

              <!-- Compact Maturity Progress Ribbon -->
              <div class="maturity-ribbon">
                <div class="maturity-bar-track">
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
                <div class="maturity-legend">
                  <span class="legend-item"><span class="dot dot-new"></span> New ({{ metrics.new_cards || 0 }})</span>
                  <span class="legend-item"><span class="dot dot-learning"></span> Learning ({{ metrics.learning_cards || 0 }})</span>
                  <span class="legend-item"><span class="dot dot-young"></span> Young ({{ metrics.young_cards || 0 }})</span>
                  <span class="legend-item"><span class="dot dot-mature"></span> Mature ({{ metrics.mature_cards || 0 }})</span>
                </div>
              </div>
            </div>

            <!-- Filter & Search Controls -->
            <div class="filter-toolbar">
              <div class="search-box">
                <BaseIcon name="search" size="14" class="search-icon" />
                <input
                  v-model="searchQuery"
                  type="text"
                  aria-label="Search flashcard questions or answers"
                  placeholder="Search questions or answers..."
                  class="search-input"
                />
                <button
                  v-if="searchQuery"
                  class="search-clear-btn"
                  type="button"
                  title="Clear search"
                  @click="searchQuery = ''"
                >
                  <BaseIcon name="x" size="12" />
                </button>
              </div>

              <div class="filter-group">
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

                <button
                  type="button"
                  class="toggle-all-btn"
                  :title="areAllNotebooksExpanded ? 'Collapse All Decks' : 'Expand All Decks'"
                  @click="toggleAllNotebooks"
                >
                  <BaseIcon :name="areAllNotebooksExpanded ? 'chevron-up' : 'chevron-down'" size="13" />
                  <span>{{ areAllNotebooksExpanded ? 'Collapse All' : 'Expand All' }}</span>
                </button>
              </div>
            </div>

            <!-- Loading & Empty States -->
            <div v-if="loading && notebooks.length === 0" class="state-container">
              <div class="loading-spinner"></div>
              <p>Loading card collection...</p>
            </div>

            <div v-else-if="filteredNotebooks.length === 0" class="state-container empty-state">
              <BaseIcon name="layers" size="32" class="empty-icon" />
              <h3>No Flashcards Found</h3>
              <p v-if="searchQuery || statusFilter !== 'all'">
                Try adjusting your search query or filter criteria.
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
                        size="15"
                      />
                    </button>
                    <BaseIcon name="book" size="15" class="nb-icon" />
                    <span class="nb-title">{{ nb.notebook_title || 'Standalone Flashcards' }}</span>
                    <span class="count-badge">{{ nb.visibleCardCount }} cards</span>
                    <span v-if="nb.dueCards > 0" class="due-badge">{{ nb.dueCards }} due</span>
                  </div>

                  <!-- Bulk Notebook Actions -->
                  <div class="deck-header-actions" @click.stop>
                    <button
                      type="button"
                      class="action-toggle-btn copy-cards-btn"
                      :title="copiedNotebookId === (nb.notebook_id || 'standalone') ? 'Copied to Clipboard!' : 'Copy this deck as Markdown'"
                      @click="copyNotebookFlashcards(nb)"
                    >
                      <BaseIcon :name="copiedNotebookId === (nb.notebook_id || 'standalone') ? 'check' : 'copy'" size="12" />
                      <span>{{ copiedNotebookId === (nb.notebook_id || 'standalone') ? 'Copied!' : 'Copy Cards' }}</span>
                    </button>
                    <button
                      v-if="nb.notebook_id"
                      type="button"
                      :class="['action-toggle-btn', nb.is_all_suspended ? 'resume-mode' : 'pause-mode']"
                      :disabled="pendingNotebookIds.has(nb.notebook_id)"
                      @click="handleToggleNotebook(nb)"
                    >
                      <BaseIcon :name="nb.is_all_suspended ? 'play' : 'pause'" size="12" />
                      <span>{{ nb.is_all_suspended ? 'Resume Deck' : 'Pause Deck' }}</span>
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
                    <!-- Formatted Clean Topic Header -->
                    <div class="topic-group-header">
                      <BaseIcon name="folder" size="12" class="topic-icon" />
                      <span class="topic-title">{{ formatTopicTitle(topic.topic_title) }}</span>
                      <span class="topic-count">({{ topic.cards.length }})</span>
                    </div>

                    <!-- Scannable Linear-Style Cards List -->
                    <div class="cards-list">
                      <div
                        v-for="card in topic.cards"
                        :key="card.id"
                        :class="[
                          'card-row-item',
                          {
                            'is-suspended': card.suspended,
                            'is-expanded': isCardExpanded(card.id),
                          }
                        ]"
                      >
                        <!-- Top Summary Row (Scannable / Click to Expand) -->
                        <div
                          class="card-row-main"
                          role="button"
                          tabindex="0"
                          :aria-expanded="isCardExpanded(card.id)"
                          @click="toggleCardExpand(card.id)"
                          @keydown.enter.self="toggleCardExpand(card.id)"
                          @keydown.space.prevent.self="toggleCardExpand(card.id)"
                        >
                          <div class="card-prompt-col" :title="card.prompt">
                            <span class="qa-indicator">Q</span>
                            <span class="card-prompt-text">{{ card.prompt }}</span>
                          </div>

                          <div class="card-row-meta" @click.stop>
                            <!-- Due / Interval / Suspended Pill -->
                            <span v-if="card.suspended" class="meta-pill pill-paused">
                              <span class="dot"></span> Paused
                            </span>
                            <span v-else-if="isCardDue(card)" class="meta-pill pill-due">
                              <span class="dot"></span> Due Now
                            </span>
                            <span v-else class="meta-pill pill-neutral">
                              {{ formatDueDate(card.due_at) }}
                            </span>

                            <span class="meta-pill pill-stability" title="Estimated Memory Interval">
                              {{ formatInterval(card.stability) }}
                            </span>

                            <!-- Pause / Resume Quick Action -->
                            <button
                              type="button"
                              :class="['card-pause-btn', card.suspended ? 'resume-btn' : 'suspend-btn']"
                              :title="card.suspended ? 'Resume card into study queue' : 'Pause / Suspend card from queue'"
                              :disabled="pendingCardIds.has(card.id)"
                              @click="handleToggleCard(card)"
                            >
                              <BaseIcon :name="card.suspended ? 'play' : 'pause'" size="12" />
                              <span>{{ card.suspended ? 'Resume' : 'Pause' }}</span>
                            </button>

                            <!-- Expand Chevron -->
                            <button
                              type="button"
                              class="row-expand-chevron"
                              :aria-label="isCardExpanded(card.id) ? 'Collapse details' : 'Expand details'"
                              @click.stop="toggleCardExpand(card.id)"
                            >
                              <BaseIcon
                                :name="isCardExpanded(card.id) ? 'chevron-up' : 'chevron-down'"
                                size="14"
                              />
                            </button>
                          </div>
                        </div>

                        <!-- Expanded Card Details Panel -->
                        <div v-if="isCardExpanded(card.id)" class="card-expanded-panel">
                          <!-- Inline Edit Mode -->
                          <div v-if="editingCardId === card.id" class="card-edit-container">
                            <div class="edit-field-group">
                              <label class="edit-field-label">
                                <span class="qa-indicator">Q</span>
                                <span>Question (Prompt)</span>
                              </label>
                              <textarea
                                v-model="editPrompt"
                                class="edit-field-textarea"
                                rows="3"
                                placeholder="Enter flashcard question..."
                                :disabled="savingEdit"
                              ></textarea>
                            </div>

                            <div class="edit-field-group">
                              <label class="edit-field-label">
                                <span class="qa-indicator ans-ind">A</span>
                                <span>Answer</span>
                              </label>
                              <textarea
                                v-model="editAnswer"
                                class="edit-field-textarea"
                                rows="4"
                                placeholder="Enter flashcard answer..."
                                :disabled="savingEdit"
                              ></textarea>
                            </div>

                            <div class="edit-form-actions">
                              <button
                                type="button"
                                class="edit-cancel-btn"
                                :disabled="savingEdit"
                                @click="cancelEditingCard"
                              >
                                Cancel
                              </button>
                              <button
                                type="button"
                                class="edit-save-btn"
                                :disabled="savingEdit"
                                @click="saveCardEdit(card)"
                              >
                                <BaseIcon v-if="savingEdit" name="refresh" size="12" class="spin-anim" />
                                <BaseIcon v-else name="check" size="12" />
                                <span>{{ savingEdit ? 'Saving...' : 'Save Changes' }}</span>
                              </button>
                            </div>
                          </div>

                          <!-- Read-only View Mode -->
                          <template v-else>
                            <!-- Full Answer -->
                            <div class="expanded-answer-box">
                              <div class="answer-badge-label">
                                <span class="qa-indicator ans-ind">A</span>
                                <span>Answer</span>
                              </div>
                              <div class="answer-content-text">{{ card.answer }}</div>
                            </div>

                            <!-- FSRS Telemetry & Safe Actions Footer -->
                            <div class="expanded-footer">
                              <div class="fsrs-stats-group">
                                <span class="fsrs-stat">
                                  <strong>Reps:</strong> {{ card.reps || 0 }}
                                </span>
                                <span class="fsrs-stat">
                                  <strong>Interval:</strong> {{ formatInterval(card.stability) }}
                                </span>
                                <span class="fsrs-stat">
                                  <strong>Next Due:</strong> {{ formatDetailedDue(card.due_at) }}
                                </span>
                                <span v-if="card.lapses !== undefined" class="fsrs-stat">
                                  <strong>Lapses:</strong> {{ card.lapses }}
                                </span>
                              </div>

                              <div class="expanded-actions">
                                <button
                                  type="button"
                                  class="guarded-edit-btn"
                                  title="Edit card prompt and answer"
                                  :disabled="pendingCardIds.has(card.id)"
                                  @click="startEditingCard(card)"
                                >
                                  <BaseIcon name="edit" size="12" />
                                  <span>Edit Card</span>
                                </button>
                                <button
                                  type="button"
                                  class="guarded-delete-btn"
                                  title="Permanently delete this card"
                                  :disabled="pendingCardIds.has(card.id)"
                                  @click="confirmDeleteCard(card)"
                                >
                                  <BaseIcon name="trash" size="12" />
                                  <span>Delete Card</span>
                                </button>
                              </div>
                            </div>
                          </template>
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
  updateFlashcardContent,
} from '../services/appApi.js'
import { copyTextToClipboard } from '../utils/clipboard'
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

const { confirm: showConfirm } = useDialog()
const { showNotice, showError } = useToast()

const loading = ref(false)
const pendingCardIds = ref(new Set())
const pendingNotebookIds = ref(new Set())
const searchQuery = ref('')
const statusFilter = ref('all') // 'all' | 'due' | 'active' | 'suspended'
const expandedNotebooks = ref(new Set())
const expandedCardIds = ref(new Set())
const copiedNotebookId = ref(null)
const copiedAll = ref(false)
let copyResetTimer = null

// Inline Edit State
const editingCardId = ref(null)
const editPrompt = ref('')
const editAnswer = ref('')
const savingEdit = ref(false)

function startEditingCard(card) {
  editingCardId.value = card.id
  editPrompt.value = card.prompt || ''
  editAnswer.value = card.answer || ''
  if (!expandedCardIds.value.has(card.id)) {
    expandedCardIds.value = new Set(expandedCardIds.value).add(card.id)
  }
}

function cancelEditingCard() {
  editingCardId.value = null
  editPrompt.value = ''
  editAnswer.value = ''
}

async function saveCardEdit(card) {
  const trimmedPrompt = editPrompt.value.trim()
  const trimmedAnswer = editAnswer.value.trim()

  if (!trimmedPrompt) {
    showError('Card question cannot be empty')
    return
  }
  if (!trimmedAnswer) {
    showError('Card answer cannot be empty')
    return
  }

  savingEdit.value = true
  try {
    const res = await updateFlashcardContent(card.id, trimmedPrompt, trimmedAnswer)
    if (res && res.error) {
      showError(res.error)
      return
    }
    if (res && res.ok) {
      card.prompt = trimmedPrompt
      card.answer = trimmedAnswer
      editingCardId.value = null
      showNotice('Flashcard updated')
      emit('updated')
    }
  } catch (err) {
    showError(err?.message || 'Failed to update flashcard')
  } finally {
    savingEdit.value = false
  }
}

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

// Format ugly raw topic slugs like NB-F353DC91-C33B-4CBD-ADD4-5B3C0C3E9E52-CH-04-ENCODING-AND-EVOLUTI
function formatWord(w) {
  const upper = w.toUpperCase()
  // Keep common study acronyms or uppercase terms intact
  const acronyms = new Set(['DNA', 'RNA', 'API', 'CPU', 'GPU', 'RAM', 'HTTP', 'HTTPS', 'SQL', 'CSS', 'HTML', 'FSRS', 'AI', 'ML', 'REST'])
  if (acronyms.has(upper)) return upper
  return w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()
}

function formatTopicTitle(rawTitle) {
  if (!rawTitle) return 'General'
  let cleaned = rawTitle.replace(/^NB-[A-F0-9-]+-?/i, '')
  const chMatch = cleaned.match(/^CH-(\d+)-(.*)$/i)
  if (chMatch) {
    const chNum = parseInt(chMatch[1], 10)
    const rest = chMatch[2]
      .split(/[-_]+/)
      .filter(Boolean)
      .map(formatWord)
      .join(' ')
    return `Chapter ${chNum}: ${rest}`
  }
  if (cleaned.includes('-') || cleaned.includes('_')) {
    return cleaned
      .split(/[-_]+/)
      .filter(Boolean)
      .map(formatWord)
      .join(' ')
  }
  return cleaned
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

function formatDetailedDue(dueAt) {
  if (!dueAt || dueAt === 0) return 'Immediate (New)'
  const date = new Date(dueAt * 1000)
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

function formatInterval(stability) {
  if (!stability || stability <= 0) return '--'
  if (stability < 1) return `${Math.round(stability * 24)}h`
  if (stability < 30) return `${Math.round(stability)}d`
  return `${(stability / 30).toFixed(1)}mo`
}

function buildNotebookMarkdown(notebook) {
  const nbTitle = notebook.notebook_title || 'Standalone Flashcards'
  const lines = [`# ${nbTitle} — Flashcards`, '']
  const topics = notebook.filteredTopics || notebook.topics || []

  for (const topic of topics) {
    if (!topic.cards || topic.cards.length === 0) continue
    const topicTitle = formatTopicTitle(topic.topic_title)
    lines.push(`## Topic: ${topicTitle}`)
    lines.push('')
    for (const card of topic.cards) {
      lines.push(`- **Q:** ${card.prompt}`)
      lines.push(`  **A:** ${card.answer}`)
      lines.push('')
    }
  }
  return lines.join('\n')
}

async function copyNotebookFlashcards(notebook) {
  if (!notebook) return
  const key = notebook.notebook_id || 'standalone'
  const markdown = buildNotebookMarkdown(notebook)
  if (!markdown.trim()) {
    showError('No flashcards available to copy in this deck.')
    return
  }
  const ok = await copyTextToClipboard(markdown)
  if (ok) {
    copiedNotebookId.value = key
    showNotice(`Copied cards for "${notebook.notebook_title || 'Standalone'}"`)
    if (copyResetTimer) clearTimeout(copyResetTimer)
    copyResetTimer = setTimeout(() => {
      if (copiedNotebookId.value === key) {
        copiedNotebookId.value = null
      }
    }, 2000)
  } else {
    showError('Failed to copy flashcards to clipboard.')
  }
}

async function copyAllFlashcards() {
  const activeNotebooks = filteredNotebooks.value
  if (!activeNotebooks || activeNotebooks.length === 0) {
    showError('No flashcards matching current filters to copy.')
    return
  }

  const sections = []
  let totalCardsCount = 0

  for (const nb of activeNotebooks) {
    const md = buildNotebookMarkdown(nb)
    if (md.trim()) {
      sections.push(md)
      totalCardsCount += nb.visibleCardCount || 0
    }
  }

  if (sections.length === 0) {
    showError('No flashcards found to copy.')
    return
  }

  const fullMarkdown = sections.join('\n\n---\n\n')
  const ok = await copyTextToClipboard(fullMarkdown)
  if (ok) {
    copiedAll.value = true
    showNotice(`Copied ${totalCardsCount} flashcards to clipboard`)
    if (copyResetTimer) clearTimeout(copyResetTimer)
    copyResetTimer = setTimeout(() => {
      copiedAll.value = false
    }, 2000)
  } else {
    showError('Failed to copy all flashcards to clipboard.')
  }
}

// Notebook accordion controls
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

const areAllNotebooksExpanded = computed(() => {
  if (filteredNotebooks.value.length === 0) return false
  return filteredNotebooks.value.every((nb) => expandedNotebooks.value.has(nb.notebook_id || 'standalone'))
})

function toggleAllNotebooks() {
  if (areAllNotebooksExpanded.value) {
    expandedNotebooks.value = new Set()
  } else {
    const nextSet = new Set()
    filteredNotebooks.value.forEach((nb) => {
      nextSet.add(nb.notebook_id || 'standalone')
    })
    expandedNotebooks.value = nextSet
  }
}

// Individual card expand/collapse
function isCardExpanded(cardId) {
  return expandedCardIds.value.has(cardId)
}

function toggleCardExpand(cardId) {
  const nextSet = new Set(expandedCardIds.value)
  if (nextSet.has(cardId)) {
    nextSet.delete(cardId)
  } else {
    nextSet.add(cardId)
  }
  expandedCardIds.value = nextSet
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
            if (filter === 'due' && !isCardDue(c, nowUnix)) return false
            if (filter === 'active' && c.suspended) return false
            if (filter === 'suspended' && !c.suspended) return false

            if (query) {
              const p = (c.prompt || '').toLowerCase()
              const a = (c.answer || '').toLowerCase()
              const nbT = (c.notebook_title || '').toLowerCase()
              const topT = (c.topic_title || '').toLowerCase()
              const textMatch =
                p.includes(query) ||
                a.includes(query) ||
                nbT.includes(query) ||
                topT.includes(query)
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

const hasAutoExpanded = ref(false)

async function loadOverview() {
  if (loading.value) return
  loading.value = true
  try {
    const res = await getFlashcardsDeckOverview()
    if (res && res.error) {
      showError(res.error)
      return
    }
    if (res && res.metrics) {
      metrics.value = res.metrics
      notebooks.value = res.notebooks || []
      // Auto-expand first 2 notebooks on first load only
      if (!hasAutoExpanded.value && notebooks.value.length > 0) {
        const initialSet = new Set()
        notebooks.value.slice(0, 2).forEach((nb) => {
          initialSet.add(nb.notebook_id || 'standalone')
        })
        expandedNotebooks.value = initialSet
        hasAutoExpanded.value = true
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
    if (res && res.error) {
      showError(res.error)
      return
    }
    if (res && res.ok) {
      card.suspended = newSuspended
      showNotice(newSuspended ? 'Card paused' : 'Card resumed')
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
    title: newSuspended ? 'Pause All Cards in Deck?' : 'Resume All Cards in Deck?',
    message: newSuspended
      ? `This will pause all flashcards in "${nb.notebook_title}" and exclude them from your daily review queue.`
      : `This will resume all flashcards in "${nb.notebook_title}" for retention review.`,
    confirmText: newSuspended ? 'Pause Deck' : 'Resume Deck',
    type: newSuspended ? 'warning' : 'info',
  })
  if (!confirmed) return

  pendingNotebookIds.value = new Set(pendingNotebookIds.value).add(nbKey)
  try {
    const res = await toggleNotebookCardsSuspension(nb.notebook_id, newSuspended)
    if (res && res.error) {
      showError(res.error)
      return
    }
    if (res && res.ok) {
      showNotice(newSuspended ? 'Deck cards paused' : 'Deck cards resumed')
      await loadOverview()
      emit('updated')
    }
  } catch (err) {
    showError(err?.message || 'Failed to toggle deck suspension')
  } finally {
    const nextSet = new Set(pendingNotebookIds.value)
    nextSet.delete(nbKey)
    pendingNotebookIds.value = nextSet
  }
}

async function confirmDeleteCard(card) {
  if (pendingCardIds.value.has(card.id)) return

  const confirmed = await showConfirm({
    title: 'Delete Flashcard Permanently?',
    message: `Warning: Deleting this card will permanently purge its FSRS review history and stability records.\n\n"${card.prompt.slice(0, 80)}${card.prompt.length > 80 ? '...' : ''}"`,
    confirmText: 'Delete Card',
    type: 'danger',
  })
  if (!confirmed) return

  pendingCardIds.value = new Set(pendingCardIds.value).add(card.id)
  try {
    const res = await deleteFlashcard(card.id)
    if (res && res.error) {
      showError(res.error)
      return
    }
    if (res && res.ok) {
      showNotice('Flashcard deleted')
      const nextExpanded = new Set(expandedCardIds.value)
      nextExpanded.delete(card.id)
      expandedCardIds.value = nextExpanded
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
    if (editingCardId.value !== null) {
      cancelEditingCard()
      return
    }
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
  border-radius: 18px;
  box-shadow: 0 24px 48px -12px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  color: var(--on-surface);
}

/* Header */
.deck-modal-header {
  padding: 16px 22px;
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
  letter-spacing: 0.05em;
  color: var(--primary);
}

.header-title {
  margin: 0;
  font-size: 1.18rem;
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
  width: 34px;
  height: 34px;
  border-radius: 9px;
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
}

.icon-action-btn:active,
.modal-close-btn:active {
  transform: scale(0.95);
}

.copy-all-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 12px;
  border-radius: 9px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  transition: all 0.15s ease;
}

.copy-all-btn:hover:not(:disabled) {
  background: var(--surface-container);
  border-color: var(--primary);
  color: var(--primary);
}

.copy-all-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.copy-cards-btn {
  background: transparent;
  color: var(--on-surface);
}

.copy-cards-btn:hover {
  background: var(--surface-container-highest);
  color: var(--primary);
  border-color: var(--primary);
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
  padding: 20px 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Executive Summary Strip */
.executive-summary-strip {
  padding: 12px 16px;
  background: var(--surface-container, #282828);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.stats-row {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
}

.stat-pill {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 4px 10px;
  background: var(--surface-container-lowest, #141617);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
}

.stat-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--muted-text);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.stat-val {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}

.stat-sub {
  font-size: 10px;
  color: var(--muted-text);
}

.stat-pill.due-pill {
  border-color: rgba(245, 158, 11, 0.3);
  background: rgba(245, 158, 11, 0.08);
}

.stat-pill.mastery-pill {
  border-color: rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.08);
}

.text-amber { color: #f59e0b; }
.text-emerald { color: #10b981; }
.text-muted { color: var(--muted-text); }

/* Maturity Ribbon */
.maturity-ribbon {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  padding-top: 6px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
}

.maturity-bar-track {
  flex: 1;
  min-width: 180px;
  height: 6px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 4px;
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

.maturity-legend {
  display: flex;
  align-items: center;
  gap: 10px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 600;
  color: var(--muted-text);
}

.legend-item .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.dot-new { background: #94a3b8; }
.dot-learning { background: #f59e0b; }
.dot-young { background: #3b82f6; }
.dot-mature { background: #10b981; }

/* Filter & Search Toolbar */
.filter-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.search-box {
  flex: 1;
  min-width: 220px;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 10px;
  color: var(--muted-text);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 7px 30px 7px 30px;
  background: var(--surface-container, #282828);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  color: var(--on-surface);
  font-size: 12px;
  outline: none;
  transition: all 0.15s ease;
}

.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px rgba(0, 91, 193, 0.15);
}

.search-clear-btn {
  position: absolute;
  right: 8px;
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  padding: 2px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-chips {
  display: flex;
  gap: 4px;
}

.filter-chip {
  padding: 6px 10px;
  border-radius: 7px;
  font-size: 11px;
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

.toggle-all-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 10px;
  border-radius: 7px;
  font-size: 11px;
  font-weight: 600;
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.toggle-all-btn:hover {
  background: var(--surface-container);
  color: var(--on-surface);
}

/* States */
.state-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 44px 20px;
  color: var(--muted-text);
  gap: 10px;
  text-align: center;
}

.loading-spinner {
  width: 28px;
  height: 28px;
  border: 2.5px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.empty-icon {
  opacity: 0.4;
}

.empty-state h3 {
  margin: 0;
  color: var(--on-surface);
  font-size: 14px;
}

.empty-state p {
  margin: 0;
  font-size: 12px;
}

/* Decks List */
.decks-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.notebook-deck-card {
  background: var(--surface-container-low, #1d2021);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  overflow: hidden;
}

.notebook-deck-header {
  padding: 12px 14px;
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
  gap: 8px;
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
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}

.count-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 10px;
  background: var(--surface-container-lowest, #141617);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.due-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 7px;
  border-radius: 10px;
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid var(--outline-variant);
}

.deck-header-actions {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
}

.action-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 9px;
  border-radius: 7px;
  font-size: 11px;
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
  color: #dc2626;
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
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-top: 1px solid var(--outline-variant);
}

.topic-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.topic-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-left: 2px;
}

.topic-icon {
  color: var(--muted-text);
}

.topic-title {
  font-size: 11px;
  font-weight: 700;
  color: var(--muted-text);
  letter-spacing: 0.02em;
}

.topic-count {
  font-size: 10px;
  color: var(--muted-text);
  opacity: 0.7;
}

/* Cards List */
.cards-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.card-row-item {
  background: var(--surface-container-lowest, #141617);
  border: 1px solid var(--outline-variant);
  border-radius: 9px;
  transition: all 0.15s ease;
  overflow: hidden;
}

.card-row-item:hover {
  border-color: rgba(255, 255, 255, 0.18);
}

.card-row-item.is-suspended {
  opacity: 0.65;
  background: rgba(0, 0, 0, 0.2);
}

.card-row-item.is-expanded {
  border-color: rgba(0, 91, 193, 0.4);
}

/* Scannable Card Row Header */
.card-row-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 12px;
  cursor: pointer;
  user-select: none;
}

.card-prompt-col {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.qa-indicator {
  font-size: 9px;
  font-weight: 800;
  padding: 1px 5px;
  border-radius: 4px;
  flex-shrink: 0;
  background: rgba(0, 91, 193, 0.15);
  color: var(--primary);
}

.qa-indicator.ans-ind {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.card-prompt-text {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-row-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.meta-pill {
  font-size: 10.5px;
  padding: 2px 6px;
  border-radius: 5px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
}

.meta-pill .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}

.pill-paused {
  background: rgba(245, 158, 11, 0.12);
  color: #f59e0b;
}
.pill-paused .dot { background: #f59e0b; }

.pill-due {
  background: rgba(239, 68, 68, 0.12);
  color: #dc2626;
}
.pill-due .dot { background: #dc2626; }

.pill-neutral,
.pill-stability {
  background: var(--surface-container, #282828);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.card-pause-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 10.5px;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--outline-variant);
  transition: all 0.15s ease;
}

.card-pause-btn.suspend-btn {
  background: transparent;
  color: var(--muted-text);
}

.card-pause-btn.suspend-btn:hover {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border-color: rgba(245, 158, 11, 0.3);
}

.card-pause-btn.resume-btn {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
}

.card-pause-btn.resume-btn:hover {
  background: rgba(16, 185, 129, 0.25);
}

.row-expand-chevron {
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 2px;
}

/* Expanded Card Panel */
.card-expanded-panel {
  padding: 10px 14px 12px 14px;
  background: rgba(0, 0, 0, 0.2);
  border-top: 1px solid var(--outline-variant);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.expanded-answer-box {
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: var(--surface-container-low, #1d2021);
  padding: 8px 10px;
  border-radius: 7px;
  border: 1px solid var(--outline-variant);
}

.answer-badge-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10.5px;
  font-weight: 700;
  color: #10b981;
}

.answer-content-text {
  font-size: 12.5px;
  line-height: 1.45;
  color: var(--on-surface);
  white-space: pre-wrap;
  word-break: break-word;
}

.expanded-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.fsrs-stats-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.fsrs-stat {
  font-size: 11px;
  color: var(--muted-text);
}

.fsrs-stat strong {
  color: var(--on-surface);
  font-weight: 600;
}

.expanded-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.guarded-edit-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 10.5px;
  font-weight: 600;
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  cursor: pointer;
  transition: all 0.15s ease;
}

.guarded-edit-btn:hover {
  background: var(--surface-container-high, #3c3836);
  border-color: var(--outline);
  color: var(--accent-color, #fabd2f);
}

.guarded-delete-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 10.5px;
  font-weight: 600;
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: #dc2626;
  cursor: pointer;
  transition: all 0.15s ease;
}

.guarded-delete-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.4);
}

/* Inline Edit Card Form */
.card-edit-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--surface-container-low, #1d2021);
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
}

.edit-field-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.edit-field-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--on-surface);
}

.edit-field-textarea {
  width: 100%;
  background: var(--surface-container-lowest, #141617);
  border: 1px solid var(--outline-variant);
  border-radius: 6px;
  color: var(--on-surface);
  font-family: inherit;
  font-size: 12.5px;
  line-height: 1.45;
  padding: 8px 10px;
  resize: vertical;
  outline: none;
  transition: border-color 0.15s ease;
  box-sizing: border-box;
}

.edit-field-textarea:focus {
  border-color: var(--accent-color, #fabd2f);
}

.edit-field-textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.edit-form-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 2px;
}

.edit-cancel-btn {
  background: transparent;
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  font-size: 11.5px;
  font-weight: 500;
  padding: 4px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.edit-cancel-btn:hover:not(:disabled) {
  background: var(--surface-container-high, #3c3836);
  color: var(--on-surface);
}

.edit-save-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  background: var(--accent-color, #fabd2f);
  color: #1a1a1a;
  border: none;
  font-size: 11.5px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: opacity 0.15s ease, transform 0.1s ease;
}

.edit-save-btn:hover:not(:disabled) {
  opacity: 0.92;
}

.edit-save-btn:active:not(:disabled) {
  transform: scale(0.97);
}

.edit-save-btn:disabled,
.edit-cancel-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
  transform: scale(0.97) translateY(6px);
}

.deck-modal-fade-leave-to .deck-modal-container {
  transform: scale(0.98) translateY(4px);
}

@media (max-width: 768px) {
  .deck-modal-overlay {
    padding: 10px;
  }
  .stats-row {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
  }
  .card-prompt-text {
    max-width: 180px;
  }
}
</style>
