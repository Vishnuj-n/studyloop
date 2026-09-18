<template>
  <Teleport to="body">
    <div v-if="visible" class="release-modal-overlay" @click.self="close">
      <div class="release-modal" role="dialog" aria-labelledby="release-modal-title">
        <header class="release-modal-header">
          <div class="header-badge">
            <span class="sparkle-icon">
              <BaseIcon name="sparkles" size="16" />
            </span>
            <span class="version-tag">v{{ version }}</span>
          </div>
          <h2 id="release-modal-title">What's New in Studyloop</h2>
        </header>

        <div class="release-modal-body">
          <div v-if="formattedNotes.length > 0" class="notes-list">
            <div v-for="(section, idx) in formattedNotes" :key="idx" class="notes-section">
              <h3 v-if="section.title" class="section-title">{{ section.title }}</h3>
              <ul class="bullet-list">
                <li v-for="(item, i) in section.items" :key="i" class="bullet-item">
                  <span class="bullet-dot">•</span>
                  <!-- eslint-disable-next-line vue/no-v-html -->
                  <span class="bullet-text" v-html="formatInlineText(item)"></span>
                </li>
              </ul>
            </div>
          </div>
          <p v-else class="raw-notes">{{ rawNotes }}</p>
        </div>

        <footer class="release-modal-footer">
          <button type="button" class="btn-ack" @click="close">
            Got it &mdash; Don't show again
          </button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  version: {
    type: String,
    default: '',
  },
  rawNotes: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['close'])

const formattedNotes = computed(() => {
  if (!props.rawNotes) return []
  const lines = props.rawNotes.split('\n')
  const sections = []
  let currentSection = { title: '', items: [] }

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) continue

    if (line.startsWith('#')) {
      if (currentSection.items.length > 0 || currentSection.title) {
        sections.push(currentSection)
      }
      currentSection = {
        title: line.replace(/^#+\s*/, ''),
        items: [],
      }
    } else if (line.startsWith('-') || line.startsWith('*')) {
      currentSection.items.push(line.replace(/^[-*]\s*/, ''))
    } else {
      if (!currentSection.title && currentSection.items.length === 0) {
        currentSection.title = line
      } else {
        currentSection.items.push(line)
      }
    }
  }

  if (currentSection.items.length > 0 || currentSection.title) {
    sections.push(currentSection)
  }

  return sections
})

function formatInlineText(text) {
  if (!text) return ''
  // Bold **text**
  let out = text.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  // Code `text`
  out = out.replace(/`(.*?)`/g, '<code>$1</code>')
  return out
}

function close() {
  emit('close')
}
</script>

<style scoped>
.release-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(15, 23, 42, 0.75);
  backdrop-filter: blur(8px);
  z-index: 10000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  animation: fadeIn 0.25s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.release-modal {
  background: var(--surface-container-lowest, #0f172a);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.1));
  border-radius: 20px;
  padding: 28px 32px;
  max-width: 520px;
  width: 100%;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  gap: 20px;
  animation: scaleUp 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes scaleUp {
  from { transform: scale(0.95); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.release-modal-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.header-badge {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sparkle-icon {
  font-size: 20px;
}

.version-tag {
  font-family: monospace;
  font-size: 12px;
  font-weight: 700;
  background: color-mix(in srgb, var(--primary, #4f46e5) 15%, transparent);
  color: var(--primary, #6366f1);
  padding: 3px 10px;
  border-radius: 99px;
  border: 1px solid color-mix(in srgb, var(--primary, #4f46e5) 30%, transparent);
}

.release-modal-header h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  font-family: 'Manrope', sans-serif;
  color: var(--on-surface, #ffffff);
  letter-spacing: -0.02em;
}

.release-modal-body {
  max-height: 340px;
  overflow-y: auto;
  padding-right: 6px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.18) transparent;
}

.release-modal-body::-webkit-scrollbar {
  width: 6px;
}

.release-modal-body::-webkit-scrollbar-track {
  background: transparent;
}

.release-modal-body::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.18);
  border-radius: 99px;
}

.release-modal-body::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.35);
}

.notes-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--on-surface, #f8fafc);
  border-bottom: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
  padding-bottom: 4px;
}

.bullet-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.bullet-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  font-size: 13px;
  color: var(--on-surface-variant, #cbd5e1);
  line-height: 1.5;
}

.bullet-dot {
  color: var(--primary, #6366f1);
  font-weight: bold;
}

.bullet-text :deep(strong) {
  color: var(--on-surface, #ffffff);
  font-weight: 700;
}

.bullet-text :deep(code) {
  font-family: monospace;
  background: var(--surface-container-high, rgba(255, 255, 255, 0.06));
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.release-modal-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.06));
}

.btn-ack {
  background: var(--primary, #4f46e5);
  color: var(--on-primary, #ffffff);
  border: none;
  border-radius: 10px;
  padding: 10px 22px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  width: 100%;
  text-align: center;
}

.btn-ack:hover {
  background: var(--primary-hover, #4338ca);
  transform: translateY(-1px);
}
</style>
