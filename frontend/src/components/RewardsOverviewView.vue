<template>
  <div class="rewards-overview">
    <!-- Rank Card -->
    <div class="card rank-card">
      <div class="rank-header">
        <div class="rank-avatar-pedestal">
          <span class="rank-avatar">{{ getTitleEmoji(profile.current_title) }}</span>
        </div>
        <div class="rank-info-block">
          <div class="rank-badge-row">
            <span class="rank-label">Current Rank</span>
            <span class="level-chip">Lvl {{ profile.level || 1 }}</span>
          </div>
          <h2 class="rank-title">{{ profile.current_title }}</h2>
        </div>
        <button
          type="button"
          class="view-roadmap-btn"
          @click="$emit('switch-tab', 'roadmap')"
        >
          View Roadmap →
        </button>
      </div>

      <div class="xp-bar-container">
        <div class="xp-bar-track">
          <div class="xp-bar-fill" :style="{ width: progressPercent + '%' }"></div>
        </div>
        <div class="xp-labels">
          <span>{{ (profile.total_xp || 0).toLocaleString() }} XP</span>
          <span>Next Rank: {{ profile.next_title }} ({{ (profile.next_title_xp || 0).toLocaleString() }} XP)</span>
        </div>
      </div>
    </div>

    <!-- Currency & Freezes Card -->
    <div class="card currency-card">
      <div class="currency-group">
        <div class="currency-item">
          <div class="icon-pedestal coin-pedestal">
            <BaseIcon name="coin" size="24" custom-class="currency-svg-icon" />
          </div>
          <div>
            <span class="currency-val">{{ profile.coins }}</span>
            <span class="currency-name">Study Coins</span>
          </div>
        </div>

        <div class="currency-item">
          <div class="icon-pedestal shield-pedestal">
            <BaseIcon name="shield" size="24" custom-class="currency-svg-icon" />
          </div>
          <div>
            <div class="freeze-val-row">
              <span class="currency-val">{{ profile.streak_freezes_owned }}</span>
              <button
                type="button"
                title="View how streak freezes work"
                class="info-icon-btn"
                @click="$emit('open-freeze-info')"
              >
                <BaseIcon name="help-circle" size="14" />
              </button>
            </div>
            <span class="currency-name">Streak Freezes</span>
          </div>
        </div>
      </div>

      <div class="shop-action-group">
        <button
          class="buy-freeze-btn"
          type="button"
          :disabled="freezeButtonDisabled"
          @click="$emit('buy-streak-freeze')"
        >
          {{ buying ? 'Purchasing...' : freezeButtonLabel }}
        </button>
      </div>
      <p v-if="buyError" class="buy-error-msg">{{ buyError }}</p>
    </div>

    <!-- Mystery Chests Vault -->
    <div class="card chests-card">
      <div class="chests-card-header">
        <div>
          <h3 class="card-section-title">Mystery Chests Vault</h3>
          <p class="chests-desc">Chests are earned by completing reading sessions, quizzes, and reviews.</p>
        </div>
        <span v-if="chests.length > 0" class="vault-counter-pill">{{ chests.length }} Unopened</span>
      </div>

      <div v-if="chests.length === 0" class="empty-chests">
        <div class="empty-chest-pedestal">
          <BaseIcon name="package" size="30" />
        </div>
        <p>No unopened chests in your vault. Complete study tasks to earn more!</p>
      </div>

      <div v-else class="chests-list">
        <button
          v-for="chest in chests"
          :key="chest.id"
          type="button"
          class="chest-vault-item"
          :class="'tier-' + chest.box_tier.toLowerCase()"
          @click="$emit('open-chest', chest)"
        >
          <div class="chest-pedestal">
            <GamificationIcon :name="'chest-' + chest.box_tier.toLowerCase()" size="44" />
          </div>
          <div class="chest-vault-info">
            <span class="chest-vault-tier">{{ chest.box_tier }} CHEST</span>
            <span class="chest-vault-tap">Click to open <BaseIcon name="sparkles" :size="13" class="inline-sparkle" /></span>
          </div>
        </button>
      </div>
    </div>

    <!-- Study Companion (Mochi) -->
    <div class="card companion-card">
      <div class="companion-header">
        <div>
          <h3 class="card-section-title">Study Companion</h3>
          <p class="companion-desc">Mochi sits on your screen and cheers you on while you study. Toggle it on/off and pick a skin.</p>
        </div>
        <button
          type="button"
          class="toggle-btn"
          :class="{ active: petState.enabled }"
          @click="togglePet()"
        >
          {{ petState.enabled ? 'Enabled' : 'Disabled' }}
        </button>
      </div>

      <div v-if="petState.enabled" class="companion-body">
        <span class="skin-heading">Companion Skin</span>
        <div class="skin-grid">
          <button
            v-for="skin in currentPet.skins"
            :key="skin.id"
            type="button"
            class="skin-card"
            :class="{ active: petState.activeSkinId === skin.id }"
            @click="setSkin(skin.id)"
          >
            <span class="skin-color-preview" :style="{ background: skin.primaryColor }">
              <span class="skin-color-dot" :style="{ background: skin.secondaryColor }"></span>
            </span>
            <span class="skin-name">{{ skin.name }}</span>
          </button>
        </div>
        <button type="button" class="reset-pos-btn" @click="resetPosition()">
          Reset Screen Position
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import BaseIcon from './BaseIcon.vue'
import GamificationIcon from './icons/GamificationIcon.vue'
import { getTitleEmoji } from '../utils/gamification'
import { usePet } from '../composables/usePet'

const { petState, currentPet, togglePet, setSkin, resetPosition } = usePet()

const props = defineProps({
  profile: {
    type: Object,
    required: true,
  },
  chests: {
    type: Array,
    default: () => [],
  },
  buying: {
    type: Boolean,
    default: false,
  },
  buyError: {
    type: String,
    default: '',
  },
})

defineEmits(['switch-tab', 'open-freeze-info', 'buy-streak-freeze', 'open-chest'])

const isWeeklyCooldown = computed(() => {
  if (!props.profile.last_freeze_purchased_at) return false
  const nowSec = Math.floor(Date.now() / 1000)
  const elapsed = nowSec - props.profile.last_freeze_purchased_at
  return elapsed < 7 * 24 * 3600
})

const cooldownDaysLeft = computed(() => {
  if (!props.profile.last_freeze_purchased_at) return 0
  const nowSec = Math.floor(Date.now() / 1000)
  const remainingSec = (7 * 24 * 3600) - (nowSec - props.profile.last_freeze_purchased_at)
  if (remainingSec <= 0) return 0
  return Math.ceil(remainingSec / 86400)
})

const freezeButtonLabel = computed(() => {
  if ((props.profile.streak_freezes_owned || 0) >= 2) {
    return 'Max Capacity (2/2 Freezes)'
  }
  if (isWeeklyCooldown.value) {
    return `Weekly Limit (Available in ${cooldownDaysLeft.value}d)`
  }
  return 'Buy Streak Freeze (150 Coins)'
})

const freezeButtonDisabled = computed(() => {
  if (props.buying) return true
  if ((props.profile.streak_freezes_owned || 0) >= 2) return true
  if (isWeeklyCooldown.value) return true
  if ((props.profile.coins || 0) < 150) return true
  return false
})

const progressPercent = computed(() => {
  const min = props.profile.current_title_min_xp || 0
  const max = props.profile.next_title_xp || 500
  const cur = props.profile.total_xp || 0
  if (cur >= max) return 100
  const range = max - min
  if (range <= 0) return 100
  const pct = ((cur - min) / range) * 100
  return Math.min(100, Math.max(0, Math.round(pct)))
})
</script>

<style scoped>
.rewards-overview {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.card {
  position: relative;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 18px;
  padding: 1.5rem 1.75rem;
  box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease;
}

.card:hover {
  box-shadow: 0 8px 24px -2px rgba(0, 0, 0, 0.14);
}

.rank-header {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  margin-bottom: 1.25rem;
}

.rank-avatar-pedestal {
  width: 58px;
  height: 58px;
  border-radius: 14px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
}

.rank-avatar {
  font-size: 2rem;
  display: inline-block;
  filter: drop-shadow(0 0 6px color-mix(in srgb, var(--primary) 65%, transparent));
}

.rank-info-block {
  flex: 1;
}

.rank-badge-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.2rem;
}

.rank-label {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted-text);
}

.level-chip {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  border: 1px solid var(--outline-variant);
  color: var(--primary);
  font-size: 0.75rem;
  font-weight: 700;
  padding: 1px 8px;
  border-radius: 12px;
}

.rank-title {
  margin: 0.2rem 0 0;
  font-family: 'Manrope', sans-serif;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--on-surface);
}

.view-roadmap-btn {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  color: var(--primary);
  font-size: 0.82rem;
  font-weight: 700;
  padding: 8px 14px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.view-roadmap-btn:hover {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border-color: var(--primary);
}

.xp-bar-container {
  margin-top: 0.75rem;
}

.xp-bar-track {
  height: 8px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 999px;
  overflow: hidden;
}

.xp-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-dim), var(--primary));
  box-shadow: 0 0 10px color-mix(in srgb, var(--primary) 50%, transparent);
  transition: width 0.4s ease;
}

.xp-labels {
  display: flex;
  justify-content: space-between;
  margin-top: 0.4rem;
  font-size: 0.8rem;
  color: var(--muted-text);
}

.currency-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1.5rem;
}

.currency-group {
  display: flex;
  align-items: center;
  gap: 2.5rem;
  flex-wrap: wrap;
}

.currency-item {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.icon-pedestal {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.coin-pedestal {
  background: radial-gradient(circle, rgba(245, 158, 11, 0.15) 0%, var(--surface-container-low) 85%);
  border-color: rgba(245, 158, 11, 0.3);
}

.shield-pedestal {
  background: radial-gradient(circle, color-mix(in srgb, var(--primary) 15%, transparent) 0%, var(--surface-container-low) 85%);
  border-color: color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
}

.currency-val {
  font-size: 1.35rem;
  font-weight: 700;
  font-family: 'Manrope', sans-serif;
  color: var(--on-surface);
  display: block;
}

.currency-name {
  font-size: 0.8rem;
  color: var(--muted-text);
}

.freeze-val-row {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.info-icon-btn {
  background: none;
  border: none;
  padding: 2px;
  margin: 0;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--muted-text);
  border-radius: 50%;
  transition: color 0.15s ease;
}

.info-icon-btn:hover {
  color: var(--primary);
}

.buy-freeze-btn {
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  padding: 0.65rem 1.25rem;
  border-radius: 10px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  box-shadow: 0 4px 14px color-mix(in srgb, var(--primary) 30%, transparent);
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease;
}

.buy-freeze-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 18px color-mix(in srgb, var(--primary) 45%, transparent);
}

.buy-freeze-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  box-shadow: none;
}

.chests-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.vault-counter-pill {
  background: color-mix(in srgb, var(--primary) 20%, transparent);
  color: var(--primary);
  border: 1px solid var(--outline-variant);
  font-size: 0.75rem;
  font-weight: 800;
  padding: 4px 10px;
  border-radius: 999px;
}

.card-section-title {
  margin: 0 0 0.25rem;
  font-family: 'Manrope', sans-serif;
  font-size: 1.15rem;
  color: var(--on-surface);
}

.chests-desc {
  margin: 0 0 1.25rem;
  font-size: 0.85rem;
  color: var(--muted-text);
}

.empty-chests {
  text-align: center;
  padding: 2.5rem 1rem;
  color: var(--muted-text);
}

.empty-chest-pedestal {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 0.75rem;
}

.empty-chest-pedestal span {
  font-size: 2rem;
  opacity: 0.85;
}

.chests-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 1.25rem;
}

.chest-vault-item {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  padding: 1.25rem 1rem;
  text-align: center;
  cursor: pointer;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), border-color 0.2s ease;
}

.chest-vault-item:hover {
  transform: translateY(-3px);
  border-color: var(--primary);
}

.chest-pedestal {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 0.75rem;
}

.chest-vault-tier {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  display: block;
  color: var(--primary);
}

.chest-vault-tap {
  font-size: 0.75rem;
  color: var(--muted-text);
  margin-top: 2px;
  display: block;
}

.tier-bronze .chest-vault-tier { color: #d97706; }
.tier-silver .chest-vault-tier { color: #cbd5e1; }
.tier-gold .chest-vault-tier { color: #fbbf24; }
.tier-mythic .chest-vault-tier { color: #c084fc; }

.buy-error-msg {
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: var(--danger, #ef4444);
  font-weight: 600;
}

/* Companion card */
.companion-card { padding: 20px; }

.companion-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.companion-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--muted-text);
  line-height: 1.4;
}

.toggle-btn {
  flex-shrink: 0;
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.07);
  color: var(--muted-text, #94a3b8);
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: all 0.2s ease;
}

.toggle-btn.active {
  background: var(--primary, #6366f1);
  color: #fff;
  border-color: var(--primary, #6366f1);
}

.companion-body {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.07);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.skin-heading {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface);
}

.skin-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 10px;
}

.skin-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.08);
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
}

.skin-card:hover { border-color: var(--primary, #6366f1); }

.skin-card.active {
  border-color: var(--primary, #6366f1);
  background: rgba(99, 102, 241, 0.12);
}

.skin-color-preview {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.skin-color-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.skin-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--on-surface);
}

.reset-pos-btn {
  align-self: flex-start;
  font-size: 11px;
  color: var(--muted-text);
  background: transparent;
  border: none;
  cursor: pointer;
  text-decoration: underline;
  padding: 4px 0;
}

.reset-pos-btn:hover { color: var(--on-surface); }
</style>
