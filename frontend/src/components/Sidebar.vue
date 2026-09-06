<template>
  <aside class="sidebar">
    <div>
      <div class="brand">
        <div class="brand-mark">S</div>
        <div>
          <p class="brand-title">The StudyLoop</p>
        </div>
      </div>

      <!-- Compact Gamification Pill -->
      <RouterLink to="/rewards" class="gamification-sidebar-pill" title="View Rewards & Progression">
        <span class="pill-title">{{ gamification.current_title }}</span>
        <div class="pill-xp-bar">
          <div class="pill-xp-fill" :style="{ width: xpPercent + '%' }"></div>
        </div>
      </RouterLink>

      <nav class="menu">
        <RouterLink v-for="item in topItems" :key="item.to" :to="item.to" class="menu-item">
          <span class="menu-icon" aria-hidden="true">{{ item.icon }}</span>
          {{ item.label }}
        </RouterLink>
      </nav>
    </div>

    <div class="bottom-actions">
      <button
        v-if="isCloudAccount"
        class="sync-link"
        :class="{ 'sync-success': syncState === 'success', 'sync-error': syncState === 'error' }"
        type="button"
        :disabled="syncing"
        title="Sync with Cloud"
        @click="handleSync"
      >
        <span class="menu-icon">{{ syncIcon }}</span>
        <span>{{ syncLabel }}</span>
      </button>
      <RouterLink to="/settings" class="menu-item bottom-item">Settings</RouterLink>
    </div>
  </aside>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { triggerCloudSync, getUserSettings, getGamificationState } from '../services/appApi'

const gamification = ref({
  current_title: 'The Apprentice',
  total_xp: 0,
  next_title_xp: 500,
  current_title_min_xp: 0,
})

const xpPercent = computed(() => {
  const min = gamification.value.current_title_min_xp || 0
  const max = gamification.value.next_title_xp || 500
  const cur = gamification.value.total_xp || 0
  if (cur >= max) return 100
  const range = max - min
  if (range <= 0) return 100
  return Math.min(100, Math.max(0, Math.round(((cur - min) / range) * 100)))
})

const syncing = ref(false)
const syncState = ref('idle')
const isCloudAccount = ref(false)
let syncTimer = null

const syncIcon = computed(() => {
  if (syncState.value === 'success') return '✓'
  if (syncState.value === 'error') return '⚠️'
  return '☁️'
})

const syncLabel = computed(() => {
  if (syncState.value === 'syncing') return 'Syncing...'
  if (syncState.value === 'success') return 'Sync successful'
  if (syncState.value === 'error') return 'Sync failed'
  return 'Sync Cloud'
})

async function checkCloudAccount() {
  const s = await getUserSettings().catch(() => null)
  isCloudAccount.value = !!(s?.cloud_api_token || s?.classroom_code)
}

async function handleSync() {
  if (syncing.value) return
  if (syncTimer) {
    clearTimeout(syncTimer)
    syncTimer = null
  }
  syncing.value = true
  syncState.value = 'syncing'
  try {
    const res = await triggerCloudSync()
    if (res?.error) {
      console.warn('[SIDEBAR] Cloud sync error:', res.error)
      syncState.value = 'error'
    } else {
      console.log('[SIDEBAR] Cloud sync completed successfully')
      syncState.value = 'success'
    }
  } catch (err) {
    console.warn('[SIDEBAR] Cloud sync exception:', err)
    syncState.value = 'error'
  } finally {
    syncing.value = false
    syncTimer = setTimeout(() => {
      syncState.value = 'idle'
      syncTimer = null
    }, 4000)
  }
}

async function loadGamification() {
  try {
    const res = await getGamificationState()
    if (res?.profile) {
      gamification.value = res.profile
    }
  } catch (err) {
    console.warn('[SIDEBAR] Failed to load gamification profile:', err)
  }
}

onMounted(() => {
  checkCloudAccount()
  loadGamification()
  window.addEventListener('settings-updated', checkCloudAccount)
  window.addEventListener('gamification-updated', loadGamification)
})

onUnmounted(() => {
  window.removeEventListener('settings-updated', checkCloudAccount)
  window.removeEventListener('gamification-updated', loadGamification)
  if (syncTimer) {
    clearTimeout(syncTimer)
    syncTimer = null
  }
})

const topItems = [
  { to: '/dashboard', label: 'Dashboard', icon: '▦' },
  { to: '/reader', label: 'Reader', icon: '◫' },
  { to: '/notebooks', label: 'Notebooks', icon: '▤' },
  { to: '/quiz', label: 'Quiz', icon: '◪' },
  { to: '/flashcards', label: 'Flashcards', icon: '◧' },
  { to: '/examiner', label: 'Examiner', icon: '✎' },
  { to: '/tutor', label: 'Tutor', icon: '◎' },
  { to: '/rewards', label: 'Rewards', icon: '✦' },
  { to: '/extensions', label: 'Extensions', icon: '❖' },
]
</script>

<style scoped>
.sidebar {
  width: 248px;
  min-width: 248px;
  background: var(--surface-container);
  padding: 24px 16px 24px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 24px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 8px;
}

.brand-mark {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: linear-gradient(15deg, var(--primary-dim), var(--primary));
  color: var(--on-primary);
  display: grid;
  place-items: center;
  font-family: 'Manrope', sans-serif;
  font-weight: 700;
}

.brand-title {
  margin: 0;
  font-family: 'Manrope', sans-serif;
  font-size: 20px;
  letter-spacing: -0.02em;
  font-weight: 700;
  color: var(--on-surface);
}

.gamification-sidebar-pill {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 14px;
  padding: 9px 12px;
  background: color-mix(in srgb, var(--on-surface) 3%, transparent);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  text-decoration: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1),
              background-color 0.2s ease,
              border-color 0.2s ease,
              box-shadow 0.2s ease;
}

.gamification-sidebar-pill:hover {
  background: var(--surface-container-low);
  border-color: color-mix(in srgb, var(--primary) 50%, var(--outline-variant));
  transform: translateY(-1.5px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.16);
}

.pill-title {
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pill-xp-bar {
  height: 4px;
  background: var(--outline-variant);
  border-radius: 999px;
  overflow: hidden;
}

.pill-xp-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-dim), var(--primary));
  transition: width 0.3s ease;
}

.menu {
  margin-top: 20px;
  display: grid;
  gap: 6px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 9px;
  border-radius: 12px;
  text-decoration: none;
  color: var(--on-surface);
  padding: 11px 12px;
  font-size: 15px;
  font-weight: 500;
  background: transparent;
  transition:
    background-color 0.2s cubic-bezier(0.16, 1, 0.3, 1),
    color 0.2s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.menu-icon {
  width: 18px;
  min-width: 18px;
  text-align: center;
  color: color-mix(in srgb, var(--muted-text) 65%, var(--on-surface));
  font-size: 12px;
  line-height: 1;
}

.menu-item:hover {
  background: var(--surface-container-low);
  transform: translateX(4px);
}

.menu-item.router-link-active {
  background: var(--surface-container-lowest);
  color: var(--primary);
}

.menu-item.router-link-active .menu-icon {
  color: var(--primary);
}

.bottom-actions {
  display: grid;
  gap: 10px;
}

.sync-link {
  display: flex;
  align-items: center;
  gap: 9px;
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  font-size: 14px;
  font-weight: 500;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  width: 100%;
}

.sync-link:hover:not(:disabled) {
  background: var(--surface-container-highest);
  color: var(--primary);
  border-color: var(--primary);
}

.sync-link.sync-success {
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.4);
  background: color-mix(in srgb, #10b981 12%, var(--surface-container-low));
}

.sync-link.sync-success .menu-icon {
  color: #10b981;
}

.sync-link.sync-error {
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.4);
  background: color-mix(in srgb, #ef4444 12%, var(--surface-container-low));
}

.sync-link.sync-error .menu-icon {
  color: #ef4444;
}

.sync-link:disabled {
  opacity: 0.6;
  cursor: wait;
}

.bottom-item {
  color: var(--muted-text);
}

@media (max-width: 960px) {
  .sidebar {
    width: 100%;
    min-width: 0;
    border-radius: 0;
    padding: 16px;
    gap: 16px;
  }

  .menu {
    margin-top: 16px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .bottom-actions {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    align-items: center;
  }

  .sync-link,
  .bottom-item {
    text-align: center;
  }
}
</style>
