<template>
  <div class="youtube-player-container">
    <div v-if="startSeconds > 0 || endSeconds > 0 || videoId || cachedVideoUrl" class="video-timecode-banner">
      <div class="timecode-info">
        <span v-if="startSeconds > 0 || endSeconds > 0" class="timecode-badge">⏱️ {{ formatTime(startSeconds) }} – {{ formatTime(endSeconds) }}</span>
        <span v-if="durationText" class="duration-badge">Duration: {{ durationText }}</span>

        <!-- Source Mode Switcher (Local Offline vs Online YouTube) -->
        <div v-if="cachedVideoUrl" class="source-toggle-group">
          <button
            type="button"
            class="source-toggle-btn"
            :class="{ active: sourceMode === 'local' }"
            @click="sourceMode = 'local'"
          >
            💾 Offline Video
          </button>
          <button
            type="button"
            class="source-toggle-btn"
            :class="{ active: sourceMode === 'youtube' }"
            @click="sourceMode = 'youtube'"
          >
            🌐 YouTube Stream
          </button>
        </div>

        <button
          v-if="externalWatchUrl"
          type="button"
          class="open-browser-btn"
          title="Play this chapter in your default browser (Chrome/Edge/Firefox)"
          @click="openInExternalBrowser"
        >
          ↗ Open in Browser
        </button>
        <span v-if="browserError" class="browser-error-msg">⚠️ {{ browserError }}</span>
      </div>
      <p v-if="startSeconds > 0 || endSeconds > 0" class="timecode-hint">
        This study session covers the video segment from {{ formatTime(startSeconds) }} to {{ formatTime(endSeconds) }}.
        <span v-if="sourceMode === 'local'" class="offline-active-hint">(Playing high-speed offline copy from disk)</span>
      </p>
    </div>

    <div class="video-wrapper">
      <!-- Local Video Player when offline/cached mode -->
      <video
        v-if="sourceMode === 'local' && cachedVideoUrl"
        ref="videoRef"
        :src="cachedVideoUrl"
        class="video-element"
        controls
        autoplay
        playsinline
        @loadedmetadata="onVideoMetadata"
        @canplay="onVideoMetadata"
        @timeupdate="onTimeUpdate"
      >
        <track
          v-if="captionTrackUrl"
          kind="subtitles"
          srclang="en"
          label="English"
          :src="captionTrackUrl"
        />
      </video>

      <!-- Remote YouTube Iframe -->
      <iframe
        v-else
        :src="embedUrl"
        class="youtube-iframe"
        frameborder="0"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
        allowfullscreen
      ></iframe>
    </div>

    <div class="transcript-drawer">
      <div class="transcript-drawer-header" @click="showTranscript = !showTranscript">
        <span class="transcript-header-title">📖 Chapter Transcript & Notes</span>
        <span class="transcript-toggle-btn">{{ showTranscript ? '▲ Hide Transcript' : '▼ View Transcript' }}</span>
      </div>
      <div v-if="showTranscript" class="transcript-body">
        <MarkdownReader
          :content="transcriptContent"
          :topic-title="topicTitle"
          :start-page="startPage"
          :end-page="endPage"
          :is-task-flow="isTaskFlow"
          :completing="completing"
          :disabled="disabled"
          @complete="$emit('complete')"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import MarkdownReader from './MarkdownReader.vue'
import { openURLInBrowser } from '../services/appApi'

const props = defineProps({
  embedUrl: {
    type: String,
    default: '',
  },
  cachedVideoUrl: {
    type: String,
    default: '',
  },
  transcriptContent: {
    type: String,
    default: '',
  },
  topicTitle: {
    type: String,
    default: '',
  },
  startPage: {
    type: Number,
    default: 1,
  },
  endPage: {
    type: Number,
    default: 1,
  },
  videoStartSeconds: {
    type: Number,
    default: 0,
  },
  videoEndSeconds: {
    type: Number,
    default: 0,
  },
  isTaskFlow: {
    type: Boolean,
    default: false,
  },
  completing: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['complete'])

const showTranscript = ref(true)
const browserError = ref('')
const videoRef = ref(null)
// Default to local if cached video is available
const sourceMode = ref(props.cachedVideoUrl ? 'local' : 'youtube')

watch(
  () => props.cachedVideoUrl,
  (newVal) => {
    if (newVal && !sourceMode.value) {
      sourceMode.value = 'local'
    }
  },
  { immediate: true }
)

const startSeconds = computed(() => {
  if (props.videoStartSeconds > 0) return props.videoStartSeconds
  try {
    const url = new URL(props.embedUrl)
    return parseInt(url.searchParams.get('start') || '0', 10)
  } catch {
    return 0
  }
})

const endSeconds = computed(() => {
  if (props.videoEndSeconds > 0) return props.videoEndSeconds
  try {
    const url = new URL(props.embedUrl)
    return parseInt(url.searchParams.get('end') || '0', 10)
  } catch {
    return 0
  }
})

let userInteractedPastEnd = false

function onVideoMetadata() {
  if (videoRef.value) {
    userInteractedPastEnd = false
    if (startSeconds.value > 0 && Math.abs(videoRef.value.currentTime - startSeconds.value) > 1) {
      videoRef.value.currentTime = startSeconds.value
    }
  }
}

// ponytail: Auto-pause when chapter ends; user can press play again if they wish to continue
function onTimeUpdate() {
  if (!videoRef.value) return
  const current = videoRef.value.currentTime

  if (endSeconds.value > 0 && current >= endSeconds.value) {
    if (!videoRef.value.paused && !userInteractedPastEnd) {
      videoRef.value.pause()
      userInteractedPastEnd = true
    }
  } else if (current < endSeconds.value) {
    userInteractedPastEnd = false
  }
}

watch(
  [() => startSeconds.value, () => sourceMode.value],
  async () => {
    userInteractedPastEnd = false
    await nextTick()
    if (sourceMode.value === 'local' && videoRef.value) {
      if (startSeconds.value > 0) {
        videoRef.value.currentTime = startSeconds.value
      } else {
        videoRef.value.currentTime = 0
      }
    }
  }
)

function formatTime(totalSeconds) {
  if (!totalSeconds || isNaN(totalSeconds) || totalSeconds < 0) return '0:00'
  const hrs = Math.floor(totalSeconds / 3600)
  const mins = Math.floor((totalSeconds % 3600) / 60)
  const secs = Math.floor(totalSeconds % 60)
  if (hrs > 0) {
    return `${hrs}:${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  }
  return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
}

// ponytail: Slice transcript into clean, bite-sized subtitle cues (1-2 lines at a time)
const captionTrackUrl = computed(() => {
  if (!props.transcriptContent) return ''
  const start = startSeconds.value || 0
  const end = endSeconds.value > start ? endSeconds.value : start + 600
  const text = props.transcriptContent.replace(/<[^>]+>/g, '').replace(/[#*`_[\]]/g, '').trim()
  if (!text) return ''

  const words = text.split(/\s+/)
  const chunkSize = 7 // 7 words per line looks neat and compact
  const totalChunks = Math.ceil(words.length / chunkSize)
  if (totalChunks === 0) return ''

  const chunkDuration = (end - start) / totalChunks
  let vtt = 'WEBVTT\n\n'

  for (let i = 0; i < totalChunks; i++) {
    const s1 = start + i * chunkDuration
    const s2 = Math.min(end, s1 + chunkDuration)
    const t1 = (formatTime(s1).length <= 4 ? '00:0' : '00:') + formatTime(s1) + '.000'
    const t2 = (formatTime(s2).length <= 4 ? '00:0' : '00:') + formatTime(s2) + '.000'
    const line = words.slice(i * chunkSize, (i + 1) * chunkSize).join(' ')
    vtt += `${t1} --> ${t2}\n${line}\n\n`
  }

  return URL.createObjectURL(new Blob([vtt], { type: 'text/vtt' }))
})

const durationText = computed(() => {
  if (!endSeconds.value || endSeconds.value <= startSeconds.value) return ''
  const durSec = endSeconds.value - startSeconds.value
  const mins = Math.floor(durSec / 60)
  const secs = durSec % 60
  if (mins > 0 && secs > 0) return `${mins}m ${secs}s`
  if (mins > 0) return `${mins} min`
  return `${secs}s`
})

const videoId = computed(() => {
  if (!props.embedUrl) return ''
  const match = props.embedUrl.match(/\/embed\/([^/?#]+)/)
  return match ? match[1] : ''
})

const externalWatchUrl = computed(() => {
  if (!videoId.value) return ''
  let url = `https://www.youtube.com/watch?v=${videoId.value}`
  if (startSeconds.value > 0) {
    url += `&t=${startSeconds.value}s`
  }
  return url
})

async function openInExternalBrowser() {
  browserError.value = ''
  if (externalWatchUrl.value) {
    try {
      const res = await openURLInBrowser(externalWatchUrl.value)
      if (res && res.error) {
        browserError.value = res.error
      }
    } catch (err) {
      browserError.value = err instanceof Error ? err.message : String(err)
    }
  }
}
</script>

<style scoped>
.youtube-player-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.video-timecode-banner {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 16px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
}

.timecode-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.browser-error-msg {
  font-size: 12px;
  color: var(--error, #ba1a1a);
  font-weight: 500;
}

.open-browser-btn {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-high);
  color: var(--on-surface);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.15s ease;
}

.open-browser-btn:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--surface-container-highest);
}

.open-browser-btn:active {
  transform: scale(0.97);
}

.timecode-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 700;
  font-size: 13.5px;
  color: var(--primary);
  background: var(--surface-container-high);
  padding: 4px 10px;
  border-radius: 6px;
  letter-spacing: 0.3px;
}

.duration-badge {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--on-surface-variant);
}

.timecode-hint {
  margin: 0;
  font-size: 12px;
  color: var(--on-surface-variant);
  line-height: 1.4;
}

.source-toggle-group {
  display: inline-flex;
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 2px;
  gap: 2px;
}

.source-toggle-btn {
  background: transparent;
  border: none;
  color: var(--on-surface-variant);
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 6px;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.15s ease;
}

.source-toggle-btn.active {
  background: var(--primary);
  color: var(--on-primary, #ffffff);
}

.source-toggle-btn:not(.active):hover {
  color: var(--on-surface);
  background: var(--surface-container-highest);
}

.offline-active-hint {
  color: var(--primary);
  font-weight: 600;
  margin-left: 6px;
}

.video-wrapper {
  position: relative;
  width: 100%;
  padding-bottom: 56.25%; /* 16:9 Aspect Ratio */
  background: #000000;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.video-element,
.youtube-iframe {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  border: none;
  object-fit: contain;
}

.video-element::cue {
  background-color: rgba(0, 0, 0, 0.75);
  color: #ffffff;
  font-size: 15px;
  line-height: 1.4;
  font-family: inherit;
  border-radius: 4px;
}

.transcript-drawer {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  overflow: hidden;
}

.transcript-drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 18px;
  cursor: pointer;
  background: var(--surface-container);
  user-select: none;
}

.transcript-header-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--on-surface);
}

.transcript-toggle-btn {
  font-size: 12.5px;
  color: var(--primary);
  font-weight: 600;
}

.transcript-body {
  padding: 16px 20px;
  max-height: 500px;
  overflow-y: auto;
}
</style>
