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

          <button type="button" class="btn-check" :disabled="checking || updating" @click="performCheck">
            {{ checking ? 'Checking...' : 'Check for Updates' }}
          </button>

          <button
            v-if="updateAvailable && !updating"
            type="button"
            class="btn-install"
            @click="startAutoUpdate"
          >
            Download & Install Update
          </button>

          <button
            v-if="updateAvailable && !updating"
            type="button"
            class="btn-redirect"
            @click="redirectToRepo"
          >
            Manual Download (GitHub)
          </button>
        </div>

        <!-- Update Download Progress -->
        <div v-if="updating" class="update-progress-box">
          <div class="progress-bar-header">
            <span>{{ updateStatusText }}</span>
            <span class="progress-pct">{{ updateProgress.toFixed(0) }}%</span>
          </div>
          <div class="progress-track">
            <div class="progress-fill" :style="{ width: updateProgress + '%' }"></div>
          </div>
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
            <svg class="card-icon-svg" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m8 2 1.88 1.88"/>
              <path d="M14.12 3.88 16 2"/>
              <path d="M9 7.13v-1a3.003 3.003 0 0 1 6 0v1"/>
              <path d="M12 20c-3.3 0-6-2.7-6-6v-3a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v3c0 3.3-2.7 6-6 6z"/>
              <path d="M12 20v-9"/>
              <path d="M6.53 9C4.6 9.8 3 11.4 3 14"/>
              <path d="M6 18c-1.8-1-3-2.7-3-5"/>
              <path d="M17.47 9c1.93.8 3.53 2.4 3.53 5"/>
              <path d="M18 18c1.8-1 3-2.7 3-5"/>
            </svg>
            <h3>Found a Bug?</h3>
          </div>
          <p>Report issues directly to our GitHub repository issue tracker.</p>
          <button type="button" class="btn-outline" @click="reportBug">
            <span>Report an Issue</span>
            <span class="arrow-icon">↗</span>
          </button>
        </div>

        <div class="community-card">
          <div class="card-header">
            <svg class="card-icon-svg" viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
            <h3>Privacy & Data</h3>
          </div>
          <p>100% local and private. Read our open-source privacy policy.</p>
          <button type="button" class="btn-outline" @click="openRepoPrivacy">
            <span>Read Policy</span>
            <span class="arrow-icon">↗</span>
          </button>
        </div>

        <div class="community-card highlight">
          <div class="card-header">
            <svg class="card-icon-svg github-star-icon" viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
              <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/>
            </svg>
            <h3>Enjoying Studyloop?</h3>
          </div>
          <p>Star our repository on GitHub to support open-source development.</p>
          <button type="button" class="btn-github-star" @click="starRepo">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
              <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/>
            </svg>
            <span>Star on GitHub</span>
            <svg class="star-mini-icon" viewBox="0 0 24 24" width="14" height="14" fill="#fbbf24">
              <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/>
            </svg>
            <span class="arrow-icon">↗</span>
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
import { ref, onMounted, onUnmounted } from 'vue'
import { checkForUpdates, getReleaseNotes, openRepoURL, openURLInBrowser, downloadAndApplyUpdate } from '../services/appApi'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import ReleaseNotesModal from './ReleaseNotesModal.vue'

const checking = ref(false)
const updateChecked = ref(false)
const updateAvailable = ref(false)
const currentVersion = ref('1.4.1')
const latestVersion = ref('')
const error = ref('')

const updating = ref(false)
const updateProgress = ref(0)
const updateStatusText = ref('')

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

async function startAutoUpdate() {
  updating.value = true
  updateProgress.value = 0
  updateStatusText.value = 'Connecting and downloading update...'
  error.value = ''

  try {
    const res = await downloadAndApplyUpdate()
    if (res?.error) {
      error.value = res.error
      updating.value = false
    } else {
      updateStatusText.value = 'Download complete! Launching installer and restarting...'
      updateProgress.value = 100
    }
  } catch (err) {
    error.value = err.message || 'Failed to install update'
    updating.value = false
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

function openRepoPrivacy() {
  openURLInBrowser('https://github.com/Vishnuj-n/studyloop#privacy')
}

let unlistenProgress = null

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

  if (EventsOn) {
    unlistenProgress = EventsOn('update:progress', (data) => {
      if (data && typeof data.percentage === 'number') {
        updateProgress.value = Math.min(100, Math.max(0, data.percentage))
        if (data.total > 0) {
          const mbDownloaded = (data.downloaded / (1024 * 1024)).toFixed(1)
          const mbTotal = (data.total / (1024 * 1024)).toFixed(1)
          updateStatusText.value = `Downloading update: ${mbDownloaded} MB / ${mbTotal} MB`
        }
      }
    })
  }
})

onUnmounted(() => {
  if (unlistenProgress && EventsOff) {
    EventsOff('update:progress')
  }
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

.btn-install {
  background: linear-gradient(135deg, var(--primary, #6366f1) 0%, #4f46e5 100%);
  color: #ffffff;
  box-shadow: 0 2px 10px color-mix(in srgb, var(--primary) 25%, transparent);
}

.btn-install:hover {
  opacity: 0.95;
  transform: translateY(-1px);
  box-shadow: 0 4px 14px color-mix(in srgb, var(--primary) 35%, transparent);
}

.btn-redirect {
  background: var(--surface-container-high);
  color: var(--primary);
  border: 1px solid color-mix(in srgb, var(--primary) 30%, transparent);
}

.btn-redirect:hover {
  background: var(--surface-container-highest);
}

.update-progress-box {
  margin-top: 8px;
  background: var(--surface-container-low, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.progress-bar-header {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface);
}

.progress-pct {
  color: var(--primary);
  font-family: monospace;
}

.progress-track {
  width: 100%;
  height: 8px;
  background: var(--surface-container-highest, rgba(255, 255, 255, 0.1));
  border-radius: 99px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--primary, #6366f1);
  border-radius: 99px;
  transition: width 0.15s ease-out;
}

/* Community Grid */
.community-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
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
  min-height: 170px;
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

.card-icon-svg {
  color: var(--primary);
  flex-shrink: 0;
}

.github-star-icon {
  color: var(--on-surface);
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
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--surface-container-highest);
  color: var(--on-surface);
  border: 1px solid var(--outline-variant);
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.btn-outline:hover {
  background: var(--surface-container);
  border-color: var(--outline-variant);
  transform: translateY(-1px);
}

.btn-github-star {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--surface-container-highest);
  color: var(--on-surface);
  border: 1px solid color-mix(in srgb, var(--primary, #6366f1) 30%, var(--outline-variant));
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  box-shadow: 0 2px 8px color-mix(in srgb, var(--primary, #6366f1) 12%, transparent);
}

.btn-github-star:hover {
  background: color-mix(in srgb, var(--primary, #6366f1) 15%, var(--surface-container-highest));
  border-color: var(--primary);
  transform: translateY(-1px);
  box-shadow: 0 4px 14px color-mix(in srgb, var(--primary, #6366f1) 25%, transparent);
}

.arrow-icon {
  font-size: 13px;
  opacity: 0.8;
}

.star-mini-icon {
  margin-left: -2px;
}

@media (max-width: 960px) {
  .community-grid {
    grid-template-columns: 1fr;
  }
}
</style>
