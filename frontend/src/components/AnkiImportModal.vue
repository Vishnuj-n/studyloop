<template>
  <div v-if="show" class="modal-backdrop" @click.self="close">
    <div class="modal-card">
      <div class="modal-header">
        <div class="header-left">
          <span class="modal-icon">🎴</span>
          <div>
            <h3 class="modal-title">Import Anki Deck</h3>
            <p class="modal-subtitle">Import cards (.apkg, .colpkg) directly into your study schedule</p>
          </div>
        </div>
        <button type="button" class="modal-close" aria-label="Close modal" @click="close">×</button>
      </div>

      <div class="modal-body">
        <!-- File Picker Group -->
        <div class="form-group">
          <label class="input-label">Anki Package File</label>
          <div class="file-picker-row">
            <div class="file-path-display" :class="{ 'has-file': !!selectedFilePath }">
              {{ selectedFileName || 'No file selected (.apkg, .colpkg)' }}
            </div>
            <button
              type="button"
              class="btn-browse"
              :disabled="isLoading"
              @click="handleBrowseFile"
            >
              Browse
            </button>
          </div>
        </div>

        <!-- Target Destination Selection -->
        <div class="form-group">
          <label class="input-label">Destination</label>
          <div class="radio-options">
            <label class="radio-option">
              <input
                v-model="targetMode"
                type="radio"
                value="standalone"
                :disabled="isLoading"
              />
              <div class="radio-label-group">
                <span class="radio-title">Create Standalone Notebook</span>
                <span class="radio-desc">Creates a dedicated flashcard-only notebook</span>
              </div>
            </label>

            <label class="radio-option" :class="{ disabled: !availableNotebooks.length }">
              <input
                v-model="targetMode"
                type="radio"
                value="existing"
                :disabled="isLoading || !availableNotebooks.length"
              />
              <div class="radio-label-group">
                <span class="radio-title">Add to Existing Notebook</span>
                <span class="radio-desc">Attaches cards to an existing study notebook</span>
              </div>
            </label>
          </div>

          <!-- Existing Notebook Dropdown -->
          <div v-if="targetMode === 'existing'" class="existing-notebook-picker">
            <label for="anki-nb-select" class="sub-label">Select Notebook</label>
            <select
              id="anki-nb-select"
              v-model="selectedNotebookID"
              class="select-input"
              :disabled="isLoading"
            >
              <option value="" disabled>— Select Notebook —</option>
              <option v-for="nb in availableNotebooks" :key="nb.id" :value="nb.id">
                {{ nb.title }}
              </option>
            </select>
          </div>
        </div>

        <div v-if="error" class="error-box">
          {{ error }}
        </div>

        <div v-if="isLoading" class="loading-box">
          <div class="spinner"></div>
          <div class="loading-info">
            <span class="loading-title">Parsing and importing Anki cards...</span>
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
          :disabled="!canSubmit || isLoading"
          @click="handleSubmit"
        >
          <span v-if="isLoading">Importing...</span>
          <span v-else>Import Deck</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { selectAnkiFile } from '../services/appApi'

const props = defineProps({
  show: { type: Boolean, default: false },
  isLoading: { type: Boolean, default: false },
  statusMessage: { type: String, default: '' },
  error: { type: String, default: '' },
  availableNotebooks: { type: Array, default: () => [] },
})

const emit = defineEmits(['close', 'submit'])

const selectedFilePath = ref('')
const selectedFileName = ref('')
const targetMode = ref('standalone')
const selectedNotebookID = ref('')

const canSubmit = computed(() => {
  if (!selectedFilePath.value) return false
  if (targetMode.value === 'existing' && !selectedNotebookID.value) return false
  return true
})

watch(
  () => props.show,
  (isOpen) => {
    if (!isOpen) {
      selectedFilePath.value = ''
      selectedFileName.value = ''
      targetMode.value = 'standalone'
      selectedNotebookID.value = ''
    }
  }
)

async function handleBrowseFile() {
  try {
    const res = await selectAnkiFile()
    if (res && res.file_path) {
      selectedFilePath.value = res.file_path
      selectedFileName.value = res.file_name || res.file_path
    }
  } catch (err) {
    console.error('Failed to select Anki file:', err)
  }
}

function close() {
  if (props.isLoading) return
  emit('close')
}

function handleSubmit() {
  if (!canSubmit.value || props.isLoading) return
  emit('submit', {
    filePath: selectedFilePath.value,
    targetNotebookID: targetMode.value === 'existing' ? selectedNotebookID.value : '',
  })
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.75);
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
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.12);
  overflow: hidden;
  animation: slideUp 0.15s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from { transform: translateY(8px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 24px;
  border-bottom: 1px solid var(--outline-variant);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.modal-icon {
  font-size: 24px;
  line-height: 1;
}

.modal-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--on-surface);
}

.modal-subtitle {
  margin: 2px 0 0 0;
  font-size: 0.8rem;
  color: var(--on-surface-variant);
}

.modal-close {
  background: transparent;
  border: none;
  font-size: 1.4rem;
  line-height: 1;
  color: var(--on-surface-variant);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  transition: all 0.15s ease;
}

.modal-close:hover {
  background: var(--surface-container);
  color: var(--on-surface);
}

.modal-body {
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-label {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--on-surface-variant);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.file-picker-row {
  display: flex;
  gap: 10px;
}

.file-path-display {
  flex: 1;
  padding: 10px 14px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  font-size: 0.88rem;
  color: var(--on-surface-variant);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-path-display.has-file {
  color: var(--on-surface);
  font-weight: 500;
}

.btn-browse {
  padding: 0 16px;
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  color: var(--on-surface);
  font-size: 0.88rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-browse:hover:not(:disabled) {
  background: var(--surface-container-highest);
}

.radio-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.radio-option {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  cursor: pointer;
  transition: border-color 0.15s ease;
}

.radio-option:hover {
  border-color: var(--primary);
}

.radio-option.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.radio-label-group {
  display: flex;
  flex-direction: column;
}

.radio-title {
  font-size: 0.88rem;
  font-weight: 600;
  color: var(--on-surface);
}

.radio-desc {
  font-size: 0.75rem;
  color: var(--on-surface-variant);
}

.existing-notebook-picker {
  margin-top: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sub-label {
  font-size: 0.75rem;
  color: var(--on-surface-variant);
}

.select-input {
  padding: 10px 12px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  font-size: 0.88rem;
  color: var(--on-surface);
}

.error-box {
  padding: 12px 14px;
  background: var(--error-container);
  color: var(--on-error-container);
  border-radius: 8px;
  font-size: 0.84rem;
}

.loading-box {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  background: var(--surface-container);
  border-radius: 10px;
}

.spinner {
  width: 22px;
  height: 22px;
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
}

.loading-title {
  font-size: 0.88rem;
  font-weight: 500;
  color: var(--on-surface);
}

.loading-subtitle {
  font-size: 0.75rem;
  color: var(--on-surface-variant);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 24px;
  border-top: 1px solid var(--outline-variant);
}

.btn-cancel {
  padding: 8px 16px;
  background: transparent;
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  font-size: 0.88rem;
  color: var(--on-surface);
  cursor: pointer;
}

.btn-submit {
  padding: 8px 18px;
  background: var(--primary);
  color: var(--on-primary);
  border: none;
  border-radius: 8px;
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s ease;
}

.btn-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
