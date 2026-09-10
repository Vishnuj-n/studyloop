<template>
  <Teleport to="body">
    <div class="toast-stack">
      <transition name="toast-fade">
        <div
          v-if="toast.show"
          class="app-toast-card"
          :class="toast.type === 'notice' ? 'is-notice' : 'is-error'"
          role="alert"
        >
          <div class="toast-glow"></div>
          <div class="toast-icon-badge">
            <svg
              v-if="toast.type === 'notice'"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
            <svg
              v-else
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
          </div>

          <div class="toast-body">
            <div class="toast-title-row">
              <span class="toast-title">{{ toast.title || (toast.type === 'notice' ? 'Notice' : 'Error') }}</span>
              <button
                type="button"
                class="toast-close-btn"
                aria-label="Dismiss"
                @click="hideToast"
              >
                ✕
              </button>
            </div>
            <p class="toast-msg">{{ toast.message }}</p>
          </div>
        </div>
      </transition>
    </div>
  </Teleport>
</template>

<script setup>
import { useToast } from '../composables/useToast'

const { toast, hideToast } = useToast()
</script>

<style scoped>
.toast-stack {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: 10px;
  pointer-events: none;
}

.toast-stack > * {
  pointer-events: auto;
}

.app-toast-card {
  position: relative;
  max-width: 380px;
  width: calc(100vw - 48px);
  padding: 12px 14px;
  background: var(--surface-container-low, #1e1e24);
  border: 1px solid var(--outline-variant);
  border-radius: 14px;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.25), 0 0 16px rgba(0, 0, 0, 0.08);
  color: var(--on-surface, #e2e8f0);
  display: flex;
  align-items: flex-start;
  gap: 12px;
  overflow: hidden;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.app-toast-card:hover {
  transform: translateY(-2px);
}

.toast-glow {
  position: absolute;
  top: -24px;
  left: -24px;
  width: 90px;
  height: 90px;
  border-radius: 50%;
  pointer-events: none;
  opacity: 0.7;
}

.app-toast-card.is-notice .toast-glow {
  background: radial-gradient(circle, rgba(16, 185, 129, 0.35) 0%, transparent 70%);
}

.app-toast-card.is-error .toast-glow {
  background: radial-gradient(circle, rgba(239, 68, 68, 0.35) 0%, transparent 70%);
}

.toast-icon-badge {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-container, rgba(255, 255, 255, 0.05));
  border: 1px solid var(--outline-variant);
}

.app-toast-card.is-notice .toast-icon-badge {
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.08);
}

.app-toast-card.is-error .toast-icon-badge {
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.08);
}

.toast-body {
  flex: 1;
  min-width: 0;
}

.toast-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 2px;
}

.toast-title {
  font-weight: 700;
  font-size: 13px;
  letter-spacing: 0.02em;
  color: var(--on-surface, #ffffff);
}

.toast-close-btn {
  background: none;
  border: none;
  color: var(--muted-text, #94a3b8);
  font-size: 12px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
  line-height: 1;
  transition: color 0.15s, background-color 0.15s;
}

.toast-close-btn:hover {
  color: #ffffff;
  background: rgba(255, 255, 255, 0.08);
}

.toast-msg {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.45;
  color: var(--on-surface-variant, #94a3b8);
  word-break: break-word;
}

.toast-fade-enter-active,
.toast-fade-leave-active {
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-fade-enter-from,
.toast-fade-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.96);
}
</style>
