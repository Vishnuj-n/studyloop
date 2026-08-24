<template>
  <div class="extensions-page">
    <!-- Header -->
    <header class="page-header">
      <div class="header-intro">
        <h1 class="page-title">Extensions Hub</h1>
        <p class="page-subtitle">
          Customize and extend your StudyLoop environment with local tools and integrations.
        </p>
      </div>
      <div class="header-actions">
        <div v-if="isPro" class="tier-pill pro">
          <span class="pro-dot"></span> PRO ACTIVE
        </div>
        <div v-else class="tier-pill free">
          FREE PLAN
        </div>
        <button v-if="!isPro" class="upgrade-btn-header" @click="handleUpgrade">
          Upgrade to Pro
        </button>
      </div>
    </header>

    <!-- Error Banner -->
    <div v-if="errorMessage" class="error-banner">
      <span>{{ errorMessage }}</span>
      <button class="close-err" @click="errorMessage = ''" aria-label="Close error">×</button>
    </div>

    <!-- Free Extensions Section -->
    <section class="extension-section">
      <div class="section-title-row">
        <div class="section-heading">
          <h2>Free Extensions</h2>
          <span class="count-badge">{{ freeExtensions.length }} available</span>
        </div>
      </div>

      <div class="extensions-grid">
        <div
          v-for="ext in freeExtensions"
          :key="ext.id"
          class="ext-card"
          :class="{ 'is-disabled': !isExtensionEnabled(ext.id) }"
        >
          <div class="card-top">
            <div class="icon-box">
              <!-- Monochromatic SVG Icon for clean editorial feel -->
              <svg v-if="ext.id === 'text_simplifier'" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <line x1="16" y1="13" x2="8" y2="13"></line>
                <line x1="16" y1="17" x2="8" y2="17"></line>
                <polyline points="10 9 9 9 8 9"></polyline>
              </svg>
              <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="3" width="7" height="7"></rect>
                <rect x="14" y="3" width="7" height="7"></rect>
                <rect x="14" y="14" width="7" height="7"></rect>
                <rect x="3" y="14" width="7" height="7"></rect>
              </svg>
            </div>

            <div class="ext-meta">
              <div class="title-line">
                <h3 class="ext-name">{{ ext.name }}</h3>
                <span class="tier-tag free">FREE</span>
              </div>
              <span class="version-tag">v{{ ext.version }} &bull; {{ ext.category || 'Reader' }}</span>
            </div>

            <!-- Toggle Switch -->
            <label class="switch" :title="isExtensionEnabled(ext.id) ? 'Disable extension' : 'Enable extension'">
              <input
                type="checkbox"
                :checked="isExtensionEnabled(ext.id)"
                :disabled="isSettingUp(ext.id)"
                @change="handleToggle(ext)"
              />
              <span class="slider"></span>
            </label>
          </div>

          <p class="ext-desc">{{ ext.description || 'Local StudyLoop reader extension.' }}</p>

          <div class="card-bottom">
            <button
              v-if="ext.id === 'text_simplifier'"
              class="action-btn primary-action"
              :disabled="!isExtensionEnabled(ext.id)"
              @click="router.push('/simplify')"
            >
              Open Simplifier ➔
            </button>
            <button
              v-else
              class="action-btn primary-action"
              :disabled="!isExtensionEnabled(ext.id) || runningId === ext.id || isSettingUp(ext.id)"
              @click="handleRun(ext)"
            >
              {{ getActionButtonLabel(ext.id) }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Pro Extensions Section -->
    <section class="extension-section pro-section">
      <div class="section-title-row">
        <div class="section-heading">
          <h2>Pro Extensions</h2>
          <span class="pro-crown-symbol" title="Pro Features">👑</span>
          <span class="count-badge pro">{{ proExtensions.length }} pro tools</span>
        </div>
      </div>

      <div class="extensions-grid">
        <div
          v-for="ext in proExtensions"
          :key="ext.id"
          class="ext-card pro-card"
          :class="{ 'pro-locked': !isPro, 'is-disabled': isPro && !isExtensionEnabled(ext.id) }"
        >
          <div class="card-top">
            <div class="icon-box pro-icon">
              <!-- Audio icon -->
              <svg v-if="ext.id === 'audio_overview' || ext.category === 'audio'" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M3 18v-6a9 9 0 0 1 18 0v6"></path>
                <path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z"></path>
              </svg>
              <!-- Video/YouTube icon -->
              <svg v-else-if="ext.id === 'youtube_transcripts' || ext.category === 'ingestion'" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polygon points="23 7 16 12 23 17 23 7"></polygon>
                <rect x="1" y="5" width="15" height="14" rx="2" ry="2"></rect>
              </svg>
              <!-- Default Pro icon -->
              <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
              </svg>
            </div>

            <div class="ext-meta">
              <div class="title-line">
                <h3 class="ext-name">{{ ext.name }}</h3>
                <span class="tier-tag pro">PRO</span>
              </div>
              <span class="version-tag">v{{ ext.version }} &bull; {{ ext.category || 'Advanced' }}</span>
            </div>

            <!-- Pro Toggle Switch -->
            <label class="switch" :title="isPro ? (isExtensionEnabled(ext.id) ? 'Disable extension' : 'Enable extension') : 'Unlock with Pro'">
              <input
                type="checkbox"
                :checked="isPro && isExtensionEnabled(ext.id)"
                :disabled="isSettingUp(ext.id)"
                @change="handleProToggle(ext)"
              />
              <span class="slider"></span>
            </label>
          </div>

          <p class="ext-desc">{{ ext.description || 'Advanced StudyLoop Pro integration.' }}</p>

          <div class="card-bottom">
            <button
              v-if="isPro"
              class="action-btn pro-action"
              :disabled="!isExtensionEnabled(ext.id) || runningId === ext.id || isSettingUp(ext.id)"
              @click="handleRun(ext)"
            >
              {{ getActionButtonLabel(ext.id) }}
            </button>
            <button
              v-else
              class="action-btn unlock-action"
              @click="handleUpgrade"
            >
              Unlock with Pro
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Reusable Setup / Verification Progress Popup Modal -->
    <ExtensionSetupModal
      :is-open="setupModalOpen"
      :extension="currentSetupExt"
      @close="setupModalOpen = false"
      @success="handleSetupSuccess"
    />

    <!-- Output Modal (Aligned with Digital Sanctuary elevation & typography) -->
    <div v-if="outputModalOpen" class="modal-overlay" @click.self="outputModalOpen = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3>Extension Output: {{ activeExtName }}</h3>
          <button class="close-modal-btn" @click="outputModalOpen = false">✕</button>
        </div>
        <div class="modal-body">
          <pre class="output-pre">{{ extensionOutput }}</pre>
        </div>
        <div class="modal-footer">
          <button class="modal-close-btn" @click="outputModalOpen = false">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listExtensions, runExtension, checkExtensionReadiness } from '../services/appApi'
import { useClerkAuth, initClerk } from '../services/clerkAuth'
import { useExtensions } from '../composables/useExtensions'
import ExtensionSetupModal from '../components/ExtensionSetupModal.vue'

const router = useRouter()
const clerkAuth = useClerkAuth()
const isPro = computed(() => clerkAuth.isPro.value)
const { isEnabled: isExtensionEnabled, setExtensionEnabled } = useExtensions()

const extensions = ref([])
const runningId = ref(null)
const settingUpMap = ref({})
const errorMessage = ref('')

const outputModalOpen = ref(false)
const activeExtName = ref('')
const extensionOutput = ref('')

// Setup Modal State
const setupModalOpen = ref(false)
const currentSetupExt = ref(null)

const freeExtensions = computed(() =>
  extensions.value.filter((e) => (e.tier || 'free').toLowerCase() === 'free')
)

const proExtensions = computed(() =>
  extensions.value.filter((e) => (e.tier || 'free').toLowerCase() === 'pro')
)

function isSettingUp(id) {
  return !!settingUpMap.value[id]
}

function getActionButtonLabel(extId) {
  if (isSettingUp(extId)) return 'Setting up...'
  if (runningId.value === extId) return 'Running...'
  return 'Run Extension'
}

async function handleToggle(ext) {
  if (isExtensionEnabled(ext.id)) {
    // User is turning it off
    setExtensionEnabled(ext.id, false)
    return
  }

  // User is turning it on: check runtime readiness
  const runtime = (ext.runtime || '').toLowerCase()
  if (runtime === 'python' || runtime === 'py') {
    try {
      const readiness = await checkExtensionReadiness(ext.id)
      if (readiness && readiness.is_ready) {
        setExtensionEnabled(ext.id, true)
        return
      }
    } catch (e) {
      console.warn('Readiness check returned error, initiating setup:', e)
    }

    // Needs setup: trigger setup popup modal
    triggerSetup(ext)
  } else {
    setExtensionEnabled(ext.id, true)
  }
}

async function handleProToggle(ext) {
  if (!isPro.value) {
    handleUpgrade()
    return
  }
  await handleToggle(ext)
}

function triggerSetup(ext) {
  currentSetupExt.value = ext
  activeExtName.value = ext.name
  setupModalOpen.value = true
}

function handleSetupSuccess(ext) {
  if (ext && ext.id) {
    setExtensionEnabled(ext.id, true)
  }
}

async function fetchExtensions() {
  try {
    const list = await listExtensions()
    extensions.value = Array.isArray(list) ? list : []
  } catch (err) {
    console.error('Failed to load extensions:', err)
  }
}

async function handleRun(ext) {
  if (!isExtensionEnabled(ext.id)) return
  errorMessage.value = ''
  runningId.value = ext.id
  try {
    const res = await runExtension(ext.id, '', isPro.value)
    if (res?.error) {
      errorMessage.value = res.error
    } else {
      activeExtName.value = ext.name
      extensionOutput.value = res.output || 'Process exited successfully with no output.'
      outputModalOpen.value = true
    }
  } catch (err) {
    errorMessage.value = String(err)
  } finally {
    runningId.value = null
  }
}

function handleUpgrade() {
  clerkAuth.openBilling()
}

onMounted(async () => {
  await initClerk()
  await fetchExtensions()
})
</script>

<style scoped>
.extensions-page {
  padding: 40px 48px;
  max-width: 1200px;
  margin: 0 auto;
  color: var(--on-surface);
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
}

/* Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 40px;
  gap: 24px;
}

.header-intro {
  max-width: 680px;
}

.page-title {
  font-family: 'Manrope', sans-serif;
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 8px 0;
  letter-spacing: -0.02em;
  color: var(--on-surface);
}

.page-subtitle {
  color: var(--muted-text);
  font-size: 14px;
  line-height: 1.5;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.tier-pill {
  padding: 6px 14px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tier-pill.free {
  background: var(--surface-container-low);
  color: var(--muted-text);
}

.tier-pill.pro {
  background: var(--surface-container-lowest);
  color: var(--primary);
  box-shadow: 0 2px 8px rgba(45, 51, 56, 0.04);
}

.pro-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--primary);
}

.upgrade-btn-header {
  background: linear-gradient(15deg, var(--primary) 0%, var(--primary-dim) 100%);
  color: var(--on-primary);
  border: none;
  padding: 8px 18px;
  border-radius: 12px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.2s ease, transform 0.15s ease;
}

.upgrade-btn-header:hover {
  opacity: 0.95;
  transform: translateY(-1px);
}

.upgrade-btn-header:active {
  transform: scale(0.97);
}

/* Error Banner */
.error-banner {
  background: rgba(159, 64, 61, 0.08);
  color: #9f403d;
  padding: 12px 16px;
  border-radius: 12px;
  margin-bottom: 28px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13.5px;
}

.close-err {
  background: transparent;
  border: none;
  color: #9f403d;
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
}

/* Sections */
.extension-section {
  margin-bottom: 44px;
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.section-heading {
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-heading h2 {
  font-family: 'Manrope', sans-serif;
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: var(--on-surface);
  letter-spacing: -0.01em;
}

.pro-crown-symbol {
  font-size: 16px;
  opacity: 0.85;
}

.count-badge {
  font-size: 12px;
  font-weight: 500;
  padding: 3px 9px;
  border-radius: 999px;
  background: var(--surface-container-low);
  color: var(--muted-text);
}

.count-badge.pro {
  background: var(--surface-container);
  color: var(--primary);
  font-weight: 600;
}

/* Grid & Cards - Clean No-Line Sanctuary System */
.extensions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 20px;
}

.ext-card {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 18px;
  box-shadow: 0 2px 10px rgba(45, 51, 56, 0.03);
  transition: transform 0.2s ease, box-shadow 0.2s ease, opacity 0.2s ease;
}

.ext-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(45, 51, 56, 0.06);
}

.ext-card.is-disabled {
  opacity: 0.7;
}

.card-top {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.icon-box {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.pro-icon {
  background: var(--surface-container);
  color: var(--primary);
}

.ext-meta {
  flex: 1;
  min-width: 0;
}

.title-line {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 3px;
}

.ext-name {
  font-family: 'Manrope', sans-serif;
  font-size: 15px;
  font-weight: 600;
  margin: 0;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.version-tag {
  font-size: 12px;
  color: var(--muted-text);
  text-transform: capitalize;
}

.tier-tag {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}

.tier-tag.free {
  background: var(--surface-container-low);
  color: var(--muted-text);
}

.tier-tag.pro {
  background: var(--surface-container);
  color: var(--primary);
}

/* Switch Component */
.switch {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  flex-shrink: 0;
  margin-top: 2px;
  cursor: pointer;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background-color: var(--surface-container-low);
  transition: 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 20px;
}

.slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background-color: var(--on-surface);
  opacity: 0.6;
  transition: 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 50%;
}

input:checked + .slider {
  background-color: var(--primary);
}

input:checked + .slider:before {
  transform: translateX(16px);
  background-color: #ffffff;
  opacity: 1;
}

/* Descriptions */
.ext-desc {
  font-size: 13.5px;
  line-height: 1.55;
  color: var(--muted-text);
  margin: 0;
  flex-grow: 1;
}

/* Actions */
.card-bottom {
  display: flex;
  gap: 8px;
}

.action-btn {
  width: 100%;
  padding: 10px 16px;
  border-radius: 12px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: all 0.2s ease;
  font-family: inherit;
}

.action-btn:active:not(:disabled) {
  transform: scale(0.98);
}

.action-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.primary-action {
  background: var(--surface-container-low);
  color: var(--on-surface);
}

.primary-action:hover:not(:disabled) {
  background: var(--surface-container);
  color: var(--primary);
}

.pro-action {
  background: var(--surface-container);
  color: var(--primary);
}

.pro-action:hover:not(:disabled) {
  background: linear-gradient(15deg, var(--primary) 0%, var(--primary-dim) 100%);
  color: var(--on-primary);
}

.unlock-action {
  background: linear-gradient(15deg, var(--primary) 0%, var(--primary-dim) 100%);
  color: var(--on-primary);
}

.unlock-action:hover {
  opacity: 0.95;
  transform: translateY(-1px);
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(45, 51, 56, 0.4);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24px;
}

.modal-card {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  width: 100%;
  max-width: 640px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 40px rgba(45, 51, 56, 0.08);
}

.modal-header {
  padding: 20px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  font-family: 'Manrope', sans-serif;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface);
}

.close-modal-btn {
  background: transparent;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--muted-text);
  padding: 4px;
}

.modal-body {
  padding: 0 24px 20px;
  overflow-y: auto;
  flex: 1;
}

.output-pre {
  margin: 0;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 12.5px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--on-surface);
  background: var(--surface-container-low);
  padding: 16px;
  border-radius: 12px;
}

.modal-footer {
  padding: 16px 24px;
  display: flex;
  justify-content: flex-end;
}

.modal-close-btn {
  padding: 8px 18px;
  border-radius: 10px;
  background: var(--surface-container-low);
  border: none;
  color: var(--on-surface);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s ease;
}

.modal-close-btn:hover {
  background: var(--surface-container);
}

/* Setup Modal Specific Styles */
.setup-card {
  max-width: 580px;
}

.setup-header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.setup-icon-spinner {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
}

.loading-spin-circle {
  width: 18px;
  height: 18px;
  border: 2.5px solid var(--surface-container);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.setup-icon-success {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #2e7d32;
  color: #ffffff;
  display: grid;
  place-items: center;
  font-size: 13px;
  font-weight: bold;
}

.setup-icon-error {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #c62828;
  color: #ffffff;
  display: grid;
  place-items: center;
  font-size: 14px;
  font-weight: bold;
}

.setup-desc {
  font-size: 13.5px;
  color: var(--muted-text);
  line-height: 1.5;
  margin: 0 0 16px 0;
}

.setup-logs-box {
  background: var(--surface-container-low);
  border-radius: 12px;
  padding: 14px 16px;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  max-height: 200px;
  overflow-y: auto;
  color: var(--on-surface);
}

.setup-log-line {
  margin-bottom: 4px;
}

.log-pending {
  color: var(--primary);
  font-style: italic;
}

.modal-action-btn.retry-btn {
  padding: 8px 18px;
  border-radius: 10px;
  background: linear-gradient(15deg, var(--primary) 0%, var(--primary-dim) 100%);
  color: var(--on-primary);
  border: none;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  margin-right: 8px;
}
</style>
