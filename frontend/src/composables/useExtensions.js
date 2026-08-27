import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'
import { listExtensions } from '../services/appApi'

const STORAGE_KEY = 'studyloop_extensions_enabled'

// Official extensions are enabled by default out-of-the-box
const DEFAULT_ENABLED_EXTENSIONS = {
  text_simplifier: true,
  fast_pdf: true,
  youtube: true,
  audio_overview: true,
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
let metadataFetched = false

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

export function useExtensions() {
  const clerkAuth = useClerkAuth()
  const isPro = computed(() => clerkAuth.isPro.value)

  if (!metadataFetched) {
    refreshExtensionsMetadata()
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

  return {
    enabledMap,
    extensionsMetadata,
    isPro,
    isEnabled,
    isExtensionActive,
    refreshExtensionsMetadata,
    setExtensionEnabled,
    toggleExtension
  }
}

