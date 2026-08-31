<template>
  <div class="audio-overview-bar" role="region" aria-label="Audio Overview Player">
    <div class="audio-bar-inner">
      <!-- Left: Status / Controls -->
      <div class="audio-controls">
        <button
          class="play-btn"
          :disabled="isLoading || (!isPlaying && chunks.length === 0)"
          :title="isPlaying ? 'Pause' : 'Play'"
          @click="togglePlay"
        >
          <span v-if="isLoading" class="spinner-icon">⏳</span>
          <span v-else-if="isPlaying">⏸</span>
          <span v-else>▶</span>
        </button>

        <button
          class="nav-btn"
          :disabled="currentIndex <= 0 || chunks.length === 0"
          title="Previous sentence"
          @click="prevChunk"
        >
          ⏮
        </button>

        <button
          class="nav-btn"
          :disabled="currentIndex >= chunks.length - 1 || chunks.length === 0"
          title="Next sentence"
          @click="nextChunk"
        >
          ⏭
        </button>

        <div class="progress-indicator">
          <span class="chunk-badge">
            {{ chunks.length > 0 ? `Sentence ${currentIndex + 1} of ${totalChunks || chunks.length}` : 'Starting...' }}
          </span>
        </div>
      </div>

      <!-- Center: Subtitle / Transcript -->
      <div class="audio-transcript">
        <p v-if="errorMessage" class="error-text">
          ⚠️ {{ errorMessage }}
        </p>
        <p v-else-if="isLoading && chunks.length === 0" class="buffering-text">
          <span class="pulse-dot">●</span> Generating fast audio briefing with Neural TTS...
        </p>
        <p v-else class="caption-text">
          "{{ currentChunkText }}"
        </p>
      </div>

      <!-- Right: Voice, Speed & Close -->
      <div class="audio-options">
        <select
          v-model="selectedVoice"
          class="voice-select"
          :disabled="isLoading && chunks.length === 0"
          title="Voice selection"
          @change="restartWithVoice"
        >
          <option value="en-US-ChristopherNeural">Christopher (Engaging Male)</option>
          <option value="en-US-GuyNeural">Guy (Conversational Male)</option>
          <option value="en-US-JennyNeural">Jenny (Clear Female)</option>
          <option value="en-US-AriaNeural">Aria (Expressive Female)</option>
        </select>

        <button class="speed-btn" title="Playback speed" @click="cycleSpeed">
          {{ playbackRate }}x
        </button>

        <button class="close-btn" title="Close Audio Overview" @click="handleClose">
          ✕
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { startTopicAudioOverview, stopTopicAudioOverview } from '../services/appApi'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useExtensions } from '../composables/useExtensions'

const props = defineProps({
  topicId: {
    type: String,
    required: true,
  },
  notebookId: {
    type: String,
    default: '',
  },
  topicTitle: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['close'])

const { getExtensionSetting } = useExtensions()

const chunks = ref([])
const currentIndex = ref(0)
const isPlaying = ref(false)
const isLoading = ref(true)
const isFinished = ref(false)
const totalChunks = ref(0)
const errorMessage = ref('')
const selectedVoice = ref(getExtensionSetting('audio_overview', 'voice', 'en-US-ChristopherNeural'))
const playbackRate = ref(Number(getExtensionSetting('audio_overview', 'speed', 1.0)))
const activeGenerationId = ref('')

let audio = null

function showError(msg) {
  errorMessage.value = msg || 'Audio overview generation failed'
  isLoading.value = false
  if (audio) {
    audio.pause()
  }
  isPlaying.value = false
}

const currentChunkText = computed(() => {
  if (chunks.value.length === 0 || currentIndex.value >= chunks.value.length) {
    return ''
  }
  return chunks.value[currentIndex.value].text
})

function initAudio() {
  if (!audio) {
    audio = new Audio()
    audio.playbackRate = playbackRate.value

    audio.onended = () => {
      if (currentIndex.value + 1 < chunks.value.length) {
        currentIndex.value++
        playChunk(currentIndex.value)
      } else if (isFinished.value) {
        isPlaying.value = false
      } else {
        // Still generating upcoming chunks
        isLoading.value = true
      }
    }

    audio.onerror = (e) => {
      // Ignore errors caused by resetting/clearing src or uninitialized state
      if (!audio || !audio.getAttribute('src')) {
        return
      }
      console.warn('[AudioOverview] Audio element error:', e)
      showError('Playback error encountered on current audio chunk.')
    }
  }
}

function playChunk(index) {
  if (!audio || index < 0 || index >= chunks.value.length) return
  const chunk = chunks.value[index]
  if (!chunk || !chunk.audio_base64) return

  audio.src = `data:audio/mp3;base64,${chunk.audio_base64}`
  audio.playbackRate = playbackRate.value
  isLoading.value = false
  audio
    .play()
    .then(() => {
      isPlaying.value = true
    })
    .catch((err) => {
      console.warn('[AudioOverview] Autoplay blocked or interrupted:', err)
      isPlaying.value = false
    })
}

function togglePlay() {
  if (!audio) return
  if (isPlaying.value) {
    audio.pause()
    isPlaying.value = false
  } else {
    if (chunks.value.length > 0) {
      if (!audio.src || audio.ended) {
        playChunk(currentIndex.value)
      } else {
        audio.play()
        isPlaying.value = true
      }
    }
  }
}

function prevChunk() {
  if (currentIndex.value > 0) {
    currentIndex.value--
    playChunk(currentIndex.value)
  }
}

function nextChunk() {
  if (currentIndex.value + 1 < chunks.value.length) {
    currentIndex.value++
    playChunk(currentIndex.value)
  }
}

function cycleSpeed() {
  const speeds = [1.0, 1.25, 1.5, 1.75, 2.0]
  const nextIdx = (speeds.indexOf(playbackRate.value) + 1) % speeds.length
  playbackRate.value = speeds[nextIdx]
  if (audio) {
    audio.playbackRate = playbackRate.value
  }
}

function handleStart() {
  const genId = 'gen_' + Date.now() + '_' + Math.random().toString(36).slice(2, 8)
  activeGenerationId.value = genId
  cleanupSession()
  initAudio()
  isLoading.value = true
  errorMessage.value = ''
  isFinished.value = false
  chunks.value = []
  currentIndex.value = 0

  startTopicAudioOverview(props.topicId, props.notebookId, selectedVoice.value)
    .then((res) => {
      if (res?.generation_id) {
        activeGenerationId.value = res.generation_id
      }
    })
    .catch((err) => {
      showError(err.message || 'Failed to start audio overview')
    })
}

function restartWithVoice() {
  handleStart()
}

function cleanupSession() {
  if (audio) {
    audio.pause()
    audio.removeAttribute('src')
    audio.load()
  }
  isPlaying.value = false
  stopTopicAudioOverview().catch(() => {})
}

function handleClose() {
  cleanupSession()
  emit('close')
}

onMounted(() => {
  initAudio()

  EventsOn('audio:overview:start', (data) => {
    if (data?.generation_id && activeGenerationId.value && data.generation_id !== activeGenerationId.value) {
      return
    }
    if (data?.generation_id) {
      activeGenerationId.value = data.generation_id
    }
    isLoading.value = true
    errorMessage.value = ''
  })

  EventsOn('audio:overview:chunk', (chunk) => {
    if (chunk?.generation_id && activeGenerationId.value && chunk.generation_id !== activeGenerationId.value) {
      return
    }

    if (!chunk || chunk.status === 'error') {
      if (chunk && chunk.error) {
        console.warn('[AudioOverview] Chunk synthesis error:', chunk.error)
        showError(chunk.error)
      }
      return
    }

    if (chunk.total_chunks) {
      totalChunks.value = chunk.total_chunks
    }

    if (errorMessage.value) {
      errorMessage.value = ''
    }

    chunks.value.push(chunk)

    // Play chunk 1 immediately as soon as it lands without waiting
    if (chunks.value.length === 1) {
      currentIndex.value = 0
      playChunk(0)
    } else if (isLoading.value && currentIndex.value === chunks.value.length - 1) {
      // Catch up if player was waiting for this chunk
      playChunk(currentIndex.value)
    }
  })

  EventsOn('audio:overview:complete', (data) => {
    if (data?.generation_id && activeGenerationId.value && data.generation_id !== activeGenerationId.value) {
      return
    }
    isFinished.value = true
    isLoading.value = false
    if (data && data.total_chunks) {
      totalChunks.value = data.total_chunks
    }
  })

  EventsOn('audio:overview:error', (data) => {
    if (data?.generation_id && activeGenerationId.value && data.generation_id !== activeGenerationId.value) {
      return
    }
    showError(data?.error || 'Audio overview generation failed')
  })

  handleStart()
})

onUnmounted(() => {
  cleanupSession()
  EventsOff('audio:overview:start')
  EventsOff('audio:overview:chunk')
  EventsOff('audio:overview:complete')
  EventsOff('audio:overview:error')
})

watch(
  () => props.topicId,
  (newId, oldId) => {
    if (newId && newId !== oldId) {
      handleStart()
    }
  }
)
</script>

<style scoped>
.audio-overview-bar {
  position: sticky;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 99;
  background: var(--surface-container-lowest, #141617);
  border-top: 1px solid var(--outline-variant, rgba(235, 219, 178, 0.1));
  box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.12);
  padding: 10px 18px;
  animation: slideUp 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  color: var(--on-surface, #ebdbb2);
}

@keyframes slideUp {
  from {
    transform: translateY(100%);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.audio-bar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  max-width: 1300px;
  margin: 0 auto;
}

.audio-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.play-btn {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--primary, #d79921);
  color: var(--on-primary, #1d2021);
  border: none;
  font-size: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--primary) 25%, transparent);
  transition: transform 0.15s ease, background 0.2s ease, box-shadow 0.2s ease;
}

.play-btn:hover:not(:disabled) {
  background: var(--primary-dim, #b57614);
  transform: scale(1.06);
}

.play-btn:active:not(:disabled) {
  transform: scale(0.95);
}

.play-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.nav-btn {
  background: var(--surface-container, #282828);
  border: 1px solid var(--outline-variant, rgba(235, 219, 178, 0.1));
  color: var(--on-surface, #ebdbb2);
  border-radius: 8px;
  padding: 7px 11px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.nav-btn:hover:not(:disabled) {
  background: var(--surface-container-highest, #3c3836);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.nav-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.progress-indicator {
  display: flex;
  align-items: center;
}

.chunk-badge {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.03em;
  color: var(--muted-text, #a89984);
  background: var(--surface-container-low, #232526);
  border: 1px solid var(--outline-variant, rgba(235, 219, 178, 0.08));
  padding: 4px 10px;
  border-radius: 20px;
  white-space: nowrap;
}

.audio-transcript {
  flex: 1;
  min-width: 0;
  padding: 0 12px;
}

.caption-text {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.45;
  color: var(--on-surface, #ebdbb2);
  font-weight: 500;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.buffering-text {
  margin: 0;
  font-size: 13px;
  color: var(--primary, #d79921);
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
}

.pulse-dot {
  animation: pulse 1.2s infinite ease-in-out;
}

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}

.error-text {
  margin: 0;
  font-size: 13px;
  color: #ef4444;
  font-weight: 500;
}

.audio-options {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.voice-select {
  padding: 6px 30px 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant, rgba(235, 219, 178, 0.12));
  background-color: var(--surface-container-low, #232526);
  color: var(--on-surface, #ebdbb2);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease;
}

.voice-select:hover:not(:disabled) {
  border-color: var(--primary);
  background-color: var(--surface-container);
}

.voice-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.speed-btn {
  padding: 6px 10px;
  border-radius: 8px;
  border: 1px solid var(--outline-variant, rgba(235, 219, 178, 0.12));
  background: var(--surface-container-low, #232526);
  color: var(--on-surface, #ebdbb2);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  min-width: 44px;
  transition: all 0.15s ease;
}

.speed-btn:hover {
  background: var(--surface-container);
  border-color: color-mix(in srgb, var(--primary) 30%, transparent);
}

.close-btn {
  background: transparent;
  border: none;
  font-size: 15px;
  color: var(--muted-text, #a89984);
  cursor: pointer;
  padding: 6px 8px;
  border-radius: 6px;
  transition: all 0.15s ease;
}

.close-btn:hover {
  color: var(--on-surface, #ebdbb2);
  background: var(--surface-container-highest, #3c3836);
}
</style>
