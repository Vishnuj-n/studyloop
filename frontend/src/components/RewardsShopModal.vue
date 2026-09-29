<template>
  <div class="shop-section-hub">
    <div class="shop-header">
      <div class="header-titles">
        <h3 class="shop-title">Theme Wardrobe & Cosmetics</h3>
        <p class="shop-subtitle">Customize your study atmosphere. Unlock palettes with study coins or earn them through achievements.</p>
      </div>

      <div v-if="coins !== undefined" class="coin-pill">
        <BaseIcon name="coin" size="16" />
        <span class="coin-balance">{{ coins }} Coins</span>
      </div>
    </div>

    <!-- Loading / Error States -->
    <div v-if="loading" class="empty-state">Loading Theme Wardrobe...</div>
    <div v-else-if="error" class="error-state">{{ error }}</div>

    <div v-else class="themes-grid">
      <div
        v-for="item in themes"
        :key="item.id"
        class="shop-item-card"
        :class="{ unlocked: item.unlocked, active: activeTheme === item.id }"
      >
        <div class="item-header">
          <span class="item-title">{{ item.name }}</span>
          <span v-if="activeTheme === item.id" class="badge active-badge">Current Theme</span>
          <span v-else-if="item.unlocked" class="badge unlocked-badge">
            <BaseIcon name="check" size="12" /> Unlocked
          </span>
          <span v-else-if="item.price > 0" class="badge price-badge">
            <BaseIcon name="coin" size="12" /> {{ item.price }}
          </span>
          <span v-else class="badge condition-badge">
            <BaseIcon name="lock" size="12" /> Locked
          </span>
        </div>

        <!-- Theme Color Swatch Preview -->
        <div
          v-if="themePreviewColors[item.id]"
          class="shop-theme-preview"
          :style="{ background: themePreviewColors[item.id].bg }"
        >
          <span class="swatch-dot" :style="{ background: themePreviewColors[item.id].primary }"></span>
          <span class="swatch-dot" :style="{ background: themePreviewColors[item.id].surface }"></span>
        </div>

        <p v-if="item.unlock_condition && !item.unlocked" class="unlock-desc">
          {{ item.unlock_condition }}
        </p>

        <div class="item-footer">
          <button
            v-if="!item.unlocked && item.price > 0"
            type="button"
            class="buy-btn"
            :disabled="buying === item.id || (coins || 0) < item.price"
            @click="buyItem(item)"
          >
            {{ buying === item.id ? 'Unlocking...' : `Unlock for ${item.price} Coins` }}
          </button>

          <button
            v-else-if="item.unlocked"
            type="button"
            class="equip-btn"
            :class="{ active: activeTheme === item.id }"
            @click="equipTheme(item.id)"
          >
            {{ activeTheme === item.id ? 'Active Theme' : 'Apply Theme' }}
          </button>

          <span v-else class="locked-hint">Earn via Achievement</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import {
  getGamificationStore,
  unlockCosmeticItem,
} from '../services/appApi'

const props = defineProps({
  activeTheme: { type: String, default: 'dark-gruvbox' },
  coins: { type: Number, default: 0 },
})

const emit = defineEmits(['theme-changed'])

const themePreviewColors = {
  'dark-gruvbox': { bg: '#1d2021', primary: '#d79921', surface: '#282828' },
  'light-classic': { bg: '#f9f9fb', primary: '#005bc1', surface: '#ebeef2' },
  'dark-indigo': { bg: '#0b0d16', primary: '#6366f1', surface: '#171a2b' },
  'dark-emerald': { bg: '#0a120d', primary: '#10b981', surface: '#152219' },
  'light-warm': { bg: '#fdfaf6', primary: '#c27d38', surface: '#f3eae1' },
  'light-sage': { bg: '#f4f7f4', primary: '#2e7d32', surface: '#e2ebe2' },
  'dark-academia': { bg: '#1c1917', primary: '#d97706', surface: '#292524' },
  'dark-cyberpunk': { bg: '#090714', primary: '#ff007f', surface: '#161228' },
  'light-zen': { bg: '#f5f5f0', primary: '#44403c', surface: '#e5e5df' },
  'dark-obsidian': { bg: '#09090b', primary: '#f4f4f5', surface: '#18181b' },
  'light-monochrome': { bg: '#f8f8f9', primary: '#18181b', surface: '#e8e8ec' },
}

const themes = ref([])
const loading = ref(true)
const error = ref('')
const buying = ref('')

async function fetchThemes() {
  loading.value = true
  error.value = ''
  try {
    const res = await getGamificationStore()
    if (res && res.error) {
      error.value = res.error
    } else if (res && res.store && res.store.themes) {
      themes.value = res.store.themes
    } else if (res && res.themes) {
      themes.value = res.themes
    }
  } catch (err) {
    if (import.meta.env.DEV || import.meta.env.MODE === 'test') {
      themes.value = [
        { id: 'dark-gruvbox', name: 'Gruvbox Dark', type: 'theme', price: 0, unlocked: true },
        { id: 'light-classic', name: 'Light Classic', type: 'theme', price: 0, unlocked: true },
        { id: 'dark-indigo', name: 'Deep Indigo', type: 'theme', price: 75, unlocked: false },
        { id: 'dark-emerald', name: 'Forest Emerald', type: 'theme', price: 75, unlocked: false },
        { id: 'light-warm', name: 'Warm Sepia', type: 'theme', price: 50, unlocked: false },
        { id: 'light-sage', name: 'Sage Garden', type: 'theme', price: 50, unlocked: false },
        { id: 'dark-academia', name: 'Dark Academia', type: 'theme', price: 400, unlocked: false },
        { id: 'dark-cyberpunk', name: 'Neon Cyberpunk', type: 'theme', price: 600, unlocked: false },
        { id: 'light-zen', name: 'Zen Minimalist', type: 'theme', price: 300, unlocked: false },
        { id: 'dark-obsidian', name: 'Obsidian Black', type: 'theme', price: 0, unlock_condition: 'Achievement: Night Scholar', unlocked: false },
        { id: 'light-monochrome', name: 'Monochrome Paper', type: 'theme', price: 0, unlock_condition: 'Achievement: Quiz Master', unlocked: false },
      ]
    } else {
      error.value = err.message || 'Wails backend bridge unavailable'
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchThemes()
})

async function buyItem(item) {
  if (buying.value) return
  buying.value = item.id
  try {
    const res = await unlockCosmeticItem(item.id, item.price)
    if (res && res.error) {
      alert(res.error)
    } else {
      await fetchThemes()
      equipTheme(item.id)
      window.dispatchEvent(new Event('gamification-updated'))
    }
  } catch (err) {
    alert(err.message || 'Purchase failed')
  } finally {
    buying.value = ''
  }
}

function equipTheme(themeId) {
  document.documentElement.setAttribute('data-theme', themeId)
  localStorage.setItem('app-theme', themeId)
  emit('theme-changed', themeId)
}
</script>

<style scoped>
.shop-section-hub {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.shop-header {
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

.shop-title {
  margin: 0 0 0.2rem;
  font-family: 'Manrope', sans-serif;
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--on-surface, #fff);
}

.shop-subtitle {
  margin: 0;
  font-size: 0.84rem;
  color: var(--muted-text, #94a3b8);
}

.coin-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: color-mix(in srgb, #f59e0b 15%, transparent);
  border: 1px solid var(--outline-variant);
  padding: 6px 14px;
  border-radius: 999px;
  font-weight: 800;
  font-size: 0.9rem;
  color: #f59e0b;
}

.themes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.25rem;
}

.shop-item-card {
  background: var(--surface-container-low, #1b1c23);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
  border-radius: 16px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-height: 180px;
  box-sizing: border-box;
  transition: border-color 0.2s ease, transform 0.2s ease, box-shadow 0.2s ease;
}

.shop-item-card:hover {
  border-color: color-mix(in srgb, var(--primary, #38bdf8) 40%, transparent);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.shop-item-card.active {
  border-color: var(--primary, #38bdf8);
  box-shadow: 0 0 16px color-mix(in srgb, var(--primary, #38bdf8) 25%, transparent);
}

.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.item-title {
  font-weight: 700;
  font-size: 1rem;
  color: var(--on-surface, #fff);
}

.shop-theme-preview {
  width: 100%;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin: 8px 0;
  border: 1px solid var(--outline-variant);
}

.swatch-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  box-shadow: 0 0 3px rgba(0, 0, 0, 0.15);
}

.badge {
  font-size: 0.76rem;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 6px;
  white-space: nowrap;
}

.active-badge {
  background: color-mix(in srgb, var(--primary, #38bdf8) 20%, transparent);
  color: var(--primary, #38bdf8);
  border: 1px solid var(--outline-variant);
}

.unlocked-badge {
  background: color-mix(in srgb, #10b981 15%, transparent);
  color: #10b981;
}

.price-badge {
  background: color-mix(in srgb, #f59e0b 15%, transparent);
  color: #f59e0b;
}

.condition-badge {
  background: var(--surface-container, rgba(255, 255, 255, 0.05));
  color: var(--muted-text, #94a3b8);
}

.unlock-desc {
  font-size: 0.8rem;
  color: var(--muted-text, #94a3b8);
  margin: 4px 0 10px;
  line-height: 1.4;
}

.item-footer {
  margin-top: auto;
  padding-top: 8px;
}

.buy-btn, .equip-btn {
  width: 100%;
  height: 40px;
  border-radius: 10px;
  font-weight: 700;
  font-size: 0.88rem;
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.buy-btn {
  background: #f59e0b;
  color: #111;
}

.buy-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.equip-btn {
  background: var(--surface-container, rgba(255, 255, 255, 0.08));
  color: var(--on-surface, #e2e8f0);
}

.equip-btn:hover:not(.active) {
  background: rgba(255, 255, 255, 0.12);
}

.equip-btn.active {
  background: var(--primary, #38bdf8);
  color: var(--on-primary, #09090b);
}

.locked-hint {
  font-size: 0.78rem;
  color: var(--muted-text, #94a3b8);
  font-style: italic;
  display: block;
  text-align: center;
}

.empty-state, .error-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--muted-text, #94a3b8);
}
</style>
