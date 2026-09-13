<template>
  <div class="settings-update-container">
    <article class="panel">
      <h2>Application Updates & Release Notes</h2>
      <div class="update-section">
        <div class="status-info">
          <p class="current-ver">
            Current Version: <strong>v{{ currentVersion }}</strong>
          </p>
          <p v-if="updateChecked && !updateAvailable" class="status-text success">
            Your application is up to date.
          </p>
          <p v-if="updateChecked && updateAvailable" class="status-text warning">
            A new version (<strong>v{{ latestVersion }}</strong>) is available.
          </p>
          <p v-if="error" class="status-text error">Error checking updates: {{ error }}</p>
        </div>

        <div class="action-buttons">
          <button type="button" class="btn-notes" @click="openNotes">
            View Release Notes
          </button>

          <button type="button" class="btn-check" :disabled="checking" @click="performCheck">
            {{ checking ? 'Checking...' : 'Check for Updates' }}
          </button>

          <button v-if="updateAvailable" type="button" class="btn-redirect" @click="redirectToRepo">
            Get Update (Redirect to Repository)
          </button>
        </div>
      </div>
    </article>

    <!-- Community & Feedback Panel -->
    <article class="panel community-panel">
      <h2>Feedback & Community</h2>
      <p class="panel-desc">
        Help improve Studyloop by reporting bugs, suggesting features, or supporting open-source development.
      </p>

      <div class="community-grid">
        <div class="community-card">
          <div class="card-header">
            <span class="card-icon">🐛</span>
            <h3>Found a Bug?</h3>
          </div>
          <p>Report issues directly to our GitHub repository issue tracker.</p>
          <button type="button" class="btn-outline" @click="reportBug">
            Report an Issue ↗
          </button>
        </div>

        <div class="community-card highlight">
          <div class="card-header">
            <span class="card-icon">⭐</span>
            <h3>Enjoying Studyloop?</h3>
          </div>
          <p>Star our project repository on GitHub to show your support!</p>
          <button type="button" class="btn-star" @click="starRepo">
            Star on GitHub ⭐ ↗
          </button>
        </div>
      </div>
    </article>

    <!-- Release Notes Modal -->
    <ReleaseNotesModal
      :visible="showNotesModal"
      :version="notesVersion"
      :raw-notes="notesContent"
      @close="showNotesModal = false"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { checkForUpdates, getReleaseNotes, openRepoURL, openURLInBrowser } from '../services/appApi'
import ReleaseNotesModal from './ReleaseNotesModal.vue'

const checking = ref(false)
const updateChecked = ref(false)
const updateAvailable = ref(false)
const currentVersion = ref('1.4.1')
const latestVersion = ref('')
const error = ref('')

const showNotesModal = ref(false)
const notesVersion = ref('')
const notesContent = ref('')

async function fetchNotes() {
  try {
    const res = await getReleaseNotes()
    if (res?.version) {
      notesVersion.value = String(res.version).replace(/^v+/, '')
    }
    if (res?.notes) {
      notesContent.value = res.notes
    }
  } catch (err) {
    console.error('Failed to fetch release notes:', err)
  }
}

async function openNotes() {
  if (!notesContent.value) {
    await fetchNotes()
  }
  showNotesModal.value = true
}

async function performCheck() {
  checking.value = true
  error.value = ''
  try {
    const res = await checkForUpdates()
    if (res?.current_version) {
      currentVersion.value = String(res.current_version).replace(/^v+/, '')
    }
    if (res?.latest_version) {
      latestVersion.value = String(res.latest_version).replace(/^v+/, '')
    }
    updateAvailable.value = !!res?.update_available
    updateChecked.value = true
    if (res?.error) {
      error.value = res.error
    }
  } catch (err) {
    error.value = err.message || 'Failed to check updates'
  } finally {
    checking.value = false
  }
}

function redirectToRepo() {
  openRepoURL()
}

function reportBug() {
  openURLInBrowser('https://github.com/Vishnuj-n/studyloop/issues/new')
}

function starRepo() {
  openURLInBrowser('https://github.com/Vishnuj-n/studyloop')
}

onMounted(() => {
  fetchNotes()
  checkForUpdates()
    .then((res) => {
      if (res?.current_version) {
        currentVersion.value = String(res.current_version).replace(/^v+/, '')
      }
      if (res?.error) {
        error.value = res.error
      }
    })
    .catch(() => {})
})
</script>

<style scoped>
.settings-update-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

h2 {
  font-size: 20px;
  margin: 0 0 16px;
  font-weight: 700;
  font-family: 'Manrope', sans-serif;
}

.panel-desc {
  font-size: 13px;
  color: var(--muted-text);
  margin: -8px 0 20px;
}

.update-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.current-ver {
  margin: 0;
  font-size: 14px;
}

.status-text {
  margin: 8px 0 0 0;
  font-size: 14px;
  font-weight: 500;
}

.status-text.success {
  color: #10b981;
}

.status-text.warning {
  color: #f59e0b;
}

.status-text.error {
  color: #ef4444;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

button {
  border: none;
  border-radius: 8px;
  padding: 10px 18px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-notes {
  background: var(--primary);
  color: var(--on-primary, #ffffff);
}

.btn-notes:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-check {
  background: var(--surface-container-highest);
  color: var(--on-surface);
}

.btn-check:hover:not(:disabled) {
  background: var(--surface-container-high);
}

.btn-check:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-redirect {
  background: var(--surface-container-high);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
}

.btn-redirect:hover {
  background: var(--surface-container-highest);
}

/* Community Grid */
.community-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.community-card {
  background: var(--surface-container-low, rgba(255, 255, 255, 0.02));
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.community-card.highlight {
  background: color-mix(in srgb, var(--primary, #4f46e5) 5%, var(--surface-container-low));
  border-color: color-mix(in srgb, var(--primary, #4f46e5) 25%, transparent);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-icon {
  font-size: 22px;
}

.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
}

.community-card p {
  margin: 0;
  font-size: 13px;
  color: var(--muted-text);
  line-height: 1.4;
  flex: 1;
}

.btn-outline {
  background: var(--surface-container-highest);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
}

.btn-outline:hover {
  background: var(--surface-container);
}

.btn-star {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #ffffff;
}

.btn-star:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}

@media (max-width: 768px) {
  .community-grid {
    grid-template-columns: 1fr;
  }
}
</style>
