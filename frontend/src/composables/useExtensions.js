import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'
import { listExtensions, getExtensionConfig, saveExtensionConfig, setupExtension, cancelExtensionSetup, getUserSettings, updateUserSettings } from '../services/appApi'
import { EventsOn } from '../../wailsjs/runtime/runtime'

const STORAGE_KEY = 'studyloop_extensions_enabled'
const SETUP_COMPLETED_KEY = 'studyloop_extensions_setup_completed'

// Built-in lightweight extensions are enabled by default; external/python tools require explicit setup & opt-in
const DEFAULT_ENABLED_EXTENSIONS = {
  text_simplifier: true,
  prompt_compressor: true,
  deep_pdf: false,
  youtube: false,
  audio_overview: false,
}

export const DEFAULT_EXTENSION_CONFIG = {
  audio_overview: {
    voice: 'en-US-ChristopherNeural',
    speed: 1.0,
  },
  text_simplifier: {
    level: 'simple',
  },
  youtube: {
    auto_download: false,
    download_quality: '720p',
  },
  prompt_compressor: {
    mode: 'OVER_LIMIT',
    rate: 0.80,
  },
}

function loadPersistedState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULT_ENABLED_EXTENSIONS }
    const parsed = JSON.parse(raw)
    return { ...DEFAULT_ENABLED_EXTENSIONS, ...parsed }
  } catch {
    return { ...DEFAULT_ENABLED_EXTENSIONS }
  }
}

function savePersistedState(state) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // Ignore storage write errors
  }
}

function loadSetupCompletedState() {
  try {
    const raw = localStorage.getItem(SETUP_COMPLETED_KEY)
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
  }
}

function saveSetupCompletedState(state) {
  try {
    localStorage.setItem(SETUP_COMPLETED_KEY, JSON.stringify(state))
  } catch {
    // Ignore storage write errors
  }
}

const setupCompletedMap = ref(loadSetupCompletedState())
const enabledMap = ref(loadPersistedState())
const extensionsMetadata = ref([])
const extensionConfig = ref(JSON.parse(JSON.stringify(DEFAULT_EXTENSION_CONFIG)))
let metadataFetched = false
let configFetched = false

// Singleton Extension Setup State across entire app
export const activeSetup = ref({
  extensionId: null,
  extensionName: '',
  status: 'idle', // 'idle' | 'running' | 'success' | 'error'
  step: 1, // 1: Runtime, 2: Requirements, 3: Probe
  logs: [],
  errorMessage: '',
  visibleToast: false,
})

export const setupModalState = ref({
  isOpen: false,
  extension: null,
})

let setupEventsInitialized = false
function initSetupEventListener() {
  if (setupEventsInitialized) return
  if (typeof EventsOn === 'function') {
    EventsOn('extension:setup:progress', (data) => {
      if (!data || !activeSetup.value.extensionId || data.id !== activeSetup.value.extensionId) return
      if (Array.isArray(data.logs) && data.logs.length > 0) {
        activeSetup.value.logs = data.logs
      } else if (data.log) {
        activeSetup.value.logs.push(data.log)
      }
      if (data.step && Number.isInteger(data.step)) {
        activeSetup.value.step = data.step
      }
    })
    setupEventsInitialized = true
  }
}
initSetupEventListener()

async function refreshExtensionsMetadata() {
  try {
    const exts = await listExtensions()
    if (Array.isArray(exts)) {
      extensionsMetadata.value = exts
      metadataFetched = true
    }
  } catch (_) {
    // Ignore metadata fetch errors
  }
}

async function refreshExtensionConfig() {
  try {
    const raw = await getExtensionConfig()
    let parsed = {}
    if (raw && typeof raw === 'string' && raw.trim() !== '') {
      try {
        parsed = JSON.parse(raw)
      } catch {}
    }

    let userMode = null
    let userRate = null
    try {
      const u = await getUserSettings()
      if (u) {
        if (u.prompt_compression_mode) userMode = u.prompt_compression_mode
        if (typeof u.prompt_compression_rate === 'number' && u.prompt_compression_rate > 0) {
          userRate = u.prompt_compression_rate
        }
      }
    } catch {}

    extensionConfig.value = {
      audio_overview: { ...DEFAULT_EXTENSION_CONFIG.audio_overview, ...(parsed.audio_overview || {}) },
      text_simplifier: { ...DEFAULT_EXTENSION_CONFIG.text_simplifier, ...(parsed.text_simplifier || {}) },
      youtube: { ...DEFAULT_EXTENSION_CONFIG.youtube, ...(parsed.youtube || {}) },
      prompt_compressor: {
        ...DEFAULT_EXTENSION_CONFIG.prompt_compressor,
        ...(parsed.prompt_compressor || {}),
        ...(userMode ? { mode: userMode } : {}),
        ...(userRate ? { rate: userRate } : {}),
      },
      ...parsed,
    }
    configFetched = true
  } catch (_) {
    // Fallback to defaults
  }
}

const configError = ref('')

export function useExtensions() {
  const clerkAuth = useClerkAuth()
  const isPro = computed(() => clerkAuth.isPro.value)

  if (!metadataFetched) {
    refreshExtensionsMetadata()
  }

  if (!configFetched) {
    refreshExtensionConfig()
  }

  function isEnabled(extensionId) {
    return Boolean(enabledMap.value[extensionId])
  }

  function hasCompletedSetup(extensionId) {
    return Boolean(setupCompletedMap.value[extensionId])
  }

  function markSetupCompleted(extensionId, completed = true) {
    setupCompletedMap.value = {
      ...setupCompletedMap.value,
      [extensionId]: Boolean(completed)
    }
    saveSetupCompletedState(setupCompletedMap.value)
  }

  function isExtensionActive(extensionId) {
    if (!isEnabled(extensionId)) {
      return false
    }
    const ext = extensionsMetadata.value.find((e) => e.id === extensionId)
    if (ext && (ext.tier || '').toLowerCase() === 'pro') {
      return Boolean(isPro.value)
    }
    return true
  }

  function setExtensionEnabled(extensionId, enabled) {
    enabledMap.value = {
      ...enabledMap.value,
      [extensionId]: Boolean(enabled)
    }
    savePersistedState(enabledMap.value)
  }

  function toggleExtension(extensionId) {
    setExtensionEnabled(extensionId, !isEnabled(extensionId))
  }

  function getExtensionSetting(extensionId, key, fallback = null) {
    const extConf = extensionConfig.value[extensionId]
    if (extConf && extConf[key] !== undefined) {
      return extConf[key]
    }
    return fallback
  }

  async function setExtensionSetting(extensionId, key, value) {
    if (!extensionConfig.value[extensionId]) {
      extensionConfig.value[extensionId] = {}
    }
    const priorValue = extensionConfig.value[extensionId][key]
    extensionConfig.value[extensionId][key] = value
    configError.value = ''
    console.log(`[useExtensions] Updating config for [${extensionId}.${key}] =>`, value)
    try {
      const payload = JSON.stringify(extensionConfig.value)
      await saveExtensionConfig(payload)
      console.log(`[useExtensions] Successfully persisted extension config:`, extensionConfig.value)

      if (extensionId === 'prompt_compressor') {
        try {
          const current = await getUserSettings()
          if (current) {
            if (key === 'mode') {
              current.prompt_compression_mode = String(value)
            } else if (key === 'rate') {
              current.prompt_compression_rate = Number(value)
            }
            await updateUserSettings(current)
          }
        } catch (syncErr) {
          console.warn('[useExtensions] Failed syncing prompt_compressor to user_settings:', syncErr)
        }
      }
    } catch (err) {
      extensionConfig.value[extensionId][key] = priorValue
      configError.value = err?.message || String(err)
      console.error(`[useExtensions] Failed to save extension setting [${extensionId}.${key}]:`, err)
      throw err
    }
  }

  function openSetupModal(ext) {
    if (!ext) return
    setupModalState.value = {
      isOpen: true,
      extension: ext,
    }
  }

  function closeSetupModal() {
    setupModalState.value = {
      isOpen: false,
      extension: null,
    }
  }

  function dismissSetupToast() {
    activeSetup.value.visibleToast = false
  }

  async function startSetup(ext) {
    if (!ext || !ext.id) return

    // If a different setup is currently running, don't clobber
    if (activeSetup.value.status === 'running' && activeSetup.value.extensionId !== ext.id) {
      console.warn('[useExtensions] Another setup is currently running:', activeSetup.value.extensionId)
      return
    }

    activeSetup.value = {
      extensionId: ext.id,
      extensionName: ext.name || 'Extension',
      status: 'running',
      step: 1,
      logs: ['Checking environment...'],
      errorMessage: '',
      visibleToast: true,
    }

    try {
      const res = await setupExtension(ext.id)
      if (res && res.success) {
        activeSetup.value.logs = res.logs || ['Setup completed successfully.']
        activeSetup.value.step = 3
        activeSetup.value.status = 'success'
        markSetupCompleted(ext.id, true)
        setExtensionEnabled(ext.id, true)
      } else if (res && res.canceled) {
        activeSetup.value.status = 'idle'
        activeSetup.value.visibleToast = false
      } else {
        activeSetup.value.status = 'error'
        activeSetup.value.logs = res?.logs || activeSetup.value.logs
        activeSetup.value.errorMessage = res?.error || 'Setup failed to complete.'
      }
    } catch (err) {
      activeSetup.value.status = 'error'
      activeSetup.value.errorMessage = err?.message || String(err)
    }
  }

  async function cancelSetup() {
    if (activeSetup.value.status === 'running') {
      try {
        await cancelExtensionSetup()
      } catch (err) {
        console.error('[useExtensions] Failed to cancel setup:', err)
      }
    }
    activeSetup.value.status = 'idle'
    activeSetup.value.visibleToast = false
    closeSetupModal()
  }

  return {
    enabledMap,
    extensionsMetadata,
    extensionConfig,
    configError,
    activeSetup,
    setupModalState,
    isPro,
    isEnabled,
    isExtensionActive,
    refreshExtensionsMetadata,
    refreshExtensionConfig,
    setExtensionEnabled,
    toggleExtension,
    hasCompletedSetup,
    markSetupCompleted,
    getExtensionSetting,
    setExtensionSetting,
    openSetupModal,
    closeSetupModal,
    dismissSetupToast,
    startSetup,
    cancelSetup,
  }
}


