import { ref, watch, computed } from 'vue'
import { PET_REGISTRY } from '../config/pets'

const STORAGE_KEY = 'studyloop_pet_settings'

const defaultState = {
  enabled: true,
  activePetId: 'cat',
  activeSkinId: 'calico',
  unlockedPets: ['cat'],
  unlockedSkins: [
    'cat:calico',
    'cat:void',
    'cat:matcha',
    'cat:lavender',
    'dog:golden',
    'dragon:ruby',
  ],
  position: { x: null, y: null },
}

function loadInitialState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return defaultState
    const parsed = JSON.parse(raw)
    return {
      ...defaultState,
      ...parsed,
      unlockedPets: Array.from(new Set([...defaultState.unlockedPets, ...(parsed.unlockedPets || [])])),
      unlockedSkins: Array.from(new Set([...defaultState.unlockedSkins, ...(parsed.unlockedSkins || [])])),
    }
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

  function isPetUnlocked(petId) {
    const pet = PET_REGISTRY.find((p) => p.id === petId)
    if (!pet) return false
    if (pet.isFree) return true
    return (petState.value.unlockedPets || []).includes(petId)
  }

  function isSkinUnlocked(petId, skinId) {
    const pet = PET_REGISTRY.find((p) => p.id === petId)
    if (!pet) return false
    const skin = pet.skins.find((s) => s.id === skinId)
    if (!skin) return false
    if (skin.isFree) return true
    const key = `${petId}:${skinId}`
    return (petState.value.unlockedSkins || []).includes(key)
  }

  function unlockPet(petId) {
    if (!petState.value.unlockedPets) {
      petState.value.unlockedPets = ['cat']
    }
    if (!petState.value.unlockedPets.includes(petId)) {
      petState.value.unlockedPets.push(petId)
    }
  }

  function unlockSkin(petId, skinId) {
    if (!petState.value.unlockedSkins) {
      petState.value.unlockedSkins = []
    }
    const key = `${petId}:${skinId}`
    if (!petState.value.unlockedSkins.includes(key)) {
      petState.value.unlockedSkins.push(key)
    }
  }

  function togglePet(val) {
    petState.value.enabled = val !== undefined ? val : !petState.value.enabled
  }

  function setPet(petId) {
    const pet = PET_REGISTRY.find((p) => p.id === petId)
    if (pet && isPetUnlocked(petId)) {
      petState.value.activePetId = petId
      petState.value.activeSkinId = pet.defaultSkin || pet.skins[0].id
    }
  }

  function setSkin(skinId) {
    const pet = currentPet.value
    if (pet.skins.some((s) => s.id === skinId) && isSkinUnlocked(pet.id, skinId)) {
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
    isPetUnlocked,
    isSkinUnlocked,
    unlockPet,
    unlockSkin,
    togglePet,
    setPet,
    setSkin,
    setPosition,
    resetPosition,
  }
}
