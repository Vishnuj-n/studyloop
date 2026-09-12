<template>
  <article class="panel form-grid">
    <div class="panel-header">
      <h2>Workspace Aesthetics</h2>
      <button type="button" class="shop-trigger-btn" @click="showShopModal = true">
        🛒 Rewards & Theme Shop
      </button>
    </div>

    <div v-if="error" class="error-msg">{{ error }}</div>

    <div class="form-group">
      <label>Aesthetic Theme</label>
      <div class="theme-grid">
        <button
          v-for="t in themes"
          :key="t.id"
          type="button"
          class="theme-card"
          :class="{ active: settings.theme === t.id, locked: !isUnlocked(t.id) }"
          :disabled="disabled"
          @click="handleThemeClick(t)"
        >
          <div class="theme-preview" :style="{ background: t.bg }">
            <span class="preview-dot" :style="{ background: t.primary }"></span>
            <span class="preview-dot" :style="{ background: t.surface }"></span>
            <span v-if="!isUnlocked(t.id)" class="lock-overlay">🔒</span>
          </div>
          <span class="theme-label">
            {{ t.label }}
            <span v-if="!isUnlocked(t.id)" class="lock-tag">Locked</span>
          </span>
        </button>
      </div>
      <p class="hint">
        Select a visual theme. Default themes are unlocked. Additional themes can be unlocked in the Rewards Shop using Coins or by completing study achievements!
      </p>
    </div>

    <RewardsShopModal
      v-if="showShopModal"
      :active-theme="settings.theme"
      @close="onShopClose"
      @theme-changed="onThemeChanged"
    />
  </article>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import RewardsShopModal from './RewardsShopModal.vue'
import { getGamificationState } from '../services/appApi'

const props = defineProps({
  settings: { type: Object, required: true },
  disabled: { type: Boolean, default: false },
})

const showShopModal = ref(false)
const unlockedCosmetics = ref(['dark-gruvbox', 'light-classic', 'light-warm', 'dark-indigo'])
const error = ref('')

const themes = [
  { id: 'dark-gruvbox', label: 'Gruvbox Dark', bg: '#1d2021', primary: '#d79921', surface: '#282828' },
  { id: 'dark-obsidian', label: 'Obsidian Black', bg: '#09090b', primary: '#f4f4f5', surface: '#18181b' },
  { id: 'dark-indigo', label: 'Deep Indigo', bg: '#0b0d16', primary: '#6366f1', surface: '#171a2b' },
  { id: 'dark-emerald', label: 'Forest Emerald', bg: '#0a120d', primary: '#10b981', surface: '#152219' },
  { id: 'light-monochrome', label: 'Monochrome Paper', bg: '#f8f8f9', primary: '#18181b', surface: '#e8e8ec' },
  { id: 'light-warm', label: 'Warm Sepia', bg: '#fdfaf6', primary: '#c27d38', surface: '#f3eae1' },
  { id: 'light-sage', label: 'Sage Garden', bg: '#f4f7f4', primary: '#2e7d32', surface: '#e2ebe2' },
  { id: 'light-classic', label: 'Light Classic', bg: '#f9f9fb', primary: '#005bc1', surface: '#ebeef2' },
]

onMounted(async () => {
  await loadUnlockedCosmetics()
})

async function loadUnlockedCosmetics() {
  error.value = ''
  try {
    const res = await getGamificationState()
    if (res && res.error) {
      unlockedCosmetics.value = []
      error.value = res.error
    } else if (res?.profile?.unlocked_cosmetics_json) {
      unlockedCosmetics.value = JSON.parse(res.profile.unlocked_cosmetics_json)
    }
  } catch (err) {
    unlockedCosmetics.value = []
    error.value = err.message || 'Failed to load unlocked cosmetics'
    console.error('Failed to load unlocked cosmetics:', err)
  }
}

function isUnlocked(themeId) {
  return unlockedCosmetics.value.includes(themeId)
}

function handleThemeClick(theme) {
  if (props.disabled) return
  if (!isUnlocked(theme.id)) {
    showShopModal.value = true
    return
  }
  props.settings.theme = theme.id
  document.documentElement.setAttribute('data-theme', theme.id)
}

function onThemeChanged(themeId) {
  props.settings.theme = themeId
  loadUnlockedCosmetics()
}

function onShopClose() {
  showShopModal.value = false
  loadUnlockedCosmetics()
}
</script>

<style scoped>
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
}

.hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--muted-text);
  line-height: 1.4;
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

h2 {
  font-size: 20px;
  margin: 0;
  font-weight: 700;
}

.shop-trigger-btn {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  font-size: 0.85rem;
  font-weight: 700;
  padding: 6px 14px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.shop-trigger-btn:hover {
  background: var(--surface-container-highest);
  border-color: var(--primary);
}

.panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  gap: 16px;
  margin-top: 8px;
}

.theme-card {
  background: var(--surface-container-low);
  border: none;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  width: 100%;
  color: var(--on-surface);
  position: relative;
}

.theme-card.locked {
  opacity: 0.75;
}

.theme-card:hover:not(:disabled) {
  background: var(--surface-container-lowest);
  box-shadow: 0 8px 16px color-mix(in srgb, var(--on-surface) 6%, transparent);
}

.theme-card.active {
  background: var(--surface-container-lowest);
  box-shadow: 0 0 0 2px var(--primary);
}

.theme-card:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.theme-preview {
  width: 100%;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  position: relative;
}

.lock-overlay {
  position: absolute;
  font-size: 1.2rem;
  background: rgba(0, 0, 0, 0.45);
  inset: 0;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(2px);
}

.preview-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.theme-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted-text);
  transition: color 0.2s ease, font-weight 0.2s ease;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.lock-tag {
  font-size: 0.7rem;
  color: #f59e0b;
  font-weight: 700;
}

.theme-card.active .theme-label {
  color: var(--on-surface);
  font-weight: 700;
}

.error-msg {
  color: var(--danger, #ef4444);
  font-size: 0.85rem;
  font-weight: 600;
}
</style>
