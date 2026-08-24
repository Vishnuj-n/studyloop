import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'

const STORAGE_KEY = 'studyloop_extensions_enabled'

function loadPersistedState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
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

