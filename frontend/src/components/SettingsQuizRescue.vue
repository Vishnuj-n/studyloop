<template>
  <article class="panel form-grid">
    <h2>Quiz Failure Rescue</h2>
    <p class="hint" style="margin-top: -10px; margin-bottom: 8px">
      Choose what happens when you fail a quiz. Customize the remediation track to match your study
      style.
    </p>

    <div class="form-group">
      <div class="strategy-options">
        <label
          class="strategy-option"
          :class="{ active: settings.default_remedial_strategy === 'CLASSIC' }"
        >
          <input
            v-model="settings.default_remedial_strategy"
            type="radio"
            value="CLASSIC"
            :disabled="disabled"
            style="cursor: pointer"
          />
          <div class="option-content">
            <span class="option-title">Classic Track</span>
            <span class="option-desc"
              >Reread first, then Socratic tutor if you fail again (dense text, sequential
              learning)</span
            >
          </div>
        </label>

        <label
          class="strategy-option"
          :class="{ active: settings.default_remedial_strategy === 'FAST' }"
        >
          <input
            v-model="settings.default_remedial_strategy"
            type="radio"
            value="FAST"
            :disabled="disabled"
            style="cursor: pointer"
          />
          <div class="option-content">
            <span class="option-title">Fast Track</span>
            <span class="option-desc"
              >Go directly to Socratic AI tutor (deeper encoding, conceptual topics)</span
            >
          </div>
        </label>
      </div>
    </div>

    <div class="form-group">
      <label>AI Tutor Style</label>
      <p class="hint" style="margin-bottom: 8px">
        Choose how the AI tutor interacts with you during concept rescue sessions.
      </p>
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
            <span class="option-desc">Asks leading questions to help you discover the answers yourself</span>
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
            <span class="option-desc">Direct explanations pointing out exactly where you went wrong</span>
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
            <span class="option-desc">Deep walkthroughs with intuitive analogies and clear examples</span>
          </div>
        </label>
      </div>
    </div>

    <SettingsToggle
      :model-value="settings.rag_enabled"
      :disabled="disabled"
      title="Enable Local AI Retrieval (RAG)"
      hint="Preloads local ONNX embeddings for context-rich Q&A. Unticking unloads RAG from memory instantly."
      @update:model-value="$emit('rag-toggle', $event)"
    />

    <div v-if="settings.rag_enabled" class="rag-sub-settings">
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
        hint="Allows general queries scoped to the selected notebook in the Tutor interface."
      />

      <SettingsToggle
        v-model="settings.rag_queue_study"
        :disabled="disabled"
        title="Enable Tutor in Queue Study Sessions"
        hint="Shows an optional Tutor panel inside active reading tasks."
      />
    </div>
  </article>
</template>

<script setup>
import SettingsToggle from './SettingsToggle.vue'

defineProps({
  settings: { type: Object, required: true },
  disabled: { type: Boolean, default: false },
})

defineEmits(['rag-toggle'])
</script>

<style scoped>
label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
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
  gap: 24px;
}

h2 {
  font-size: 20px;
  margin: 0 0 16px;
  font-weight: 700;
}

.panel {
  background: var(--surface-container-lowest);
  border-radius: 16px;
  padding: 28px;
  border: 1px solid var(--outline-variant);
  box-shadow: 0 4px 20px color-mix(in srgb, var(--on-surface) 3%, transparent);
}

.rag-sub-settings {
  margin-left: 28px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 8px;
}

.strategy-options {
  display: flex;
  gap: 16px;
  margin-top: 8px;
}

.strategy-option {
  flex: 1;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--surface-container-low);
  border: none;
  color: var(--on-surface);
}

.strategy-option:hover {
  background: var(--surface-container-lowest);
  box-shadow: 0 8px 16px color-mix(in srgb, var(--on-surface) 6%, transparent);
}

.strategy-option.active {
  background: var(--surface-container-lowest);
  box-shadow: 0 0 0 2px var(--primary);
}

.strategy-option input[type='radio'] {
  margin-top: 4px;
  accent-color: var(--primary);
  cursor: pointer;
}

.option-title {
  display: block;
  font-size: 1rem;
  font-weight: 600;
  color: var(--on-surface);
}

input[type='number'] {
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  padding: 12px 14px;
  font-size: 14px;
  font-family: inherit;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
  width: 100%;
  box-sizing: border-box;
}

input[type='number']:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 15%, transparent);
  outline: none;
}

.option-desc {
  display: block;
  font-size: 0.85rem;
  color: var(--muted-text);
  margin-top: 4px;
  line-height: 1.4;
}
</style>
