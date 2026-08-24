import { ref, computed } from 'vue'
import { useClerkAuth } from '../services/clerkAuth'

// Reactive state shared across all components
const enabledMap = ref({
  text_simplifier: true,
  audio_overview: true,
  youtube_transcripts: true
})

export function useExtensions() {
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

