<template>
  <div class="youtube-player-container">
    <div v-if="startSeconds > 0 || endSeconds > 0 || videoId" class="video-timecode-banner">
      <div class="timecode-info">
        <span v-if="startSeconds > 0 || endSeconds > 0" class="timecode-badge">⏱️ {{ formatTime(startSeconds) }} – {{ formatTime(endSeconds) }}</span>
        <span v-if="durationText" class="duration-badge">Duration: {{ durationText }}</span>
        <button
          v-if="externalWatchUrl"
          type="button"
          class="open-browser-btn"
          title="Play this chapter in your default browser (Chrome/Edge/Firefox)"
          @click="openInExternalBrowser"
        >
          ↗ Open in Browser
        </button>
      </div>
      <p v-if="startSeconds > 0 || endSeconds > 0" class="timecode-hint">This study session covers the video segment from {{ formatTime(startSeconds) }} to {{ formatTime(endSeconds) }}.</p>
    </div>

    <div class="video-wrapper">
      <iframe
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
import { ref, computed } from 'vue'
import MarkdownReader from './MarkdownReader.vue'
import { openURLInBrowser } from '../services/appApi'

const props = defineProps({
  embedUrl: {
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

function openInExternalBrowser() {
  if (externalWatchUrl.value) {
    openURLInBrowser(externalWatchUrl.value)
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

.video-wrapper {
  position: relative;
  width: 100%;
  padding-bottom: 56.25%; /* 16:9 Aspect Ratio */
  background: #000000;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.youtube-iframe {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  border: none;
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
