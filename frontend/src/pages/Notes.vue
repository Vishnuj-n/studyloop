<template>
  <div class="notes-page">
    <div class="notes-layout">
      <!-- Left Sidebar: Topic / Notebook Navigator -->
      <aside class="notes-sidebar">
        <div class="notes-sidebar-header">
          <div class="sidebar-title-row">
            <h2 class="sidebar-title">Study Notes</h2>
            <span class="notes-count-badge">{{ totalNotesCount }} Notes</span>
          </div>
          <p class="sidebar-subtitle">Knowledge base & topic summaries</p>

          <!-- Notebook Selector -->
          <div class="notebook-filter-group">
            <label class="filter-label" for="notes-notebook-select">Notebook</label>
            <select
              id="notes-notebook-select"
              v-model="selectedNotebookID"
              class="notes-select"
              :disabled="loading"
              @change="onNotebookChange"
            >
              <option value="">All Notebooks</option>
              <option
                v-for="nb in notebooks"
                :key="nb.id"
                :value="nb.id"
              >
                {{ nb.title }}
              </option>
            </select>
          </div>

          <!-- Search Box -->
          <div class="shared-search-wrapper">
            <BaseIcon name="search" size="14" class="search-icon" />
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search topics or notes..."
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

        <!-- Topic List -->
        <div class="topics-list-container">
          <div v-if="loading" class="loading-topics">
            <span>Loading notes...</span>
          </div>

          <div v-else-if="filteredTopics.length === 0" class="empty-topics">
            <p v-if="searchQuery">No topics match "{{ searchQuery }}"</p>
            <p v-else>No topics found for this notebook.</p>
          </div>

          <div v-else class="topics-scroll-list">
            <button
              v-for="item in filteredTopics"
              :key="item.topic_id"
              type="button"
              class="topic-list-item"
              :class="{ active: selectedTopicID === item.topic_id }"
              @click="selectTopic(item)"
            >
              <div class="topic-item-header">
                <span class="topic-item-title">{{ item.title }}</span>
                <span
                  v-if="hasNote(item.topic_id)"
                  class="status-dot dot-has-note"
                  title="Study Note Available"
                ></span>
                <span
                  v-else
                  class="status-dot dot-missing-note"
                  title="No Study Note Yet"
                ></span>
              </div>
              <div class="topic-item-meta">
                <span v-if="item.notebook_title" class="topic-nb-tag">{{ item.notebook_title }}</span>
                <span v-if="item.last_reviewed_text" class="topic-reviewed-tag">
                  Reviewed {{ item.last_reviewed_text }}
                </span>
                <span v-else-if="hasNote(item.topic_id)" class="topic-not-reviewed-tag">
                  Not reviewed
                </span>
              </div>
            </button>
          </div>
        </div>
      </aside>

      <!-- Right Pane: Session Cards PPT-Style Feed -->
      <main class="notes-content-pane">
        <template v-if="!selectedTopicID">
          <div class="notes-empty-selection">
            <BaseIcon name="book" size="48" class="empty-icon" />
            <h3>Select a Chapter</h3>
            <p>Choose a chapter from the sidebar to view, edit, or generate per-session study notes.</p>
          </div>
        </template>

        <template v-else>
          <!-- Topic Header -->
          <header class="note-view-header">
            <div class="note-title-group">
              <div class="note-meta-badges">
                <span v-if="activeTopic?.notebook_title" class="chip-tag">
                  {{ activeTopic.notebook_title }}
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

          <!-- Interop Tip Callout -->
          <div class="notes-obsidian-tip">
            <BaseIcon name="info" size="14" class="tip-icon" />
            <span>
              <strong>Obsidian & Logseq Compatible:</strong> Notes are synced to <code>dev_data/notes/</code>. Use <em>Open Folder</em> to add images (<code>![alt](./assets/pic.png)</code>) or edit notes in external markdown tools.
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
              :key="`${slot.start_page}-${slot.end_page}`"
              class="session-note-card"
            >
              <!-- Card Header -->
              <div class="session-card-header">
                <div class="session-card-badge-row">
                  <span class="session-index-pill">Session {{ idx + 1 }}</span>
                  <span class="session-range-pill">
                    <BaseIcon name="book-open" size="12" />
                    <span>{{ formatPageRange(slot.start_page, slot.end_page) }}</span>
                  </span>
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
                      <span>{{ generatingSlotKey === getSlotKey(slot) ? 'Generating...' : (slot.content ? 'Regenerate' : 'Generate Note') }}</span>
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
import { ref, computed, onMounted } from 'vue'
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

// Flat topic list with notebook title & note status
const allTopics = computed(() => {
  const list = []
  for (const nb of notebookTree.value) {
    const nbTitle = nb.title || 'Untitled Notebook'
    const nbID = nb.notebook_id || nb.id
    if (Array.isArray(nb.topics)) {
      for (const t of nb.topics) {
        const tid = t.topic_id || t.id
        const note = notesMap.value[tid]
        let reviewedText = ''
        if (note && note.last_reviewed_at > 0) {
          reviewedText = formatTimeAgo(note.last_reviewed_at)
        }
        list.push({
          topic_id: tid,
          title: t.title || tid,
          notebook_id: nbID,
          notebook_title: nbTitle,
          start_page: t.start_page || 0,
          end_page: t.end_page || 0,
          last_reviewed_at: note?.last_reviewed_at || 0,
          last_reviewed_text: reviewedText,
          has_note: Boolean(note && note.content && note.content.trim().length > 0),
        })
      }
    }
  }
  return list
})

const filteredTopics = computed(() => {
  let list = allTopics.value

  if (selectedNotebookID.value) {
    list = list.filter((t) => t.notebook_id === selectedNotebookID.value)
  }

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase().trim()
    list = list.filter((t) => {
      const titleMatch = t.title.toLowerCase().includes(q)
      const nbMatch = t.notebook_title.toLowerCase().includes(q)
      const noteContent = notesMap.value[t.topic_id]?.content || ''
      const contentMatch = noteContent.toLowerCase().includes(q)
      return titleMatch || nbMatch || contentMatch
    })
  }

  return list
})

const totalNotesCount = computed(() => {
  return Object.values(notesMap.value).filter((n) => n && n.content && n.content.trim().length > 0).length
})

const activeTopic = computed(() => {
  if (!selectedTopicID.value) return null
  return allTopics.value.find((t) => t.topic_id === selectedTopicID.value) || {
    topic_id: selectedTopicID.value,
    title: selectedTopicID.value,
    notebook_id: selectedNotebookID.value,
  }
})

function hasNote(topicID) {
  const note = notesMap.value[topicID]
  return Boolean(note && note.content && note.content.trim().length > 0)
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

    const nMap = {}
    if (notesRes && Array.isArray(notesRes.notes)) {
      for (const n of notesRes.notes) {
        if (n.topic_id) {
          nMap[n.topic_id] = n
        }
      }
    }
    notesMap.value = nMap

    // Check query params for initial selection
    const queryTopic = route.query.topicId || route.query.topic_id
    const queryNotebook = route.query.notebookId || route.query.notebook_id
    if (queryNotebook) {
      selectedNotebookID.value = queryNotebook
    }
    if (queryTopic) {
      const found = allTopics.value.find((t) => t.topic_id === queryTopic)
      if (found) {
        selectTopic(found)
      } else {
        selectedTopicID.value = queryTopic
        loadTopicSlots(queryTopic)
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

async function onNotebookChange() {
  if (selectedTopicID.value) {
    const exists = filteredTopics.value.some((t) => t.topic_id === selectedTopicID.value)
    if (!exists && filteredTopics.value.length > 0) {
      selectTopic(filteredTopics.value[0])
    }
  }
}

async function selectTopic(topic) {
  if (!topic || !topic.topic_id) return
  selectedTopicID.value = topic.topic_id
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
}

/* Left Sidebar */
.notes-sidebar {
  width: 320px;
  min-width: 300px;
  max-width: 360px;
  background: var(--surface-container-low);
  border-right: 1px solid var(--outline-variant);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.notes-sidebar-header {
  padding: 20px 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  border-bottom: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
}

.sidebar-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sidebar-title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.notes-count-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
}

.sidebar-subtitle {
  margin: -6px 0 0;
  font-size: 12px;
  color: var(--muted-text);
}

.notebook-filter-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.filter-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted-text);
}

.notes-select {
  width: 100%;
  padding: 8px 10px;
  font-size: 13px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  outline: none;
}

.notes-select:focus {
  border-color: var(--primary);
}

/* Topics list */
.topics-list-container {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.loading-topics,
.empty-topics {
  padding: 24px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--muted-text);
}

.topics-scroll-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.topic-list-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border-radius: 8px;
  background: transparent;
  border: 1px solid transparent;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
  width: 100%;
}

.topic-list-item:hover {
  background: var(--surface-container);
  border-color: var(--outline-variant);
}

.topic-list-item.active {
  background: var(--surface-container-high);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
}

.topic-item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.topic-item-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topic-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}

.topic-nb-tag {
  color: var(--muted-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 130px;
}

.topic-reviewed-tag {
  color: #10b981;
  font-weight: 500;
}

.topic-not-reviewed-tag {
  color: var(--muted-text);
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

/* Topic Header */
.note-view-header {
  padding: 24px 32px 16px;
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.note-title-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 800px;
}

.note-meta-badges {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.note-topic-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--on-surface);
}

.note-header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.open-folder-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 8px;
  font-weight: 600;
}

/* Obsidian / Logseq Tip */
.notes-obsidian-tip {
  margin: 14px 32px 0;
  padding: 10px 16px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--primary) 8%, var(--surface-container));
  border: 1px solid color-mix(in srgb, var(--primary) 20%, transparent);
  font-size: 12px;
  line-height: 1.4;
  color: var(--on-surface-variant);
  display: flex;
  align-items: center;
  gap: 10px;
}

.notes-obsidian-tip code {
  font-family: monospace;
  background: var(--surface-container-high);
  padding: 2px 5px;
  border-radius: 4px;
  font-size: 11px;
}

.tip-icon {
  color: var(--primary);
  flex-shrink: 0;
}

/* Banners */
.note-banner {
  margin: 12px 32px 0;
}

/* Session Cards PPT-Style Feed */
.session-cards-feed {
  padding: 24px 32px 48px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 860px;
}

.slots-loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px;
  color: var(--muted-text);
  gap: 12px;
}

/* Individual Session Card (Presentation / Deck Style) */
.session-note-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.session-note-card:hover {
  border-color: color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.07);
}

/* Card Header */
.session-card-header {
  padding: 14px 20px;
  background: var(--surface-container);
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.session-card-badge-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.session-index-pill {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 3px 8px;
  border-radius: 6px;
  background: var(--surface-container-highest);
  color: var(--on-surface);
}

.session-range-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 25%, transparent);
}

.session-review-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  color: #10b981;
}

.session-card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.icon-btn-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  padding: 5px 12px;
  border-radius: 8px;
  background: var(--surface-container-high);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  cursor: pointer;
  transition: all 0.15s ease;
}

.icon-btn-pill:hover:not(:disabled) {
  background: var(--surface-container-highest);
  border-color: var(--outline);
}

.icon-btn-pill.highlight {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
  border-color: color-mix(in srgb, var(--primary) 35%, transparent);
}

.icon-btn-pill.highlight:hover:not(:disabled) {
  background: color-mix(in srgb, var(--primary) 25%, transparent);
}

.icon-btn-pill.success {
  background: color-mix(in srgb, #10b981 12%, transparent);
  color: #10b981;
  border-color: color-mix(in srgb, #10b981 30%, transparent);
}

.icon-btn-pill:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Card Body */
.session-card-body {
  padding: 20px 24px;
}

.slot-markdown-content {
  font-size: 14px;
  line-height: 1.65;
}

.slot-markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  margin: 12px 0;
  display: block;
}

.slot-markdown-content :deep(video) {
  max-width: 100%;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  margin: 12px 0;
}

.generating-slot-state {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 0;
  color: var(--muted-text);
  font-size: 13px;
}

.slot-editor-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.slot-textarea {
  width: 100%;
  font-size: 14px;
  line-height: 1.5;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  resize: vertical;
}

.slot-empty-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  gap: 16px;
  flex-wrap: wrap;
}

.empty-hint {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
}

.slot-empty-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.add-note-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 8px;
  font-weight: 600;
}

.add-note-new-card {
  border-color: color-mix(in srgb, var(--primary) 40%, var(--outline-variant));
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.new-badge {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 40%, transparent);
}

.page-range-inputs {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--muted-text);
  margin-left: 6px;
}

.page-input {
  width: 58px;
  padding: 3px 6px;
  font-size: 12px;
  border-radius: 6px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--on-surface);
  text-align: center;
  outline: none;
}

.page-input:focus {
  border-color: var(--primary);
}

.btn-sm {
  padding: 5px 12px;
  font-size: 12px;
}
</style>
