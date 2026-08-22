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

const chunks = ref([])
const currentIndex = ref(0)
const isPlaying = ref(false)
const isLoading = ref(true)
const isFinished = ref(false)
const totalChunks = ref(0)
const errorMessage = ref('')
const selectedVoice = ref('en-US-ChristopherNeural')
const playbackRate = ref(1.0)

let audio = null

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
      console.error('[AudioOverview] Audio element error:', e)
      errorMessage.value = 'Playback error encountered on current audio chunk.'
      isPlaying.value = false
      isLoading.value = false
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
  cleanupSession()
  initAudio()
  isLoading.value = true
  errorMessage.value = ''
  isFinished.value = false
  chunks.value = []
  currentIndex.value = 0

  startTopicAudioOverview(props.topicId, props.notebookId, selectedVoice.value).catch((err) => {
    errorMessage.value = err.message || 'Failed to start audio overview'
    isLoading.value = false
  })
}

function restartWithVoice() {
  handleStart()
}

function cleanupSession() {
  if (audio) {
    audio.pause()
    audio.src = ''
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

  EventsOn('audio:overview:start', () => {
    isLoading.value = true
    errorMessage.value = ''
  })

  EventsOn('audio:overview:chunk', (chunk) => {
    if (!chunk || chunk.status === 'error') {
      if (chunk && chunk.error) {
        errorMessage.value = chunk.error
      }
      isLoading.value = false
      return
    }

    if (chunk.total_chunks) {
      totalChunks.value = chunk.total_chunks
    }

    chunks.value.push(chunk)

    // ponytail: play chunk 1 immediately as it lands (< 1.5s latency)
    if (chunks.value.length === 1) {
      currentIndex.value = 0
      playChunk(0)
    } else if (isLoading.value && currentIndex.value === chunks.value.length - 1) {
      // Catch up if player was waiting for this chunk
      playChunk(currentIndex.value)
    }
  })

  EventsOn('audio:overview:complete', (data) => {
    isFinished.value = true
    isLoading.value = false
    if (data && data.total_chunks) {
      totalChunks.value = data.total_chunks
    }
  })

  EventsOn('audio:overview:error', (data) => {
    isLoading.value = false
    errorMessage.value = data?.error || 'Audio overview generation failed'
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
  background: var(--bg-card, #ffffff);
  border-top: 1px solid var(--border-color, #e5e7eb);
  box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.08);
  padding: 10px 16px;
  animation: slideUp 0.25s ease-out;
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
  max-width: 1200px;
  margin: 0 auto;
}

.audio-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.play-btn {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-primary, #3b82f6);
  color: #ffffff;
  border: none;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: transform 0.1s ease, background 0.2s ease;
}

.play-btn:hover:not(:disabled) {
  background: var(--color-primary-hover, #2563eb);
  transform: scale(1.05);
}

.play-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.nav-btn {
  background: transparent;
  border: 1px solid var(--border-color, #e5e7eb);
  color: var(--text-primary, #1f2937);
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.nav-btn:hover:not(:disabled) {
  background: var(--bg-hover, #f3f4f6);
}

.nav-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.progress-indicator {
  display: flex;
  align-items: center;
}

.chunk-badge {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary, #6b7280);
  background: var(--bg-secondary, #f3f4f6);
  padding: 4px 8px;
  border-radius: 12px;
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
  line-height: 1.4;
  color: var(--text-primary, #111827);
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
  color: var(--color-primary, #3b82f6);
  font-weight: 500;
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
  color: #dc2626;
  font-weight: 500;
}

.audio-options {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.voice-select {
  padding: 5px 8px;
  border-radius: 6px;
  border: 1px solid var(--border-color, #e5e7eb);
  background: var(--bg-card, #ffffff);
  color: var(--text-primary, #1f2937);
  font-size: 12.5px;
  cursor: pointer;
}

.speed-btn {
  padding: 5px 8px;
  border-radius: 6px;
  border: 1px solid var(--border-color, #e5e7eb);
  background: var(--bg-secondary, #f3f4f6);
  color: var(--text-primary, #1f2937);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  min-width: 42px;
}

.speed-btn:hover {
  background: var(--bg-hover, #e5e7eb);
}

.close-btn {
  background: transparent;
  border: none;
  font-size: 16px;
  color: var(--text-secondary, #9ca3af);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: color 0.15s ease;
}

.close-btn:hover {
  color: var(--text-primary, #111827);
  background: var(--bg-hover, #f3f4f6);
}
</style>
