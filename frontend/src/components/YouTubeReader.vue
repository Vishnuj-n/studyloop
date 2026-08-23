<template>
  <div class="youtube-player-container">
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
import { ref } from 'vue'
import MarkdownReader from './MarkdownReader.vue'

defineProps({
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
</script>

<style scoped>
.youtube-player-container {
  display: flex;
  flex-direction: column;
  gap: 18px;
  width: 100%;
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
