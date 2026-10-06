<template>
  <div class="pet-sanctuary-hub">
    <!-- Sanctuary Header -->
    <div class="sanctuary-header">
      <div class="header-titles">
        <h3 class="sanctuary-title">Pet Sanctuary & Companions</h3>
        <p class="sanctuary-subtitle">
          Adopt cozy study companions, unlock custom skins with your hard-earned study coins, and let them cheer you on.
        </p>
      </div>

      <div class="header-actions">
        <div v-if="coins !== undefined" class="coin-pill">
          <BaseIcon name="coin" size="16" />
          <span class="coin-balance">{{ coins }} Coins</span>
        </div>

        <button
          type="button"
          class="companion-toggle-btn"
          :class="{ active: petState.enabled }"
          @click="togglePet()"
        >
          <BaseIcon :name="petState.enabled ? 'check' : 'x'" size="14" />
          <span>Floating Companion: {{ petState.enabled ? 'Enabled' : 'Disabled' }}</span>
        </button>
      </div>
    </div>

    <!-- Error Alert if any -->
    <div v-if="actionError" class="sanctuary-error-banner">
      <BaseIcon name="alert-circle" size="16" />
      <span>{{ actionError }}</span>
      <button class="banner-close" @click="actionError = ''">×</button>
    </div>

    <!-- Main Sanctuary Grid: Showcase & Selection -->
    <div class="sanctuary-layout">
      <!-- Left Column: Pet Showcase Stage -->
      <div class="showcase-card">
        <div class="showcase-header">
          <div class="showcase-badge-row">
            <span class="pet-species-badge">{{ selectedPet.title || selectedPet.name }}</span>
            <span v-if="petState.activePetId === selectedPet.id" class="active-equipped-badge">
              <BaseIcon name="sparkles" size="12" /> Active Companion
            </span>
          </div>
          <h4 class="showcase-pet-name">{{ selectedPet.name }}</h4>
          <p class="showcase-pet-desc">{{ selectedPet.description }}</p>
        </div>

        <!-- Live Interactive Stage -->
        <div class="preview-stage">
          <div
            class="stage-avatar-wrapper"
            :class="[`action-${previewAction}`]"
          >
            <!-- Speech Bubble on Stage -->
            <transition name="bubble-pop">
              <div v-if="stageMessage" class="stage-speech-bubble" @click="stageMessage = null">
                <span>{{ stageMessage }}</span>
                <div class="stage-bubble-tail"></div>
              </div>
            </transition>

            <!-- Render Vector Avatar for Selected Pet & Skin -->
            <svg
              viewBox="0 0 100 100"
              class="stage-pet-svg"
              xmlns="http://www.w3.org/2000/svg"
            >
              <ellipse cx="50" cy="88" rx="28" ry="6" class="stage-shadow" />

              <!-- Cat (Mochi) -->
              <g v-if="selectedPet.id === 'cat'">
                <path d="M 24 72 C 14 70 10 58 12 50 C 14 54 20 62 26 66" :fill="previewSkin.accentColor" class="pet-tail" />
                <path d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z" :fill="previewSkin.primaryColor" class="pet-body" />
                <ellipse cx="50" cy="62" rx="15" ry="14" :fill="previewSkin.secondaryColor" />
                <polygon points="30,34 38,14 48,30" :fill="previewSkin.primaryColor" class="pet-ear ear-left" />
                <polygon points="33,31 38,18 45,28" :fill="previewSkin.secondaryColor" />
                <polygon points="70,34 62,14 52,30" :fill="previewSkin.primaryColor" class="pet-ear ear-right" />
                <polygon points="67,31 62,18 55,28" :fill="previewSkin.secondaryColor" />

                <g v-if="previewAction === 'sleep'">
                  <path d="M 38 46 Q 42 50 46 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                  <path d="M 54 46 Q 58 50 62 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                </g>
                <g v-else-if="previewAction === 'blink'">
                  <line x1="38" y1="46" x2="46" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                  <line x1="54" y1="46" x2="62" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                </g>
                <g v-else>
                  <ellipse cx="42" cy="45" rx="3.5" ry="4.5" fill="#1E293B" />
                  <circle cx="43.5" cy="43.5" r="1.5" fill="#FFFFFF" />
                  <ellipse cx="58" cy="45" rx="3.5" ry="4.5" fill="#1E293B" />
                  <circle cx="59.5" cy="43.5" r="1.5" fill="#FFFFFF" />
                </g>
                <polygon points="48,51 52,51 50,54" fill="#F43F5E" />
                <path d="M 46 55 Q 50 58 50 55 Q 50 58 54 55" fill="none" stroke="#1E293B" stroke-width="1.5" stroke-linecap="round" />
                <line x1="28" y1="48" x2="36" y2="50" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
                <line x1="28" y1="54" x2="36" y2="53" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
                <line x1="64" y1="50" x2="72" y2="48" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
                <line x1="64" y1="53" x2="72" y2="54" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
              </g>

              <!-- Dog (Buster) -->
              <g v-else-if="selectedPet.id === 'dog'">
                <path d="M 74 72 C 84 68 88 56 86 48 C 84 52 78 60 72 66" :fill="previewSkin.accentColor" class="pet-tail" />
                <path d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z" :fill="previewSkin.primaryColor" class="pet-body" />
                <path d="M 30 32 C 20 34 16 46 18 56 C 20 60 25 60 27 54 C 29 48 31 38 33 34 Z" :fill="previewSkin.accentColor" class="pet-ear ear-left" />
                <path d="M 70 32 C 80 34 84 46 82 56 C 80 60 75 60 73 54 C 71 48 69 38 67 34 Z" :fill="previewSkin.accentColor" class="pet-ear ear-right" />
                <ellipse cx="50" cy="63" rx="16" ry="14" :fill="previewSkin.secondaryColor" />
                <ellipse cx="50" cy="52" rx="12" ry="9" :fill="previewSkin.secondaryColor" />

                <g v-if="previewAction === 'sleep'">
                  <path d="M 38 45 Q 42 49 46 45" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                  <path d="M 54 45 Q 58 49 62 45" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                </g>
                <g v-else-if="previewAction === 'blink'">
                  <line x1="38" y1="45" x2="46" y2="45" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                  <line x1="54" y1="45" x2="62" y2="45" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                </g>
                <g v-else>
                  <circle cx="41" cy="44" r="4" fill="#1E293B" />
                  <circle cx="42.5" cy="42.5" r="1.5" fill="#FFFFFF" />
                  <circle cx="59" cy="44" r="4" fill="#1E293B" />
                  <circle cx="60.5" cy="42.5" r="1.5" fill="#FFFFFF" />
                </g>
                <ellipse cx="50" cy="50" rx="4.5" ry="3.5" fill="#1E293B" />
                <path d="M 46 54 Q 50 57 50 54 Q 50 57 54 54" fill="none" stroke="#1E293B" stroke-width="1.6" stroke-linecap="round" />
              </g>

              <!-- Dragon (Ignis) -->
              <g v-else-if="selectedPet.id === 'dragon'">
                <path d="M 28 50 C 14 38 12 24 24 22 C 22 30 20 40 28 48 Z" :fill="previewSkin.accentColor" class="pet-ear ear-left" />
                <path d="M 72 50 C 86 38 88 24 76 22 C 78 30 80 40 72 48 Z" :fill="previewSkin.accentColor" class="pet-ear ear-right" />
                <path d="M 22 74 C 10 72 6 56 10 46 C 14 52 18 64 24 68" :fill="previewSkin.primaryColor" class="pet-tail" />
                <polygon points="10,46 6,40 14,44" :fill="previewSkin.accentColor" />
                <path d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z" :fill="previewSkin.primaryColor" class="pet-body" />
                <polygon points="32,32 26,12 40,26" :fill="previewSkin.accentColor" class="pet-ear ear-left" />
                <polygon points="68,32 74,12 60,26" :fill="previewSkin.accentColor" class="pet-ear ear-right" />
                <path d="M 40 50 Q 50 54 60 50 Q 50 62 40 50 Z" :fill="previewSkin.secondaryColor" />
                <path d="M 38 60 Q 50 65 62 60 Q 50 72 38 60 Z" :fill="previewSkin.secondaryColor" />
                <path d="M 42 70 Q 50 74 58 70 Q 50 79 42 70 Z" :fill="previewSkin.secondaryColor" />

                <g v-if="previewAction === 'sleep'">
                  <path d="M 37 46 Q 42 50 47 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                  <path d="M 53 46 Q 58 50 63 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                </g>
                <g v-else-if="previewAction === 'blink'">
                  <line x1="37" y1="46" x2="47" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                  <line x1="53" y1="46" x2="63" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
                </g>
                <g v-else>
                  <polygon points="42,42 46,45 42,48 38,45" fill="#FEF08A" stroke="#1E293B" stroke-width="1" />
                  <line x1="42" y1="43" x2="42" y2="47" stroke="#1E293B" stroke-width="1.8" />
                  <polygon points="58,42 62,45 58,48 54,45" fill="#FEF08A" stroke="#1E293B" stroke-width="1" />
                  <line x1="58" y1="43" x2="58" y2="47" stroke="#1E293B" stroke-width="1.8" />
                </g>
                <ellipse cx="47" cy="52" rx="1.5" ry="1" fill="#1E293B" />
                <ellipse cx="53" cy="52" rx="1.5" ry="1" fill="#1E293B" />
              </g>

              <!-- Coffee Mug for coffee action -->
              <g v-if="previewAction === 'coffee'">
                <rect x="44" y="66" width="12" height="11" rx="2" fill="#E2E8F0" stroke="#475569" stroke-width="1" />
                <path d="M 56 68 Q 60 71 56 74" fill="none" stroke="#475569" stroke-width="1" />
                <path d="M 48 63 Q 50 60 50 58" fill="none" stroke="#94A3B8" stroke-width="1" stroke-linecap="round" class="steam-line" />
              </g>

              <ellipse cx="40" cy="78" rx="5" ry="4" :fill="previewSkin.secondaryColor" />
              <ellipse cx="60" cy="78" rx="5" ry="4" :fill="previewSkin.secondaryColor" />
            </svg>
          </div>
        </div>

        <!-- Animation Testing Controls -->
        <div class="stage-controls">
          <span class="controls-label">Test Behavior:</span>
          <div class="action-btn-group">
            <button
              v-for="act in (selectedPet.actions || ['idle', 'blink', 'coffee', 'wiggle', 'cheer', 'sleep'])"
              :key="act"
              type="button"
              class="stage-act-btn"
              :class="{ active: previewAction === act }"
              @click="triggerAction(act)"
            >
              {{ act }}
            </button>
            <button type="button" class="stage-act-btn msg-btn" @click="testSpeechBubble">
              💬 Tip
            </button>
          </div>
        </div>

        <!-- Stage Footer: Equip / Buy Action -->
        <div class="stage-footer">
          <div v-if="!isPetUnlocked(selectedPet.id)" class="stage-unlock-box">
            <div class="cost-callout">
              <span>Adoption Fee:</span>
              <strong><BaseIcon name="coin" size="14" /> {{ selectedPet.priceCoins }} Coins</strong>
            </div>
            <button
              type="button"
              class="primary-adopt-btn"
              :disabled="unlocking === selectedPet.id || (coins || 0) < selectedPet.priceCoins"
              @click="handleUnlockPet(selectedPet)"
            >
              {{ unlocking === selectedPet.id ? 'Adopting Companion...' : `Adopt ${selectedPet.name} for ${selectedPet.priceCoins} Coins` }}
            </button>
          </div>

          <div v-else class="stage-equipped-box">
            <button
              v-if="petState.activePetId !== selectedPet.id"
              type="button"
              class="primary-equip-btn"
              @click="equipSelectedPet()"
            >
              Set as Active Companion
            </button>
            <div v-else class="active-pill-note">
              <BaseIcon name="check" size="14" /> {{ selectedPet.name }} is currently by your side
            </div>
            <button type="button" class="stage-reset-btn" @click="resetPosition()">
              Reset Screen Coordinates
            </button>
          </div>
        </div>
      </div>

      <!-- Right Column: Pet Roster & Skin Atelier -->
      <div class="roster-column">
        <!-- 1. Companion Pet Selection -->
        <div class="card roster-card">
          <div class="card-head">
            <h4 class="card-title">Study Companions</h4>
            <span class="card-subtitle">Choose a friend to study with</span>
          </div>

          <div class="pets-grid">
            <div
              v-for="pet in PET_REGISTRY"
              :key="pet.id"
              class="pet-select-card"
              :class="{
                selected: selectedPetId === pet.id,
                locked: !isPetUnlocked(pet.id),
                active: petState.activePetId === pet.id
              }"
              @click="selectPet(pet.id)"
            >
              <div class="pet-card-top">
                <div class="pet-mini-preview" :style="{ background: pet.skins[0].primaryColor }">
                  <span class="pet-mini-accent" :style="{ background: pet.skins[0].secondaryColor }"></span>
                </div>
                <div class="pet-card-meta">
                  <span class="pet-card-name">{{ pet.name }}</span>
                  <span class="pet-card-title">{{ pet.title || 'Companion' }}</span>
                </div>
              </div>

              <div class="pet-card-bottom">
                <span v-if="petState.activePetId === pet.id" class="badge equipped-tag">
                  Active
                </span>
                <span v-else-if="isPetUnlocked(pet.id)" class="badge unlocked-tag">
                  Unlocked
                </span>
                <span v-else class="badge price-tag">
                  <BaseIcon name="coin" size="12" /> {{ pet.priceCoins }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 2. Skin Wardrobe for Selected Pet -->
        <div class="card wardrobe-card">
          <div class="card-head">
            <h4 class="card-title">{{ selectedPet.name }}'s Skin Wardrobe</h4>
            <span class="card-subtitle">Change appearances & color palettes</span>
          </div>

          <div class="skins-grid">
            <div
              v-for="skin in selectedPet.skins"
              :key="skin.id"
              class="skin-shop-item"
              :class="{
                selected: selectedSkinId === skin.id,
                active: petState.activePetId === selectedPet.id && petState.activeSkinId === skin.id,
                locked: !isSkinUnlocked(selectedPet.id, skin.id)
              }"
              @click="selectSkin(skin.id)"
            >
              <div class="skin-swatch-box" :style="{ background: skin.primaryColor }">
                <span class="swatch-secondary" :style="{ background: skin.secondaryColor }"></span>
                <span class="swatch-accent" :style="{ background: skin.accentColor }"></span>
              </div>

              <div class="skin-info">
                <span class="skin-name">{{ skin.name }}</span>
                <div class="skin-status-row">
                  <span
                    v-if="petState.activePetId === selectedPet.id && petState.activeSkinId === skin.id"
                    class="badge active-badge"
                  >
                    Equipped
                  </span>
                  <span v-else-if="isSkinUnlocked(selectedPet.id, skin.id)" class="badge unlocked-badge">
                    Owned
                  </span>
                  <span v-else class="badge price-badge">
                    <BaseIcon name="coin" size="11" /> {{ skin.priceCoins }}
                  </span>
                </div>
              </div>

              <!-- Inline Action for Skin -->
              <div class="skin-action-col">
                <button
                  v-if="!isSkinUnlocked(selectedPet.id, skin.id)"
                  type="button"
                  class="skin-buy-btn"
                  :disabled="unlocking === `${selectedPet.id}:${skin.id}` || (coins || 0) < skin.priceCoins"
                  @click.stop="handleUnlockSkin(selectedPet, skin)"
                >
                  {{ unlocking === `${selectedPet.id}:${skin.id}` ? '...' : 'Buy' }}
                </button>
                <button
                  v-else-if="petState.activePetId === selectedPet.id && petState.activeSkinId !== skin.id"
                  type="button"
                  class="skin-equip-btn"
                  @click.stop="applySkin(skin.id)"
                >
                  Equip
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { usePet } from '../composables/usePet'
import { PET_REGISTRY, PET_MESSAGES } from '../config/pets'
import { unlockCosmeticItem } from '../services/appApi'

const props = defineProps({
  coins: { type: Number, default: 0 },
})

const emit = defineEmits(['coins-updated'])

const {
  petState,
  isPetUnlocked,
  isSkinUnlocked,
  unlockPet,
  unlockSkin,
  setPet,
  setSkin,
  togglePet,
  resetPosition,
} = usePet()

const selectedPetId = ref(petState.value.activePetId || 'cat')
const selectedSkinId = ref(petState.value.activeSkinId || 'calico')
const previewAction = ref('idle')
const stageMessage = ref(null)
const unlocking = ref(null)
const actionError = ref('')

let actionTimer = null
let messageTimer = null

const selectedPet = computed(() => {
  return PET_REGISTRY.find((p) => p.id === selectedPetId.value) || PET_REGISTRY[0]
})

const previewSkin = computed(() => {
  const pet = selectedPet.value
  return pet.skins.find((s) => s.id === selectedSkinId.value) || pet.skins[0]
})

watch(selectedPetId, (newPetId) => {
  const pet = PET_REGISTRY.find((p) => p.id === newPetId)
  if (pet) {
    if (newPetId === petState.value.activePetId) {
      selectedSkinId.value = petState.value.activeSkinId
    } else {
      selectedSkinId.value = pet.defaultSkin || pet.skins[0].id
    }
  }
})

function selectPet(petId) {
  selectedPetId.value = petId
}

function selectSkin(skinId) {
  selectedSkinId.value = skinId
}

function triggerAction(action) {
  if (actionTimer) clearTimeout(actionTimer)
  previewAction.value = action
  actionTimer = setTimeout(() => {
    previewAction.value = 'idle'
  }, 2200)
}

function testSpeechBubble() {
  if (messageTimer) clearTimeout(messageTimer)
  const randomMsg = PET_MESSAGES[Math.floor(Math.random() * PET_MESSAGES.length)]
  stageMessage.value = randomMsg
  messageTimer = setTimeout(() => {
    stageMessage.value = null
  }, 3500)
}

function equipSelectedPet() {
  if (!isPetUnlocked(selectedPetId.value)) return
  setPet(selectedPetId.value)
  if (isSkinUnlocked(selectedPetId.value, selectedSkinId.value)) {
    setSkin(selectedSkinId.value)
  }
}

function applySkin(skinId) {
  selectedSkinId.value = skinId
  if (petState.value.activePetId === selectedPetId.value) {
    setSkin(skinId)
  }
}

async function handleUnlockPet(pet) {
  if ((props.coins || 0) < pet.priceCoins) {
    actionError.value = `You need ${pet.priceCoins} coins to adopt ${pet.name}. Complete more study sessions to earn coins!`
    return
  }

  actionError.value = ''
  unlocking.value = pet.id
  try {
    const res = await unlockCosmeticItem(`pet:${pet.id}`, pet.priceCoins)
    if (res && res.error) {
      actionError.value = res.error
      return
    }
    unlockPet(pet.id)
    setPet(pet.id)
    emit('coins-updated')
  } catch (err) {
    actionError.value = err?.message || 'Failed to adopt companion.'
  } finally {
    unlocking.value = null
  }
}

async function handleUnlockSkin(pet, skin) {
  if ((props.coins || 0) < skin.priceCoins) {
    actionError.value = `You need ${skin.priceCoins} coins to purchase this skin.`
    return
  }

  actionError.value = ''
  const key = `${pet.id}:${skin.id}`
  unlocking.value = key
  try {
    const res = await unlockCosmeticItem(`skin:${pet.id}:${skin.id}`, skin.priceCoins)
    if (res && res.error) {
      actionError.value = res.error
      return
    }
    unlockSkin(pet.id, skin.id)
    if (petState.value.activePetId === pet.id) {
      setSkin(skin.id)
    }
    emit('coins-updated')
  } catch (err) {
    actionError.value = err?.message || 'Failed to unlock skin.'
  } finally {
    unlocking.value = null
  }
}
</script>

<style scoped>
.pet-sanctuary-hub {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.sanctuary-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.sanctuary-title {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-color, #f8fafc);
  margin: 0 0 4px;
}

.sanctuary-subtitle {
  font-size: 0.88rem;
  color: var(--text-muted, #94a3b8);
  margin: 0;
  max-width: 650px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.coin-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(234, 179, 8, 0.12);
  border: 1px solid rgba(234, 179, 8, 0.35);
  color: #facc15;
  padding: 6px 12px;
  border-radius: 9999px;
  font-weight: 700;
  font-size: 0.88rem;
}

.companion-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-surface, #1e293b);
  border: 1px solid var(--border-color, #334155);
  color: var(--text-muted, #94a3b8);
  padding: 6px 14px;
  border-radius: 9999px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s ease;
}

.companion-toggle-btn.active {
  background: rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.4);
  color: #34d399;
}

.sanctuary-error-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.35);
  color: #f87171;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 0.86rem;
}

.banner-close {
  margin-left: auto;
  background: none;
  border: none;
  color: #f87171;
  font-size: 1.1rem;
  cursor: pointer;
}

/* Layout */
.sanctuary-layout {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 20px;
}

@media (max-width: 900px) {
  .sanctuary-layout {
    grid-template-columns: 1fr;
  }
}

/* Left Column Stage */
.showcase-card {
  background: var(--bg-surface, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.showcase-badge-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.pet-species-badge {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 700;
  color: #38bdf8;
  background: rgba(56, 189, 248, 0.12);
  padding: 2px 8px;
  border-radius: 4px;
}

.active-equipped-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.75rem;
  font-weight: 700;
  color: #34d399;
  background: rgba(16, 185, 129, 0.15);
  padding: 2px 8px;
  border-radius: 4px;
}

.showcase-pet-name {
  font-size: 1.4rem;
  font-weight: 800;
  color: var(--text-color, #f8fafc);
  margin: 0 0 4px;
}

.showcase-pet-desc {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
  margin: 0;
  line-height: 1.4;
}

.preview-stage {
  background: radial-gradient(circle at center, rgba(30, 41, 59, 0.8) 0%, rgba(15, 23, 42, 0.95) 100%);
  border: 1px dashed var(--border-color, #334155);
  border-radius: 10px;
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: visible;
}

.stage-avatar-wrapper {
  width: 120px;
  height: 120px;
  position: relative;
}

.stage-pet-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
}

.stage-speech-bubble {
  position: absolute;
  top: -38px;
  left: 50%;
  transform: translateX(-50%);
  background: #ffffff;
  color: #0f172a;
  padding: 5px 10px;
  border-radius: 8px;
  font-size: 0.78rem;
  font-weight: 600;
  white-space: nowrap;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
  cursor: pointer;
  z-index: 10;
}

.stage-bubble-tail {
  position: absolute;
  bottom: -5px;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border-left: 5px solid transparent;
  border-right: 5px solid transparent;
  border-top: 5px solid #ffffff;
}

.stage-controls {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.controls-label {
  font-size: 0.76rem;
  text-transform: uppercase;
  font-weight: 700;
  color: var(--text-muted, #64748b);
}

.action-btn-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.stage-act-btn {
  background: var(--bg-surface-elevated, #242c3d);
  border: 1px solid var(--border-color, #334155);
  color: var(--text-muted, #94a3b8);
  font-size: 0.78rem;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 6px;
  cursor: pointer;
  text-transform: capitalize;
  transition: all 0.15s ease;
}

.stage-act-btn:hover {
  background: #334155;
  color: #f8fafc;
}

.stage-act-btn.active {
  background: #38bdf8;
  color: #0f172a;
  border-color: #38bdf8;
  font-weight: 700;
}

.stage-act-btn.msg-btn {
  background: rgba(234, 179, 8, 0.12);
  border-color: rgba(234, 179, 8, 0.35);
  color: #facc15;
}

.stage-footer {
  margin-top: auto;
  border-top: 1px solid var(--border-color, #334155);
  padding-top: 14px;
}

.cost-callout {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  font-size: 0.88rem;
  color: var(--text-muted, #94a3b8);
}

.cost-callout strong {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #facc15;
  font-size: 1rem;
}

.primary-adopt-btn {
  width: 100%;
  background: #f59e0b;
  border: none;
  color: #0f172a;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.primary-adopt-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.primary-equip-btn {
  width: 100%;
  background: #38bdf8;
  border: none;
  color: #0f172a;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;
}

.active-pill-note {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: rgba(16, 185, 129, 0.12);
  color: #34d399;
  font-size: 0.84rem;
  font-weight: 600;
  padding: 8px;
  border-radius: 6px;
  margin-bottom: 8px;
}

.stage-reset-btn {
  width: 100%;
  background: transparent;
  border: 1px dashed var(--border-color, #475569);
  color: var(--text-muted, #94a3b8);
  font-size: 0.78rem;
  padding: 6px;
  border-radius: 6px;
  cursor: pointer;
}

/* Right Column */
.roster-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: var(--bg-surface, #1e293b);
  border: 1px solid var(--border-color, #334155);
  border-radius: 12px;
  padding: 18px;
}

.card-head {
  margin-bottom: 14px;
}

.card-title {
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--text-color, #f8fafc);
  margin: 0 0 2px;
}

.card-subtitle {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

/* Pets Grid */
.pets-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.pet-select-card {
  background: var(--bg-surface-elevated, #242c3d);
  border: 2px solid transparent;
  border-radius: 10px;
  padding: 12px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: all 0.18s ease;
}

.pet-select-card:hover {
  border-color: rgba(56, 189, 248, 0.4);
}

.pet-select-card.selected {
  border-color: #38bdf8;
  background: rgba(56, 189, 248, 0.08);
}

.pet-card-top {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pet-mini-preview {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  position: relative;
  flex-shrink: 0;
}

.pet-mini-accent {
  position: absolute;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  bottom: 4px;
  right: 4px;
}

.pet-card-meta {
  display: flex;
  flex-direction: column;
}

.pet-card-name {
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--text-color, #f8fafc);
}

.pet-card-title {
  font-size: 0.75rem;
  color: var(--text-muted, #94a3b8);
}

.pet-card-bottom {
  margin-top: auto;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.72rem;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 4px;
}

.badge.equipped-tag {
  background: rgba(16, 185, 129, 0.2);
  color: #34d399;
}

.badge.unlocked-tag {
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
}

.badge.price-tag,
.badge.price-badge {
  background: rgba(234, 179, 8, 0.15);
  color: #facc15;
}

.badge.active-badge {
  background: rgba(16, 185, 129, 0.2);
  color: #34d399;
}

.badge.unlocked-badge {
  background: rgba(148, 163, 184, 0.15);
  color: #94a3b8;
}

/* Skins Grid */
.skins-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.skin-shop-item {
  background: var(--bg-surface-elevated, #242c3d);
  border: 1px solid var(--border-color, #334155);
  border-radius: 8px;
  padding: 10px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.skin-shop-item:hover {
  border-color: #475569;
}

.skin-shop-item.selected {
  border-color: #38bdf8;
  background: rgba(56, 189, 248, 0.06);
}

.skin-swatch-box {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  position: relative;
  flex-shrink: 0;
}

.swatch-secondary {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.swatch-accent {
  position: absolute;
  bottom: 4px;
  left: 4px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.skin-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.skin-name {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-color, #f8fafc);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.skin-action-col {
  margin-left: auto;
}

.skin-buy-btn {
  background: #f59e0b;
  border: none;
  color: #0f172a;
  font-size: 0.76rem;
  font-weight: 700;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
}

.skin-buy-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.skin-equip-btn {
  background: #38bdf8;
  border: none;
  color: #0f172a;
  font-size: 0.76rem;
  font-weight: 700;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
}

/* Animations */
.action-blink .pet-body {
  transform-origin: center bottom;
}

.action-cheer {
  animation: cheer-hop 0.6s ease-in-out infinite alternate;
}

@keyframes cheer-hop {
  0% { transform: translateY(0); }
  100% { transform: translateY(-8px); }
}

.action-wiggle .pet-ear {
  animation: ear-wiggle 0.35s ease-in-out infinite alternate;
}

@keyframes ear-wiggle {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(8deg); }
}
</style>
