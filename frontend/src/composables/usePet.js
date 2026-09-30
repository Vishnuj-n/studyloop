import { ref, watch, computed } from 'vue'
import { PET_REGISTRY } from '../config/pets'

const STORAGE_KEY = 'studyloop_pet_settings'

const defaultState = {
  enabled: true,
  activePetId: 'cat',
  activeSkinId: 'calico',
  position: { x: null, y: null },
}

function loadInitialState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return defaultState
    const parsed = JSON.parse(raw)
    return { ...defaultState, ...parsed }
  } catch {
    return defaultState
  }
}

const petState = ref(loadInitialState())

watch(
  petState,
  (newVal) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(newVal))
    } catch (err) {
      console.debug('[usePet] Failed to persist pet state:', err)
    }
  },
  { deep: true }
)

export function usePet() {
  const currentPet = computed(() => {
    return PET_REGISTRY.find((p) => p.id === petState.value.activePetId) || PET_REGISTRY[0]
  })

  const currentSkin = computed(() => {
    const pet = currentPet.value
    return pet.skins.find((s) => s.id === petState.value.activeSkinId) || pet.skins[0]
  })

  function togglePet(val) {
    petState.value.enabled = val !== undefined ? val : !petState.value.enabled
  }

  function setPet(petId) {
    const pet = PET_REGISTRY.find((p) => p.id === petId)
    if (pet) {
      petState.value.activePetId = petId
      petState.value.activeSkinId = pet.defaultSkin || pet.skins[0].id
    }
  }

  function setSkin(skinId) {
    const pet = currentPet.value
    if (pet.skins.some((s) => s.id === skinId)) {
      petState.value.activeSkinId = skinId
    }
  }

  function setPosition(x, y) {
    petState.value.position = { x, y }
  }

  function resetPosition() {
    petState.value.position = { x: null, y: null }
  }

  return {
    petState,
    currentPet,
    currentSkin,
    togglePet,
    setPet,
    setSkin,
    setPosition,
    resetPosition,
  }
}
