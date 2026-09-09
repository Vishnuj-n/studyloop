<template>
  <article class="panel form-grid">
    <h2>Workspace Aesthetics</h2>
    <div class="form-group">
      <label>Aesthetic Theme</label>
      <div class="theme-grid">
        <button
          v-for="t in themes"
          :key="t.id"
          type="button"
          class="theme-card"
          :class="{ active: settings.theme === t.id }"
          :disabled="disabled"
          @click="selectTheme(t.id)"
        >
          <div class="theme-preview" :style="{ background: t.bg }">
            <span class="preview-dot" :style="{ background: t.primary }"></span>
            <span class="preview-dot" :style="{ background: t.surface }"></span>
          </div>
          <span class="theme-label">{{ t.label }}</span>
        </button>
      </div>
      <p class="hint">
        Select a visual theme. Changing themes alters the colors of your study desk instantly.
      </p>
    </div>
  </article>
</template>

<script setup>
const props = defineProps({
  settings: { type: Object, required: true },
  disabled: { type: Boolean, default: false },
})

const themes = [
  { id: 'light-classic', label: 'Light Classic', bg: '#f9f9fb', primary: '#005bc1', surface: '#ebeef2' },
  { id: 'light-warm', label: 'Warm Sepia', bg: '#fdfaf6', primary: '#c27d38', surface: '#f3eae1' },
  { id: 'light-sage', label: 'Sage Garden', bg: '#f4f7f4', primary: '#2e7d32', surface: '#e2ebe2' },
  { id: 'dark-gruvbox', label: 'Gruvbox Dark', bg: '#1d2021', primary: '#d79921', surface: '#282828' },
  { id: 'dark-indigo', label: 'Deep Indigo', bg: '#0b0d16', primary: '#6366f1', surface: '#171a2b' },
  { id: 'dark-emerald', label: 'Forest Emerald', bg: '#0a120d', primary: '#10b981', surface: '#152219' },
]

function selectTheme(themeId) {
  if (props.disabled) return
  props.settings.theme = themeId
  document.documentElement.setAttribute('data-theme', themeId)
}
</script>

<style scoped>
label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
}

.hint {
  margin: 8px 0 0;
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

.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  gap: 16px;
  margin-top: 8px;
}

.theme-card {
  background: var(--surface-container-low);
  border: none;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  width: 100%;
  color: var(--on-surface);
}

.theme-card:hover:not(:disabled) {
  background: var(--surface-container-lowest);
  box-shadow: 0 8px 16px color-mix(in srgb, var(--on-surface) 6%, transparent);
}

.theme-card.active {
  background: var(--surface-container-lowest);
  box-shadow: 0 0 0 2px var(--primary);
}

.theme-card:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.theme-preview {
  width: 100%;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.preview-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.theme-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted-text);
  transition: color 0.2s ease, font-weight 0.2s ease;
}

.theme-card.active .theme-label {
  color: var(--on-surface);
  font-weight: 700;
}
</style>
