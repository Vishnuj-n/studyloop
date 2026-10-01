<template>
  <div class="settings-rag-container">
    <!-- Local AI Retrieval (RAG) -->
    <article class="panel form-grid">
      <div class="panel-header-row">
        <div>
          <h2>Local AI Retrieval (RAG)</h2>
          <p class="hint">
            Preload local ONNX embedding models and SQLite vector index for instant, context-grounded Q&amp;A.
          </p>
        </div>
        <div class="panel-header-actions">
          <div
            v-if="isSettingUpRag"
            class="rag-status-badge status-setting-up"
            title="Local embedding engine initialization is in progress"
          >
            <span class="status-dot pulse"></span>
            <span>Setting Up{{ ragStatus ? ` (${ragStatus})` : '...' }}</span>
          </div>
          <div
            v-else-if="settings.rag_enabled"
            class="rag-status-badge status-ready"
            title="ONNX vector embeddings loaded and active in RAM"
          >
            <span class="status-dot active"></span>
            <span>Active in RAM / Ready</span>
          </div>
          <div
            v-else
            class="rag-status-badge status-dormant"
            title="RAG embeddings are currently disabled"
          >
            <span class="status-dot"></span>
            <span>Dormant / Off</span>
          </div>

          <button
            v-if="settings.rag_enabled"
            type="button"
            class="rag-setup-trigger-btn"
            :disabled="disabled || isSettingUpRag"
            @click="$emit('open-setup')"
          >
            <BaseIcon name="refresh-cw" size="12" />
            <span>Re-index / Setup</span>
          </button>
        </div>
      </div>

      <SettingsToggle
        :model-value="settings.rag_enabled"
        :disabled="disabled"
        title="Enable Local AI Retrieval (RAG)"
        hint="Preloads local embedding models for deep contextual Q&A across your textbooks. Unticking unloads embeddings from memory."
        @update:model-value="$emit('rag-toggle', $event)"
      />

      <!-- Sub-settings for RAG Scopes -->
      <div v-if="settings.rag_enabled" class="rag-sub-settings animate-fade-in">
        <h4 class="sub-heading">Retrieval Access Scopes</h4>
        <SettingsToggle
          v-model="settings.rag_notebook_chapter"
          :disabled="disabled"
          title="Enable Tutor from Notebook Chapters"
          hint="Allows accessing Socratic RAG directly from notebook chapter details."
        />

        <SettingsToggle
          v-model="settings.rag_entire_notebook"
          :disabled="disabled"
          title="Enable RAG for Entire Book"
          hint="Allows general queries scoped to the selected textbook in the Tutor interface."
        />

        <SettingsToggle
          v-model="settings.rag_queue_study"
          :disabled="disabled"
          title="Enable Tutor in Queue Study Sessions"
          hint="Shows an optional Tutor panel inside active reading study sessions."
        />
      </div>
    </article>

    <!-- AI Tutor Persona & Style -->
    <article class="panel form-grid">
      <div class="panel-header-row">
        <div>
          <h2>AI Tutor Interaction Style</h2>
          <p class="hint">
            Customize how the Socratic AI tutor responds and guides you during concept discussions and remedial study.
          </p>
        </div>
        <span class="global-badge"><BaseIcon name="globe" size="12" /> Persona</span>
      </div>

      <div class="form-group">
        <div class="strategy-options">
          <label
            class="strategy-option"
            :class="{ active: (settings.tutor_style || 'socratic') === 'socratic' }"
          >
            <input
              v-model="settings.tutor_style"
              type="radio"
              value="socratic"
              :disabled="disabled"
              style="cursor: pointer"
            />
            <div class="option-content">
              <span class="option-title">Socratic Guide</span>
              <span class="option-desc">Asks leading questions to help you discover the answers and build intuition.</span>
            </div>
          </label>

          <label
            class="strategy-option"
            :class="{ active: settings.tutor_style === 'direct' }"
          >
            <input
              v-model="settings.tutor_style"
              type="radio"
              value="direct"
              :disabled="disabled"
              style="cursor: pointer"
            />
            <div class="option-content">
              <span class="option-title">Direct &amp; Concise</span>
              <span class="option-desc">Clear, direct explanations pinpointing exactly where misconceptions lie.</span>
            </div>
          </label>

          <label
            class="strategy-option"
            :class="{ active: settings.tutor_style === 'detailed' }"
          >
            <input
              v-model="settings.tutor_style"
              type="radio"
              value="detailed"
              :disabled="disabled"
              style="cursor: pointer"
            />
            <div class="option-content">
              <span class="option-title">Step-by-Step</span>
              <span class="option-desc">Deep breakdowns with comprehensive analogies, examples, and derivations.</span>
            </div>
          </label>
        </div>
      </div>
    </article>
  </div>
</template>

<script setup>
import BaseIcon from './BaseIcon.vue'
import SettingsToggle from './SettingsToggle.vue'

defineProps({
  settings: { type: Object, required: true },
  disabled: { type: Boolean, default: false },
  ragStatus: { type: String, default: '' },
  isSettingUpRag: { type: Boolean, default: false },
})

defineEmits(['rag-toggle', 'open-setup'])
</script>

<style scoped>
.settings-rag-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

.panel-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

h2 {
  font-size: 20px;
  margin: 0;
  font-weight: 700;
  font-family: 'Manrope', sans-serif;
}

.sub-heading {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted-text);
  margin: 0 0 6px;
}

.hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--muted-text);
  line-height: 1.4;
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.rag-sub-settings {
  margin-left: 20px;
  padding-left: 16px;
  border-left: 2px solid var(--outline-variant);
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-top: 4px;
}

.panel-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.rag-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--outline-variant);
  white-space: nowrap;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--muted-text);
}

.status-ready {
  color: #10b981;
  background: color-mix(in srgb, #10b981 12%, transparent);
  border-color: color-mix(in srgb, #10b981 30%, transparent);
}

.status-ready .status-dot {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
}

.status-setting-up {
  color: #f59e0b;
  background: color-mix(in srgb, #f59e0b 12%, transparent);
  border-color: color-mix(in srgb, #f59e0b 30%, transparent);
}

.status-setting-up .status-dot {
  background: #f59e0b;
  box-shadow: 0 0 8px rgba(245, 158, 11, 0.6);
}

.status-dormant {
  color: var(--muted-text);
  background: var(--surface-container-high);
}

.status-dot.pulse {
  animation: pulseDot 1.4s infinite ease-in-out;
}

@keyframes pulseDot {
  0%, 100% {
    transform: scale(0.8);
    opacity: 0.6;
  }
  50% {
    transform: scale(1.2);
    opacity: 1;
  }
}

.rag-setup-trigger-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-high);
  color: var(--on-surface);
  cursor: pointer;
  transition: all 0.15s ease;
}

.rag-setup-trigger-btn:hover:not(:disabled) {
  background: var(--surface-container-highest);
  border-color: var(--primary);
  color: var(--primary);
}

.rag-setup-trigger-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.global-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border: 1px solid var(--outline-variant);
  padding: 3px 8px;
  border-radius: 999px;
  white-space: nowrap;
}

.strategy-options {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-top: 4px;
}

.strategy-option {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
}

.strategy-option:hover {
  background: var(--surface-container-lowest);
  box-shadow: 0 4px 12px color-mix(in srgb, var(--on-surface) 6%, transparent);
}

.strategy-option.active {
  background: var(--surface-container-lowest);
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 20%, transparent);
}

.strategy-option input[type='radio'] {
  margin-top: 3px;
  accent-color: var(--primary);
  cursor: pointer;
}

.option-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.option-title {
  display: block;
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface);
}

.option-desc {
  display: block;
  font-size: 12px;
  color: var(--muted-text);
  line-height: 1.35;
}

.animate-fade-in {
  animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 860px) {
  .strategy-options {
    grid-template-columns: 1fr;
  }
}
</style>
