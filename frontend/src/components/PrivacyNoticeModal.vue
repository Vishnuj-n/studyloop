<template>
  <Teleport to="body">
    <div v-if="visible" class="privacy-modal-overlay">
      <div class="privacy-modal" role="dialog" aria-labelledby="privacy-modal-title">
        <header class="privacy-modal-header">
          <div class="header-icon-badge">
            <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            </svg>
          </div>
          <h2 id="privacy-modal-title">Anonymous Usage Notice</h2>
        </header>

        <div class="privacy-modal-body">
          <p class="main-msg">
            Studyloop is 100% private and offline-capable. To help us maintain and improve the app across platforms, it sends a minimal anonymous startup ping (operating system and app version only).
          </p>
          <div class="privacy-highlights">
            <div class="highlight-item positive">
              <span class="status-icon">✓</span>
              <span><strong>Zero personal data:</strong> No files, notes, prompts, or study content ever leave your computer.</span>
            </div>
            <div class="highlight-item positive">
              <span class="status-icon">✓</span>
              <span><strong>Open Source:</strong> Completely verifiable privacy policies and code.</span>
            </div>
          </div>
          <p class="sub-msg">
            You can read our full privacy principles in <code>PRIVACY.md</code> on GitHub or review settings anytime.
          </p>
        </div>

        <footer class="privacy-modal-footer">
          <button type="button" class="btn-link" @click="openPrivacyDoc">
            Read PRIVACY.md ↗
          </button>
          <button type="button" class="btn-confirm" @click="handleAcknowledge">
            Got it
          </button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { openURLInBrowser } from '../services/appApi'

defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['close'])

function openPrivacyDoc() {
  openURLInBrowser('https://github.com/Vishnuj-n/studyloop/blob/main/PRIVACY.md')
}

function handleAcknowledge() {
  emit('close')
}
</script>

<style scoped>
.privacy-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(10, 15, 29, 0.45);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  z-index: 10001;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  animation: fadeIn 0.25s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.privacy-modal {
  background: color-mix(in srgb, var(--surface-container-lowest, #0f172a) 92%, transparent);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.12));
  border-radius: 20px;
  padding: 28px 32px;
  max-width: 480px;
  width: 100%;
  box-shadow: 0 20px 50px -10px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255, 255, 255, 0.05);
  display: flex;
  flex-direction: column;
  gap: 20px;
  animation: scaleUp 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes scaleUp {
  from { transform: scale(0.95); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.privacy-modal-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon-badge {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--primary, #6366f1) 15%, transparent);
  color: var(--primary, #6366f1);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.privacy-modal-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  font-family: 'Manrope', sans-serif;
  color: var(--on-surface, #ffffff);
}

.privacy-modal-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.main-msg {
  margin: 0;
  font-size: 14px;
  color: var(--on-surface, #f8fafc);
  line-height: 1.5;
}

.privacy-highlights {
  background: var(--surface-container-low, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.08));
  border-radius: 12px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.highlight-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  color: var(--on-surface-variant, #cbd5e1);
  line-height: 1.4;
}

.status-icon {
  color: #10b981;
  font-weight: bold;
}

.sub-msg {
  margin: 0;
  font-size: 12px;
  color: var(--muted-text, #94a3b8);
  line-height: 1.4;
}

.privacy-modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
}

.btn-link {
  background: transparent;
  border: none;
  color: var(--primary, #6366f1);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  padding: 6px 10px;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.btn-link:hover {
  background: color-mix(in srgb, var(--primary, #6366f1) 10%, transparent);
}

.btn-confirm {
  background: var(--primary, #4f46e5);
  color: var(--on-primary, #ffffff);
  border: none;
  border-radius: 10px;
  padding: 9px 20px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-confirm:hover {
  background: var(--primary-hover, #4338ca);
  transform: translateY(-1px);
}
</style>
