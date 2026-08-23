<template>
  <div v-if="show" class="modal-backdrop" @click.self="close">
    <div class="modal-card">
      <div class="modal-header">
        <div class="header-left">
          <span class="modal-icon">🎥</span>
          <div>
            <h3 class="modal-title">Import YouTube Lecture</h3>
            <p class="modal-subtitle">Generate structured study chapters and notes from video</p>
          </div>
        </div>
        <button type="button" class="modal-close" @click="close" aria-label="Close modal">×</button>
      </div>

      <div class="modal-body">
        <div class="form-group">
          <label for="yt-url-input" class="input-label">YouTube Video URL</label>
          <div class="input-wrapper">
            <input
              id="yt-url-input"
              ref="inputRef"
              v-model="url"
              type="text"
              placeholder="https://www.youtube.com/watch?v=..."
              class="url-input"
              :disabled="isLoading"
              @keydown.enter="handleSubmit"
            />
            <button
              v-if="url"
              type="button"
              class="clear-btn"
              :disabled="isLoading"
              @click="url = ''"
            >
              ✕
            </button>
          </div>
          <p class="field-hint">
            Supports YouTube videos with English or auto-generated captions.
          </p>
        </div>

        <div v-if="error" class="error-box">
          {{ error }}
        </div>

        <div v-if="isLoading" class="loading-box">
          <div class="spinner"></div>
          <div class="loading-info">
            <span class="loading-title">Ingesting video lecture...</span>
            <span v-if="statusMessage" class="loading-subtitle">{{ statusMessage }}</span>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button type="button" class="btn-cancel" :disabled="isLoading" @click="close">
          Cancel
        </button>
        <button
          type="button"
          class="btn-submit"
          :disabled="!url.trim() || isLoading"
          @click="handleSubmit"
        >
          <span v-if="isLoading">Ingesting...</span>
          <span v-else>Import Lecture</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  isLoading: { type: Boolean, default: false },
  statusMessage: { type: String, default: '' },
  error: { type: String, default: '' },
})

const emit = defineEmits(['close', 'submit'])

const url = ref('')
const inputRef = ref(null)

watch(
  () => props.show,
  (isOpen) => {
    if (isOpen) {
      nextTick(() => {
        inputRef.value?.focus()
      })
    } else {
      url.value = ''
    }
  }
)

function close() {
  if (props.isLoading) return
  emit('close')
}

function handleSubmit() {
  const trimmed = url.value.trim()
  if (!trimmed || props.isLoading) return
  emit('submit', trimmed)
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 16px;
  animation: fadeIn 0.15s ease;
}

.modal-card {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  width: 100%;
  max-width: 520px;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.35);
  overflow: hidden;
  animation: slideUp 0.15s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--outline-variant);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.modal-icon {
  font-size: 24px;
}

.modal-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--on-surface);
}

.modal-subtitle {
  margin: 2px 0 0;
  font-size: 12.5px;
  color: var(--muted-text);
}

.modal-close {
  background: transparent;
  border: none;
  font-size: 22px;
  line-height: 1;
  color: var(--muted-text);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  transition: all 0.15s;
}

.modal-close:hover {
  color: var(--on-surface);
  background: var(--surface-container);
}

.modal-body {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface);
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.url-input {
  width: 100%;
  padding: 10px 36px 10px 14px;
  border-radius: 10px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-lowest);
  color: var(--on-surface);
  font-size: 13.5px;
  font-family: inherit;
  transition: border-color 0.15s;
}

.url-input:focus {
  outline: none;
  border-color: var(--primary);
}

.clear-btn {
  position: absolute;
  right: 10px;
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  font-size: 12px;
}

.clear-btn:hover {
  color: var(--on-surface);
}

.field-hint {
  margin: 0;
  font-size: 12px;
  color: var(--muted-text);
  opacity: 0.85;
}

.error-box {
  padding: 10px 14px;
  background: rgba(239, 68, 68, 0.12);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  border-radius: 8px;
  font-size: 13px;
}

.loading-box {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--surface-container-lowest);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--outline-variant);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.loading-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface);
}

.loading-subtitle {
  font-size: 11.5px;
  color: var(--muted-text);
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--outline-variant);
  background: var(--surface-container-lowest);
}

.btn-cancel {
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant);
  background: transparent;
  color: var(--on-surface);
  font-size: 13.5px;
  font-family: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-cancel:hover:not(:disabled) {
  background: var(--surface-container);
}

.btn-submit {
  padding: 8px 18px;
  border-radius: 8px;
  border: none;
  background: var(--primary);
  color: var(--on-primary);
  font-size: 13.5px;
  font-family: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-submit:hover:not(:disabled) {
  filter: brightness(1.08);
}

.btn-submit:active:not(:disabled) {
  transform: scale(0.97);
}

.btn-submit:disabled,
.btn-cancel:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
