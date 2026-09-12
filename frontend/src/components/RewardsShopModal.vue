<template>
  <Teleport to="body">
    <div v-if="visible" class="shop-modal-backdrop" @click.self="closeModal">
      <div class="shop-modal-card">
        <button class="close-btn" type="button" aria-label="Close" @click="closeModal">✕</button>

        <div class="modal-header">
          <div class="header-titles">
            <h2 class="modal-title">Rewards & Achievement Shop</h2>
            <p class="modal-subtitle">Spend coins on desk aesthetics or earn exclusive themes!</p>
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
            🎨 Workspace Themes
          </button>
          <button
            type="button"
            class="tab-btn"
            :class="{ active: activeTab === 'achievements' }"
            @click="activeTab = 'achievements'"
          >
            🏆 Achievements
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
              <div class="ach-icon">{{ ach.icon }}</div>
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
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  activeTheme: { type: String, default: 'dark-gruvbox' },
})

const emit = defineEmits(['close', 'theme-changed'])

const visible = ref(true)
const activeTab = ref('themes')
const store = ref(null)
const loading = ref(true)
const error = ref('')
const buying = ref('')

async function fetchStore() {
  loading.value = true
  error.value = ''
  try {
    if (window.go?.main?.App?.GetGamificationStore) {
      const res = await window.go.main.App.GetGamificationStore()
      if (res.error) {
        error.value = res.error
      } else {
        store.value = res.store
      }
    } else {
      // Mock fallback in dev mode if Wails bindings not present
      store.value = {
        profile: { coins: 120 },
        themes: [
          { id: 'dark-gruvbox', name: 'Gruvbox Dark', type: 'theme', price: 0, unlocked: true },
          { id: 'light-classic', name: 'Light Classic', type: 'theme', price: 0, unlocked: true },
          { id: 'dark-indigo', name: 'Deep Indigo', type: 'theme', price: 75, unlocked: false },
          { id: 'dark-emerald', name: 'Forest Emerald', type: 'theme', price: 75, unlocked: false },
          { id: 'light-warm', name: 'Warm Sepia', type: 'theme', price: 50, unlocked: false },
          { id: 'light-sage', name: 'Sage Garden', type: 'theme', price: 50, unlocked: false },
          { id: 'dark-obsidian', name: 'Obsidian Black', type: 'theme', price: 0, unlocked: false, unlock_condition: 'Achievement: Night Scholar' },
          { id: 'light-monochrome', name: 'Monochrome Paper', type: 'theme', price: 0, unlocked: false, unlock_condition: 'Achievement: Quiz Master' },
        ],
        achievements: [
          { id: 'first_step', title: 'First Steps', description: 'Complete 1 reading session', icon: '📖', current_value: 1, target_value: 1, completed: true, reward_coins: 20 },
          { id: 'night_scholar', title: 'Night Scholar', description: 'Complete 5 reading sessions', icon: '🦉', current_value: 2, target_value: 5, completed: false, reward_coins: 50, reward_item: 'dark-obsidian' },
          { id: 'quiz_master', title: 'Quiz Master', description: 'Pass 5 quizzes', icon: '🎯', current_value: 3, target_value: 5, completed: false, reward_coins: 75, reward_item: 'light-monochrome' },
          { id: 'memory_monk', title: 'Memory Monk', description: 'Review 25 flashcards', icon: '🧠', current_value: 12, target_value: 25, completed: false, reward_coins: 50 },
        ],
      }
    }
  } catch (err) {
    error.value = err.message || 'Failed to load store'
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
    if (window.go?.main?.App?.UnlockCosmeticItem) {
      const res = await window.go.main.App.UnlockCosmeticItem(item.id, item.price)
      if (res.error) {
        alert(res.error)
      } else {
        await fetchStore()
        equipTheme(item.id)
      }
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

function closeModal() {
  visible.value = false
  emit('close')
}
</script>

<style scoped>
.shop-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.68);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
}

.shop-modal-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  width: 100%;
  max-width: 620px;
  max-height: 85vh;
  padding: 1.75rem;
  display: flex;
  flex-direction: column;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  color: var(--on-surface);
  overflow: hidden;
}

.close-btn {
  position: absolute;
  top: 16px;
  right: 18px;
  background: none;
  border: none;
  color: var(--muted-text);
  font-size: 1.25rem;
  cursor: pointer;
  border-radius: 8px;
  padding: 4px 8px;
}

.close-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
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
  gap: 8px;
  margin-bottom: 1.25rem;
  border-bottom: 1px solid var(--outline-variant);
  padding-bottom: 8px;
}

.tab-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  font-weight: 700;
  font-size: 0.9rem;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
}

.tab-btn.active {
  background: var(--surface-container);
  color: var(--on-surface);
}

.tab-content {
  overflow-y: auto;
  flex: 1;
  padding-right: 4px;
}

.themes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 12px;
}

.shop-item-card {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.item-title {
  font-weight: 700;
  font-size: 0.95rem;
}

.badge {
  font-size: 0.75rem;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 6px;
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
  font-size: 0.78rem;
  color: var(--muted-text);
  margin: 0;
}

.item-footer {
  margin-top: auto;
}

.buy-btn, .equip-btn {
  width: 100%;
  padding: 8px;
  border-radius: 8px;
  font-weight: 700;
  font-size: 0.85rem;
  cursor: pointer;
  border: none;
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

.ach-icon {
  font-size: 2rem;
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
