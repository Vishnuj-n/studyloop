<template>
  <Teleport to="body">
    <transition name="banner-slide">
      <aside
        v-if="rateLimitBanner.show"
        class="rate-limit-banner-card"
        role="alert"
        aria-live="assertive"
      >
        <div class="banner-top-row">
          <div class="banner-badge-title">
            <span class="banner-alert-icon">
              <BaseIcon name="alert-triangle" size="16" />
            </span>
            <div class="banner-heading">
              <h3 class="banner-title">{{ rateLimitBanner.title || 'AI Rate Limit Reached (HTTP 429)' }}</h3>
              <p class="banner-subtitle">
                Your AI provider exceeded its request or token quota. Use these quick fixes:
              </p>
            </div>
          </div>
          <button
            type="button"
            class="banner-close-btn"
            title="Dismiss notification"
            aria-label="Dismiss banner"
            @click="hideRateLimitBanner(false)"
          >
            <BaseIcon name="x" size="14" />
          </button>
        </div>

        <div class="banner-tips-grid">
          <div class="tip-item">
            <BaseIcon name="zap" size="13" class="tip-icon" />
            <span><strong>Prompt Compression:</strong> Shrink prompt sizes by 30–60% using local BERT token pruning.</span>
          </div>
          <div class="tip-item">
            <BaseIcon name="refresh-cw" size="13" class="tip-icon" />
            <span><strong>Heavy Model &amp; Pacing:</strong> Configure a dedicated Heavy model (or 45s cooldown pacing) for uninterrupted study sessions.</span>
          </div>
          <div class="tip-item">
            <BaseIcon name="coin" size="13" class="tip-icon" />
            <span><strong>Paid API Key:</strong> Add a paid provider key (e.g. OpenAI, OpenRouter, or paid Gemini) for higher token quotas without limits.</span>
          </div>
        </div>

        <div class="banner-actions-row">
          <div class="primary-actions">
            <button
              type="button"
              class="action-btn compression-btn"
              @click="goToExtensions"
            >
              <BaseIcon name="zap" size="13" />
              <span>Configure Compression</span>
            </button>
            <button
              type="button"
              class="action-btn settings-btn"
              @click="goToSettings('ai')"
            >
              <BaseIcon name="settings" size="13" />
              <span>AI Provider &amp; Model Swap</span>
            </button>
          </div>

          <div class="secondary-actions">
            <button
              type="button"
              class="dont-show-btn"
              title="Don't show this advisory again on future 429 errors"
              @click="hideRateLimitBanner(true)"
            >
              Don't show again
            </button>
          </div>
        </div>
      </aside>
    </transition>
  </Teleport>
</template>

<script setup>
import { useRouter } from 'vue-router'
import BaseIcon from './BaseIcon.vue'
import { useToast } from '../composables/useToast'

const router = useRouter()
const { rateLimitBanner, hideRateLimitBanner } = useToast()

function goToExtensions() {
  hideRateLimitBanner(false)
  router.push('/extensions')
}

function goToSettings(category) {
  hideRateLimitBanner(false)
  router.push(`/settings?category=${category}`)
}
</script>

<style scoped>
.rate-limit-banner-card {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  width: min(680px, calc(100vw - 32px));
  z-index: 10001;
  background: var(--surface-container-high, #1c1c1e);
  color: var(--on-surface, #f2f2f7);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.12));
  border-left: 4px solid #f59e0b;
  border-radius: 12px;
  padding: 16px 18px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  gap: 12px;
  backdrop-filter: blur(12px);
  pointer-events: auto;
}

.banner-top-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.banner-badge-title {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.banner-alert-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  flex-shrink: 0;
  margin-top: 1px;
}

.banner-heading {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.banner-title {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface, #ffffff);
  letter-spacing: -0.01em;
}

.banner-subtitle {
  margin: 0;
  font-size: 12.5px;
  color: var(--on-surface-variant, #a1a1aa);
  line-height: 1.4;
}

.banner-close-btn {
  background: transparent;
  border: none;
  color: var(--muted-text, #71717a);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.banner-close-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: var(--on-surface, #ffffff);
}

.banner-tips-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--surface-container, rgba(255, 255, 255, 0.03));
  border-radius: 8px;
  padding: 8px 12px;
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
}

.tip-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--on-surface-variant, #d4d4d8);
  line-height: 1.35;
}

.tip-icon {
  color: #f59e0b;
  flex-shrink: 0;
}

.banner-actions-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 4px;
}

.primary-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid var(--outline-variant);
}

.compression-btn {
  background: rgba(245, 158, 11, 0.15);
  color: #d97706;
  border-color: rgba(245, 158, 11, 0.3);
}

.compression-btn:hover {
  background: rgba(245, 158, 11, 0.25);
  border-color: #f59e0b;
}

.settings-btn {
  background: var(--surface-container-highest, rgba(255, 255, 255, 0.08));
  color: var(--on-surface, #f4f4f5);
  border-color: var(--outline-variant, rgba(255, 255, 255, 0.1));
}

.settings-btn:hover {
  background: var(--surface-container-high, rgba(255, 255, 255, 0.14));
}

.secondary-actions {
  display: flex;
  align-items: center;
}

.dont-show-btn {
  background: transparent;
  border: none;
  color: var(--muted-text, #71717a);
  font-size: 11.5px;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
  padding: 4px 6px;
  transition: color 0.15s ease;
}

.dont-show-btn:hover {
  color: var(--on-surface, #ffffff);
}

/* Transitions */
.banner-slide-enter-active,
.banner-slide-leave-active {
  transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.22s ease;
}

.banner-slide-enter-from {
  opacity: 0;
  transform: translate(-50%, -20px);
}

.banner-slide-leave-to {
  opacity: 0;
  transform: translate(-50%, -10px);
}
</style>
