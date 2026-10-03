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

      <!-- Right Pane: Note Viewer / Editor -->
      <main class="notes-content-pane">
        <template v-if="!selectedTopicID">
          <div class="notes-empty-selection">
            <BaseIcon name="book" size="48" class="empty-icon" />
            <h3>Select a Topic</h3>
            <p>Choose a chapter or topic from the sidebar to view, edit, or generate high-yield study notes.</p>
          </div>
        </template>

        <template v-else>
          <!-- Note Header -->
          <header class="note-view-header">
            <div class="note-title-group">
              <div class="note-meta-badges">
                <span v-if="activeTopic?.notebook_title" class="chip-tag">
                  {{ activeTopic.notebook_title }}
                </span>
                <span v-if="activeNote?.last_reviewed_at" class="chip-tag success">
                  <BaseIcon name="clock" size="12" />
                  <span>Last reviewed {{ formatTimeAgo(activeNote.last_reviewed_at) }}</span>
                </span>
                <span v-else class="chip-tag">
                  <BaseIcon name="circle" size="10" />
                  <span>Never reviewed</span>
                </span>
              </div>
              <h1 class="note-topic-title">{{ activeTopic?.title || selectedTopicID }}</h1>
            </div>

            <div class="note-actions">
              <template v-if="isEditing">
                <button
                  type="button"
                  class="secondary-btn"
                  :disabled="saving"
                  @click="cancelEdit"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  class="primary-btn"
                  :disabled="saving"
                  @click="saveNote"
                >
                  {{ saving ? 'Saving...' : 'Save Note' }}
                </button>
              </template>

              <template v-else>
                <button
                  v-if="hasCurrentNote"
                  type="button"
                  class="secondary-btn"
                  :disabled="generating || saving"
                  title="Edit note text"
                  @click="startEdit"
                >
                  <BaseIcon name="edit" size="14" />
                  <span>Edit Note</span>
                </button>

                <button
                  type="button"
                  class="primary-btn"
                  :disabled="generating || saving"
                  :title="hasCurrentNote ? 'Regenerate study note using AI' : 'Generate structured study note'"
                  @click="generateNote"
                >
                  <BaseIcon name="sparkles" size="14" />
                  <span>{{ generating ? 'Generating...' : (hasCurrentNote ? 'Regenerate' : 'Generate Note') }}</span>
                </button>

                <button
                  v-if="hasCurrentNote"
                  type="button"
                  class="secondary-btn"
                  :disabled="marking"
                  title="Mark note as reviewed today"
                  @click="markReviewed"
                >
                  <BaseIcon name="check-circle" size="14" />
                  <span>{{ marking ? 'Marking...' : 'Mark Reviewed' }}</span>
                </button>
              </template>
            </div>
          </header>

          <!-- Banner messages -->
          <div v-if="errorMsg" class="note-banner error-banner">
            <BaseIcon name="alert-triangle" size="16" />
            <span>{{ errorMsg }}</span>
          </div>

          <div v-if="successMsg" class="note-banner success-banner">
            <BaseIcon name="check" size="16" />
            <span>{{ successMsg }}</span>
          </div>

          <!-- Note Body -->
          <div class="note-body-wrapper">
            <!-- Generating State -->
            <div v-if="generating" class="generating-state">
              <div class="spinner generating-spinner"></div>
              <h3>Generating Structured Study Note...</h3>
              <p>Extracting high-yield concepts, mechanisms, and rules of thumb from compressed chapter text.</p>
            </div>

            <!-- Edit Mode -->
            <div v-else-if="isEditing" class="edit-mode-container">
              <div class="editor-header">
                <span class="editor-tip">Markdown supported. Keep summaries concise (~150 words).</span>
              </div>
              <textarea
                v-model="editBuffer"
                class="shared-textarea"
                placeholder="Write your study note or summary here (Markdown supported)..."
                rows="18"
              ></textarea>
            </div>

            <!-- Display Mode -->
            <div v-else-if="hasCurrentNote" class="markdown-view-container shared-markdown-content">
              <div class="note-rendered-markdown" v-html="renderedNoteContent"></div>
            </div>

            <!-- No Note Yet Empty State -->
            <div v-else class="empty-box-card">
              <BaseIcon name="file-text" size="40" class="empty-box-icon" />
              <h3>No Study Note Yet</h3>
              <p>Generate a 100–150 word high-yield summary from your chapter text, or write your own notes.</p>
              <div class="no-note-actions">
                <button
                  type="button"
                  class="primary-btn"
                  :disabled="generating"
                  @click="generateNote"
                >
                  <BaseIcon name="sparkles" size="14" />
                  <span>Generate with AI</span>
                </button>
                <button
                  type="button"
                  class="secondary-btn"
                  @click="startEdit"
                >
                  <BaseIcon name="edit" size="14" />
                  <span>Write Note Manually</span>
                </button>
              </div>
            </div>
          </div>
        </template>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseIcon from '../components/BaseIcon.vue'
import {
  getNotebooks,
  getNotebookTopicTree,
  getNotesByNotebook,
  getTopicStudyNote,
  generateTopicStudyNote,
  updateTopicStudyNote,
  markTopicReviewed as apiMarkTopicReviewed,
} from '../services/appApi'
import { renderMarkdown } from '../services/markdown'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const generating = ref(false)
const saving = ref(false)
const marking = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const notebooks = ref([])
const notebookTree = ref([])
const selectedNotebookID = ref('')
const selectedTopicID = ref('')
const searchQuery = ref('')

const notesMap = ref({}) // topicID -> TopicStudyNote
const activeNote = ref(null)
const isEditing = ref(false)
const editBuffer = ref('')

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

const hasCurrentNote = computed(() => {
  return Boolean(activeNote.value && activeNote.value.content && activeNote.value.content.trim().length > 0)
})

const renderedNoteContent = computed(() => {
  if (!activeNote.value || !activeNote.value.content) return ''
  return renderMarkdown(activeNote.value.content)
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
        loadTopicNote(queryTopic)
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
  isEditing.value = false
  errorMsg.value = ''
  successMsg.value = ''
  await loadTopicNote(topic.topic_id)
}

async function loadTopicNote(topicID) {
  try {
    const res = await getTopicStudyNote(topicID)
    if (res && res.note) {
      activeNote.value = res.note
      notesMap.value[topicID] = res.note
    } else {
      activeNote.value = null
    }
  } catch (err) {
    console.warn('[NOTES] Failed to fetch topic note:', err)
    activeNote.value = null
  }
}

function startEdit() {
  editBuffer.value = activeNote.value?.content || ''
  isEditing.value = true
  errorMsg.value = ''
  successMsg.value = ''
}

function cancelEdit() {
  isEditing.value = false
  editBuffer.value = ''
  errorMsg.value = ''
}

async function saveNote() {
  if (!selectedTopicID.value) return
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await updateTopicStudyNote(selectedTopicID.value, editBuffer.value)
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    if (res && res.note) {
      activeNote.value = res.note
      notesMap.value[selectedTopicID.value] = res.note
    } else {
      if (activeNote.value) {
        activeNote.value.content = editBuffer.value
      } else {
        activeNote.value = {
          topic_id: selectedTopicID.value,
          notebook_id: activeTopic.value?.notebook_id || '',
          content: editBuffer.value,
          last_reviewed_at: 0,
        }
      }
      notesMap.value[selectedTopicID.value] = activeNote.value
    }
    isEditing.value = false
    successMsg.value = 'Note saved successfully.'
    setTimeout(() => {
      successMsg.value = ''
    }, 3000)
  } catch (err) {
    console.error('[NOTES] Failed to save note:', err)
    errorMsg.value = 'Failed to save note.'
  } finally {
    saving.value = false
  }
}

async function generateNote() {
  if (!selectedTopicID.value) return
  generating.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const notebookID = activeTopic.value?.notebook_id || selectedNotebookID.value || ''
    const res = await generateTopicStudyNote(selectedTopicID.value, notebookID)
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    if (res && res.note) {
      activeNote.value = res.note
      notesMap.value[selectedTopicID.value] = res.note
      successMsg.value = 'Study note generated!'
      setTimeout(() => {
        successMsg.value = ''
      }, 3000)
    }
  } catch (err) {
    console.error('[NOTES] Failed to generate note:', err)
    errorMsg.value = err.message || 'Failed to generate study note.'
  } finally {
    generating.value = false
  }
}

async function markReviewed() {
  if (!selectedTopicID.value) return
  marking.value = true
  errorMsg.value = ''
  try {
    const res = await apiMarkTopicReviewed(selectedTopicID.value)
    if (res && res.error) {
      errorMsg.value = res.error
      return
    }
    const nowSec = Math.floor(Date.now() / 1000)
    if (activeNote.value) {
      activeNote.value.last_reviewed_at = nowSec
    }
    if (notesMap.value[selectedTopicID.value]) {
      notesMap.value[selectedTopicID.value].last_reviewed_at = nowSec
    }
    successMsg.value = 'Marked as reviewed.'
    setTimeout(() => {
      successMsg.value = ''
    }, 3000)
  } catch (err) {
    console.error('[NOTES] Failed to mark reviewed:', err)
    errorMsg.value = 'Failed to mark reviewed.'
  } finally {
    marking.value = false
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

/* Note Header */
.note-view-header {
  padding: 24px 32px 16px;
  border-bottom: 1px solid var(--outline-variant);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
}

.note-title-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 700px;
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

.note-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* Banners */
.note-banner {
  margin: 12px 32px 0;
}

/* Note Body */
.note-body-wrapper {
  padding: 24px 32px 40px;
  flex: 1;
}

.generating-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  color: var(--muted-text);
}

.generating-spinner {
  width: 36px;
  height: 36px;
  margin-bottom: 16px;
}

.generating-state h3 {
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 6px;
}

.generating-state p {
  font-size: 13px;
  max-width: 440px;
  margin: 0;
  line-height: 1.5;
}

/* Edit Mode */
.edit-mode-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.editor-header {
  font-size: 12px;
  color: var(--muted-text);
}

/* Markdown View */
.markdown-view-container {
  max-width: 800px;
}

.no-note-actions {
  display: flex;
  gap: 10px;
}
</style>
