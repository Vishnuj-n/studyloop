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
            <PetAvatar
              :pet-id="selectedPet.id"
              :skin="previewSkin"
              :action="previewAction"
            />
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
            <div v-else class="active-companion-actions">
              <button
                type="button"
                class="stage-companion-toggle-btn"
                :class="{ enabled: petState.enabled }"
                @click="togglePet()"
              >
                <BaseIcon :name="petState.enabled ? 'check' : 'eye'" size="16" />
                <span>{{ petState.enabled ? `${selectedPet.name} is on Screen (Click to Hide)` : `Summon ${selectedPet.name} to Screen` }}</span>
              </button>
              <button
                v-if="petState.enabled"
                type="button"
                class="stage-reset-btn"
                title="Reset floating pet to bottom-right corner"
                @click="resetPosition()"
              >
                Reset Position
              </button>
            </div>
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
                <div class="pet-card-avatar">
                  <PetAvatar
                    :pet-id="pet.id"
                    :skin="pet.skins[0]"
                    action="idle"
                    :show-shadow="false"
                  />
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
              <div class="skin-avatar-thumb">
                <PetAvatar
                  :pet-id="selectedPet.id"
                  :skin="skin"
                  action="idle"
                  :show-shadow="false"
                />
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
import PetAvatar from './PetAvatar.vue'
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
  togglePet(true)
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
  gap: 1.25rem;
}

.sanctuary-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 1.25rem 1.5rem;
}

.sanctuary-title {
  margin: 0 0 0.2rem;
  font-family: 'Manrope', sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--on-surface);
}

.sanctuary-subtitle {
  margin: 0;
  font-family: 'Inter', sans-serif;
  font-size: 0.84rem;
  color: var(--muted-text);
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
  background: color-mix(in srgb, #f59e0b 12%, var(--surface-container-lowest));
  border: 1px solid color-mix(in srgb, #f59e0b 25%, transparent);
  color: #d97706;
  padding: 6px 14px;
  border-radius: 9999px;
  font-weight: 700;
  font-size: 0.88rem;
}

.companion-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  padding: 6px 14px;
  border-radius: 9999px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.companion-toggle-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container-highest);
}

.companion-toggle-btn.active {
  background: color-mix(in srgb, #10b981 14%, var(--surface-container-lowest));
  border-color: color-mix(in srgb, #10b981 35%, transparent);
  color: #059669;
  font-weight: 700;
}

.sanctuary-error-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  background: color-mix(in srgb, var(--danger, #ef4444) 12%, var(--surface-container-low));
  border: 1px solid color-mix(in srgb, var(--danger, #ef4444) 30%, transparent);
  color: var(--danger, #ef4444);
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 0.86rem;
}

.banner-close {
  margin-left: auto;
  background: none;
  border: none;
  color: var(--danger, #ef4444);
  font-size: 1.1rem;
  cursor: pointer;
}

/* Layout */
.sanctuary-layout {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 1.25rem;
}

@media (max-width: 900px) {
  .sanctuary-layout {
    grid-template-columns: 1fr;
  }
}

/* Left Column Stage */
.showcase-card {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 1.25rem;
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
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 700;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 14%, transparent);
  padding: 3px 8px;
  border-radius: 6px;
}

.active-equipped-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.72rem;
  font-weight: 700;
  color: #059669;
  background: color-mix(in srgb, #10b981 14%, transparent);
  padding: 3px 8px;
  border-radius: 6px;
}

.showcase-pet-name {
  font-family: 'Manrope', sans-serif;
  font-size: 1.3rem;
  font-weight: 800;
  color: var(--on-surface);
  margin: 0 0 4px;
}

.showcase-pet-desc {
  font-family: 'Inter', sans-serif;
  font-size: 0.85rem;
  color: var(--muted-text);
  margin: 0;
  line-height: 1.45;
}

.preview-stage {
  background: var(--surface-container-lowest);
  border: 1px dashed var(--outline-variant);
  border-radius: 14px;
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

.stage-speech-bubble {
  position: absolute;
  top: -38px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  padding: 5px 10px;
  border-radius: 8px;
  font-size: 0.78rem;
  font-weight: 600;
  white-space: nowrap;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
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
  border-top: 5px solid var(--surface-container-highest);
}

.stage-controls {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.controls-label {
  font-size: 0.74rem;
  text-transform: uppercase;
  font-weight: 700;
  color: var(--muted-text);
  letter-spacing: 0.05em;
}

.action-btn-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.stage-act-btn {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
  font-size: 0.78rem;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 8px;
  cursor: pointer;
  text-transform: capitalize;
  transition: all 0.15s ease;
}

.stage-act-btn:hover {
  background: var(--surface-container-lowest);
  color: var(--on-surface);
}

.stage-act-btn.active {
  background: var(--primary);
  color: var(--on-primary);
  border-color: var(--primary);
  font-weight: 700;
}

.stage-act-btn.msg-btn {
  background: color-mix(in srgb, #f59e0b 12%, var(--surface-container-low));
  border-color: color-mix(in srgb, #f59e0b 25%, transparent);
  color: #d97706;
}

.stage-footer {
  margin-top: auto;
  border-top: 1px solid var(--outline-variant);
  padding-top: 14px;
}

.cost-callout {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  font-size: 0.88rem;
  color: var(--muted-text);
}

.cost-callout strong {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #d97706;
  font-size: 1rem;
  font-weight: 700;
}

.primary-adopt-btn {
  width: 100%;
  background: linear-gradient(135deg, #d97706, #b45309);
  border: none;
  color: #ffffff;
  font-family: 'Manrope', sans-serif;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 10px;
  border-radius: 12px;
  cursor: pointer;
  transition: opacity 0.15s, transform 0.15s;
}

.primary-adopt-btn:hover:not(:disabled) {
  opacity: 0.92;
  transform: translateY(-1px);
}

.primary-adopt-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.primary-equip-btn {
  width: 100%;
  background: linear-gradient(135deg, var(--primary), var(--primary-dim));
  border: none;
  color: var(--on-primary);
  font-family: 'Manrope', sans-serif;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 10px;
  border-radius: 12px;
  cursor: pointer;
  transition: opacity 0.15s, transform 0.15s;
}

.primary-equip-btn:hover {
  opacity: 0.92;
  transform: translateY(-1px);
}

.active-companion-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stage-companion-toggle-btn {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  font-size: 0.88rem;
  font-weight: 700;
  padding: 10px 14px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.stage-companion-toggle-btn.enabled {
  background: color-mix(in srgb, #10b981 14%, var(--surface-container-lowest));
  border-color: color-mix(in srgb, #10b981 35%, transparent);
  color: #059669;
}

.stage-reset-btn {
  width: 100%;
  background: transparent;
  border: 1px dashed var(--outline-variant);
  color: var(--muted-text);
  font-size: 0.78rem;
  padding: 6px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.stage-reset-btn:hover {
  color: var(--on-surface);
  border-color: var(--muted-text);
}

/* Right Column */
.roster-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 1.25rem;
}

.card-head {
  margin-bottom: 14px;
}

.card-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 2px;
}

.card-subtitle {
  font-family: 'Inter', sans-serif;
  font-size: 0.8rem;
  color: var(--muted-text);
}

/* Pets Grid */
.pets-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.pet-select-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 12px;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: all 0.18s ease;
}

.pet-select-card:hover {
  background: var(--surface-container-lowest);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.04);
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
}

.pet-select-card.selected {
  border-color: var(--primary);
  background: var(--surface-container-lowest);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
}

.pet-card-top {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pet-card-avatar {
  width: 46px;
  height: 46px;
  border-radius: 10px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 3px;
  transition: transform 0.15s ease;
}

.pet-select-card:hover .pet-card-avatar {
  transform: scale(1.06);
}

.pet-card-meta {
  display: flex;
  flex-direction: column;
}

.pet-card-name {
  font-family: 'Manrope', sans-serif;
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--on-surface);
}

.pet-card-title {
  font-family: 'Inter', sans-serif;
  font-size: 0.75rem;
  color: var(--muted-text);
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
  border-radius: 6px;
}

.badge.equipped-tag,
.badge.active-badge {
  background: color-mix(in srgb, #10b981 14%, transparent);
  color: #059669;
}

.badge.unlocked-tag {
  background: color-mix(in srgb, var(--primary) 14%, transparent);
  color: var(--primary);
}

.badge.price-tag,
.badge.price-badge {
  background: color-mix(in srgb, #f59e0b 14%, transparent);
  color: #d97706;
}

.badge.unlocked-badge {
  background: var(--surface-container-highest);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

/* Skins Grid */
.skins-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.skin-shop-item {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.skin-shop-item:hover {
  background: var(--surface-container-lowest);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.skin-shop-item.selected {
  border-color: var(--primary);
  background: var(--surface-container-lowest);
}

.skin-avatar-thumb {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 2px;
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
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.skin-action-col {
  margin-left: auto;
}

.skin-buy-btn {
  background: linear-gradient(135deg, #d97706, #b45309);
  border: none;
  color: #ffffff;
  font-size: 0.76rem;
  font-weight: 700;
  padding: 5px 10px;
  border-radius: 6px;
  cursor: pointer;
}

.skin-buy-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.skin-equip-btn {
  background: var(--primary);
  border: none;
  color: var(--on-primary);
  font-size: 0.76rem;
  font-weight: 700;
  padding: 5px 10px;
  border-radius: 6px;
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
