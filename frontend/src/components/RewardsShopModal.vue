<template>
  <div class="card floating-card shop-section-card">
    <div class="modal-header">
      <div class="header-titles">
        <h2 class="modal-title">Study Shop & Progression</h2>
        <p class="modal-subtitle">Spend study coins on workspace themes and track your achievements!</p>
      </div>

      <div v-if="store" class="coin-pill">
        <span class="coin-icon">🪙</span>
        <span class="coin-balance">{{ store.profile?.coins || 0 }} Coins</span>
      </div>
    </div>

        <!-- Navigation Tabs -->
        <div class="tab-nav">
          <button
            type="button"
            class="tab-btn"
            :class="{ active: activeTab === 'themes' }"
            @click="activeTab = 'themes'"
          >
            Themes
          </button>
          <button
            type="button"
            class="tab-btn"
            :class="{ active: activeTab === 'achievements' }"
            @click="activeTab = 'achievements'"
          >
            Achievements
          </button>
        </div>

        <!-- Loading / Error States -->
        <div v-if="loading" class="empty-state">Loading Store...</div>
        <div v-else-if="error" class="error-state">{{ error }}</div>

        <div v-else-if="store" class="tab-content">
          <!-- Themes Tab -->
          <div v-if="activeTab === 'themes'" class="themes-grid">
            <div
              v-for="item in store.themes"
              :key="item.id"
              class="shop-item-card"
              :class="{ unlocked: item.unlocked }"
            >
              <div class="item-header">
                <span class="item-title">{{ item.name }}</span>
                <span v-if="item.unlocked" class="badge unlocked-badge">✓ Unlocked</span>
                <span v-else-if="item.price > 0" class="badge price-badge">🪙 {{ item.price }}</span>
                <span v-else class="badge condition-badge">🔒 Locked</span>
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
                  :disabled="buying === item.id || (store.profile?.coins || 0) < item.price"
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

          <!-- Achievements Tab -->
          <div v-else-if="activeTab === 'achievements'" class="achievements-list">
            <div
              v-for="ach in store.achievements"
              :key="ach.id"
              class="achievement-card"
              :class="{ completed: ach.completed }"
            >
              <div class="ach-badge-icon">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                </svg>
              </div>
              <div class="ach-info">
                <h4 class="ach-title">{{ ach.title }}</h4>
                <p class="ach-desc">{{ ach.description }}</p>

                <!-- Progress Bar -->
                <div class="progress-track">
                  <div
                    class="progress-fill"
                    :style="{ width: Math.min(100, Math.round((ach.current_value / ach.target_value) * 100)) + '%' }"
                  ></div>
                </div>
                <span class="progress-label">{{ ach.current_value }} / {{ ach.target_value }}</span>
              </div>

              <div class="ach-reward">
                <span v-if="ach.completed" class="ach-claimed">✓ Completed</span>
                <div v-else class="reward-tag">
                  <span v-if="ach.reward_coins">+{{ ach.reward_coins }} 🪙</span>
                  <span v-if="ach.reward_item" class="reward-item-label">Unlock Theme</span>
                </div>
              </div>
            </div>
          </div>
        </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  getGamificationStore,
  unlockCosmeticItem,
} from '../services/appApi'

defineProps({
  activeTheme: { type: String, default: 'dark-gruvbox' },
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

const activeTab = ref('themes')
const store = ref(null)
const loading = ref(true)
const error = ref('')
const buying = ref('')

async function fetchStore() {
  loading.value = true
  error.value = ''
  try {
    const res = await getGamificationStore()
    if (res && res.error) {
      error.value = res.error
    } else if (res && res.store) {
      store.value = res.store
    } else if (res) {
      store.value = res
    }
  } catch (err) {
    if (import.meta.env.DEV || import.meta.env.MODE === 'test') {
      store.value = {
        profile: { coins: 350 },
        themes: [
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
        ],
        achievements: [],
      }
    } else {
      error.value = err.message || 'Wails backend bridge unavailable'
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchStore()
})

async function buyItem(item) {
  if (buying.value) return
  buying.value = item.id
  try {
    const res = await unlockCosmeticItem(item.id, item.price)
    if (res && res.error) {
      alert(res.error)
    } else {
      await fetchStore()
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
  emit('theme-changed', themeId)
}
</script>

<style scoped>
.shop-section-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  width: 100%;
  padding: 1.75rem;
  display: flex;
  flex-direction: column;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.12);
  color: var(--on-surface);
}


.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.25rem;
  padding-right: 2rem;
}

.modal-title {
  font-size: 1.35rem;
  font-weight: 800;
  margin: 0 0 0.2rem;
}

.modal-subtitle {
  font-size: 0.82rem;
  color: var(--muted-text);
  margin: 0;
}

.coin-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: color-mix(in srgb, #f59e0b 15%, transparent);
  border: 1px solid color-mix(in srgb, #f59e0b 30%, transparent);
  padding: 6px 14px;
  border-radius: 999px;
  font-weight: 800;
  font-size: 0.9rem;
  color: #f59e0b;
}

.tab-nav {
  display: flex;
  gap: 6px;
  margin-bottom: 1.5rem;
  background: var(--surface-container-highest, rgba(255, 255, 255, 0.04));
  padding: 5px;
  border-radius: 12px;
  border: 1px solid var(--outline-variant);
  overflow-x: auto;
}

.tab-btn {
  flex: 1;
  background: transparent;
  border: none;
  color: var(--muted-text);
  font-weight: 700;
  font-size: 0.88rem;
  padding: 10px 14px;
  border-radius: 8px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.tab-btn:hover {
  color: var(--on-surface);
  background: rgba(255, 255, 255, 0.05);
}

.tab-btn.active {
  background: var(--surface-container-high, #34383b);
  color: #f59e0b;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
  border: 1px solid color-mix(in srgb, #f59e0b 30%, transparent);
}

.tab-content {
  min-height: 340px;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
}

.themes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
  width: 100%;
}

.shop-item-card {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-height: 190px;
  box-sizing: border-box;
  transition: border-color 0.2s ease, transform 0.2s ease, box-shadow 0.2s ease;
}

.shop-item-card:hover {
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
}

.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.item-title {
  font-weight: 700;
  font-size: 1.05rem;
  line-height: 1.3;
}

.shop-theme-preview {
  width: 100%;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin: 10px 0;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.swatch-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  box-shadow: 0 0 3px rgba(0, 0, 0, 0.3);
}

.badge {
  font-size: 0.8rem;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 8px;
  white-space: nowrap;
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
  background: var(--surface-container-highest);
  color: var(--muted-text);
}

.unlock-desc {
  font-size: 0.85rem;
  color: var(--muted-text);
  margin: 4px 0 12px;
  line-height: 1.45;
}

.item-footer {
  margin-top: auto;
  padding-top: 8px;
}

.buy-btn, .equip-btn {
  width: 100%;
  height: 42px;
  border-radius: 10px;
  font-weight: 700;
  font-size: 0.9rem;
  cursor: pointer;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.buy-btn {
  background: #f59e0b;
  color: var(--on-surface);
}

.buy-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.equip-btn {
  background: var(--surface-container-highest);
  color: var(--on-surface);
}

.equip-btn.active {
  background: var(--primary);
  color: var(--on-primary);
}

.locked-hint {
  font-size: 0.78rem;
  color: var(--muted-text);
  font-style: italic;
}

.achievements-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.achievement-card {
  display: flex;
  align-items: center;
  gap: 14px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 12px 16px;
}

.achievement-card.completed {
  border-color: color-mix(in srgb, #10b981 40%, transparent);
}

.ach-badge-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: color-mix(in srgb, #f59e0b 15%, transparent);
  border: 1px solid color-mix(in srgb, #f59e0b 30%, transparent);
  color: #f59e0b;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.achievement-card.completed .ach-badge-icon {
  background: color-mix(in srgb, #10b981 15%, transparent);
  border-color: color-mix(in srgb, #10b981 30%, transparent);
  color: #10b981;
}

.ach-info {
  flex: 1;
}

.ach-title {
  margin: 0 0 2px;
  font-size: 0.95rem;
  font-weight: 700;
}

.ach-desc {
  margin: 0 0 8px;
  font-size: 0.8rem;
  color: var(--muted-text);
}

.progress-track {
  height: 6px;
  background: var(--surface-container-highest);
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 4px;
}

.progress-fill {
  height: 100%;
  background: var(--primary);
  transition: width 0.3s ease;
}

.progress-label {
  font-size: 0.72rem;
  color: var(--muted-text);
  font-weight: 600;
}

.ach-claimed {
  font-size: 0.8rem;
  font-weight: 700;
  color: #10b981;
}

.reward-tag {
  font-size: 0.8rem;
  font-weight: 700;
  color: #f59e0b;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.reward-item-label {
  font-size: 0.72rem;
  color: var(--primary);
}

.empty-state, .error-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--muted-text);
}
</style>
