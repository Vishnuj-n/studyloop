<template>
  <StudyPageLayout
    eyebrow="Progression"
    title="Rewards & Milestones"
    subtitle="Track your scholar ranks, open mystery chests, complete trophies, and customize your study atmosphere."
  >
    <div v-if="loading" class="state-panel">
      <p>Loading your study rewards…</p>
    </div>

    <div v-else-if="loadError" class="state-panel error-panel">
      <p class="error-msg">{{ loadError }}</p>
      <button class="retry-btn" type="button" @click="loadData">Retry</button>
    </div>

    <div v-else class="rewards-container">
      <!-- Primary Tabs -->
      <div class="rewards-tab-bar">
        <button
          type="button"
          class="rewards-tab-btn"
          :class="{ active: currentView === 'overview' }"
          @click="currentView = 'overview'"
        >
          <BaseIcon name="zap" size="18" custom-class="tab-icon" />
          <span>Overview & Vault</span>
          <span v-if="chests.length > 0" class="tab-badge">{{ chests.length }}</span>
        </button>

        <button
          type="button"
          class="rewards-tab-btn"
          :class="{ active: currentView === 'roadmap' }"
          @click="currentView = 'roadmap'"
        >
          <BaseIcon name="map" size="18" custom-class="tab-icon" />
          <span>Scholar Roadmap</span>
        </button>

        <button
          type="button"
          class="rewards-tab-btn"
          :class="{ active: currentView === 'achievements' }"
          @click="currentView = 'achievements'"
        >
          <BaseIcon name="trophy" size="18" custom-class="tab-icon" />
          <span>Achievements</span>
          <span v-if="claimableAchievementsCount > 0" class="tab-badge">
            {{ claimableAchievementsCount }} ready
          </span>
          <span v-else-if="completedAchievementsCount > 0" class="tab-badge-emerald">
            {{ completedAchievementsCount }}/{{ achievements.length }}
          </span>
        </button>

        <button
          type="button"
          class="rewards-tab-btn"
          :class="{ active: currentView === 'shop' }"
          @click="currentView = 'shop'"
        >
          <BaseIcon name="palette" size="18" custom-class="tab-icon" />
          <span>Theme Wardrobe</span>
        </button>

        <button
          type="button"
          class="rewards-tab-btn"
          :class="{ active: currentView === 'sanctuary' }"
          @click="currentView = 'sanctuary'"
        >
          <BaseIcon name="sparkles" size="18" custom-class="tab-icon" />
          <span>Pet Sanctuary</span>
        </button>
      </div>

      <!-- Tab Content Views -->
      <RewardsOverviewView
        v-if="currentView === 'overview'"
        :profile="profile"
        :chests="chests"
        :buying="buying"
        :buy-error="buyError"
        @switch-tab="(tab) => currentView = tab"
        @open-freeze-info="openFreezeInfoModal"
        @buy-streak-freeze="handleBuyStreakFreeze"
        @open-chest="openChest"
      />

      <RankRoadmapView
        v-else-if="currentView === 'roadmap'"
        :total-xp="profile.total_xp"
        :current-title="profile.current_title"
        @preview-celebration="(tier) => activeMilestoneCelebration = tier"
      />

      <AchievementsView
        v-else-if="currentView === 'achievements'"
        :achievements="achievements"
        :claiming-id="claimingAchievementId"
        @claim="handleClaimAchievement"
      />

      <div v-else-if="currentView === 'shop'">
        <RewardsShopModal
          id="shop-section"
          :active-theme="activeTheme"
          :coins="profile.coins"
          @theme-changed="onThemeChanged"
        />
        <p v-if="themeError" class="buy-error-msg">{{ themeError }}</p>
      </div>

      <PetSanctuaryView
        v-else-if="currentView === 'sanctuary'"
        :coins="profile.coins"
        @coins-updated="loadData"
      />
    </div>

    <!-- Modals -->
    <MilestoneCelebrationModal
      v-if="activeMilestoneCelebration"
      :title="activeMilestoneCelebration.baseTitle || activeMilestoneCelebration.title"
      :xp="activeMilestoneCelebration.minXP || activeMilestoneCelebration.xp"
      @close="activeMilestoneCelebration = null"
    />

    <MysteryChestModal
      v-if="activeChest"
      :box="activeChest"
      @claimed="onChestClaimed"
      @close="activeChest = null"
    />

    <StreakFreezeModal
      v-if="showFreezeModal"
      :total-freezes="profile.streak_freezes_owned"
      :mode="freezeModalMode"
      @close="showFreezeModal = false"
    />
  </StudyPageLayout>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BaseIcon from '../components/BaseIcon.vue'
import StudyPageLayout from '../components/StudyPageLayout.vue'
import RewardsOverviewView from '../components/RewardsOverviewView.vue'
import RankRoadmapView from '../components/RankRoadmapView.vue'
import AchievementsView from '../components/AchievementsView.vue'
import RewardsShopModal from '../components/RewardsShopModal.vue'
import PetSanctuaryView from '../components/PetSanctuaryView.vue'
import MilestoneCelebrationModal from '../components/MilestoneCelebrationModal.vue'
import MysteryChestModal from '../components/MysteryChestModal.vue'
import StreakFreezeModal from '../components/StreakFreezeModal.vue'
import {
  getGamificationState,
  buyStreakFreeze,
  getUserSettings,
  updateUserSettings,
  getGamificationStore,
  claimAchievement,
} from '../services/appApi'

const loading = ref(true)
const buying = ref(false)
const claimingAchievementId = ref('')
const showFreezeModal = ref(false)
const freezeModalMode = ref('purchase')
const activeTheme = ref('dark-gruvbox')
const userSettings = ref(null)
const loadError = ref('')
const buyError = ref('')
const themeError = ref('')
const currentView = ref('overview')

const profile = ref({
  level: 1,
  total_xp: 0,
  coins: 0,
  current_title: 'The Apprentice I',
  next_title: 'The Scholar I',
  next_title_xp: 1500,
  current_title_min_xp: 0,
  streak_freezes_owned: 1,
  last_freeze_purchased_at: 0,
})

const chests = ref([])
const activeChest = ref(null)
const achievements = ref([])
const activeMilestoneCelebration = ref(null)

const claimableAchievementsCount = computed(() => {
  return achievements.value.filter((a) => a.claimable).length
})

const completedAchievementsCount = computed(() => {
  return achievements.value.filter((a) => (a.claimed_tier || 0) > 0).length
})

function openFreezeInfoModal() {
  freezeModalMode.value = 'info'
  showFreezeModal.value = true
}

function openChest(chest) {
  activeChest.value = chest
}

async function onChestClaimed(claimedBox) {
  chests.value = chests.value.filter((c) => c.id !== claimedBox.id)
  await loadData()
}

async function handleClaimAchievement(ach) {
  if (!ach || !ach.id || claimingAchievementId.value) return
  claimingAchievementId.value = ach.id
  try {
    const res = await claimAchievement(ach.id)
    if (res && res.error) {
      console.error('Failed to claim achievement:', res.error)
      return
    }
    if (res && res.store) {
      if (res.store.profile) profile.value = res.store.profile
      if (res.store.achievements) achievements.value = res.store.achievements
    } else {
      await loadData()
    }
    window.dispatchEvent(new Event('gamification-updated'))
  } catch (err) {
    console.error('Error claiming achievement:', err)
  } finally {
    claimingAchievementId.value = ''
  }
}

async function loadData(silent = false) {
  if (!silent && !achievements.value.length) {
    loading.value = true
  }
  loadError.value = ''
  try {
    const res = await getGamificationState()
    if (res && res.profile) {
      profile.value = res.profile
    }
    if (res && res.pending_chests) {
      chests.value = res.pending_chests
    }

    let storeRes = null
    try {
      if (typeof getGamificationStore === 'function') {
        storeRes = await getGamificationStore()
      }
    } catch {
      storeRes = null
    }
    if (storeRes && storeRes.achievements) {
      achievements.value = storeRes.achievements
    } else if (storeRes && storeRes.store && storeRes.store.achievements) {
      achievements.value = storeRes.store.achievements
    }

    const settings = await (getUserSettings ? getUserSettings().catch(() => null) : Promise.resolve(null))
    if (settings && !settings.error) {
      userSettings.value = settings
      if (settings.theme) {
        activeTheme.value = settings.theme
      }
    }
  } catch (err) {
    console.error('Failed to load gamification state:', err)
    if (!silent) {
      loadError.value = err?.message || 'Failed to load rewards. Please try again.'
    }
  } finally {
    loading.value = false
  }
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
  const prevTheme = activeTheme.value
  activeTheme.value = newTheme
  localStorage.setItem('app-theme', newTheme)
  document.documentElement.setAttribute('data-theme', newTheme)

  function rollback(errMsg) {
    activeTheme.value = prevTheme
    localStorage.setItem('app-theme', prevTheme)
    document.documentElement.setAttribute('data-theme', prevTheme)
    themeError.value = errMsg || 'Failed to save theme setting.'
  }

  try {
    let current = userSettings.value
    if (!current) {
      current = await getUserSettings().catch(() => null)
    }
    if (!current || current.error) {
      rollback(current?.error || 'Failed to load user settings for theme update.')
      return
    }

    const updated = { ...current, theme: newTheme }
    const res = await updateUserSettings(updated)
    if (res && res.error) {
      rollback(res.error)
    } else {
      userSettings.value = updated
      themeError.value = ''
      window.dispatchEvent(new CustomEvent('settings-updated'))
    }
  } catch (err) {
    console.error('Failed to update theme setting:', err)
    rollback(err?.message || 'Failed to save theme setting.')
  }
}

const handleSilentUpdate = () => loadData(true)

onMounted(() => {
  loadData()
  window.addEventListener('gamification-updated', handleSilentUpdate)
})

onUnmounted(() => {
  window.removeEventListener('gamification-updated', handleSilentUpdate)
})
</script>

<style scoped>
.rewards-container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.rewards-tab-bar {
  display: flex;
  gap: 8px;
  background: var(--surface-container-low);
  padding: 6px;
  border-radius: 16px;
  border: 1px solid var(--outline-variant);
  overflow-x: auto;
}

.rewards-tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: transparent;
  border: none;
  color: var(--muted-text);
  font-family: 'Manrope', sans-serif;
  font-size: 0.9rem;
  font-weight: 700;
  padding: 10px 16px;
  border-radius: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.rewards-tab-btn:hover {
  color: var(--on-surface);
  background: rgba(255, 255, 255, 0.05);
}

.rewards-tab-btn.active {
  background: var(--surface-container);
  color: var(--primary);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.08);
}

.tab-icon {
  font-size: 1.1rem;
}

.tab-badge {
  background: #f59e0b;
  color: #111;
  font-size: 0.72rem;
  font-weight: 800;
  padding: 1px 7px;
  border-radius: 999px;
}

.tab-badge-emerald {
  background: color-mix(in srgb, #10b981 18%, transparent);
  color: #10b981;
  font-size: 0.72rem;
  font-weight: 800;
  padding: 1px 7px;
  border-radius: 999px;
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
