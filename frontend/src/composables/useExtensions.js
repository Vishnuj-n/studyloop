import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'
import { listExtensions, getExtensionConfig, saveExtensionConfig } from '../services/appApi'

const STORAGE_KEY = 'studyloop_extensions_enabled'

// Built-in lightweight extensions are enabled by default; external/python tools require explicit setup & opt-in
const DEFAULT_ENABLED_EXTENSIONS = {
  text_simplifier: true,
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
    level: 'eli15',
  },
  youtube: {
    auto_download: false,
    download_quality: '720p',
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

const enabledMap = ref(loadPersistedState())
const extensionsMetadata = ref([])
const extensionConfig = ref(JSON.parse(JSON.stringify(DEFAULT_EXTENSION_CONFIG)))
let metadataFetched = false
let configFetched = false

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
    if (raw && typeof raw === 'string' && raw.trim() !== '') {
      const parsed = JSON.parse(raw)
      extensionConfig.value = {
        audio_overview: { ...DEFAULT_EXTENSION_CONFIG.audio_overview, ...(parsed.audio_overview || {}) },
        text_simplifier: { ...DEFAULT_EXTENSION_CONFIG.text_simplifier, ...(parsed.text_simplifier || {}) },
        youtube: { ...DEFAULT_EXTENSION_CONFIG.youtube, ...(parsed.youtube || {}) },
        ...parsed,
      }
    }
    configFetched = true
  } catch (_) {
    // Fallback to defaults
  }
}

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
    extensionConfig.value[extensionId][key] = value
    console.log(`[useExtensions] Updating config for [${extensionId}.${key}] =>`, value)
    try {
      const payload = JSON.stringify(extensionConfig.value)
      await saveExtensionConfig(payload)
      console.log(`[useExtensions] Successfully persisted extension config:`, extensionConfig.value)
    } catch (err) {
      console.error(`[useExtensions] Failed to save extension setting [${extensionId}.${key}]:`, err)
    }
  }

  return {
    enabledMap,
    extensionsMetadata,
    extensionConfig,
    isPro,
    isEnabled,
    isExtensionActive,
    refreshExtensionsMetadata,
    refreshExtensionConfig,
    setExtensionEnabled,
    toggleExtension,
    getExtensionSetting,
    setExtensionSetting,
  }
}


