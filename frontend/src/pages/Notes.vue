<template>
  <div class="notes-page">
    <div class="notes-layout" :class="{ 'sidebar-collapsed': isSidebarCollapsed }">
      <!-- Left Sidebar: Collapsible Book & Chapter Tree Navigator -->
      <aside v-show="!isSidebarCollapsed" class="notes-sidebar">
        <div class="notes-sidebar-header">
          <div class="sidebar-title-row">
            <div class="title-with-badge">
              <h2 class="sidebar-title">Study Notes</h2>
              <span class="notes-count-badge" :title="`${totalNotesCount} notes generated across all textbooks`">
                {{ totalNotesCount }} Notes
              </span>
            </div>
            <button
              type="button"
              class="sidebar-toggle-btn"
              title="Collapse sidebar (Zen reading mode)"
              aria-label="Collapse sidebar"
              @click="isSidebarCollapsed = true"
            >
              <BaseIcon name="chevron-left" size="14" />
            </button>
          </div>
          <p class="sidebar-subtitle">Knowledge base & topic summaries</p>

          <!-- Search Box -->
          <div class="shared-search-wrapper">
            <BaseIcon name="search" size="14" class="search-icon" />
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search books, chapters, or notes..."
            />
            <button
              v-if="searchQuery"
              type="button"
              class="clear-search-btn"
              title="Clear search"
              @click="searchQuery = ''"
            >
              <BaseIcon name="x" size="12" />
            </button>
          </div>
        </div>

        <!-- Book & Chapter Tree List -->
        <div class="topics-list-container">
          <div v-if="loading" class="loading-topics">
            <div class="spinner-sm"></div>
            <span>Loading notes library...</span>
          </div>

          <div v-else-if="treeNotebooks.length === 0" class="empty-topics">
            <p v-if="searchQuery">No books or chapters match "{{ searchQuery }}"</p>
            <p v-else>No textbooks uploaded yet. Upload textbooks in the Library to start generating notes.</p>
          </div>

          <div v-else class="book-tree-list">
            <div
              v-for="nb in treeNotebooks"
              :key="nb.notebook_id"
              class="book-tree-node"
            >
              <!-- Book / Notebook Header Item -->
              <div
                class="book-node-header"
                :class="{ expanded: isNotebookExpanded(nb.notebook_id) }"
                @click="toggleNotebook(nb.notebook_id)"
              >
                <button
                  type="button"
                  class="chevron-btn"
                  aria-label="Toggle book expansion"
                  @click.stop="toggleNotebook(nb.notebook_id)"
                >
                  <BaseIcon
                    :name="isNotebookExpanded(nb.notebook_id) ? 'chevron-down' : 'chevron-right'"
                    size="13"
                  />
                </button>
                <BaseIcon name="book" size="14" class="book-node-icon" />
                <span class="book-node-title" :title="nb.title">{{ nb.title }}</span>
                <span
                  class="book-progress-pill"
                  :class="{ 'all-ready': nb.notesCount > 0 && nb.notesCount === nb.totalTopics }"
                  :title="`${nb.notesCount} of ${nb.totalTopics} chapters have study notes`"
                >
                  {{ nb.notesCount }}/{{ nb.totalTopics }}
                </span>
              </div>

              <!-- Chapter Children List -->
              <div
                v-if="isNotebookExpanded(nb.notebook_id)"
                class="chapter-sublist"
              >
                <button
                  v-for="topic in nb.topics"
                  :key="topic.topic_id"
                  type="button"
                  class="chapter-tree-item"
                  :class="{ active: selectedTopicID === topic.topic_id }"
                  @click="selectTopic(topic)"
                >
                  <div class="chapter-item-header">
                    <span
                      class="status-dot"
                      :class="topic.has_note ? 'dot-has-note' : 'dot-missing-note'"
                      :title="topic.has_note ? 'Study Note Ready' : 'No Study Note Yet'"
                    ></span>
                    <span class="chapter-item-title" :title="topic.title">{{ topic.title }}</span>
                  </div>
                  <div class="chapter-item-meta">
                    <span v-if="topic.start_page && topic.end_page" class="chapter-page-tag">
                      p. {{ topic.start_page }}–{{ topic.end_page }}
                    </span>
                    <span v-if="topic.last_reviewed_text" class="chapter-reviewed-tag">
                      Reviewed {{ topic.last_reviewed_text }}
                    </span>
                    <span v-else-if="topic.has_note" class="chapter-ready-tag">
                      Note ready
                    </span>
                  </div>
                </button>
              </div>
            </div>
          </div>
        </div>
      </aside>

      <!-- Right Pane: Session Cards PPT-Style Feed -->
      <main class="notes-content-pane">
        <template v-if="!selectedTopicID">
          <div class="notes-empty-selection">
            <button
              v-if="isSidebarCollapsed"
              type="button"
              class="restore-sidebar-btn"
              @click="isSidebarCollapsed = false"
            >
              <BaseIcon name="layers" size="14" />
              <span>Show Book Chapters</span>
            </button>
            <BaseIcon name="book" size="48" class="empty-icon" />
            <h3>Select a Chapter</h3>
            <p>Choose a chapter from your textbooks to view, edit, or generate per-session study notes.</p>
          </div>
        </template>

        <template v-else>
          <!-- Topic Header -->
          <header class="note-view-header">
            <div class="note-title-group">
              <div class="note-meta-badges">
                <!-- Sidebar Expand Button when Collapsed -->
                <button
                  v-if="isSidebarCollapsed"
                  type="button"
                  class="restore-sidebar-pill-btn"
                  title="Expand sidebar"
                  @click="isSidebarCollapsed = false"
                >
                  <BaseIcon name="sidebar" size="13" />
                  <span>Chapters</span>
                </button>

                <span v-if="activeTopic?.notebook_title" class="chip-tag book-chip">
                  <BaseIcon name="book" size="12" />
                  <span>{{ activeTopic.notebook_title }}</span>
                </span>
                <span class="chip-tag info">
                  <BaseIcon name="layers" size="12" />
                  <span>{{ sessionSlots.length }} {{ sessionSlots.length === 1 ? 'Card' : 'Cards' }}</span>
                </span>
              </div>
              <h1 class="note-topic-title">{{ activeTopic?.title || selectedTopicID }}</h1>
            </div>
            <div class="note-header-actions">
              <button
                v-if="isFromFlashcards"
                type="button"
                class="secondary-btn btn-sm back-flashcards-btn"
                title="Return to Flashcards review session"
                @click="goBackToFlashcards"
              >
                <BaseIcon name="arrow-left" size="14" />
                <span>Back to Flashcards</span>
              </button>
              <button
                type="button"
                class="secondary-btn btn-sm open-folder-btn"
                title="Open notes folder in File Explorer (Obsidian / Logseq compatible)"
                @click="handleOpenNotesFolder"
              >
                <BaseIcon name="folder" size="14" />
                <span>Open Folder</span>
              </button>
              <button
                type="button"
                class="primary-btn btn-sm add-note-btn"
                :disabled="showAddNoteCard"
                @click="openAddNoteCard"
              >
                <BaseIcon name="plus" size="14" />
                <span>Add Note</span>
              </button>
            </div>
          </header>

          <!-- Flashcard Navigation Context Banner -->
          <div v-if="isFromFlashcards" class="notes-flashcard-context-banner">
            <div class="banner-left">
              <BaseIcon name="book-open" size="15" class="banner-icon" />
              <span>{{ flashcardBannerText }}</span>
            </div>
            <button
              type="button"
              class="banner-return-btn"
              title="Return to Flashcards review session"
              @click="goBackToFlashcards"
            >
              <BaseIcon name="arrow-left" size="13" />
              <span>Return to Flashcards</span>
            </button>
          </div>

          <!-- Interop Tip Callout -->
          <div v-else class="notes-obsidian-tip">
            <BaseIcon name="info" size="14" class="tip-icon" />
            <span>
              <strong>Obsidian & Logseq Compatible:</strong> Notes are synced to local markdown files. Use <em>Open Folder</em> to add images (<code>![alt](./assets/pic.png)</code>) or edit notes in external markdown tools.
            </span>
          </div>

          <!-- Banner messages -->
          <div v-if="errorMsg" class="note-banner error-banner">
            <BaseIcon name="alert-triangle" size="16" />
            <span>{{ errorMsg }}</span>
          </div>

          <div v-if="successMsg" class="note-banner success-banner">
            <BaseIcon name="check" size="16" />
            <span>{{ successMsg }}</span>
          </div>

          <!-- Session Cards Feed -->
          <div class="session-cards-feed">
            <!-- Custom Add Note Form Card -->
            <div v-if="showAddNoteCard" class="session-note-card add-note-new-card">
              <div class="session-card-header">
                <div class="session-card-badge-row">
                  <span class="session-index-pill new-badge">New Note</span>
                  <div class="page-range-inputs">
                    <label>Pages:</label>
                    <input
                      v-model.number="newNoteStartPage"
                      type="number"
                      min="0"
                      class="page-input"
                      placeholder="Start"
                    />
                    <span>–</span>
                    <input
                      v-model.number="newNoteEndPage"
                      type="number"
                      min="0"
                      class="page-input"
                      placeholder="End"
                    />
                  </div>
                </div>
                <div class="session-card-actions">
                  <button
                    type="button"
                    class="secondary-btn btn-sm"
                    :disabled="savingNewNote"
                    @click="closeAddNoteCard"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    class="primary-btn btn-sm"
                    :disabled="savingNewNote"
                    @click="saveNewNote"
                  >
                    {{ savingNewNote ? 'Saving...' : 'Save Note' }}
                  </button>
                </div>
              </div>
              <div class="session-card-body">
                <div class="slot-editor-box">
                  <textarea
                    v-model="newNoteBuffer"
                    class="shared-textarea slot-textarea"
                    placeholder="Write markdown notes or summaries for this session/chapter..."
                    rows="8"
                  ></textarea>
                </div>
              </div>
            </div>

            <div v-if="loadingSlots" class="slots-loading-state">
              <div class="spinner"></div>
              <p>Loading reading session notes...</p>
            </div>

            <div v-else-if="sessionSlots.length === 0 && !showAddNoteCard" class="empty-box-card">
              <BaseIcon name="book-open" size="40" class="empty-box-icon" />
              <h3>No Study Notes Yet</h3>
              <p>Start reading or click "Add Note" to write custom notes for this chapter.</p>
              <button
                type="button"
                class="primary-btn btn-sm"
                style="margin-top: 12px;"
                @click="openAddNoteCard"
              >
                <BaseIcon name="plus" size="14" />
                <span>Add Note</span>
              </button>
            </div>

            <!-- PPT / Deck of Cards for each Reading Session -->
            <div
              v-for="(slot, idx) in sessionSlots"
              :id="`session-slot-${slot.start_page}-${slot.end_page}`"
              :key="`${slot.start_page}-${slot.end_page}`"
              class="session-note-card"
              :class="{ 'target-highlight': isTargetSlot(slot, idx) }"
            >
              <!-- Card Header -->
              <div class="session-card-header">
                <div class="session-card-badge-row">
                  <span class="session-index-pill">Session {{ idx + 1 }}</span>
                  <span class="session-range-pill">
                    <BaseIcon :name="isCurrentTopicYouTube ? 'video' : 'book-open'" size="12" />
                    <span>{{ formatPageRange(slot.start_page, slot.end_page) }}</span>
                  </span>
                  <!-- Flashcard Source Badge when targeted from Flashcards -->
                  <span v-if="isTargetSlot(slot, idx)" class="session-target-badge" title="Referenced by your current flashcard">
                    <BaseIcon name="target" size="12" />
                    <span>{{ targetPageFromRoute ? `Flashcard Source (p. ${targetPageFromRoute})` : 'Flashcard Source' }}</span>
                  </span>
                  <!-- Jump to Reader Source Badge Button -->
                  <button
                    v-if="canJumpToSource(slot)"
                    type="button"
                    class="jump-source-pill-btn"
                    :title="isCurrentTopicYouTube ? 'Jump to video segment in Reader' : `Open Reader at page ${slot.start_page || 1}`"
                    @click="jumpToReader(slot)"
                  >
                    <BaseIcon :name="isCurrentTopicYouTube ? 'play' : 'external-link'" size="11" />
                    <span>{{ isCurrentTopicYouTube ? 'Watch ↗' : `Jump to p. ${slot.start_page || 1} ↗` }}</span>
                  </button>
                  <span v-if="slot.last_reviewed_at" class="session-review-pill">
                    <BaseIcon name="check-circle" size="12" />
                    <span>Reviewed {{ formatTimeAgo(slot.last_reviewed_at) }}</span>
                  </span>
                </div>

                <div class="session-card-actions">
                  <!-- Actions when editing this slot -->
                  <template v-if="editingSlotKey === getSlotKey(slot)">
                    <button
                      type="button"
                      class="secondary-btn btn-sm"
                      :disabled="savingSlotKey === getSlotKey(slot)"
                      @click="cancelEditSlot"
                    >
                      Cancel
                    </button>
                    <button
                      type="button"
                      class="primary-btn btn-sm"
                      :disabled="savingSlotKey === getSlotKey(slot)"
                      @click="saveSlot(slot)"
                    >
                      {{ savingSlotKey === getSlotKey(slot) ? 'Saving...' : 'Save' }}
                    </button>
                  </template>

                  <!-- Actions when viewing this slot -->
                  <template v-else>
                    <button
                      v-if="slot.content"
                      type="button"
                      class="icon-btn-pill"
                      title="Edit note"
                      :disabled="generatingSlotKey === getSlotKey(slot)"
                      @click="startEditSlot(slot)"
                    >
                      <BaseIcon name="edit" size="14" />
                      <span>Edit</span>
                    </button>

                    <button
                      type="button"
                      class="icon-btn-pill highlight"
                      :disabled="generatingSlotKey === getSlotKey(slot)"
                      :title="slot.content ? 'Regenerate session note' : 'Generate session note with AI'"
                      @click="generateSlot(slot)"
                    >
                      <BaseIcon name="sparkles" size="14" />
                      <span>{{ getSlotGenerateButtonText(slot) }}</span>
                    </button>

                    <button
                      v-if="slot.content"
                      type="button"
                      class="icon-btn-pill success"
                      title="Mark as reviewed"
                      :disabled="markingSlotKey === getSlotKey(slot)"
                      @click="markSlotReviewed(slot)"
                    >
                      <BaseIcon name="check" size="14" />
                      <span>{{ markingSlotKey === getSlotKey(slot) ? 'Marking...' : 'Reviewed' }}</span>
                    </button>
                  </template>
                </div>
              </div>

              <!-- Card Body -->
              <div class="session-card-body">
                <!-- Generating Spinner -->
                <div v-if="generatingSlotKey === getSlotKey(slot)" class="generating-slot-state">
                  <div class="spinner"></div>
                  <p>Synthesizing high-yield summary for {{ formatPageRange(slot.start_page, slot.end_page) }}...</p>
                </div>

                <!-- Editor Mode -->
                <div v-else-if="editingSlotKey === getSlotKey(slot)" class="slot-editor-box">
                  <textarea
                    v-model="editSlotBuffer"
                    class="shared-textarea slot-textarea"
                    placeholder="Write or edit notes for this reading session..."
                    rows="8"
                  ></textarea>
                </div>

                <!-- Rendered Note -->
                <div
                  v-else-if="slot.content"
                  class="slot-markdown-content shared-markdown-content"
                  v-html="renderMarkdown(slot.content)"
                ></div>

                <!-- Empty State within Card -->
                <div v-else class="slot-empty-state">
                  <p class="empty-hint">No notes for {{ formatPageRange(slot.start_page, slot.end_page) }} yet.</p>
                  <div class="slot-empty-buttons">
                    <button
                      type="button"
                      class="primary-btn btn-sm"
                      :disabled="generatingSlotKey === getSlotKey(slot)"
                      @click="generateSlot(slot)"
                    >
                      <BaseIcon name="sparkles" size="13" />
                      <span>Generate with AI</span>
                    </button>
                    <button
                      type="button"
                      class="secondary-btn btn-sm"
                      @click="startEditSlot(slot)"
                    >
                      <BaseIcon name="edit" size="13" />
                      <span>Write Manually</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseIcon from '../components/BaseIcon.vue'
import {
  getNotebooks,
  getNotebookTopicTree,
  getNotesByNotebook,
  getTopicStudyNoteSlots,
  generateTopicStudyNoteForRange,
  updateTopicStudyNote,
  markTopicReviewed as apiMarkTopicReviewed,
  openNotesFolder,
} from '../services/appApi'
import { renderMarkdown } from '../services/markdown'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const loadingSlots = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const notebooks = ref([])
const notebookTree = ref([])
const selectedNotebookID = ref('')
const selectedTopicID = ref('')
const searchQuery = ref('')
const expandedNotebooks = ref(new Set())
const isSidebarCollapsed = ref(false)

const notesMap = ref({}) // topicID -> TopicStudyNote
const sessionSlots = ref([])

// Slot editing & action state
const editingSlotKey = ref(null)
const editSlotBuffer = ref('')
const savingSlotKey = ref(null)
const generatingSlotKey = ref(null)
const markingSlotKey = ref(null)

function getSlotKey(slot) {
  return `${slot.start_page}_${slot.end_page}`
}

function getSlotGenerateButtonText(slot) {
  if (generatingSlotKey.value === getSlotKey(slot)) {
    return 'Generating...'
  }
  return slot.content ? 'Regenerate' : 'Generate Note'
}

function toggleNotebook(notebookID) {
  if (expandedNotebooks.value.has(notebookID)) {
    expandedNotebooks.value.delete(notebookID)
  } else {
    expandedNotebooks.value.add(notebookID)
  }
}

function isNotebookExpanded(notebookID) {
  if (searchQuery.value.trim().length > 0) return true
  return expandedNotebooks.value.has(notebookID)
}

// Flat topic list with notebook title & note status
const allTopics = computed(() => {
  const list = []
  for (const nb of notebookTree.value) {
    const nbTitle = nb.title || 'Untitled Notebook'
    const nbID = nb.notebook_id || nb.id
    if (Array.isArray(nb.topics)) {
      for (const t of nb.topics) {
        const tid = t.topic_id || t.id
        const topicNotes = notesMap.value[tid] || []
        const hasNoteVal = topicNotes.some((n) => n && n.content && n.content.trim().length > 0)
        let maxReviewedAt = 0
        for (const n of topicNotes) {
          if (n?.last_reviewed_at && n.last_reviewed_at > maxReviewedAt) {
            maxReviewedAt = n.last_reviewed_at
          }
        }
        const reviewedText = maxReviewedAt > 0 ? formatTimeAgo(maxReviewedAt) : ''
        // A topic is active/reached if it has a note, is completed, or has active page reading progress
        const hasStartedReading = (t.status === 'completed') || (t.current_page_cursor && t.current_page_cursor > 0)
        if (!hasNoteVal && !hasStartedReading) {
          continue
        }
        list.push({
          topic_id: tid,
          title: t.title || tid,
          notebook_id: nbID,
          notebook_title: nbTitle,
          start_page: t.start_page || 0,
          end_page: t.end_page || 0,
          last_reviewed_at: maxReviewedAt,
          last_reviewed_text: reviewedText,
          has_note: hasNoteVal,
        })
      }
    }
  }
  return list
})

// Grouped Tree for Sidebar: Books with nested chapters and progress counts
const treeNotebooks = computed(() => {
  const query = searchQuery.value.toLowerCase().trim()
  const result = []

  for (const nb of notebookTree.value) {
    const nbTitle = nb.title || 'Untitled Notebook'
    const nbID = nb.notebook_id || nb.id
    const rawTopics = Array.isArray(nb.topics) ? nb.topics : []

    let totalChaptersWithNotes = 0
    const processedTopics = []

    for (const t of rawTopics) {
      const tid = t.topic_id || t.id
      const topicNotes = notesMap.value[tid] || []
      const hasNoteVal = topicNotes.some((n) => n && n.content && n.content.trim().length > 0)
      if (hasNoteVal) {
        totalChaptersWithNotes++
      }

      let maxReviewedAt = 0
      for (const n of topicNotes) {
        if (n?.last_reviewed_at && n.last_reviewed_at > maxReviewedAt) {
          maxReviewedAt = n.last_reviewed_at
        }
      }
      const reviewedText = maxReviewedAt > 0 ? formatTimeAgo(maxReviewedAt) : ''

      const hasStartedReading = (t.status === 'completed') || (t.current_page_cursor && t.current_page_cursor > 0)
      // Only include chapters that have existing study notes or have been reached/started
      if (!hasNoteVal && !hasStartedReading) {
        continue
      }

      const topicObj = {
        topic_id: tid,
        title: t.title || tid,
        notebook_id: nbID,
        notebook_title: nbTitle,
        start_page: t.start_page || 0,
        end_page: t.end_page || 0,
        last_reviewed_at: maxReviewedAt,
        last_reviewed_text: reviewedText,
        has_note: hasNoteVal,
      }

      if (query) {
        const titleMatch = topicObj.title.toLowerCase().includes(query)
        const nbMatch = nbTitle.toLowerCase().includes(query)
        const contentMatch = topicNotes.some((n) => (n?.content || '').toLowerCase().includes(query))
        if (titleMatch || nbMatch || contentMatch) {
          processedTopics.push(topicObj)
        }
      } else {
        processedTopics.push(topicObj)
      }
    }

    if (processedTopics.length > 0) {
      result.push({
        notebook_id: nbID,
        title: nbTitle,
        totalTopics: rawTopics.length,
        notesCount: totalChaptersWithNotes,
        topics: processedTopics,
      })
    }
  }

  return result
})

const totalNotesCount = computed(() => {
  return Object.values(notesMap.value)
    .flat()
    .filter((n) => n && n.content && n.content.trim().length > 0).length
})

const activeTopic = computed(() => {
  if (!selectedTopicID.value) return null
  return allTopics.value.find((t) => t.topic_id === selectedTopicID.value) || {
    topic_id: selectedTopicID.value,
    title: selectedTopicID.value,
    notebook_id: selectedNotebookID.value,
  }
})

// Navigation & Loop helpers
const isFromFlashcards = computed(() => {
  return route.query.from === 'flashcards'
})

const activeNotebook = computed(() => {
  const nbId = activeTopic.value?.notebook_id || selectedNotebookID.value
  if (!nbId) return null
  return notebooks.value.find((n) => (n.id || n.notebook_id) === nbId) || null
})

const isCurrentTopicYouTube = computed(() => {
  const fileType = (activeNotebook.value?.file_type || '').toLowerCase()
  return fileType === 'youtube'
})

const targetPageFromRoute = computed(() => {
  const p = route.query.page || route.query.startPage || route.query.start_page
  return p ? Number(p) : null
})

const flashcardBannerText = computed(() => {
  const topicTitle = activeTopic.value?.title || activeNotebook.value?.title || ''
  const page = targetPageFromRoute.value

  let matchedSlot = null
  let sessionIndex = -1

  if (page && sessionSlots.value.length > 0) {
    const idx = sessionSlots.value.findIndex(
      (s) => s.start_page > 0 && s.end_page > 0 && page >= s.start_page && page <= s.end_page
    )
    if (idx !== -1) {
      matchedSlot = sessionSlots.value[idx]
      sessionIndex = idx + 1
    }
  }

  if (!matchedSlot && sessionSlots.value.length > 0) {
    matchedSlot = sessionSlots.value[0]
    sessionIndex = 1
  }

  if (matchedSlot && matchedSlot.start_page > 0 && matchedSlot.end_page > 0) {
    const sessionPages =
      matchedSlot.start_page === matchedSlot.end_page
        ? `p. ${matchedSlot.start_page}`
        : `pp. ${matchedSlot.start_page}–${matchedSlot.end_page}`
    const sessionLabel =
      sessionSlots.value.length > 1
        ? `Session ${sessionIndex} (${sessionPages})`
        : `(${sessionPages})`

    if (page && page >= matchedSlot.start_page && page <= matchedSlot.end_page) {
      if (topicTitle) {
        return `Reviewing notes for ${topicTitle} — Page ${page} in ${sessionLabel}`
      }
      return `Reviewing notes for Page ${page} in ${sessionLabel}`
    }

    if (topicTitle) {
      return `Reviewing notes for ${topicTitle} — ${sessionLabel}`
    }
    return `Reviewing notes for ${sessionLabel}`
  }

  if (page) {
    if (topicTitle) {
      return `Reviewing notes for ${topicTitle} — Page ${page}`
    }
    return `Reviewing notes for Page ${page}`
  }

  if (topicTitle) {
    return `Reviewing notes for ${topicTitle}`
  }

  return 'Reviewing study notes for your active flashcard session'
})

function isTargetSlot(slot, idx = 0) {
  if (!isFromFlashcards.value || !slot) return false
  const target = targetPageFromRoute.value
  if (target && slot.start_page > 0 && slot.end_page > 0) {
    return target >= slot.start_page && target <= slot.end_page
  }
  // When navigated from flashcard and either no specific page or single slot / chapter note
  if (sessionSlots.value.length === 1 || idx === 0) {
    return true
  }
  return false
}

function goBackToFlashcards() {
  router.push({
    path: '/flashcards',
    query: {
      taskId: route.query.taskId || undefined,
      notebookId: activeTopic.value?.notebook_id || selectedNotebookID.value || undefined,
    },
  })
}

function canJumpToSource(slot) {
  // Can jump to reader if we have an active topic with a valid notebook
  return Boolean(activeTopic.value?.notebook_id || selectedNotebookID.value)
}

function jumpToReader(slot) {
  const nbId = slot?.notebook_id || activeTopic.value?.notebook_id || selectedNotebookID.value
  const topicId = slot?.topic_id || activeTopic.value?.topic_id || selectedTopicID.value
  const targetPage = slot?.start_page || activeTopic.value?.start_page || 1

  router.push({
    path: '/reader',
    query: {
      notebookId: nbId || undefined,
      topicId: topicId || undefined,
      page: targetPage,
      from: 'notes',
    },
  })
}

function formatTimeAgo(unixSec) {
  if (!unixSec || unixSec <= 0) return 'Never'
  const nowSec = Math.floor(Date.now() / 1000)
  const diffSec = nowSec - unixSec
  if (diffSec < 60) return 'Just now'
  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHours = Math.floor(diffMin / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays === 1) return 'Yesterday'
  if (diffDays < 30) return `${diffDays} days ago`
  const diffMonths = Math.floor(diffDays / 30)
  return `${diffMonths} month${diffMonths > 1 ? 's' : ''} ago`
}

async function loadInitialData() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [nbList, tree, notesRes] = await Promise.all([
      getNotebooks().catch(() => []),
      getNotebookTopicTree().catch(() => []),
      getNotesByNotebook('').catch(() => ({ notes: [] })),
    ])

    notebooks.value = Array.isArray(nbList) ? nbList : []
    notebookTree.value = Array.isArray(tree) ? tree : []

    // Expand all notebooks by default for quick scanning
    const defaultExpanded = new Set()
    for (const nb of notebookTree.value) {
      const nid = nb.notebook_id || nb.id
      if (nid) defaultExpanded.add(nid)
    }
    expandedNotebooks.value = defaultExpanded

    const nMap = {}
    if (notesRes && Array.isArray(notesRes.notes)) {
      for (const n of notesRes.notes) {
        if (n.topic_id) {
          if (!nMap[n.topic_id]) {
            nMap[n.topic_id] = []
          }
          nMap[n.topic_id].push(n)
        }
      }
    }
    notesMap.value = nMap

    // Check query params for initial selection
    const queryTopic = route.query.topicId || route.query.topic_id
    const queryNotebook = route.query.notebookId || route.query.notebook_id
    if (queryNotebook) {
      selectedNotebookID.value = queryNotebook
      expandedNotebooks.value.add(queryNotebook)
    }
    if (queryTopic) {
      const found = allTopics.value.find((t) => t.topic_id === queryTopic)
      if (found) {
        selectTopic(found)
      } else {
        selectedTopicID.value = queryTopic
        loadTopicSlots(queryTopic)
      }
    } else if (queryNotebook) {
      const nbTopic = allTopics.value.find((t) => t.notebook_id === queryNotebook)
      if (nbTopic) {
        selectTopic(nbTopic)
      } else {
        selectedTopicID.value = queryNotebook
        loadTopicSlots(queryNotebook)
      }
    } else if (allTopics.value.length > 0 && !selectedTopicID.value) {
      selectTopic(allTopics.value[0])
    }
  } catch (err) {
    console.error('[NOTES] Failed to load data:', err)
    errorMsg.value = 'Failed to load study notes data.'
  } finally {
    loading.value = false
  }
}

async function selectTopic(topic) {
  if (!topic || !topic.topic_id) return
  selectedTopicID.value = topic.topic_id
  if (topic.notebook_id) {
    selectedNotebookID.value = topic.notebook_id
    expandedNotebooks.value.add(topic.notebook_id)
  }
  cancelEditSlot()
  errorMsg.value = ''
  successMsg.value = ''
  await loadTopicSlots(topic.topic_id)
}

async function loadTopicSlots(topicID) {
  loadingSlots.value = true
  try {
    const res = await getTopicStudyNoteSlots(topicID)
    const dbSlots = (res && Array.isArray(res.slots)) ? res.slots : []
    sessionSlots.value = dbSlots

    if (targetPageFromRoute.value && dbSlots.length > 0) {
      nextTick(() => {
        const targetSlot = dbSlots.find((s) => isTargetSlot(s))
        if (targetSlot) {
          const el = document.getElementById(`session-slot-${targetSlot.start_page}-${targetSlot.end_page}`)
          if (el) {
            el.scrollIntoView({ behavior: 'smooth', block: 'center' })
          }
        }
      })
    }
  } catch (err) {
    console.warn('[NOTES] Failed to fetch topic slots:', err)
    sessionSlots.value = []
  } finally {
    loadingSlots.value = false
  }
}

function startEditSlot(slot) {
  editingSlotKey.value = getSlotKey(slot)
  editSlotBuffer.value = slot.content || ''
  errorMsg.value = ''
  successMsg.value = ''
}

function cancelEditSlot() {
  editingSlotKey.value = null
  editSlotBuffer.value = ''
}

async function saveSlot(slot) {
  const key = getSlotKey(slot)
  savingSlotKey.value = key
  errorMsg.value = ''
  try {
    const res = await updateTopicStudyNote(
      slot.topic_id,
      slot.start_page,
      slot.end_page,
      editSlotBuffer.value
    )
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    slot.content = editSlotBuffer.value
    editingSlotKey.value = null
    successMsg.value = 'Note saved.'
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (err) {
    console.error('[NOTES] Failed to save slot note:', err)
    errorMsg.value = 'Failed to save note.'
  } finally {
    savingSlotKey.value = null
  }
}

async function generateSlot(slot) {
  const key = getSlotKey(slot)
  generatingSlotKey.value = key
  errorMsg.value = ''
  try {
    const nbID = slot.notebook_id || activeTopic.value?.notebook_id || selectedNotebookID.value || ''
    const res = await generateTopicStudyNoteForRange(
      slot.topic_id,
      nbID,
      slot.start_page,
      slot.end_page
    )
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    if (res && res.note) {
      slot.content = res.note.content
      slot.last_reviewed_at = res.note.last_reviewed_at
      successMsg.value = 'Note generated!'
      setTimeout(() => { successMsg.value = '' }, 3000)
    }
  } catch (err) {
    console.error('[NOTES] Failed to generate slot note:', err)
    errorMsg.value = err.message || 'Failed to generate note.'
  } finally {
    generatingSlotKey.value = null
  }
}

// Add Note state
const showAddNoteCard = ref(false)
const newNoteStartPage = ref(0)
const newNoteEndPage = ref(0)
const newNoteBuffer = ref('')
const savingNewNote = ref(false)

function formatPageRange(startPage, endPage) {
  if ((!startPage && !endPage) || (startPage === 0 && endPage === 0)) {
    return 'Chapter Note'
  }
  if (startPage === endPage) {
    return `Page ${startPage}`
  }
  return `Pages ${startPage}–${endPage}`
}

function openAddNoteCard() {
  showAddNoteCard.value = true
  if (activeTopic.value) {
    newNoteStartPage.value = activeTopic.value.start_page || 0
    newNoteEndPage.value = activeTopic.value.end_page || 0
  } else {
    newNoteStartPage.value = 0
    newNoteEndPage.value = 0
  }
  newNoteBuffer.value = ''
  errorMsg.value = ''
}

function closeAddNoteCard() {
  showAddNoteCard.value = false
  newNoteBuffer.value = ''
}

async function saveNewNote() {
  if (!selectedTopicID.value) return
  if (!newNoteBuffer.value.trim()) {
    errorMsg.value = 'Please enter note content.'
    return
  }
  savingNewNote.value = true
  errorMsg.value = ''
  try {
    const res = await updateTopicStudyNote(
      selectedTopicID.value,
      newNoteStartPage.value || 0,
      newNoteEndPage.value || 0,
      newNoteBuffer.value.trim()
    )
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    showAddNoteCard.value = false
    newNoteBuffer.value = ''
    successMsg.value = 'Note added successfully.'
    setTimeout(() => { successMsg.value = '' }, 3000)
    await loadTopicSlots(selectedTopicID.value)
  } catch (err) {
    console.error('[NOTES] Failed to create custom note:', err)
    errorMsg.value = 'Failed to create note.'
  } finally {
    savingNewNote.value = false
  }
}

async function handleOpenNotesFolder() {
  errorMsg.value = ''
  try {
    const nbID = activeTopic.value?.notebook_id || selectedNotebookID.value || ''
    const tID = selectedTopicID.value || ''
    const res = await openNotesFolder(nbID, tID)
    if (res && res.error) {
      errorMsg.value = res.error
    }
  } catch (err) {
    console.error('[NOTES] Failed to open notes directory:', err)
    errorMsg.value = 'Failed to open notes folder.'
  }
}

async function markSlotReviewed(slot) {
  const key = getSlotKey(slot)
  markingSlotKey.value = key
  errorMsg.value = ''
  try {
    const res = await apiMarkTopicReviewed(slot.topic_id, slot.start_page, slot.end_page)
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    const nowSec = Math.floor(Date.now() / 1000)
    slot.last_reviewed_at = nowSec
    successMsg.value = 'Marked as reviewed.'
    setTimeout(() => { successMsg.value = '' }, 3000)
  } catch (err) {
    console.error('[NOTES] Failed to mark reviewed:', err)
    errorMsg.value = 'Failed to mark reviewed.'
  } finally {
    markingSlotKey.value = null
  }
}

onMounted(() => {
  loadInitialData()
})
</script>

<style scoped>
.notes-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  color: var(--on-surface);
  overflow: hidden;
}

.notes-layout {
  display: flex;
  flex: 1;
  height: 100%;
  overflow: hidden;
  position: relative;
}

/* Left Sidebar */
.notes-sidebar {
  width: 330px;
  min-width: 300px;
  max-width: 380px;
  background: var(--surface-container-low);
  border-right: 1px solid var(--outline-variant);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: width 0.2s ease, opacity 0.2s ease;
}

.notes-sidebar-header {
  padding: 16px 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-bottom: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
}

.sidebar-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.title-with-badge {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sidebar-title {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.notes-count-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
  border: 1px solid var(--outline-variant);
}

.sidebar-toggle-btn {
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-color: transparent;
  color: var(--muted-text);
  border-radius: 6px;
  padding: 4px 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.sidebar-toggle-btn:hover {
  background: var(--surface-container);
  color: var(--on-surface);
  border-color: var(--outline-variant);
}

.sidebar-subtitle {
  margin: -4px 0 2px;
  font-size: 12px;
  color: var(--muted-text);
}

/* Tree list container */
.topics-list-container {
  flex: 1;
  overflow-y: auto;
  padding: 10px 8px;
}

.loading-topics,
.empty-topics {
  padding: 24px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--muted-text);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.spinner-sm {
  width: 20px;
  height: 20px;
  border: 2px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.book-tree-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.book-tree-node {
  display: flex;
  flex-direction: column;
}

/* Book Node Header */
.book-node-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  user-select: none;
  background: color-mix(in srgb, var(--surface-container) 60%, transparent);
  border: 1px solid var(--outline-variant);
  transition: all 0.15s ease;
}

.book-node-header:hover {
  background: var(--surface-container);
  border-color: var(--outline-variant);
}

.chevron-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  padding: 2px;
  cursor: pointer;
  display: flex;
  align-items: center;
}

.book-node-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.book-node-title {
  flex: 1;
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  letter-spacing: -0.01em;
}

.book-progress-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--surface-container-high);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.book-progress-pill.all-ready {
  background: color-mix(in srgb, #10b981 15%, transparent);
  color: #10b981;
  border-color: color-mix(in srgb, #10b981 30%, transparent);
}

/* Chapter Sublist */
.chapter-sublist {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 4px;
  padding-left: 14px;
  border-left: 2px solid color-mix(in srgb, var(--outline-variant) 40%, transparent);
  margin-left: 12px;
}

.chapter-tree-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 7px 10px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-color: transparent;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
  width: 100%;
}

.chapter-tree-item:hover {
  background: var(--surface-container);
  border-color: var(--outline-variant);
}

.chapter-tree-item.active {
  background: var(--surface-container-high);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}

.chapter-item-header {
  display: flex;
  align-items: center;
  gap: 7px;
}

.chapter-item-title {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--on-surface);
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chapter-tree-item.active .chapter-item-title {
  font-weight: 700;
  color: var(--primary);
}

.chapter-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  padding-left: 14px;
}

.chapter-page-tag {
  color: var(--muted-text);
  font-size: 10.5px;
}

.chapter-reviewed-tag {
  color: #10b981;
  font-weight: 600;
  font-size: 10.5px;
}

.chapter-ready-tag {
  color: var(--muted-text);
  font-size: 10.5px;
}

/* Status dots */
.status-dot {
  width: 6.5px;
  height: 6.5px;
  border-radius: 50%;
  flex-shrink: 0;
}

.dot-has-note {
  background: #10b981;
  box-shadow: 0 0 5px rgba(16, 185, 129, 0.4);
}

.dot-missing-note {
  background: var(--outline-variant);
}

/* Right Content Pane */
.notes-content-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  background: var(--surface);
}

.notes-empty-selection {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 40px;
  text-align: center;
  color: var(--muted-text);
}

.empty-icon {
  color: var(--outline-variant);
  margin-bottom: 16px;
}

.notes-empty-selection h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 8px;
}

.notes-empty-selection p {
  font-size: 14px;
  max-width: 400px;
  margin: 0;
  line-height: 1.5;
}

.restore-sidebar-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: var(--surface-container);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 24px;
  transition: all 0.15s ease;
}

.restore-sidebar-btn:hover {
  background: var(--surface-container-high);
  border-color: var(--primary);
}

/* Topic Header */
.note-view-header {
  padding: 20px 32px 16px;
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.note-title-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 800px;
}

.note-meta-badges {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.restore-sidebar-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  font-size: 11.5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.restore-sidebar-pill-btn:hover {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
}

.chip-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11.5px;
  font-weight: 600;
  background: var(--surface-container);
  color: var(--on-surface-variant);
  border: 1px solid var(--outline-variant);
}

.chip-tag.book-chip {
  background: color-mix(in srgb, var(--surface-container-high) 80%, transparent);
  color: var(--on-surface);
}

.chip-tag.info {
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 20%, transparent);
}

.note-topic-title {
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--on-surface);
  line-height: 1.25;
}

.note-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

/* Interop tip & Flashcard Context banners */
.notes-flashcard-context-banner {
  margin: 16px 32px 0;
  padding: 10px 16px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--primary) 14%, var(--surface-container));
  border: 1px solid color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
  color: var(--on-surface);
  font-size: 13px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.notes-flashcard-context-banner .banner-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.notes-flashcard-context-banner .banner-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.banner-return-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 6px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  color: var(--primary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
}

.banner-return-btn:hover {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  border-color: var(--primary);
}

.notes-obsidian-tip {
  margin: 16px 32px 0;
  padding: 10px 14px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--primary) 8%, var(--surface));
  border: 1px solid var(--outline-variant);
  color: var(--on-surface-variant);
  font-size: 12.5px;
  display: flex;
  align-items: center;
  gap: 10px;
  line-height: 1.4;
}

.tip-icon {
  color: var(--primary);
  flex-shrink: 0;
}

.notes-obsidian-tip code {
  background: var(--surface-container-high);
  padding: 1px 5px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 11.5px;
}

/* Banner notifications */
.note-banner {
  margin: 12px 32px 0;
  padding: 10px 14px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
}

.error-banner {
  background: color-mix(in srgb, #ef4444 15%, var(--surface));
  color: #dc2626;
  border: 1px solid var(--outline-variant);
}

.success-banner {
  background: color-mix(in srgb, #10b981 15%, var(--surface));
  color: #059669;
  border: 1px solid var(--outline-variant);
}

/* Session Cards Feed */
.session-cards-feed {
  padding: 20px 32px 40px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 960px;
}

.session-note-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  transition: border-color 0.3s ease, box-shadow 0.3s ease;
}

.session-note-card:hover {
  border-color: color-mix(in srgb, var(--primary) 30%, var(--outline-variant));
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.session-note-card.target-highlight {
  border-color: var(--primary);
}

.session-note-card.target-highlight .session-card-header {
  background: color-mix(in srgb, var(--primary) 12%, var(--surface-container));
  border-bottom-color: color-mix(in srgb, var(--primary) 30%, var(--outline-variant));
  animation: noteTargetHeaderGlow 4s ease-out forwards;
}

@keyframes noteTargetHeaderGlow {
  0% {
    background: color-mix(in srgb, var(--primary) 22%, var(--surface-container));
  }
  60% {
    background: color-mix(in srgb, var(--primary) 14%, var(--surface-container));
  }
  100% {
    background: color-mix(in srgb, var(--primary) 8%, var(--surface-container));
  }
}

.session-target-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--primary) 18%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 40%, transparent);
  font-size: 11.5px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.session-card-header {
  padding: 12px 18px;
  background: var(--surface-container);
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  transition: background 0.3s ease;
}

.session-card-badge-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.session-index-pill {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--surface-container-high);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
}

.session-index-pill.new-badge {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.session-range-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--muted-text);
}

.jump-source-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
  font-size: 11.5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.jump-source-pill-btn:hover {
  background: color-mix(in srgb, var(--primary) 22%, transparent);
  border-color: var(--primary);
  transform: translateY(-1px);
}

.jump-source-pill-btn:active {
  transform: scale(0.96);
}

.back-flashcards-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
  font-weight: 600;
}

.back-flashcards-btn:hover {
  background: color-mix(in srgb, var(--primary) 22%, transparent);
}

.session-review-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: #10b981;
}

.session-card-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.icon-btn-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  background: var(--surface-container-high);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  cursor: pointer;
  transition: all 0.15s ease;
}

.icon-btn-pill:hover:not(:disabled) {
  background: var(--surface-container-highest);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
}

.icon-btn-pill.highlight {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.icon-btn-pill.highlight:hover:not(:disabled) {
  background: color-mix(in srgb, var(--primary) 25%, transparent);
}

.icon-btn-pill.success {
  background: color-mix(in srgb, #10b981 12%, transparent);
  color: #10b981;
  border-color: color-mix(in srgb, #10b981 25%, transparent);
}

.icon-btn-pill.success:hover:not(:disabled) {
  background: color-mix(in srgb, #10b981 20%, transparent);
}

/* Card Body */
.session-card-body {
  padding: 18px 20px;
}

.slot-markdown-content {
  line-height: 1.6;
  font-size: 13.5px;
  color: var(--on-surface);
}

.slot-editor-box {
  width: 100%;
}

.slot-textarea {
  width: 100%;
  padding: 12px 14px;
  border-radius: 8px;
  font-size: 13.5px;
  line-height: 1.6;
  font-family: inherit;
  background: var(--surface-container);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  outline: none;
  resize: vertical;
  box-sizing: border-box;
}

.slot-textarea:focus {
  border-color: var(--primary);
}

.page-range-inputs {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted-text);
}

.page-input {
  width: 60px;
  padding: 3px 6px;
  border-radius: 6px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  font-size: 12px;
  outline: none;
}

.slot-empty-state {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  padding: 8px 0;
}

.empty-hint {
  font-size: 13px;
  color: var(--muted-text);
  margin: 0;
}

.slot-empty-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slots-loading-state,
.generating-slot-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  gap: 12px;
  color: var(--muted-text);
  font-size: 13px;
}

.empty-box-card {
  padding: 36px 20px;
  border: 1px dashed var(--outline-variant);
  border-radius: 12px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.empty-box-icon {
  color: var(--outline-variant);
  margin-bottom: 12px;
}

.empty-box-card h3 {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface);
}

.empty-box-card p {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

/* Spinner */
.spinner {
  width: 24px;
  height: 24px;
  border: 3px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Buttons */
.primary-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 8px;
  background: var(--primary);
  color: var(--on-primary, #ffffff);
  border: none;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s ease;
}

.primary-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.primary-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.secondary-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  background: var(--surface-container);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.secondary-btn:hover:not(:disabled) {
  background: var(--surface-container-high);
}

.secondary-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 5px 10px;
  font-size: 12px;
}

/* Shared Search Input */
.shared-search-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.shared-search-wrapper input {
  width: 100%;
  padding: 7px 28px 7px 28px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  font-size: 12.5px;
  outline: none;
  box-sizing: border-box;
}

.shared-search-wrapper input:focus {
  border-color: var(--primary);
}

.shared-search-wrapper .search-icon {
  position: absolute;
  left: 8px;
  color: var(--muted-text);
  pointer-events: none;
}

.clear-search-btn {
  position: absolute;
  right: 6px;
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
}
</style>
