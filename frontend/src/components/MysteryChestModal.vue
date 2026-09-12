<template>
  <Teleport to="body">
    <div v-if="visible" class="chest-modal-backdrop" @click.self="closeModal">
      <div class="chest-modal-card" :class="[tierClass]">
        <div class="modal-ambient-glow"></div>
        <button class="close-btn" type="button" aria-label="Close" @click="closeModal">✕</button>

        <div class="chest-header">
          <span class="tier-pill">{{ chestTier }} CHEST</span>
          <h2 class="chest-title">{{ opened ? 'Reward Unlocked!' : 'Tap Chest to Open' }}</h2>
        </div>

        <div class="chest-interactive-area" @click="handleOpen">
          <div
            class="chest-icon-wrapper"
            :class="{ rattle: isRattling, pop: opened }"
            @mouseenter="handleHover"
          >
            <div class="chest-modal-pedestal">
              <GamificationIcon v-if="!opened" :name="'chest-' + chestTier.toLowerCase()" size="80" />
              <div v-else class="opened-burst">
                <span class="reward-icon">{{ rewardEmoji }}</span>
              </div>
            </div>
          </div>

          <div v-if="claimError" class="chest-error-banner">
            {{ claimError }}
          </div>

          <div v-if="opened" class="reward-reveal-text">

            <span class="reward-amount">+{{ chestRewardAmount }} {{ rewardLabel }}</span>
            <p v-if="newTitle" class="title-unlock-banner">
              🎉 New Title Unlocked: <strong>{{ newTitle }}</strong>!
            </p>
          </div>
          <div v-else class="chest-tap-hint">
            <span>✨ Click to claim loot ✨</span>
          </div>
        </div>

        <div class="chest-footer">
          <button
            v-if="!opened"
            class="action-btn claim-btn"
            type="button"
            :disabled="opening"
            @click="handleOpen"
          >
            {{ opening ? 'Unlocking...' : 'Open Chest' }}
          </button>
          <button v-else class="action-btn collect-btn" type="button" @click="closeModal">
            Collect & Continue
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed } from 'vue'
import GamificationIcon from './icons/GamificationIcon.vue'
import { playChestRattle, playChestOpenFanfare } from '../utils/audioJuice'
import { claimLootBox } from '../services/appApi'

const props = defineProps({
  box: {
    type: Object,
    default: null,
  },
  newTitle: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['claimed', 'close'])

const visible = ref(true)
const opened = ref(false)
const opening = ref(false)
const isRattling = ref(false)
const claimError = ref('')

const chestTier = computed(() => (props.box?.box_tier || 'BRONZE').toUpperCase())
const chestRewardType = computed(() => props.box?.reward_type || 'XP')
const chestRewardAmount = computed(() => props.box?.reward_amount || 50)

const tierClass = computed(() => `tier-${chestTier.value.toLowerCase()}`)

const chestEmoji = computed(() => {
  switch (chestTier.value) {
    case 'SILVER':
      return '🥈'
    case 'GOLD':
      return '🎁'
    case 'MYTHIC':
      return '👑'
    default:
      return '📦'
  }
})

const rewardEmoji = computed(() => {
  switch (chestRewardType.value) {
    case 'COINS':
      return '🪙'
    case 'STREAK_FREEZE':
      return '🛡️'
    default:
      return '⭐'
  }
})

const rewardLabel = computed(() => {
  switch (chestRewardType.value) {
    case 'COINS':
      return 'Coins'
    case 'STREAK_FREEZE':
      return 'Streak Freeze'
    default:
      return 'XP'
  }
})

function handleHover() {
  if (!opened.value && !isRattling.value) {
    isRattling.value = true
    playChestRattle()
    setTimeout(() => {
      isRattling.value = false
    }, 400)
  }
}

async function handleOpen() {
  if (opened.value || opening.value) return
  opening.value = true
  isRattling.value = true
  claimError.value = ''
  playChestRattle()

  try {
    if (props.box?.id) {
      await claimLootBox(props.box.id)
    }
    opened.value = true
    playChestOpenFanfare()
    emit('claimed', props.box)
    window.dispatchEvent(new Event('gamification-updated'))
  } catch (err) {
    console.error('Failed to claim loot box:', err)
    claimError.value = err?.message || 'Failed to open chest. Please try again.'
  } finally {
    opening.value = false
    isRattling.value = false
  }
}

function closeModal() {
  visible.value = false
  emit('close')
}
</script>

<style scoped>
.chest-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.chest-modal-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  width: 100%;
  max-width: 400px;
  padding: 2.25rem 1.75rem;
  text-align: center;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.15), 0 0 30px rgba(0, 0, 0, 0.08);
  color: var(--on-surface);
  overflow: hidden;
}

.modal-ambient-glow {
  position: absolute;
  top: -60px;
  left: 50%;
  transform: translateX(-50%);
  width: 220px;
  height: 220px;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--primary) 25%, transparent) 0%, transparent 70%);
  pointer-events: none;
}

.tier-mythic .modal-ambient-glow {
  background: radial-gradient(circle, rgba(192, 132, 252, 0.3) 0%, transparent 70%);
}

.tier-gold .modal-ambient-glow {
  background: radial-gradient(circle, rgba(251, 191, 36, 0.25) 0%, transparent 70%);
}

.close-btn {
  position: absolute;
  top: 14px;
  right: 16px;
  background: none;
  border: none;
  color: var(--muted-text);
  font-size: 1.25rem;
  cursor: pointer;
  border-radius: 8px;
  padding: 4px 8px;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.close-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}

.tier-pill {
  display: inline-block;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.1em;
  padding: 0.3rem 0.85rem;
  border-radius: 999px;
  margin-bottom: 0.75rem;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container);
  color: var(--muted-text);
}

.tier-bronze .tier-pill {
  border-color: rgba(217, 119, 6, 0.4);
  color: #d97706;
}
.tier-silver .tier-pill {
  border-color: rgba(203, 213, 225, 0.4);
  color: #cbd5e1;
}
.tier-gold .tier-pill {
  border-color: rgba(251, 191, 36, 0.4);
  color: #fbbf24;
}
.tier-mythic .tier-pill {
  border-color: rgba(192, 132, 252, 0.4);
  color: #c084fc;
}

.chest-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.45rem;
  font-weight: 700;
  margin: 0 0 1rem;
}

.chest-interactive-area {
  cursor: pointer;
  padding: 1.25rem 0;
}

.chest-modal-pedestal {
  width: 96px;
  height: 96px;
  border-radius: 24px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  box-shadow: 0 10px 25px -4px rgba(0, 0, 0, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto;
}

.chest-emoji,
.reward-icon {
  font-size: 3.5rem;
  filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.12));
}

.chest-icon-wrapper {
  display: inline-block;
  transition: transform 0.2s ease;
  user-select: none;
}

.chest-icon-wrapper.rattle {
  animation: chest-rattle 0.35s ease-in-out infinite;
}

.chest-icon-wrapper.pop {
  animation: reward-pop 0.5s cubic-bezier(0.175, 0.885, 0.32, 1.275) forwards;
}

@keyframes chest-rattle {
  0%, 100% { transform: rotate(0deg) scale(1); }
  25% { transform: rotate(-8deg) scale(1.06); }
  75% { transform: rotate(8deg) scale(1.06); }
}

@keyframes reward-pop {
  0% { transform: scale(0.4); opacity: 0; }
  70% { transform: scale(1.15); opacity: 1; }
  100% { transform: scale(1); }
}

.reward-reveal-text {
  margin-top: 1.25rem;
}

.reward-amount {
  font-family: 'Manrope', sans-serif;
  font-size: 2rem;
  font-weight: 800;
  color: var(--primary);
  display: block;
}

.title-unlock-banner {
  margin-top: 0.75rem;
  font-size: 0.95rem;
  color: var(--on-surface);
}

.chest-tap-hint {
  margin-top: 1rem;
  font-size: 0.85rem;
  color: var(--muted-text);
}

.action-btn {
  width: 100%;
  padding: 0.8rem;
  font-weight: 700;
  font-size: 1rem;
  border-radius: 12px;
  cursor: pointer;
  border: none;
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease, opacity 0.2s ease;
}

.claim-btn {
  background: var(--primary);
  color: var(--on-primary);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--primary) 35%, transparent);
}

.claim-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--primary) 50%, transparent);
}

.collect-btn {
  background: var(--primary);
  color: var(--on-primary);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--primary) 35%, transparent);
}

.collect-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--primary) 50%, transparent);
}

.chest-error-banner {
  margin-top: 1rem;
  padding: 0.6rem 0.8rem;
  background: color-mix(in srgb, var(--danger, #ef4444) 15%, transparent);
  color: var(--danger, #ef4444);
  border: 1px solid color-mix(in srgb, var(--danger, #ef4444) 30%, transparent);
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 600;
  text-align: center;
}
</style>
