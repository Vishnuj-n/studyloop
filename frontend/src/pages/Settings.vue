<template>
  <section class="settings-workspace">
    <!-- Solid Left Category Sidebar Rail -->
    <aside class="settings-category-rail" aria-label="Settings Categories">
      <div class="rail-header">
        <p class="eyebrow">Settings</p>
        <h1 class="rail-title">Configuration</h1>
      </div>

      <!-- Quick Search / Filter Bar -->
      <div class="rail-search">
        <div class="search-input-wrapper">
          <BaseIcon name="search" size="14" class="search-icon" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search settings..."
            class="settings-search-input"
          />
          <button
            v-if="searchQuery"
            type="button"
            class="search-clear-btn"
            title="Clear search"
            @click="searchQuery = ''"
          >
            <BaseIcon name="x" size="12" />
          </button>
        </div>
      </div>

      <nav class="rail-nav">
        <button
          v-for="cat in filteredCategories"
          :key="cat.id"
          type="button"
          class="rail-item"
          :class="{ active: activeCategory === cat.id }"
          @click="activeCategory = cat.id"
        >
          <div class="rail-item-top">
            <span class="rail-item-title">{{ cat.label }}</span>
            <span v-if="searchQuery && categoryMatches[cat.id]" class="match-badge">
              Match
            </span>
          </div>
          <span class="rail-item-desc">{{ cat.desc }}</span>
        </button>

        <div v-if="filteredCategories.length === 0" class="no-search-results">
          No settings match "{{ searchQuery }}"
        </div>
      </nav>

      <div class="rail-footer">
        <button
          v-if="isCloudAccount"
          type="button"
          class="rail-sync-btn"
          :disabled="syncing"
          @click="runManualSync"
        >
          {{ syncing ? 'Syncing...' : 'Sync Cloud' }}
        </button>
        <p v-if="error" class="error-text">{{ error }}</p>
        <p v-if="success" class="success-text">{{ success }}</p>
      </div>
    </aside>

    <!-- Main Content Viewport -->
    <main class="settings-content-viewport">
      <div class="settings-content-scroll">
        <!-- Study & Routine Category -->
        <div v-show="activeCategory === 'study'" class="category-pane">
          <header class="pane-header">
            <h2>Study &amp; Routine</h2>
            <p class="pane-subtitle">
              Configure daily reading volume, review limits, study schedules, and remediation paths.
            </p>
          </header>

          <div class="pane-body">
            <SettingsStudyBudget
              :settings="settings"
              :study-duration="studyDuration"
              :max-input-tokens="llmSettings.fast?.max_input_tokens || 4000"
              :disabled="loading || saving"
              @apply-duration-preset="applyDurationPreset"
            />

            <SettingsQuizRescue
              :settings="settings"
              :disabled="loading || saving"
            />
          </div>
        </div>

        <!-- AI & Retrieval Category -->
        <div v-show="activeCategory === 'ai'" class="category-pane">
          <header class="pane-header">
            <h2>AI &amp; Retrieval</h2>
            <p class="pane-subtitle">
              Manage fast and heavy LLM endpoints, credentials, local vector retrieval, and AI tutor persona.
            </p>
          </header>

          <div class="pane-body">
            <SettingsAIProvider
              :llm-settings="llmSettings"
              :llm-fast-key="llmFastKey"
              :llm-heavy-key="llmHeavyKey"
              :target-session-words="settings.target_session_words || 3000"
              :disabled="loading || savingLLM"
              @apply-preset="applyProviderPreset"
              @remove-keys="removeLLMKeys"
              @update:llm-fast-key="llmFastKey = $event"
              @update:llm-heavy-key="llmHeavyKey = $event"
            />

            <SettingsRagRetrieval
              :settings="settings"
              :rag-status="ragStatus"
              :is-setting-up-rag="isSettingUpRag"
              :disabled="loading || saving"
              @rag-toggle="onRagToggle"
              @open-setup="openRagModal"
            />
          </div>
        </div>

        <!-- Profiles & Notebooks Category -->
        <div v-show="activeCategory === 'profiles'" class="category-pane">
          <header class="pane-header">
            <h2>Study Profiles &amp; Notebooks</h2>
            <p class="pane-subtitle">
              Switch exam targets, manage study profiles, and assign notebooks to specific profiles.
            </p>
          </header>

          <div class="pane-body profiles-layout">
            <SettingsProfilesPanel
              :profiles="profiles"
              :active-profile-id="settings.active_profile_id"
              :format-unix-date="formatUnixDate"
              @add="showAddModal = true"
              @select="setActiveProfile"
              @edit="openEditModal"
              @delete="handleDeleteProfile"
            />

            <SettingsTextbooksPanel
              :notebooks="notebooks"
              :profiles="profiles"
              @assign="handleAssignProfile"
            />
          </div>
        </div>

        <!-- Extensions & Tools Category -->
        <div v-show="activeCategory === 'extensions'" class="category-pane">
          <header class="pane-header">
            <h2>Extensions &amp; Tools</h2>
            <p class="pane-subtitle">
              Configure local AI tools, podcast voice personas, and simplifier comprehension levels.
            </p>
          </header>

          <div class="pane-body">
            <SettingsExtensions :disabled="loading || saving" />
          </div>
        </div>

        <!-- System & Account Category -->
        <div v-show="activeCategory === 'system'" class="category-pane">
          <header class="pane-header">
            <h2>System &amp; Account</h2>
            <p class="pane-subtitle">
              Manage desktop appearance, authentication, subscription tier, and software updates.
            </p>
          </header>

          <div class="pane-body">
            <SettingsTheme :settings="settings" :disabled="loading || saving" />

            <SettingsAccount
              :settings="settings"
              :is-dev="isDev"
              :disabled="loading || saving"
              :active-profile-name="activeProfileName"
              :is-sign-up-mode="isSignUpMode"
              :login-username="loginUsername"
              :login-password="loginPassword"
              :signup-username="signupUsername"
              :signup-password="signupPassword"
              :signup-classroom-code="signupClassroomCode"
              :login-error="loginError"
              :logging-in="loggingIn"
              @login="handleLogin"
              @signup="handleSignUp"
              @logout="handleLogout"
              @toggle-mode="toggleAuthMode"
              @update:login-username="loginUsername = $event"
              @update:login-password="loginPassword = $event"
              @update:signup-username="signupUsername = $event"
              @update:signup-password="signupPassword = $event"
              @update:signup-classroom-code="signupClassroomCode = $event"
            />

            <SettingsUpdate />
          </div>
        </div>

        <!-- Developer Mode Category -->
        <div v-show="activeCategory === 'dev'" class="category-pane">
          <header class="pane-header">
            <h2>Developer Mode</h2>
            <p class="pane-subtitle">
              Low-level diagnostics, task history logs, and raw reading page range verification.
            </p>
          </header>

          <div class="pane-body">
            <SettingsDeveloperPanel />
          </div>
        </div>
      </div>
    </main>


    <!-- Add Profile Modal -->
    <SettingsProfileModal
      v-if="showAddModal"
      @close="showAddModal = false"
      @save="handleAddProfile"
    />

    <!-- Edit Profile Modal -->
    <SettingsProfileModal
      v-if="showEditModal"
      is-edit
      :initial-name="editProfileName"
      :initial-deadline="editProfileDeadline"
      @close="closeEditModal"
      @save="handleUpdateProfile"
    />

    <!-- RAG Setup Modal -->
    <SettingsRagModal
      v-if="showRagModal"
      :is-setting-up-rag="isSettingUpRag"
      :rag-status="ragStatus"
      :rag-percent="ragPercent"
      :rag-message="ragMessage"
      :rag-detail="ragDetail"
      :rag-error="ragError"
      :rag-setup-completed="ragSetupCompleted"
      @dismiss="handleRagModalDismiss"
      @start="startRagSetup"
      @finish="closeRagModal"
    />
  </section>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getAppEnv, getCloudConfig, triggerCloudSync, setActiveProfileID } from '../services/appApi'

import { useSettings } from '../composables/useSettings'
import { useLLM } from '../composables/useLLM'
import { useProfiles } from '../composables/useProfiles'
import { useRAG } from '../composables/useRAG'
import { useAuth } from '../composables/useAuth'

import BaseIcon from '../components/BaseIcon.vue'
import SettingsStudyBudget from '../components/SettingsStudyBudget.vue'
import SettingsQuizRescue from '../components/SettingsQuizRescue.vue'
import SettingsAIProvider from '../components/SettingsAIProvider.vue'
import SettingsRagRetrieval from '../components/SettingsRagRetrieval.vue'
import SettingsExtensions from '../components/SettingsExtensions.vue'
import SettingsTheme from '../components/SettingsTheme.vue'
import SettingsUpdate from '../components/SettingsUpdate.vue'
import SettingsAccount from '../components/SettingsAccount.vue'
import SettingsProfilesPanel from '../components/SettingsProfilesPanel.vue'
import SettingsTextbooksPanel from '../components/SettingsTextbooksPanel.vue'
import SettingsProfileModal from '../components/SettingsProfileModal.vue'
import SettingsRagModal from '../components/SettingsRagModal.vue'
import SettingsDeveloperPanel from '../components/SettingsDeveloperPanel.vue'

const route = useRoute()

const categories = [
  {
    id: 'study',
    label: 'Study & Routine',
    desc: 'Budgets, schedules, quiz rules',
    keywords: ['study', 'budget', 'tokens', 'duration', 'quiz', 'rescue', 'remediation', 'reading', 'schedule', 'volume', 'review limits', 'pomodoro']
  },
  {
    id: 'ai',
    label: 'AI & Retrieval',
    desc: 'LLM models, API keys, RAG',
    keywords: ['ai', 'llm', 'retrieval', 'rag', 'onnx', 'vector', 'embeddings', 'api key', 'keys', 'fast', 'heavy', 'gemini', 'openai', 'groq', 'openrouter', 'ollama', 'provider', 'tutor', 'persona', 'socratic', 'limits']
  },
  {
    id: 'profiles',
    label: 'Profiles & Notebooks',
    desc: 'Exam goals and book mapping',
    keywords: ['profile', 'profiles', 'notebooks', 'textbooks', 'books', 'exam', 'deadline', 'pace', 'syllabus', 'assign']
  },
  {
    id: 'extensions',
    label: 'Extensions & Tools',
    desc: 'Podcast voice, simplifier, AI tools',
    keywords: ['extension', 'extensions', 'tools', 'podcast', 'voice', 'audio', 'simplifier', 'comprehension', 'tts', 'speech']
  },
  {
    id: 'system',
    label: 'System & Account',
    desc: 'Themes, cloud sync, updates',
    keywords: ['system', 'account', 'theme', 'dark mode', 'light mode', 'appearance', 'cloud', 'sync', 'student', 'login', 'signup', 'classroom', 'update', 'version']
  },
  {
    id: 'dev',
    label: 'Developer Mode',
    desc: 'Diagnostics, reading history & dev logs',
    keywords: ['developer', 'dev', 'diagnostics', 'logs', 'history', 'debug', 'vector db', 'tasks', 'reading history']
  },
]

const searchQuery = ref('')
const activeCategory = ref('study')
const isDev = ref(false)
const cloudConfigured = ref(false)
const syncing = ref(false)
const error = ref('')
const success = ref('')

const categoryMatches = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return {}

  const matches = {}
  categories.forEach((cat) => {
    let count = 0
    if (cat.label.toLowerCase().includes(q)) count += 2
    if (cat.desc.toLowerCase().includes(q)) count += 1
    cat.keywords?.forEach((kw) => {
      if (kw.includes(q)) count += 1
    })
    if (count > 0) {
      matches[cat.id] = count
    }
  })
  return matches
})

const filteredCategories = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return categories
  return categories.filter((cat) => categoryMatches.value[cat.id] > 0)
})

watch(searchQuery, (newQuery) => {
  const q = newQuery.trim().toLowerCase()
  if (!q) return
  if (!categoryMatches.value[activeCategory.value]) {
    const firstMatch = filteredCategories.value[0]
    if (firstMatch) {
      activeCategory.value = firstMatch.id
    }
  }
})

// Composables — all share the same error/success refs
const {
  settings,
  loading,
  saving,
  studyDuration,
  applyDurationPreset,
  loadSettings,
  cleanup: cleanupSettings,
} = useSettings(error, success)

const {
  llmSettings,
  llmFastKey,
  llmHeavyKey,
  savingLLM,
  applyProviderPreset,
  removeLLMKeys,
  loadLLM,
  cleanup: cleanupLLM,
} = useLLM(loading, error, success)

const {
  profiles,
  notebooks,
  showAddModal,
  showEditModal,
  editProfileName,
  editProfileDeadline,
  openEditModal,
  closeEditModal,
  handleAddProfile,
  handleUpdateProfile,
  handleDeleteProfile,
  handleAssignProfile,
  loadProfiles,
  loadNotebooks,
  formatUnixDate,
} = useProfiles(error)

const {
  showRagModal,
  isSettingUpRag,
  ragStatus,
  ragPercent,
  ragMessage,
  ragDetail,
  ragError,
  ragSetupCompleted,
  onRagToggle,
  openRagModal,
  startRagSetup,
  handleRagModalDismiss,
  closeRagModal,
  cleanup: cleanupRAG,
} = useRAG(settings)

const {
  isSignUpMode,
  loginUsername,
  loginPassword,
  signupUsername,
  signupPassword,
  signupClassroomCode,
  loginError,
  loggingIn,
  toggleAuthMode,
  handleLogin,
  handleSignUp,
  handleLogout,
} = useAuth(loadAllData, error, success)

const activeProfileName = computed(() => {
  const p = profiles.value.find((pr) => pr.id === settings.value.active_profile_id)
  return p ? p.name : ''
})

const isCloudAccount = computed(() => {
  return cloudConfigured.value && !!(
    settings.value?.classroom_code ||
    settings.value?.cloud_api_token ||
    settings.value?.cloud_student_id ||
    settings.value?.cloud_session_token
  )
})

async function loadAllData() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    await Promise.all([loadSettings(), loadLLM(), loadProfiles(), loadNotebooks()])
  } catch (err) {
    error.value = err.message || 'Failed to load settings data'
    console.error('loadAllData error:', err)
  } finally {
    loading.value = false
  }
}

async function setActiveProfile(profileID) {
  const prev = settings.value.active_profile_id
  settings.value.active_profile_id = profileID
  try {
    const res = await setActiveProfileID(profileID)
    if (res?.error) {
      settings.value.active_profile_id = prev
      error.value = res.error
      return
    }
    await loadSettings()
    window.dispatchEvent(new CustomEvent('settings-updated'))
  } catch (err) {
    settings.value.active_profile_id = prev
    error.value = 'Failed to switch profile'
  }
}

async function runManualSync() {
  error.value = ''
  success.value = ''
  syncing.value = true
  try {
    const res = await triggerCloudSync()
    if (res.error) {
      error.value = res.error
      return
    }
    await loadAllData()
    success.value = 'Sync completed successfully!'
    setTimeout(() => (success.value = ''), 4000)
  } catch (err) {
    error.value = err.message || 'Failed to sync with cloud'
  } finally {
    syncing.value = false
  }
}

watch(
  () => settings.value.theme,
  (newTheme) => {
    if (!newTheme || loading.value) return
    document.documentElement.setAttribute('data-theme', newTheme)
  }
)

onMounted(async () => {
  if (route.query.category && categories.some((c) => c.id === route.query.category)) {
    activeCategory.value = route.query.category
  } else if (route.query.tab && categories.some((c) => c.id === route.query.tab)) {
    activeCategory.value = route.query.tab
  }

  const [envRes, cfgRes] = await Promise.all([
    getAppEnv().catch(() => null),
    getCloudConfig().catch(() => null),
  ])
  isDev.value = envRes?.env === 'dev'
  cloudConfigured.value = cfgRes?.configured === true
  await loadAllData()
})

onUnmounted(() => {
  cleanupSettings()
  cleanupLLM()
  cleanupRAG()
})
</script>

<style scoped>
.settings-workspace {
  display: flex;
  margin: -16px -20px;
  height: calc(100% + 32px);
  min-height: calc(100vh);
  overflow: hidden;
  font-family: 'Inter', sans-serif;
  color: var(--on-surface);
  background: var(--background);
}

/* Solid Left Category Sidebar Rail */
.settings-category-rail {
  width: 260px;
  min-width: 260px;
  background: var(--surface-container-low);
  border-right: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
  display: flex;
  flex-direction: column;
  padding: 24px 16px;
  gap: 20px;
  height: 100%;
  overflow-y: auto;
  flex-shrink: 0;
}

.rail-header {
  padding: 0 8px 12px;
  border-bottom: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.05));
}

.rail-search {
  padding: 0 4px;
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 6px 10px;
  transition: all 0.15s ease;
}

.search-input-wrapper:focus-within {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 20%, transparent);
}

.search-icon {
  color: var(--muted-text);
  flex-shrink: 0;
}

.settings-search-input {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--on-surface);
  font-size: 12px;
  font-family: inherit;
}

.settings-search-input::placeholder {
  color: var(--muted-text);
}

.search-clear-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.search-clear-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container-highest);
}

.eyebrow {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
  color: var(--muted-text);
  margin: 0 0 4px;
}

.rail-title {
  margin: 0;
  font-size: 22px;
  font-family: 'Manrope', sans-serif;
  letter-spacing: -0.02em;
}

.rail-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.rail-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  border-color: transparent;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: all 0.15s ease;
  width: 100%;
}

.rail-item:hover {
  background: var(--surface-container);
}

.rail-item.active {
  background: var(--surface-container-highest);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.rail-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 6px;
}

.match-badge {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 2px 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--primary) 18%, transparent);
  color: var(--primary);
  border: 1px solid var(--outline-variant);
}

.no-search-results {
  font-size: 12px;
  color: var(--muted-text);
  padding: 16px 8px;
  text-align: center;
  line-height: 1.4;
}

.rail-item-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface);
}

.rail-item.active .rail-item-title {
  color: var(--primary);
}

.rail-item-desc {
  font-size: 11px;
  color: var(--muted-text);
  line-height: 1.3;
}

.rail-footer {
  padding-top: 12px;
  border-top: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.05));
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rail-sync-btn {
  width: 100%;
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.08));
  border-radius: 8px;
  padding: 8px 14px;
  color: var(--primary);
  font-size: 12px;
  font-weight: 700;
  background: var(--surface-container);
  cursor: pointer;
  transition: all 0.15s ease;
}

.rail-sync-btn:hover:not(:disabled) {
  background: var(--surface-container-highest);
}

.rail-sync-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Main Content Area */
.settings-content-viewport {
  flex: 1;
  min-width: 0;
  height: 100%;
  overflow-y: auto;
  padding: 24px 40px 48px;
  display: flex;
  flex-direction: column;
}

.settings-content-scroll {
  width: 100%;
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.category-pane {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.pane-header {
  border-bottom: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
  padding-bottom: 12px;
}

.pane-header h2 {
  font-size: 22px;
  font-family: 'Manrope', sans-serif;
  margin: 0 0 4px;
}

.pane-subtitle {
  font-size: 13px;
  color: var(--muted-text);
  margin: 0;
}

.pane-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.profiles-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.error-text {
  color: #ef4444;
  font-size: 12px;
  margin: 0;
}

.success-text {
  color: #10b981;
  font-size: 12px;
  margin: 0;
}

@media (max-width: 900px) {
  .settings-workspace {
    flex-direction: column;
    margin: 0;
    height: auto;
    min-height: 100vh;
    overflow: visible;
  }
  .settings-category-rail {
    width: 100%;
    min-width: 100%;
    height: auto;
    border-right: none;
    border-bottom: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
    overflow: visible;
  }
  .profiles-layout {
    grid-template-columns: 1fr;
  }
  .settings-content-viewport {
    height: auto;
    overflow: visible;
    padding: 20px 0;
  }
}
</style>
