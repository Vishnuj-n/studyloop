import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'

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
  } catch {}
}

const enabledMap = ref(loadPersistedState())

export function useExtensions() {
  const clerkAuth = useClerkAuth()
  const isPro = computed(() => clerkAuth.isPro.value)

  function isEnabled(extensionId) {
    return Boolean(enabledMap.value[extensionId])
  }

  function isExtensionActive(extensionId) {
    if (extensionId === 'audio_overview') {
      return isEnabled(extensionId) && Boolean(isPro.value)
    }
    return isEnabled(extensionId)
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
    isPro,
    isEnabled,
    isExtensionActive,
    setExtensionEnabled,
    toggleExtension
  }
}

