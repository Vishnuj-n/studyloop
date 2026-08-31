<template>
  <div class="settings-extensions-container">
    <!-- AI Audio Overview -->
    <article class="panel form-grid">
      <h2>AI Audio Overview (Edge-TTS)</h2>
      <div class="form-group">
        <label for="audio-voice">Voice Persona</label>
        <select
          id="audio-voice"
          v-model="audioVoice"
          class="setting-select"
          :disabled="disabled"
        >
          <option value="en-US-ChristopherNeural">Christopher (Narrator Male)</option>
          <option value="en-US-JennyNeural">Jenny (Engaging Female)</option>
          <option value="en-US-GuyNeural">Guy (Casual Male)</option>
          <option value="en-GB-SoniaNeural">Sonia (British Female)</option>
          <option value="en-US-AriaNeural">Aria (Expressive Female)</option>
          <option value="en-US-EricNeural">Eric (Dynamic Male)</option>
        </select>
        <p class="hint">Edge-TTS neural voice used for podcast summaries.</p>
      </div>

      <div class="form-group">
        <label for="audio-speed">Speech Pace</label>
        <select
          id="audio-speed"
          v-model.number="audioSpeed"
          class="setting-select"
          :disabled="disabled"
        >
          <option :value="0.85">0.85x (Slow)</option>
          <option :value="1.0">1.0x (Normal)</option>
          <option :value="1.25">1.25x (Study Pace)</option>
          <option :value="1.5">1.5x (Fast)</option>
          <option :value="1.75">1.75x (Speed Study)</option>
        </select>
        <p class="hint">Default playback tempo for new audio overviews.</p>
      </div>
    </article>

    <!-- Text Simplifier -->
    <article class="panel form-grid">
      <h2>Text Simplifier</h2>
      <div class="form-group">
        <label for="simplifier-level">Target Comprehension Level</label>
        <select
          id="simplifier-level"
          v-model="simplifierLevel"
          class="setting-select"
          :disabled="disabled"
        >
          <option value="eli15">High School (ELI15 with analogies)</option>
          <option value="eli10">Middle School (ELI10 simple phrasing)</option>
          <option value="bullet">Executive Bullet Summary (Key takeaways)</option>
          <option value="academic">Academic Precision (Rigorous terms intact)</option>
        </select>
        <p class="hint">Rewrites dense textbooks to match your target comprehension level.</p>
      </div>
    </article>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useExtensions } from '../composables/useExtensions'

defineProps({
  disabled: {
    type: Boolean,
    default: false,
  },
})

const { extensionConfig, setExtensionSetting } = useExtensions()

const audioVoice = computed({
  get: () => extensionConfig.value?.audio_overview?.voice || 'en-US-ChristopherNeural',
  set: (val) => setExtensionSetting('audio_overview', 'voice', val),
})

const audioSpeed = computed({
  get: () => extensionConfig.value?.audio_overview?.speed || 1.0,
  set: (val) => setExtensionSetting('audio_overview', 'speed', Number(val)),
})

const simplifierLevel = computed({
  get: () => extensionConfig.value?.text_simplifier?.level || 'eli15',
  set: (val) => setExtensionSetting('text_simplifier', 'level', val),
})
</script>

<style scoped>
.settings-extensions-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.panel {
  border: 1px solid var(--outline-variant);
  border-radius: 16px;
  background: var(--surface-container-lowest, transparent);
  padding: 24px;
}

h2 {
  font-size: 20px;
  margin: 0 0 16px;
  font-weight: 700;
  color: var(--on-surface);
}

.form-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
}

label {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
  margin-bottom: 6px;
}

select {
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  background: var(--surface-container-low);
  color: var(--on-surface);
  padding: 12px 14px;
  font-size: 14px;
  font-family: inherit;
  width: 100%;
  box-sizing: border-box;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 15%, transparent);
  outline: none;
}

.hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--muted-text);
  line-height: 1.4;
}
</style>

