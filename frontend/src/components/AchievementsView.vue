<template>
  <div class="achievements-hub">
    <!-- Header Controls & Filters -->
    <div class="achievements-header">
      <div class="header-titles">
        <h3 class="hub-title">Study Milestones & Trophies</h3>
        <p class="hub-subtitle">Complete daily and cumulative goals to earn coins, title prestige, and exclusive themes.</p>
      </div>

      <!-- Filter Tabs -->
      <div class="filter-pills">
        <button
          type="button"
          class="filter-pill-btn"
          :class="{ active: currentFilter === 'ALL' }"
          @click="currentFilter = 'ALL'"
        >
          All ({{ totalCount }})
        </button>
        <button
          v-if="claimableCount > 0"
          type="button"
          class="filter-pill-btn pill-claimable"
          :class="{ active: currentFilter === 'CLAIMABLE' }"
          @click="currentFilter = 'CLAIMABLE'"
        >
          <span class="pulse-dot"></span>
          Ready ({{ claimableCount }})
        </button>
        <button
          type="button"
          class="filter-pill-btn"
          :class="{ active: currentFilter === 'IN_PROGRESS' }"
          @click="currentFilter = 'IN_PROGRESS'"
        >
          In Progress ({{ inProgressCount }})
        </button>
        <button
          type="button"
          class="filter-pill-btn"
          :class="{ active: currentFilter === 'COMPLETED' }"
          @click="currentFilter = 'COMPLETED'"
        >
          Completed ({{ completedCount }})
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="filteredAchievements.length === 0" class="empty-ach-panel">
      <BaseIcon name="trophy" size="40" custom-class="empty-icon" />
      <p>No achievements match this filter.</p>
    </div>

    <!-- Achievements Grid -->
    <div v-else class="achievements-grid">
      <div
        v-for="ach in filteredAchievements"
        :key="ach.id"
        class="ach-card"
        :class="{
          claimable: ach.claimable,
          completed: ach.completed && !ach.claimable,
          'tier-gold': (ach.reward_coins || 0) >= 100 || ach.reward_item,
          'tier-silver': (ach.reward_coins || 0) >= 50 && (ach.reward_coins || 0) < 100,
        }"
      >
        <div class="ach-icon-col">
          <div class="ach-pedestal" :class="{ 'pedestal-pulse': ach.claimable }">
            <BaseIcon :name="getAchBaseIcon(ach)" size="26" custom-class="ach-svg-icon" />
          </div>
          <span v-if="ach.claimable" class="ach-status-badge badge-claimable">Claimable</span>
          <span v-else-if="ach.claimed_tier > 0" class="ach-status-badge badge-unlocked">
            Tier {{ toRomanNumeral(ach.claimed_tier) }}
          </span>
        </div>

        <div class="ach-content-col">
          <div class="ach-heading-row">
            <h4 class="ach-name">{{ ach.title }}</h4>
            <div class="ach-reward-badge">
              <span v-if="ach.reward_coins" class="coin-reward">
                +{{ ach.reward_coins }}
                <BaseIcon name="coin" size="14" />
              </span>
              <span v-if="ach.reward_item" class="theme-reward-tag">
                <BaseIcon name="palette" size="13" /> Theme Unlock
              </span>
            </div>
          </div>

          <p class="ach-description">{{ ach.description }}</p>

          <!-- Interactive Progress Meter & Claim Section -->
          <div class="ach-progress-section">
            <div class="ach-progress-bar-track">
              <div
                class="ach-progress-bar-fill"
                :class="{ 'fill-claimable': ach.claimable }"
                :style="{ width: getProgressPercent(ach) + '%' }"
              ></div>
            </div>
            <div class="ach-progress-text">
              <span>{{ ach.current_value }} / {{ ach.target_value }}</span>
              <span>{{ getProgressPercent(ach) }}%</span>
            </div>
          </div>

          <!-- Claim Action Button -->
          <div v-if="ach.claimable" class="ach-action-row">
            <button
              type="button"
              class="claim-ach-btn"
              :disabled="claimingId === ach.id"
              @click.stop="$emit('claim', ach)"
            >
              <BaseIcon name="gift" size="15" />
              <span v-if="claimingId === ach.id">Claiming…</span>
              <span v-else>Claim +{{ ach.reward_coins }} Coins</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  achievements: {
    type: Array,
    default: () => [],
  },
  claimingId: {
    type: String,
    default: '',
  },
})

defineEmits(['claim'])

const currentFilter = ref('ALL')

const totalCount = computed(() => props.achievements.length)
const claimableCount = computed(() => props.achievements.filter((a) => a.claimable).length)
const completedCount = computed(() => props.achievements.filter((a) => (a.claimed_tier || 0) > 0).length)
const inProgressCount = computed(() => props.achievements.filter((a) => !a.claimable).length)

const filteredAchievements = computed(() => {
  if (currentFilter.value === 'CLAIMABLE') {
    return props.achievements.filter((a) => a.claimable)
  }
  if (currentFilter.value === 'COMPLETED') {
    return props.achievements.filter((a) => (a.claimed_tier || 0) > 0 || a.completed)
  }
  if (currentFilter.value === 'IN_PROGRESS') {
    return props.achievements.filter((a) => !a.claimable)
  }
  return props.achievements
})

function getProgressPercent(ach) {
  if (ach.claimable) return 100
  const cur = ach.current_value || 0
  const tgt = ach.target_value || 1
  return Math.min(100, Math.max(0, Math.round((cur / tgt) * 100)))
}

function toRomanNumeral(num) {
  if (!num || num <= 0) return 'I'
  const vals = [10, 9, 5, 4, 1]
  const syms = ['X', 'IX', 'V', 'IV', 'I']
  let res = ''
  let n = num
  for (let i = 0; i < vals.length; i++) {
    while (n >= vals[i]) {
      n -= vals[i]
      res += syms[i]
    }
  }
  return res || 'I'
}

function getAchBaseIcon(ach) {
  const t = (ach.title || '').toLowerCase()
  if (t.includes('night') || t.includes('scholar')) return 'brain'
  if (t.includes('quiz') || t.includes('master') || t.includes('ace')) return 'target'
  if (t.includes('streak') || t.includes('fire')) return 'flame'
  if (t.includes('speed') || t.includes('quick')) return 'zap'
  if (t.includes('vault') || t.includes('chest')) return 'gift'
  if (t.includes('reading') || t.includes('book') || t.includes('steps') || t.includes('diver')) return 'book'
  if (t.includes('flashcard') || t.includes('memory')) return 'cards'
  return 'award'
}
</script>

<style scoped>
.achievements-hub {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.achievements-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
  background: var(--surface-container-low, #1e1f26);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
  border-radius: 16px;
  padding: 1.25rem 1.5rem;
}

.hub-title {
  margin: 0 0 0.2rem;
  font-family: 'Manrope', sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--on-surface, #fff);
}

.hub-subtitle {
  margin: 0;
  font-size: 0.84rem;
  color: var(--muted-text, #94a3b8);
}

.filter-pills {
  display: flex;
  gap: 6px;
  background: var(--surface-container, rgba(255, 255, 255, 0.04));
  padding: 4px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.08));
}

.filter-pill-btn {
  background: transparent;
  border: none;
  color: var(--muted-text, #94a3b8);
  font-size: 0.8rem;
  font-weight: 700;
  padding: 6px 12px;
  border-radius: 8px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
}

.filter-pill-btn:hover {
  color: var(--on-surface, #fff);
  background: rgba(255, 255, 255, 0.05);
}

.filter-pill-btn.active {
  background: var(--surface-container-high, #2a2d36);
  color: var(--primary, #38bdf8);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
}

.filter-pill-btn.pill-claimable {
  color: #f59e0b;
}

.filter-pill-btn.pill-claimable.active {
  background: color-mix(in srgb, #f59e0b 20%, var(--surface-container-high));
  color: #f59e0b;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
  animation: pulse-animation 1.5s infinite;
}

@keyframes pulse-animation {
  0% { transform: scale(0.9); opacity: 0.8; }
  50% { transform: scale(1.4); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.8; }
}

.achievements-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 1.25rem;
}

.ach-card {
  position: relative;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 1.25rem;
  display: flex;
  gap: 1.25rem;
  align-items: flex-start;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.06);
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease, border-color 0.2s ease;
}

.ach-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.1);
}

.ach-card.claimable {
  border-color: #f59e0b;
  background: color-mix(in srgb, var(--surface-container) 92%, #f59e0b 8%);
  box-shadow: 0 0 16px rgba(245, 158, 11, 0.15);
}

.ach-card.completed {
  border-color: color-mix(in srgb, #10b981 35%, var(--outline-variant));
  background: color-mix(in srgb, var(--surface-container) 94%, #10b981 6%);
}

.ach-icon-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.ach-pedestal {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  color: var(--primary);
  transition: all 0.2s ease;
}

.ach-card.claimable .ach-pedestal {
  background: color-mix(in srgb, #f59e0b 16%, var(--surface-container-low));
  color: #f59e0b;
  border-color: rgba(245, 158, 11, 0.4);
}

.ach-card.completed .ach-pedestal {
  background: color-mix(in srgb, #10b981 12%, var(--surface-container-low));
  color: #10b981;
}

.ach-svg-icon {
  display: inline-block;
}

.ach-status-badge {
  font-size: 0.65rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  padding: 2px 6px;
  border-radius: 6px;
}

.badge-claimable {
  background: color-mix(in srgb, #f59e0b 20%, transparent);
  color: #f59e0b;
}

.badge-unlocked {
  background: color-mix(in srgb, #10b981 15%, transparent);
  color: #10b981;
}

.ach-content-col {
  flex: 1;
  min-width: 0;
}

.ach-heading-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}

.ach-name {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 1rem;
  font-weight: 700;
  color: var(--on-surface);
  line-height: 1.3;
}

.ach-reward-badge {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex-shrink: 0;
}

.coin-reward {
  font-size: 0.78rem;
  font-weight: 800;
  color: #f59e0b;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.theme-reward-tag {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--primary);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.ach-description {
  margin: 0 0 10px;
  font-size: 0.82rem;
  color: var(--muted-text);
  line-height: 1.4;
}

.ach-progress-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ach-progress-bar-track {
  height: 6px;
  background: var(--surface-container-low);
  border-radius: 999px;
  overflow: hidden;
}

.ach-progress-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-dim), var(--primary));
  transition: width 0.4s ease;
}

.ach-progress-bar-fill.fill-claimable {
  background: linear-gradient(90deg, #f59e0b, #fbbf24);
}

.ach-card.completed .ach-progress-bar-fill {
  background: #10b981;
}

.ach-progress-text {
  display: flex;
  justify-content: space-between;
  font-size: 0.74rem;
  color: var(--muted-text);
  font-weight: 600;
}

.ach-action-row {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
}

.claim-ach-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #fff;
  border: none;
  font-size: 0.82rem;
  font-weight: 800;
  padding: 6px 14px;
  border-radius: 8px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(245, 158, 11, 0.35);
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.claim-ach-btn:hover:not(:disabled) {
  transform: translateY(-1px) scale(1.02);
  box-shadow: 0 4px 12px rgba(245, 158, 11, 0.45);
}

.claim-ach-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.empty-ach-panel {
  padding: 3rem 1rem;
  text-align: center;
  background: var(--surface-container-low);
  border: 1px dashed var(--outline-variant);
  border-radius: 16px;
  color: var(--muted-text);
}

.empty-icon {
  font-size: 2.5rem;
  display: block;
  margin-bottom: 0.5rem;
}
</style>

