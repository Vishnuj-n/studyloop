<template>
  <article class="panel form-grid">
    <!-- Master Developer Mode Toggle -->
    <div class="dev-toggle-card">
      <SettingsToggle
        v-model="devModeEnabled"
        title="Enable Developer Mode"
        hint="Unlocks raw queue inspection, page range diagnostics, and system log details."
      />
    </div>

    <!-- Developer Tools & History Panel -->
    <div v-if="devModeEnabled" class="dev-content-box animate-fade-in">
      <!-- LLM Prompt Logging Toggle Card -->
      <div class="dev-toggle-card">
        <SettingsToggle
          v-model="llmPromptLogEnabled"
          title="Enable LLM Prompt Logging"
          hint="Appends raw LLM prompt inputs and model parameters to dev_data/logs/llm_prompt.log in real time."
        />
      </div>

      <!-- Quick Developer Actions Card -->
      <div class="dev-actions-card">
        <div class="actions-header">
          <h4>Developer Shortcuts</h4>
          <p class="hint">Open SQLite database directory (dev_data / %APPDATA%) or logs folder in File Explorer.</p>
        </div>
        <div class="actions-buttons">
          <button type="button" class="action-btn" @click="handleOpenDataDir('')">
            📁 Open Data Directory
          </button>
          <button type="button" class="action-btn secondary" @click="handleOpenDataDir('logs')">
            📁 Open Logs Directory
          </button>
        </div>
      </div>

      <div class="section-title-row">
        <div>
          <h3>Reading Queue &amp; Task History</h3>
          <p class="hint">
            Inspect historical reading sessions, bound page ranges, and state anomalies across textbooks.
          </p>
        </div>
        <button
          type="button"
          class="refresh-btn"
          :disabled="loading"
          @click="resetAndFetch"
        >
          {{ loading ? 'Refreshing...' : 'Refresh Logs' }}
        </button>
      </div>

      <!-- Error alert -->
      <div v-if="error" class="error-banner">
        {{ error }}
      </div>

      <!-- Reading Logs Table -->
      <div class="logs-table-container">
        <table class="logs-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>Notebook / Topic</th>
              <th>Type</th>
              <th>Page Range</th>
              <th>Cursor</th>
              <th>Diagnostics</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="logs.length === 0 && !loading">
              <td colspan="6" class="empty-state">
                No reading history tasks found in database.
              </td>
            </tr>

            <tr
              v-for="log in logs"
              :key="log.task_id"
              :class="{ 'anomaly-row': log.has_anomaly }"
            >
              <td>
                <span :class="['status-badge', log.status.toLowerCase()]">
                  {{ log.status }}
                </span>
              </td>
              <td class="topic-col">
                <div class="notebook-title">{{ log.notebook_title }}</div>
                <div class="topic-title">{{ log.topic_title }}</div>
              </td>
              <td>
                <span class="type-tag">{{ log.task_type }}</span>
              </td>
              <td class="page-range">
                <strong>P. {{ log.start_page }} – {{ log.end_page }}</strong>
              </td>
              <td class="page-cursor">
                <span>P. {{ log.current_page }}</span>
              </td>
              <td>
                <span v-if="log.has_anomaly" class="anomaly-tag" :title="log.anomaly_reason">
                  ⚠ {{ log.anomaly_reason }}
                </span>
                <span v-else class="ok-tag">
                  ✓ Valid Bounds
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Footer Pagination / Load More -->
      <div class="table-footer">
        <span class="count-info">
          Showing {{ logs.length }} of {{ totalCount }} sessions
        </span>
        <button
          v-if="hasMore"
          type="button"
          class="load-more-btn"
          :disabled="loading"
          @click="fetchMore"
        >
          {{ loading ? 'Loading...' : 'Load More (50 Older Sessions)' }}
        </button>
      </div>
    </div>
  </article>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import SettingsToggle from './SettingsToggle.vue'
import {
  getReadingTaskHistory,
  getLLMPromptLogging,
  setLLMPromptLogging,
  openDataDirectory,
} from '../services/appApi'

const DEV_MODE_STORAGE_KEY = 'studyloop_dev_mode_enabled'

const devModeEnabled = ref(localStorage.getItem(DEV_MODE_STORAGE_KEY) === 'true')
const llmPromptLogEnabled = ref(false)
const logs = ref([])
const totalCount = ref(0)
const hasMore = ref(false)
const offset = ref(0)
const limit = 50
const loading = ref(false)
const error = ref('')

async function handleOpenDataDir(subDir = '') {
  error.value = ''
  const res = await openDataDirectory(subDir)
  if (res?.error) {
    error.value = res.error
  }
}

onMounted(async () => {
  if (devModeEnabled.value) {
    resetAndFetch()
  }
  try {
    llmPromptLogEnabled.value = await getLLMPromptLogging()
  } catch (err) {
    console.warn('Failed fetching LLM prompt logging status:', err)
  }
})

watch(llmPromptLogEnabled, async (newVal) => {
  try {
    const updated = await setLLMPromptLogging(newVal)
    if (typeof updated === 'boolean') {
      llmPromptLogEnabled.value = updated
    }
  } catch (err) {
    console.warn('Failed setting LLM prompt logging:', err)
  }
})

watch(devModeEnabled, (newVal) => {
  localStorage.setItem(DEV_MODE_STORAGE_KEY, String(newVal))
  if (newVal && logs.value.length === 0) {
    resetAndFetch()
  }
})

async function resetAndFetch() {
  offset.value = 0
  logs.value = []
  await fetchLogs()
}

async function fetchLogs() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await getReadingTaskHistory('', limit, offset.value)
    if (res.error) {
      error.value = res.error
      return
    }
    if (Array.isArray(res.records)) {
      if (offset.value === 0) {
        logs.value = res.records
      } else {
        logs.value = [...logs.value, ...res.records]
      }
    }
    totalCount.value = res.total_count || 0
    hasMore.value = Boolean(res.has_more)
  } catch (err) {
    console.error('Failed loading reading history:', err)
    error.value = err.message || 'Failed to fetch reading logs'
  } finally {
    loading.value = false
  }
}

async function fetchMore() {
  if (!hasMore.value || loading.value) return
  offset.value += limit
  await fetchLogs()
}
</script>

<style scoped>
.dev-toggle-card {
  padding: 16px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
}

.dev-actions-card {
  padding: 16px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.actions-header h4 {
  margin: 0 0 2px;
  font-size: 14px;
  font-family: 'Manrope', sans-serif;
}

.actions-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 8px;
  border: 1px solid var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  color: var(--primary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover {
  background: var(--primary);
  color: #ffffff;
}

.action-btn.secondary {
  border-color: var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface);
}

.action-btn.secondary:hover {
  background: var(--surface-container-highest);
  color: var(--primary);
}

.dev-content-box {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--outline-variant);
}

.section-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section-title-row h3 {
  margin: 0 0 2px;
  font-size: 16px;
  font-family: 'Manrope', sans-serif;
}

.refresh-btn {
  padding: 8px 14px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.refresh-btn:hover:not(:disabled) {
  background: var(--surface-container-highest);
  color: var(--primary);
}

.error-banner {
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid #ef4444;
  color: #ef4444;
  border-radius: 8px;
  font-size: 13px;
}

.logs-table-container {
  overflow-x: auto;
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  background: var(--surface-container-lowest);
}

.logs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  text-align: left;
}

.logs-table th {
  background: var(--surface-container-low);
  padding: 10px 14px;
  font-weight: 700;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted-text);
  border-bottom: 1px solid var(--outline-variant);
}

.logs-table td {
  padding: 12px 14px;
  border-bottom: 1px solid var(--outline-variant);
  vertical-align: middle;
}

.logs-table tr:last-child td {
  border-bottom: none;
}

.anomaly-row {
  background: rgba(239, 68, 68, 0.04);
}

.status-badge {
  display: inline-block;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.status-badge.completed {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.status-badge.active {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.status-badge.pending {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.status-badge.skipped {
  background: rgba(156, 163, 175, 0.15);
  color: #9ca3af;
}

.notebook-title {
  font-weight: 600;
  color: var(--on-surface);
}

.topic-title {
  font-size: 11px;
  color: var(--muted-text);
  margin-top: 2px;
}

.type-tag {
  font-size: 11px;
  font-family: monospace;
  background: var(--surface-container);
  padding: 2px 6px;
  border-radius: 4px;
}

.anomaly-tag {
  color: #ef4444;
  font-size: 11px;
  font-weight: 600;
}

.ok-tag {
  color: #10b981;
  font-size: 11px;
}

.empty-state {
  text-align: center;
  padding: 24px;
  color: var(--muted-text);
}

.table-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 4px 0;
}

.count-info {
  font-size: 12px;
  color: var(--muted-text);
}

.load-more-btn {
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid var(--primary);
  background: color-mix(in srgb, var(--primary) 10%, transparent);
  color: var(--primary);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.15s ease;
}

.load-more-btn:hover:not(:disabled) {
  background: var(--primary);
  color: #ffffff;
}

.load-more-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
