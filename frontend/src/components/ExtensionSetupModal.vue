<template>
  <div v-if="isOpen" class="modal-overlay" @click.self="handleOverlayClick">
    <div class="modal-card setup-card">
      <div class="modal-header">
        <div class="setup-header-title">
          <div v-if="status === 'running'" class="setup-icon-spinner">
            <div class="loading-spin-circle"></div>
          </div>
          <div v-else-if="status === 'success'" class="setup-icon-success">✓</div>
          <div v-else class="setup-icon-error">!</div>
          <h3>
            {{
              status === 'running'
                ? 'Configuring Extension'
                : status === 'success'
                ? 'Extension Ready'
                : 'Setup Failed'
            }}: {{ extension?.name || 'Python Tool' }}
          </h3>
        </div>
        <button v-if="status !== 'running'" class="close-modal-btn" @click="emitClose">✕</button>
      </div>

      <div class="modal-body">
        <p class="setup-desc">
          {{
            status === 'running'
              ? 'Setting up isolated Python virtual environment via uv and verifying dependencies...'
              : status === 'success'
              ? 'Extension dependencies and smoke test verified successfully!'
              : errorMessage
          }}
        </p>

        <!-- Progress Steps -->
        <div class="setup-steps-list">
          <div class="step-item" :class="getStepClass(1)">
            <span class="step-num">1</span>
            <span class="step-text">Python Runtime Environment</span>
          </div>
          <div class="step-item" :class="getStepClass(2)">
            <span class="step-num">2</span>
            <span class="step-text">Package Dependencies (requirements.txt)</span>
          </div>
          <div class="step-item" :class="getStepClass(3)">
            <span class="step-num">3</span>
            <span class="step-text">Extension Self-Test Probe</span>
          </div>
        </div>

        <div class="setup-logs-box">
          <div v-for="(log, idx) in logs" :key="idx" class="setup-log-line">
            {{ log }}
          </div>
          <div v-if="status === 'running'" class="setup-log-line log-pending">
            Running setup pipeline...
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button
          v-if="status === 'error'"
          class="modal-action-btn retry-btn"
          @click="startSetup"
        >
          Retry Setup
        </button>
        <button
          v-if="status !== 'running'"
          class="modal-close-btn"
          @click="emitClose"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { setupExtension } from '../services/appApi'

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false
  },
  extension: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'success', 'error'])

const status = ref('running') // 'running' | 'success' | 'error'
const logs = ref([])
const errorMessage = ref('')
const currentStep = ref(1)

function getStepClass(stepNum) {
  if (status.value === 'success') return 'completed'
  if (status.value === 'error' && currentStep.value === stepNum) return 'error'
  if (currentStep.value > stepNum) return 'completed'
  if (currentStep.value === stepNum) return 'active'
  return 'pending'
}

function handleOverlayClick() {
  if (status.value !== 'running') {
    emitClose()
  }
}

function emitClose() {
  emit('close')
}

async function startSetup() {
  if (!props.extension || !props.extension.id) return

  status.value = 'running'
  logs.value = ['Checking environment...']
  errorMessage.value = ''
  currentStep.value = 1

  try {
    const res = await setupExtension(props.extension.id)
    if (res && res.success) {
      logs.value = res.logs || ['Setup completed successfully.']
      currentStep.value = 3
      status.value = 'success'
      emit('success', props.extension)

      setTimeout(() => {
        if (props.isOpen && status.value === 'success') {
          emitClose()
        }
      }, 1200)
    } else {
      status.value = 'error'
      logs.value = res?.logs || []
      errorMessage.value = res?.error || 'Setup failed to complete.'
      emit('error', errorMessage.value)
    }
  } catch (err) {
    status.value = 'error'
    errorMessage.value = String(err)
    emit('error', errorMessage.value)
  }
}

watch(
  () => [props.isOpen, props.extension],
  ([isOpen, ext]) => {
    if (isOpen && ext) {
      startSetup()
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(45, 51, 56, 0.4);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24px;
}

.modal-card.setup-card {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  width: 100%;
  max-width: 560px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 40px rgba(45, 51, 56, 0.08);
}

.modal-header {
  padding: 20px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.setup-header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.setup-header-title h3 {
  font-family: 'Manrope', sans-serif;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--on-surface);
}

.setup-icon-spinner {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
}

.loading-spin-circle {
  width: 18px;
  height: 18px;
  border: 2.5px solid var(--surface-container);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.setup-icon-success {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #2e7d32;
  color: #ffffff;
  display: grid;
  place-items: center;
  font-size: 13px;
  font-weight: bold;
}

.setup-icon-error {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #c62828;
  color: #ffffff;
  display: grid;
  place-items: center;
  font-size: 14px;
  font-weight: bold;
}

.close-modal-btn {
  background: transparent;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--muted-text);
  padding: 4px;
}

.modal-body {
  padding: 0 24px 20px;
  overflow-y: auto;
  flex: 1;
}

.setup-desc {
  font-size: 13.5px;
  color: var(--muted-text);
  line-height: 1.5;
  margin: 0 0 16px 0;
}

.setup-steps-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.step-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12.5px;
  color: var(--muted-text);
}

.step-num {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--surface-container-low);
  display: grid;
  place-items: center;
  font-size: 11px;
  font-weight: 600;
}

.step-item.active {
  color: var(--primary);
  font-weight: 600;
}

.step-item.active .step-num {
  background: var(--primary);
  color: var(--on-primary);
}

.step-item.completed {
  color: var(--on-surface);
}

.step-item.completed .step-num {
  background: #2e7d32;
  color: #ffffff;
}

.setup-logs-box {
  background: var(--surface-container-low);
  border-radius: 12px;
  padding: 14px 16px;
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  max-height: 180px;
  overflow-y: auto;
  color: var(--on-surface);
}

.setup-log-line {
  margin-bottom: 3px;
}

.log-pending {
  color: var(--primary);
  font-style: italic;
}

.modal-footer {
  padding: 16px 24px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.modal-action-btn.retry-btn {
  padding: 8px 18px;
  border-radius: 10px;
  background: linear-gradient(15deg, var(--primary) 0%, var(--primary-dim) 100%);
  color: var(--on-primary);
  border: none;
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
}

.modal-close-btn {
  padding: 8px 18px;
  border-radius: 10px;
  background: var(--surface-container-low);
  border: none;
  color: var(--on-surface);
  font-weight: 600;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s ease;
}

.modal-close-btn:hover {
  background: var(--surface-container);
}
</style>
