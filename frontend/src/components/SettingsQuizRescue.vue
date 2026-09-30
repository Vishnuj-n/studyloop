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
  </article>
</template>

<script setup>
defineProps({
  settings: { type: Object, required: true },
  disabled: { type: Boolean, default: false },
})
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
