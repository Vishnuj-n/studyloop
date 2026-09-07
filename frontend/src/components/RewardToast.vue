<template>
  <Teleport to="body">
    <Transition name="toast-slide">
      <div
        v-if="visible"
        class="reward-toast"
        :class="[tierClass]"
        @mouseenter="pauseTimer"
        @mouseleave="resumeTimer"
      >
        <div class="toast-glow"></div>
        <button class="toast-close" type="button" aria-label="Close" @click="handleDismiss">✕</button>

        <div class="toast-content">
          <div class="toast-icon-col">
            <span class="chest-badge-icon">{{ chestEmoji }}</span>
          </div>

          <div class="toast-details">
            <div class="toast-header-row">
              <span class="toast-title">Task Complete!</span>
              <span class="tier-tag">{{ chestTier }} CHEST</span>
            </div>

            <div class="toast-pills-row">
              <span v-if="xpEarned > 0" class="reward-pill xp-pill">+{{ xpEarned }} XP</span>
              <span v-if="coinsEarned > 0" class="reward-pill coin-pill">+{{ coinsEarned }} Coins</span>
              <span v-if="newTitle" class="reward-pill title-pill">🎖️ {{ newTitle }}</span>
            </div>

            <p class="toast-desc">
              {{ box ? 'A mystery chest was added to your vault.' : 'Progress saved to your profile.' }}
            </p>

            <div v-if="box" class="toast-actions">
              <button class="toast-btn primary-btn" type="button" @click="handleOpen">
                Open Now ✨
              </button>
              <button class="toast-btn secondary-btn" type="button" @click="handleDismiss">
                Save to Vault
              </button>
            </div>
          </div>
        </div>

        <div class="toast-progress-bar" :style="{ width: progressPercent + '%' }"></div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getUserSettings } from '../services/appApi'

const props = defineProps({
  rewards: {
    type: Object,
    default: () => ({}),
  },
  durationMs: {
    type: Number,
    default: 4000,
  },
})

const emit = defineEmits(['open-chest', 'dismiss'])

const visible = ref(false)
const notificationsEnabled = ref(false)
const remainingMs = ref(props.durationMs)
let intervalId = null
let isPaused = false

const box = computed(() => props.rewards?.loot_box || null)
const xpEarned = computed(() => props.rewards?.xp_earned || 0)
const coinsEarned = computed(() => props.rewards?.coins_earned || 0)
const newTitle = computed(() => props.rewards?.new_title_unlocked || '')

const chestTier = computed(() => (box.value?.box_tier || 'BRONZE').toUpperCase())
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

const progressPercent = computed(() => {
  return Math.max(0, Math.min(100, (remainingMs.value / props.durationMs) * 100))
})

function startTimer() {
  const tick = 100
  intervalId = setInterval(() => {
    if (!isPaused) {
      remainingMs.value -= tick
      if (remainingMs.value <= 0) {
        handleDismiss()
      }
    }
  }, tick)
}

function pauseTimer() {
  isPaused = true
}

function resumeTimer() {
  isPaused = false
}

function handleOpen() {
  clearTimer()
  visible.value = false
  emit('open-chest', box.value)
}

function handleDismiss() {
  clearTimer()
  visible.value = false
  emit('dismiss')
}

function clearTimer() {
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

onMounted(async () => {
  try {
    const settings = await getUserSettings()
    notificationsEnabled.value = settings?.show_reward_notifications !== false
  } catch (err) {
    console.warn('[REWARD_TOAST] Failed to load notification preference:', err)
    notificationsEnabled.value = true
  }

  if (notificationsEnabled.value) {
    visible.value = true
    startTimer()
  }
})

onUnmounted(() => {
  clearTimer()
})
</script>

<style scoped>
.reward-toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 9998;
  max-width: 380px;
  width: calc(100vw - 48px);
  background: var(--surface-container-low, #1e1e24);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.15), 0 0 20px rgba(0, 0, 0, 0.06);
  color: var(--on-surface, #e2e8f0);
  overflow: hidden;
  padding: 16px;
}

.toast-glow {
  position: absolute;
  top: -20px;
  right: -20px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  pointer-events: none;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.08) 0%, transparent 70%);
}

.tier-bronze .toast-glow {
  background: radial-gradient(circle, rgba(217, 119, 6, 0.2) 0%, transparent 70%);
}
.tier-silver .toast-glow {
  background: radial-gradient(circle, rgba(203, 213, 225, 0.25) 0%, transparent 70%);
}
.tier-gold .toast-glow {
  background: radial-gradient(circle, rgba(251, 191, 36, 0.25) 0%, transparent 70%);
}
.tier-mythic .toast-glow {
  background: radial-gradient(circle, rgba(192, 132, 252, 0.3) 0%, transparent 70%);
}

.toast-close {
  position: absolute;
  top: 10px;
  right: 10px;
  background: none;
  border: none;
  color: var(--muted-text, #94a3b8);
  font-size: 1rem;
  cursor: pointer;
  border-radius: 6px;
  padding: 2px 6px;
  line-height: 1;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.toast-close:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.1);
}

.toast-content {
  display: flex;
  gap: 14px;
  align-items: flex-start;
}

.toast-icon-col {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: var(--surface-container, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.chest-badge-icon {
  font-size: 1.6rem;
  line-height: 1;
}

.toast-details {
  flex: 1;
  min-width: 0;
}

.toast-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  padding-right: 18px;
}

.toast-title {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--on-surface, #fff);
  letter-spacing: 0.01em;
}

.tier-tag {
  font-size: 0.65rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  padding: 2px 7px;
  border-radius: 999px;
  border: 1px solid var(--outline-variant);
}

.tier-bronze .tier-tag {
  color: #d97706;
  border-color: rgba(217, 119, 6, 0.4);
}
.tier-silver .tier-tag {
  color: #cbd5e1;
  border-color: rgba(203, 213, 225, 0.4);
}
.tier-gold .tier-tag {
  color: #fbbf24;
  border-color: rgba(251, 191, 36, 0.4);
}
.tier-mythic .tier-tag {
  color: #c084fc;
  border-color: rgba(192, 132, 252, 0.4);
}

.toast-pills-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}

.reward-pill {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
}

.xp-pill {
  background: rgba(59, 130, 246, 0.15);
  color: #60a5fa;
}

.coin-pill {
  background: rgba(234, 179, 8, 0.15);
  color: #facc15;
}

.title-pill {
  background: rgba(168, 85, 247, 0.15);
  color: #c084fc;
}

.toast-desc {
  font-size: 0.8rem;
  color: var(--muted-text, #94a3b8);
  margin: 0 0 10px 0;
  line-height: 1.35;
}

.toast-actions {
  display: flex;
  gap: 8px;
}

.toast-btn {
  padding: 6px 12px;
  font-size: 0.8rem;
  font-weight: 600;
  border-radius: 8px;
  cursor: pointer;
  transition: transform 0.1s ease, filter 0.15s ease, background 0.15s ease;
}

.primary-btn {
  background: var(--primary, #6366f1);
  color: #fff;
  border: none;
}

.primary-btn:hover {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

.secondary-btn {
  background: var(--surface-container, rgba(255, 255, 255, 0.08));
  color: var(--muted-text, #cbd5e1);
  border: 1px solid var(--outline-variant);
}

.secondary-btn:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.12);
}

.toast-progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 3px;
  background: var(--primary, #6366f1);
  opacity: 0.8;
  transition: width 0.1s linear;
}

.toast-slide-enter-active,
.toast-slide-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-slide-enter-from {
  opacity: 0;
  transform: translateY(20px) scale(0.95);
}

.toast-slide-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.96);
}
</style>
