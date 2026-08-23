<template>
  <div class="extensions-page">
    <header class="page-header">
      <div>
        <h1 class="page-title">Extensions Hub</h1>
        <p class="page-subtitle">
          Customize and extend your StudyLoop environment with local tools and integrations.
        </p>
      </div>
      <div class="header-actions">
        <div v-if="isPro" class="tier-pill pro">
          <span class="sparkle">★</span> PRO ACTIVE
        </div>
        <div v-else class="tier-pill free">
          FREE PLAN
        </div>
        <button v-if="!isPro" class="upgrade-btn-header" @click="handleUpgrade">
          Upgrade to Pro
        </button>
      </div>
    </header>

    <!-- Error message banner if any -->
    <div v-if="errorMessage" class="error-banner">
      {{ errorMessage }}
      <button class="close-err" @click="errorMessage = ''">×</button>
    </div>

    <!-- Free Extensions Section -->
    <section class="extension-section">
      <div class="section-title-row">
        <h2>Free Extensions</h2>
        <span class="count-badge">{{ freeExtensions.length }} available</span>
      </div>

      <div class="extensions-grid">
        <div v-for="ext in freeExtensions" :key="ext.id" class="ext-card">
          <div class="card-header">
            <div class="icon-box">🧩</div>
            <div class="ext-meta">
              <h3 class="ext-name">{{ ext.name }}</h3>
              <span class="version-tag">v{{ ext.version }} &bull; {{ ext.category || 'Utility' }}</span>
            </div>
            <span class="tier-badge free">FREE</span>
          </div>

          <p class="ext-desc">{{ ext.description || 'Local StudyLoop extension.' }}</p>

          <div class="card-actions">
            <button
              v-if="ext.id === 'text_simplifier'"
              class="run-btn"
              @click="router.push('/simplify')"
            >
              Open Simplifier ➔
            </button>
            <button
              v-else
              class="run-btn"
              :disabled="runningId === ext.id"
              @click="handleRun(ext)"
            >
              {{ runningId === ext.id ? 'Running...' : 'Run Extension' }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Pro Extensions Section -->
    <section class="extension-section pro-section">
      <div class="section-title-row">
        <div class="pro-heading">
          <h2>Pro Extensions</h2>
          <span class="pro-crown">👑</span>
        </div>
        <span class="count-badge pro">{{ proExtensions.length }} pro tools</span>
      </div>

      <div class="extensions-grid">
        <div
          v-for="ext in proExtensions"
          :key="ext.id"
          class="ext-card pro-card"
          :class="{ 'pro-locked': !isPro }"
        >
          <div class="card-header">
            <div class="icon-box pro-icon">⚡</div>
            <div class="ext-meta">
              <h3 class="ext-name">{{ ext.name }}</h3>
              <span class="version-tag">v{{ ext.version }} &bull; {{ ext.category || 'Advanced' }}</span>
            </div>
            <span class="tier-badge pro">PRO</span>
          </div>

          <p class="ext-desc">{{ ext.description || 'Advanced StudyLoop Pro integration.' }}</p>

          <div class="card-actions">
            <button
              v-if="isPro"
              class="run-btn pro-run"
              :disabled="runningId === ext.id"
              @click="handleRun(ext)"
            >
              {{ runningId === ext.id ? 'Running...' : 'Run Extension' }}
            </button>
            <button
              v-else
              class="upgrade-card-btn"
              @click="handleUpgrade"
            >
              Unlock with Pro
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Output Modal -->
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
          <button class="secondary-btn" @click="outputModalOpen = false">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listExtensions, runExtension } from '../services/appApi'
import { useClerkAuth, initClerk } from '../services/clerkAuth'

const router = useRouter()
const clerkAuth = useClerkAuth()
const isPro = computed(() => clerkAuth.isPro.value)

const extensions = ref([])
const runningId = ref(null)
const errorMessage = ref('')
const outputModalOpen = ref(false)
const activeExtName = ref('')
const extensionOutput = ref('')

const freeExtensions = computed(() =>
  extensions.value.filter((e) => (e.tier || 'free').toLowerCase() === 'free')
)

const proExtensions = computed(() =>
  extensions.value.filter((e) => (e.tier || 'free').toLowerCase() === 'pro')
)

async function fetchExtensions() {
  try {
    const list = await listExtensions()
    extensions.value = Array.isArray(list) ? list : []
  } catch (err) {
    console.error('Failed to load extensions:', err)
  }
}

async function handleRun(ext) {
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
  padding: 32px 40px;
  max-width: 1200px;
  margin: 0 auto;
  color: var(--on-surface);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 36px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 8px 0;
  letter-spacing: -0.02em;
}

.page-subtitle {
  color: var(--muted-text);
  font-size: 15px;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tier-pill {
  padding: 6px 14px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.tier-pill.free {
  background: var(--surface-container-high);
  color: var(--on-surface-variant);
  border: 1px solid var(--outline-variant);
}

.tier-pill.pro {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #ffffff;
  box-shadow: 0 2px 8px rgba(245, 158, 11, 0.3);
}

.sparkle {
  color: #fff;
  margin-right: 2px;
}

.upgrade-btn-header {
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  padding: 8px 16px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.upgrade-btn-header:hover {
  opacity: 0.9;
}

.error-banner {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  padding: 12px 16px;
  border-radius: 10px;
  margin-bottom: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.close-err {
  background: transparent;
  border: none;
  color: #ef4444;
  font-size: 18px;
  cursor: pointer;
}

.extension-section {
  margin-bottom: 40px;
}

.section-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
}

.section-title-row h2 {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.pro-heading {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pro-crown {
  font-size: 18px;
}

.count-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--surface-container-high);
  color: var(--muted-text);
}

.count-badge.pro {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.extensions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.ext-card {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 16px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.ext-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.08);
}

.pro-card {
  border-color: color-mix(in srgb, #f59e0b 30%, var(--outline-variant));
}

.card-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.icon-box {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: var(--surface-container-highest);
  display: grid;
  place-items: center;
  font-size: 18px;
  flex-shrink: 0;
}

.pro-icon {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
}

.ext-meta {
  flex: 1;
  min-width: 0;
}

.ext-name {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.version-tag {
  font-size: 12px;
  color: var(--muted-text);
  text-transform: capitalize;
}

.tier-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 6px;
  letter-spacing: 0.05em;
}

.tier-badge.free {
  background: var(--surface-container-highest);
  color: var(--on-surface-variant);
}

.tier-badge.pro {
  background: #f59e0b;
  color: #000;
}

.ext-desc {
  font-size: 13.5px;
  line-height: 1.5;
  color: var(--muted-text);
  margin: 0;
  flex-grow: 1;
}

.card-actions {
  display: flex;
  gap: 8px;
}

.run-btn {
  width: 100%;
  padding: 10px;
  border-radius: 10px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  color: var(--primary);
  font-weight: 600;
  font-size: 13.5px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.run-btn:hover:not(:disabled) {
  background: var(--primary);
  color: var(--on-primary);
}

.run-btn.pro-run {
  background: color-mix(in srgb, #f59e0b 20%, transparent);
  border-color: #f59e0b;
  color: #f59e0b;
}

.run-btn.pro-run:hover:not(:disabled) {
  background: #f59e0b;
  color: #000;
}

.upgrade-card-btn {
  width: 100%;
  padding: 10px;
  border-radius: 10px;
  background: linear-gradient(135deg, #f59e0b, #d97706);
  border: none;
  color: #ffffff;
  font-weight: 600;
  font-size: 13.5px;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.upgrade-card-btn:hover {
  opacity: 0.9;
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24px;
}

.modal-card {
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  width: 100%;
  max-width: 680px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  padding: 18px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--outline-variant);
}

.modal-header h3 {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
}

.close-modal-btn {
  background: transparent;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--muted-text);
}

.modal-body {
  padding: 20px 24px;
  overflow-y: auto;
  flex: 1;
}

.output-pre {
  margin: 0;
  font-family: monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--on-surface);
  background: var(--surface-container-low);
  padding: 14px;
  border-radius: 10px;
}

.modal-footer {
  padding: 14px 24px;
  border-top: 1px solid var(--outline-variant);
  display: flex;
  justify-content: flex-end;
}

.secondary-btn {
  padding: 8px 16px;
  border-radius: 8px;
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  font-weight: 600;
  cursor: pointer;
}
</style>
