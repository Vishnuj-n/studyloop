<template>
  <div class="settings-extensions-container">
    <!-- AI Audio Overview -->
    <article class="panel form-grid">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">
        <h2 style="margin: 0;">AI Audio Overview (Edge-TTS)</h2>
        <span class="global-badge"><BaseIcon name="globe" size="12" /> Global</span>
      </div>
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

    <!-- Prompt & Context Compressor -->
    <article class="panel form-grid">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">
        <h2 style="margin: 0;">Prompt &amp; Context Compressor (LLMLingua-2)</h2>
        <span class="global-badge"><BaseIcon name="globe" size="12" /> Global</span>
      </div>
      <div class="form-group">
        <label for="compressor-mode">Compression Operating Mode</label>
        <select
          id="compressor-mode"
          v-model="compressorMode"
          class="setting-select"
          :disabled="disabled"
        >
          <option value="OVER_LIMIT">Dynamic Budget (Default - Compresses only when exceeding session/model limit)</option>
          <option value="ALWAYS">Always Compress (Max efficiency - Compresses all chunks)</option>
          <option value="DISABLED">Disabled (Bypass compression - Full raw text)</option>
        </select>
        <p v-if="compressorMode === 'OVER_LIMIT'" class="hint">
           <strong>Dynamic Budget:</strong> Automatically evaluates token counts and triggers LLMLingua-2 compression only if notes exceed your session budget or 70% of LLM max context.
        </p>
        <p v-else-if="compressorMode === 'ALWAYS'" class="hint">
           <strong>Always Compress:</strong> Unconditionally runs background BERT compression on all study chunks (≥ 25 words) down to the target retention rate.
        </p>
        <p v-else class="hint">
           <strong>Disabled:</strong> Background token compression is disabled. Raw notes and textbook text are passed directly to LLM prompts.
        </p>
      </div>

      <div v-if="compressorMode !== 'DISABLED'" class="form-group">
        <label for="compressor-rate">Target Token Retention Rate</label>
        <select
          id="compressor-rate"
          v-model.number="compressorRate"
          class="setting-select"
          :disabled="disabled"
        >
          <option :value="0.80">80% Retention (Recommended - ~20% token savings, full nuance preserved)</option>
          <option :value="0.70">70% Retention (~30% token savings, balanced)</option>
          <option :value="0.60">60% Retention (~40% token savings, high pruning)</option>
          <option :value="0.50">50% Retention (~50% token savings, maximum compactness)</option>
        </select>
        <p class="hint">Target token retention ratio. LLMLingua-2 removes redundant syntax while preserving core entities and logic.</p>
      </div>
    </article>

    <!-- Text Simplifier -->
    <article class="panel form-grid">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">
        <h2 style="margin: 0;">Text Simplifier</h2>
        <span class="global-badge"><BaseIcon name="globe" size="12" /> Global</span>
      </div>
      <div class="form-group">
        <label for="simplifier-level">Target Comprehension Level</label>
        <select
          id="simplifier-level"
          v-model="simplifierLevel"
          class="setting-select"
          :disabled="disabled"
        >
          <option value="eli5">ELI5 (Explain Like I'm 5 - Ultra simple, fun analogies &amp; basic concepts)</option>
          <option value="very_simple">Very Simple (Everyday language, short sentences &amp; analogies)</option>
          <option value="simple">Simple (Clear, straightforward language with core terms)</option>
          <option value="academic">Academic (Precise terminology, definitions &amp; distinctions)</option>
          <option value="summary">Summary (Essential ideas, arguments &amp; key conclusions)</option>
        </select>
        <p class="hint">Rewrites dense textbooks to match your target comprehension level.</p>
      </div>
    </article>

    <!-- YouTube & Video Ingestion -->
    <article class="panel form-grid">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px;">
        <h2 style="margin: 0;">YouTube Ingestion &amp; Video Playback</h2>
        <span class="global-badge"><BaseIcon name="globe" size="12" /> Global</span>
      </div>

      <div class="form-group checkbox-group">
        <label class="checkbox-label">
          <input
            v-model="youtubeAutoDownload"
            type="checkbox"
            :disabled="disabled"
          />
          <span>Auto-download video for offline study</span>
        </label>
        <p class="hint">Streams immediately on import, then caches a local copy in the background for zero-buffering offline playback.</p>
      </div>

      <div v-if="youtubeAutoDownload" class="form-group">
        <label for="youtube-quality">Video Download Quality</label>
        <select
          id="youtube-quality"
          v-model="youtubeQuality"
          class="setting-select"
          :disabled="disabled"
        >
          <option value="max">Max / Best (Up to 4K / 2160p)</option>
          <option value="1440p">1440p (2K Quad HD)</option>
          <option value="1080p">1080p (Full HD)</option>
          <option value="720p">720p (Recommended - Fast download &amp; compact size)</option>
          <option value="480p">480p (Low bandwidth)</option>
        </select>
      </div>

      <div v-if="configError" class="error-banner">
        {{ configError }}
      </div>
    </article>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { useExtensions } from '../composables/useExtensions'

defineProps({
  disabled: {
    type: Boolean,
    default: false,
  },
})

const { extensionConfig, configError, setExtensionSetting } = useExtensions()

const audioVoice = computed({
  get: () => extensionConfig.value?.audio_overview?.voice || 'en-US-ChristopherNeural',
  set: (val) => setExtensionSetting('audio_overview', 'voice', val),
})

const audioSpeed = computed({
  get: () => extensionConfig.value?.audio_overview?.speed || 1.0,
  set: (val) => setExtensionSetting('audio_overview', 'speed', Number(val)),
})

const simplifierLevel = computed({
  get: () => extensionConfig.value?.text_simplifier?.level || 'simple',
  set: (val) => setExtensionSetting('text_simplifier', 'level', val),
})

const compressorMode = computed({
  get: () => extensionConfig.value?.prompt_compressor?.mode || 'OVER_LIMIT',
  set: (val) => setExtensionSetting('prompt_compressor', 'mode', val),
})

const compressorRate = computed({
  get: () => extensionConfig.value?.prompt_compressor?.rate ?? 0.80,
  set: (val) => setExtensionSetting('prompt_compressor', 'rate', Number(val)),
})

const youtubeAutoDownload = computed({
  get: () => Boolean(extensionConfig.value?.youtube?.auto_download),
  set: (val) => {
    console.log('[SettingsExtensions] Setting youtube.auto_download:', val)
    return setExtensionSetting('youtube', 'auto_download', Boolean(val))
  },
})

const youtubeQuality = computed({
  get: () => extensionConfig.value?.youtube?.download_quality || '720p',
  set: (val) => {
    console.log('[SettingsExtensions] Setting youtube.download_quality:', val)
    return setExtensionSetting('youtube', 'download_quality', val)
  },
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

.checkbox-group {
  margin-top: 4px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: var(--on-surface);
  user-select: none;
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
  accent-color: var(--primary);
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

.error-banner {
  color: #ef4444;
  font-size: 13px;
  font-weight: 600;
  padding: 8px 12px;
  border-radius: 8px;
  background: color-mix(in srgb, #ef4444 10%, transparent);
  border: 1px solid var(--outline-variant);
}
</style>

