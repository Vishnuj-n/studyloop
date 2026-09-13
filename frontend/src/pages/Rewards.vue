<template>
  <StudyPageLayout
    eyebrow="Progression"
    title="Rewards"
    subtitle="Track your rank, open mystery chests, and protect your study streaks."
  >
    <div v-if="loading" class="state-panel">
      <p>Loading your study rewards…</p>
    </div>

    <div v-else-if="loadError" class="state-panel error-panel">
      <p class="error-msg">{{ loadError }}</p>
      <button class="retry-btn" type="button" @click="loadData">Retry</button>
    </div>

    <div v-else class="rewards-grid">
      <!-- Title & XP Rank Card -->
      <div class="card floating-card rank-card">
        <div class="card-glow rank-glow"></div>
        <div class="rank-header">
          <div class="rank-avatar-pedestal">
            <span class="rank-avatar">{{ getTitleEmoji(profile.current_title) }}</span>
          </div>
          <div>
            <span class="rank-label">Current Title</span>
            <h2 class="rank-title">{{ profile.current_title }}</h2>
          </div>
        </div>

        <div class="xp-bar-container">
          <div class="xp-bar-track">
            <div class="xp-bar-fill" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="xp-labels">
            <span>{{ profile.total_xp }} XP</span>
            <span>Next Rank: {{ profile.next_title }} ({{ profile.next_title_xp }} XP)</span>
          </div>
        </div>
      </div>

      <!-- Currency & Inventory Card -->
      <div class="card floating-card currency-card">
        <div class="currency-group">
          <div class="currency-item">
            <div class="icon-pedestal coin-pedestal">
              <span class="currency-icon">🪙</span>
            </div>
            <div>
              <span class="currency-val">{{ profile.coins }}</span>
              <span class="currency-name">Study Coins</span>
            </div>
          </div>
          <div class="currency-item">
            <div class="icon-pedestal shield-pedestal">
              <span class="currency-icon">🛡️</span>
            </div>
            <div>
              <div style="display: flex; align-items: center; gap: 0.35rem;">
                <span class="currency-val">{{ profile.streak_freezes_owned }}</span>
                <button
                  type="button"
                  title="View how streak freezes work"
                  class="info-icon-btn"
                  @click="openFreezeInfoModal"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="12" y1="16" x2="12" y2="12"></line>
                    <line x1="12" y1="8" x2="12.01" y2="8"></line>
                  </svg>
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
            :disabled="profile.coins < 50 || buying"
            @click="handleBuyStreakFreeze"
          >
            {{ buying ? 'Purchasing...' : 'Buy Streak Freeze (50 Coins)' }}
          </button>
          <button
            class="shop-trigger-btn"
            type="button"
            @click="showShopModal = true"
          >
            🛒 Rewards & Theme Shop
          </button>
        </div>
        <p v-if="buyError" class="buy-error-msg">{{ buyError }}</p>
      </div>

      <!-- Pending Loot Boxes -->
      <div class="card floating-card chests-card">
        <h3 class="card-section-title">Mystery Chests Vault</h3>
        <p class="chests-desc">Chests are earned by completing reading sessions, quizzes, and reviews.</p>

        <div v-if="chests.length === 0" class="empty-chests">
          <div class="empty-chest-pedestal">
            <span>📦</span>
          </div>
          <p>No unopened chests. Complete study tasks to earn more!</p>
        </div>

        <div v-else class="chests-list">
          <button
            v-for="chest in chests"
            :key="chest.id"
            type="button"
            class="chest-vault-item"
            :class="'tier-' + chest.box_tier.toLowerCase()"
            @click="openChest(chest)"
          >
            <div class="chest-pedestal">
              <GamificationIcon :name="'chest-' + chest.box_tier.toLowerCase()" size="40" />
            </div>
            <div class="chest-vault-info">
              <span class="chest-vault-tier">{{ chest.box_tier }} CHEST</span>
              <span class="chest-vault-tap">Click to open</span>
            </div>
          </button>
        </div>
      </div>
    </div>

    <!-- Theme & Achievement Shop Modal -->
    <RewardsShopModal
      v-if="showShopModal"
      :active-theme="activeTheme"
      @close="onShopClose"
      @theme-changed="onThemeChanged"
    />

    <!-- Mystery Chest Modal -->
    <MysteryChestModal
      v-if="activeChest"
      :box="activeChest"
      @claimed="onChestClaimed"
      @close="activeChest = null"
    />

    <!-- Streak Freeze Modal -->
    <StreakFreezeModal
      v-if="showFreezeModal"
      :total-freezes="profile.streak_freezes_owned"
      :mode="freezeModalMode"
      @close="showFreezeModal = false"
    />
  </StudyPageLayout>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import StudyPageLayout from '../components/StudyPageLayout.vue'
import { getGamificationState, buyStreakFreeze, getUserSettings, updateUserSettings } from '../services/appApi'
import MysteryChestModal from '../components/MysteryChestModal.vue'
import GamificationIcon from '../components/icons/GamificationIcon.vue'
import StreakFreezeModal from '../components/StreakFreezeModal.vue'
import RewardsShopModal from '../components/RewardsShopModal.vue'

const showShopModal = ref(false)

const loading = ref(true)
const buying = ref(false)
const showFreezeModal = ref(false)
const freezeModalMode = ref('purchase')
const activeTheme = ref('dark-gruvbox')
const loadError = ref('')
const buyError = ref('')
const profile = ref({
  total_xp: 0,
  coins: 0,
  current_title: 'The Apprentice',
  next_title: 'The Scholar',
  next_title_xp: 500,
  current_title_min_xp: 0,
  streak_freezes_owned: 1,
})
const chests = ref([])
const activeChest = ref(null)

const progressPercent = computed(() => {
  const min = profile.value.current_title_min_xp || 0
  const max = profile.value.next_title_xp || 500
  const cur = profile.value.total_xp || 0
  if (cur >= max) return 100
  const range = max - min
  if (range <= 0) return 100
  const pct = ((cur - min) / range) * 100
  return Math.min(100, Math.max(0, Math.round(pct)))
})

function getChestEmoji(tier) {
  switch ((tier || '').toUpperCase()) {
    case 'SILVER':
      return '🥈'
    case 'GOLD':
      return '🎁'
    case 'MYTHIC':
      return '👑'
    default:
      return '📦'
  }
}

function getTitleEmoji(title) {
  const t = (title || '').toLowerCase()
  if (t.includes('mythic') || t.includes('sage')) return '🐉' // Mythic Dragon
  if (t.includes('luminary')) return '🦸'                      // Superhero Luminary
  if (t.includes('paragon')) return '🤴'                       // Royal King
  if (t.includes('grandmaster')) return '🐦‍🔥'                  // Legendary Phoenix
  if (t.includes('polymath')) return '🦖'                      // Apex T-Rex Dinosaur
  if (t.includes('archivist')) return '🦄'                     // Magical Unicorn
  if (t.includes('inquisitor')) return '🦚'                    // Flamboyant Peacock
  if (t.includes('scholar')) return '🦁'                      // Mighty Lion
  if (t.includes('apprentice') || t.includes('novice')) return '🧑‍🎓' // Student
  return '🧑‍🎓'
}

function openFreezeInfoModal() {
  freezeModalMode.value = 'info'
  showFreezeModal.value = true
}

async function loadData() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await getGamificationState()
    if (res && res.profile) {
      profile.value = res.profile
    }
    if (res && res.pending_chests) {
      chests.value = res.pending_chests
    }
    const settings = await getUserSettings().catch(() => null)
    if (settings && settings.theme) {
      activeTheme.value = settings.theme
    }
  } catch (err) {
    console.error('Failed to load gamification state:', err)
    loadError.value = err?.message || 'Failed to load rewards. Please try again.'
  } finally {
    loading.value = false
  }
}

function openChest(chest) {
  activeChest.value = chest
}

async function onChestClaimed(claimedBox) {
  chests.value = chests.value.filter((c) => c.id !== claimedBox.id)
  await loadData()
}

async function handleBuyStreakFreeze() {
  buying.value = true
  buyError.value = ''
  try {
    const res = await buyStreakFreeze()
    if (res && res.profile) {
      profile.value = res.profile
      freezeModalMode.value = 'purchase'
      if (localStorage.getItem('hideStreakFreezeModal') !== 'true') {
        showFreezeModal.value = true
      }
      window.dispatchEvent(new Event('gamification-updated'))
    }
  } catch (err) {
    console.error('Failed to buy streak freeze:', err)
    buyError.value = err?.message || 'Failed to purchase streak freeze.'
  } finally {
    buying.value = false
  }
}

async function onThemeChanged(newTheme) {
  activeTheme.value = newTheme
  try {
    await updateUserSettings({ theme: newTheme })
  } catch (err) {
    console.error('Failed to update theme setting:', err)
  }
}

function onShopClose() {
  showShopModal.value = false
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.shop-action-group {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-top: 16px;
  flex-wrap: wrap;
}

.shop-trigger-btn {
  background: var(--surface-card, #25282a);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.15));
  color: var(--text-color, #e0e0e0);
  padding: 10px 18px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.shop-trigger-btn:hover {
  background: var(--surface-hover, #2f3336);
  border-color: var(--accent-color, #d79921);
  color: var(--accent-color, #d79921);
}

.rewards-grid {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  margin-top: 0.5rem;
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
  transition: color 0.15s ease, background-color 0.15s ease, transform 0.15s ease;
}

.info-icon-btn:hover {
  color: #38bdf8;
  background: color-mix(in srgb, #38bdf8 15%, transparent);
  transform: scale(1.1);
}

.floating-card {
  position: relative;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 1.5rem 1.75rem;
  box-shadow: 0 10px 30px -8px rgba(0, 0, 0, 0.12), 0 4px 12px -2px rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(12px);
  overflow: hidden;
  transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.25s ease;
}

.floating-card:hover {
  box-shadow: 0 16px 36px -8px rgba(0, 0, 0, 0.15), 0 6px 16px -2px rgba(0, 0, 0, 0.08);
}

.card-glow {
  position: absolute;
  top: -50px;
  right: -50px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  pointer-events: none;
  opacity: 0.35;
}

.rank-glow {
  background: radial-gradient(circle, color-mix(in srgb, var(--primary) 18%, transparent) 0%, transparent 70%);
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
  background: color-mix(in srgb, var(--surface-container-low) 80%, transparent);
  border: 1px solid var(--outline-variant);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.25s ease, border-color 0.25s ease;
}

.rank-card:hover .rank-avatar-pedestal {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--primary) 40%, var(--outline-variant));
}

.rank-avatar {
  font-size: 2rem;
  display: inline-block;
  filter: drop-shadow(0 0 6px color-mix(in srgb, var(--primary) 65%, transparent));
  transition: transform 0.25s ease, filter 0.25s ease;
}

.rank-card:hover .rank-avatar {
  transform: scale(1.08);
  filter: drop-shadow(0 0 12px color-mix(in srgb, var(--primary) 90%, transparent));
}

.rank-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--muted-text);
  text-transform: uppercase;
  letter-spacing: 0.12em;
}

.rank-title {
  margin: 0.2rem 0 0;
  font-family: 'Manrope', sans-serif;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--on-surface);
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
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.currency-item:hover .icon-pedestal {
  transform: translateY(-2px);
}

.coin-pedestal {
  background: radial-gradient(circle, rgba(245, 158, 11, 0.15) 0%, var(--surface-container-low) 85%);
  border-color: rgba(245, 158, 11, 0.3);
}

.shield-pedestal {
  background: radial-gradient(circle, color-mix(in srgb, var(--primary) 15%, transparent) 0%, var(--surface-container-low) 85%);
  border-color: color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
}

.currency-icon {
  font-size: 1.6rem;
  display: inline-block;
  transition: transform 0.2s ease, filter 0.2s ease;
}

.coin-pedestal .currency-icon {
  filter: drop-shadow(0 0 6px rgba(245, 158, 11, 0.75));
}

.currency-item:hover .coin-pedestal .currency-icon {
  transform: scale(1.1);
  filter: drop-shadow(0 0 12px rgba(245, 158, 11, 1));
}

.shield-pedestal .currency-icon {
  filter: drop-shadow(0 0 6px rgba(56, 189, 248, 0.75));
}

.currency-item:hover .shield-pedestal .currency-icon {
  transform: scale(1.1);
  filter: drop-shadow(0 0 12px rgba(56, 189, 248, 1));
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
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease, opacity 0.2s ease;
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
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 0.75rem;
}

.empty-chest-pedestal span {
  font-size: 2rem;
  opacity: 0.85;
  display: inline-block;
  filter: drop-shadow(0 0 5px rgba(217, 119, 6, 0.4));
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
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
  transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1),
              box-shadow 0.25s ease,
              border-color 0.25s ease;
}

.chest-vault-item:hover {
  transform: translateY(-4px);
  border-color: var(--primary);
  box-shadow: 0 12px 24px -4px rgba(0, 0, 0, 0.15), 0 0 14px color-mix(in srgb, var(--primary) 20%, transparent);
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
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s ease;
}

.chest-vault-item:hover .chest-pedestal {
  transform: scale(1.05);
}

.chest-vault-icon {
  font-size: 2.2rem;
  display: inline-block;
  transition: transform 0.2s ease, filter 0.2s ease;
}

.tier-bronze .chest-vault-icon {
  filter: drop-shadow(0 0 6px rgba(217, 119, 6, 0.75));
}
.chest-vault-item.tier-bronze:hover .chest-vault-icon {
  transform: scale(1.1);
  filter: drop-shadow(0 0 14px rgba(217, 119, 6, 1));
}

.tier-silver .chest-vault-icon {
  filter: drop-shadow(0 0 6px rgba(203, 213, 225, 0.8));
}
.chest-vault-item.tier-silver:hover .chest-vault-icon {
  transform: scale(1.1);
  filter: drop-shadow(0 0 14px rgba(203, 213, 225, 1));
}

.tier-gold .chest-vault-icon {
  filter: drop-shadow(0 0 8px rgba(251, 191, 36, 0.85));
}
.chest-vault-item.tier-gold:hover .chest-vault-icon {
  transform: scale(1.12);
  filter: drop-shadow(0 0 16px rgba(251, 191, 36, 1));
}

.tier-mythic .chest-vault-icon {
  filter: drop-shadow(0 0 10px rgba(192, 132, 252, 0.9));
}
.chest-vault-item.tier-mythic:hover .chest-vault-icon {
  transform: scale(1.14);
  filter: drop-shadow(0 0 20px rgba(192, 132, 252, 1));
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

.tier-mythic:hover {
  box-shadow: 0 12px 28px -4px rgba(0, 0, 0, 0.15), 0 0 16px rgba(192, 132, 252, 0.3);
}

.state-panel {
  padding: 3rem 1rem;
  text-align: center;
  color: var(--muted-text);
}

.error-panel {
  color: var(--danger, #ef4444);
}

.error-msg {
  margin-bottom: 1rem;
  font-weight: 600;
}

.retry-btn {
  padding: 0.5rem 1.25rem;
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 600;
}

.buy-error-msg {
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: var(--danger, #ef4444);
  font-weight: 600;
}
</style>
