<template>
  <div class="roadmap-container">
    <div class="roadmap-header">
      <div>
        <h3 class="roadmap-title">Scholar Progression Path</h3>
        <p class="roadmap-subtitle">From humble apprentice to Mythic Sage. Climb the ranks as you accumulate Study XP.</p>
      </div>

      <div class="current-xp-pill">
        <BaseIcon name="zap" size="16" />
        <span class="xp-value">{{ totalXPFormatted }} Total XP</span>
      </div>
    </div>

    <div class="tiers-roadmap-grid">
      <div
        v-for="(tier, idx) in GAMIFICATION_TIERS"
        :key="tier.baseTitle"
        class="tier-step-card"
        :class="{
          unlocked: currentXP >= tier.minXP,
          'current-tier': isCurrentTier(tier, idx),
          'next-tier': isNextTier(tier, idx),
          locked: currentXP < tier.minXP && !isNextTier(tier, idx),
        }"
      >
        <div class="step-badge">
          <span v-if="currentXP >= tier.minXP" class="badge-status unlocked">
            <BaseIcon name="check" size="12" /> Achieved
          </span>
          <span v-else-if="isNextTier(tier, idx)" class="badge-status next">
            <BaseIcon name="target" size="12" /> Next Goal
          </span>
          <span v-else class="badge-status locked">
            <BaseIcon name="lock" size="12" /> Locked
          </span>
        </div>

        <div class="tier-pedestal">
          <span class="tier-avatar">{{ tier.emoji }}</span>
        </div>

        <div class="tier-info">
          <h4 class="tier-name">{{ tier.baseTitle }}</h4>
          <span class="tier-xp-target">{{ tier.minXP.toLocaleString() }} XP</span>
          <p class="tier-desc">{{ tier.description }}</p>
        </div>

        <div class="tier-card-footer">
          <button
            v-if="currentXP >= tier.minXP"
            type="button"
            class="preview-celebration-btn"
            title="Replay unlock celebration fanfare"
            @click="$emit('preview-celebration', tier)"
          >
            <BaseIcon name="sparkles" size="14" /> Replay Celebration
          </button>

          <div v-else-if="isNextTier(tier, idx)" class="locked-requirement-pill next-pill">
            <BaseIcon name="zap" size="13" />
            <span>{{ (tier.minXP - currentXP).toLocaleString() }} XP remaining</span>
          </div>

          <div v-else class="locked-requirement-pill">
            <BaseIcon name="lock" size="13" />
            <span>Unlock at {{ tier.minXP.toLocaleString() }} XP</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { GAMIFICATION_TIERS } from '../utils/gamification'

const props = defineProps({
  totalXp: {
    type: Number,
    default: 0,
  },
  currentTitle: {
    type: String,
    default: 'The Apprentice I',
  },
})

defineEmits(['preview-celebration'])

const currentXP = computed(() => props.totalXp || 0)
const totalXPFormatted = computed(() => Number(currentXP.value).toLocaleString())

function isCurrentTier(tier, idx) {
  if (currentXP.value < tier.minXP) return false
  const nextTier = GAMIFICATION_TIERS[idx + 1]
  if (!nextTier) return true
  return currentXP.value < nextTier.minXP
}

function isNextTier(tier, idx) {
  const prevTier = GAMIFICATION_TIERS[idx - 1]
  if (!prevTier) return false
  return currentXP.value >= prevTier.minXP && currentXP.value < tier.minXP
}
</script>

<style scoped>
.roadmap-container {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.roadmap-header {
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

.roadmap-title {
  margin: 0 0 0.2rem;
  font-family: 'Manrope', sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--on-surface);
}

.roadmap-subtitle {
  margin: 0;
  font-size: 0.84rem;
  color: var(--muted-text);
}

.current-xp-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border: 1px solid var(--outline-variant);
  padding: 6px 14px;
  border-radius: 999px;
  font-weight: 800;
  font-size: 0.88rem;
  color: var(--primary);
}

.tiers-roadmap-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1.25rem;
}

.tier-step-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 18px;
  padding: 1.5rem 1.25rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), border-color 0.2s ease, box-shadow 0.2s ease;
}

.tier-step-card:hover {
  transform: translateY(-3px);
}

/* 1. Unlocked / Achieved Cards */
.tier-step-card.unlocked {
  background: var(--surface-container-low);
  border-color: var(--outline-variant);
}

.tier-step-card.current-tier {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--surface-container-low) 92%, var(--primary) 8%);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--primary) 15%, transparent);
}

/* 2. Next Goal (Hero Active Target Card) */
.tier-step-card.next-tier {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--surface-container-low) 90%, var(--primary) 10%);
  box-shadow: 0 6px 22px color-mix(in srgb, var(--primary) 16%, transparent);
}

.tier-step-card.next-tier .tier-name {
  color: var(--on-surface);
}

.tier-step-card.next-tier .tier-xp-target {
  color: var(--primary);
}

/* 3. Locked Cards: Subdued, clean, high-contrast tokens */
.tier-step-card.locked {
  background: color-mix(in srgb, var(--surface-container-low) 98%, var(--on-surface) 2%);
  border: 1px dashed var(--outline-variant);
}

.tier-step-card.locked:hover {
  border-color: color-mix(in srgb, var(--primary) 50%, var(--outline-variant));
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.08);
}

.tier-step-card.locked .tier-pedestal {
  opacity: 0.75;
  filter: grayscale(20%);
}

.tier-step-card.locked:hover .tier-pedestal {
  opacity: 1;
  filter: none;
}

.tier-step-card.locked .tier-name {
  color: var(--on-surface);
  opacity: 0.85;
}

.tier-step-card.locked .tier-xp-target {
  color: var(--muted-text);
}

.tier-pedestal {
  width: 64px;
  height: 64px;
  border-radius: 18px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0.25rem 0 1rem;
  transition: all 0.2s ease;
}

.tier-avatar {
  font-size: 2.2rem;
}

.step-badge {
  margin-bottom: 0.75rem;
}

.badge-status {
  font-size: 0.72rem;
  font-weight: 800;
  padding: 3px 10px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.badge-status.unlocked {
  background: color-mix(in srgb, #10b981 15%, transparent);
  color: #10b981;
  border: 1px solid var(--outline-variant);
}

.badge-status.next {
  background: color-mix(in srgb, var(--primary) 18%, transparent);
  color: var(--primary);
  border: 1px solid var(--outline-variant);
}

.badge-status.locked {
  background: var(--surface-container);
  color: var(--muted-text);
  border: 1px solid var(--outline-variant);
}

.tier-info {
  flex: 1;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.tier-name {
  margin: 0 0 2px;
  font-family: 'Manrope', sans-serif;
  font-size: 1.15rem;
  font-weight: 800;
  color: var(--on-surface);
}

.tier-xp-target {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--primary);
  margin-bottom: 0.6rem;
}

.tier-desc {
  font-size: 0.8rem;
  color: var(--muted-text);
  line-height: 1.4;
  margin: 0 0 1rem;
}

.tier-card-footer {
  width: 100%;
  margin-top: auto;
}

.preview-celebration-btn {
  width: 100%;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  padding: 8px 12px;
  border-radius: 10px;
  font-size: 0.8rem;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  transition: all 0.2s ease;
}

.preview-celebration-btn:hover {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  border-color: var(--primary);
  color: var(--primary);
}

.locked-requirement-pill {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 10px;
  font-size: 0.76rem;
  font-weight: 700;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  color: var(--muted-text);
}

.locked-requirement-pill.next-pill {
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-color: color-mix(in srgb, var(--primary) 28%, transparent);
  color: var(--primary);
}
</style>

