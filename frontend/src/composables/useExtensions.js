import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'

const STORAGE_KEY = 'studyloop_extensions_enabled'

// Reactive state shared across all components
const enabledMap = ref({
  text_simplifier: true,
  audio_overview: true,
  youtube_transcripts: true
})

let isInitialized = false

function loadSettings() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      enabledMap.value = { ...enabledMap.value, ...JSON.parse(saved) }
    }
  } catch (e) {
    console.error('Failed to load extension settings from localStorage', e)
  }
}

function saveSettings() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(enabledMap.value))
  } catch (e) {
    console.error('Failed to save extension settings to localStorage', e)
  }
}

export function useExtensions() {
  if (!isInitialized) {
    loadSettings()
    isInitialized = true
  }

  const clerkAuth = useClerkAuth()
  const isPro = computed(() => clerkAuth.isPro.value)

  function isEnabled(extensionId) {
    return enabledMap.value[extensionId] !== false
  }

  // Returns true only if enabled and (if pro) user has pro plan
  function isExtensionActive(extensionId) {
    const enabled = isEnabled(extensionId)
    if (!enabled) return false

    if (extensionId === 'audio_overview' || extensionId === 'youtube_transcripts') {
      return isPro.value
    }
    return true
  }

  function setExtensionEnabled(extensionId, enabled) {
    enabledMap.value = {
      ...enabledMap.value,
      [extensionId]: !!enabled
    }
    saveSettings()
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
