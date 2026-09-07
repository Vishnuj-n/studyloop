<template>
  <Teleport to="body">
    <div v-if="visible" class="freeze-modal-backdrop" @click.self="closeModal">
      <div class="freeze-modal-card">
        <div class="modal-ambient-glow"></div>
        <button class="close-btn" type="button" aria-label="Close" @click="closeModal">✕</button>

        <div class="modal-header">
          <span class="shield-pill">SHIELD ACQUIRED</span>
          <h2 class="modal-title">Streak Freeze Equipped!</h2>
          <p class="modal-subtitle">Your study streak is now actively protected.</p>
        </div>

        <div class="shield-display-area">
          <div class="shield-pedestal">
            <span class="shield-icon">🛡️</span>
          </div>
        </div>

        <!-- How It Works Explanation -->
        <div class="info-box">
          <h4 class="info-title">How Streak Freezes Work:</h4>
          <ul class="info-list">
            <li>
              <strong>100% Passive Protection:</strong> You don't need to manually click or activate anything. As long as you own a freeze, it sits in your inventory as an insurance shield.
            </li>
            <li>
              <strong>Missed Day Safety Net:</strong> If life gets busy and you miss studying for a day, 1 freeze is automatically consumed so your streak doesn't reset to zero.
            </li>
            <li>
              <strong>Shield Indicator:</strong> When protected, your streak icon displays a <strong>🛡️ Shield</strong> until today's study tasks are completed.
            </li>
          </ul>
        </div>

        <!-- Inventory Stats -->
        <div class="transaction-summary">
          <div class="summary-item cost">
            <span class="summary-icon">🪙</span>
            <span class="summary-text">-50 Coins</span>
          </div>
          <div class="summary-divider"></div>
          <div class="summary-item balance">
            <span class="summary-icon">🛡️</span>
            <span class="summary-text">{{ totalFreezes }} Owned</span>
          </div>
        </div>

        <div class="modal-footer">
          <button class="action-btn" type="button" @click="closeModal">
            Got It, Keep Studying
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { playCorrectChime } from '../utils/audioJuice'

defineProps({
  totalFreezes: {
    type: Number,
    default: 1,
  },
})

const emit = defineEmits(['close'])

const visible = ref(true)

onMounted(() => {
  playCorrectChime()
})

function closeModal() {
  visible.value = false
  emit('close')
}
</script>

<style scoped>
.freeze-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.68);
  backdrop-filter: blur(8px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.freeze-modal-card {
  position: relative;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 20px;
  width: 100%;
  max-width: 440px;
  padding: 2rem 1.75rem;
  text-align: center;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.15), 0 0 35px rgba(56, 189, 248, 0.1);
  color: var(--on-surface);
  overflow: hidden;
  animation: card-appear 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes card-appear {
  from {
    opacity: 0;
    transform: scale(0.92) translateY(12px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.modal-ambient-glow {
  position: absolute;
  top: -70px;
  left: 50%;
  transform: translateX(-50%);
  width: 250px;
  height: 250px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(56, 189, 248, 0.22) 0%, transparent 70%);
  pointer-events: none;
}

.close-btn {
  position: absolute;
  top: 14px;
  right: 16px;
  background: none;
  border: none;
  color: var(--muted-text);
  font-size: 1.25rem;
  cursor: pointer;
  border-radius: 8px;
  padding: 4px 8px;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.close-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}

.shield-pill {
  display: inline-block;
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  padding: 0.3rem 0.85rem;
  border-radius: 999px;
  margin-bottom: 0.75rem;
  border: 1px solid var(--outline-variant);
  background: color-mix(in srgb, rgba(56, 189, 248, 0.12) 100%, transparent);
  color: #38bdf8;
}

.modal-title {
  font-family: 'Manrope', sans-serif;
  font-size: 1.45rem;
  font-weight: 800;
  margin: 0 0 0.35rem;
  color: var(--on-surface);
}

.modal-subtitle {
  font-size: 0.85rem;
  color: var(--muted-text);
  margin: 0;
}

.shield-display-area {
  padding: 1.25rem 0 0.85rem;
}

.shield-pedestal {
  width: 84px;
  height: 84px;
  border-radius: 22px;
  background: radial-gradient(circle, rgba(56, 189, 248, 0.2) 0%, var(--surface-container) 80%);
  border: 1px solid var(--outline-variant);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12), 0 0 20px rgba(56, 189, 248, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto;
  animation: float-shield 3s ease-in-out infinite alternate;
}

@keyframes float-shield {
  from { transform: translateY(0); }
  to { transform: translateY(-6px); }
}

.shield-icon {
  font-size: 3rem;
  filter: drop-shadow(0 0 12px rgba(56, 189, 248, 0.8));
}

.info-box {
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 1rem 1.15rem;
  text-align: left;
  margin: 0.75rem 0 1.25rem;
}

.info-title {
  margin: 0 0 0.5rem;
  font-size: 0.82rem;
  font-weight: 700;
  color: #38bdf8;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-list {
  margin: 0;
  padding-left: 1.1rem;
  font-size: 0.8rem;
  line-height: 1.45;
  color: var(--on-surface);
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.info-list strong {
  color: var(--on-surface);
}

.transaction-summary {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1.25rem;
  padding: 0.65rem 1rem;
  background: var(--surface-container-highest);
  border-radius: 10px;
  margin-bottom: 1.25rem;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.88rem;
  font-weight: 700;
  font-family: 'Manrope', sans-serif;
}

.summary-item.cost .summary-text {
  color: #f59e0b;
}

.summary-item.balance .summary-text {
  color: #38bdf8;
}

.summary-divider {
  width: 1px;
  height: 18px;
  background: var(--outline-variant);
}

.modal-footer {
  width: 100%;
}

.action-btn {
  width: 100%;
  padding: 0.85rem;
  font-weight: 700;
  font-size: 0.95rem;
  border-radius: 12px;
  cursor: pointer;
  border: none;
  background: var(--primary);
  color: var(--on-primary);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--primary) 35%, transparent);
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s ease;
}

.action-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--primary) 50%, transparent);
}
</style>
