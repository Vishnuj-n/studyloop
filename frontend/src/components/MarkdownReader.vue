<template>
  <div class="markdown-container">
    <!-- Earlier chapters (if studying chapter 2 or later) -->
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-if="renderedBefore" class="markdown-viewport before-viewport" v-html="renderedBefore"></div>

    <!-- Start of Assigned Reading Banner -->
    <div class="markdown-boundary-tag top-boundary">
      <div class="boundary-info">
        <span class="boundary-badge">📖 Start of Assigned Reading</span>
        <span class="boundary-sub">{{ topicTitle }} (Pages {{ validStartPage }}–{{ validEndPage }})</span>
      </div>
    </div>

    <!-- Assigned Reading Content -->
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div class="markdown-viewport assigned-viewport" v-html="renderedAssigned"></div>

    <!-- End of Assigned Reading Banner -->
    <div class="markdown-boundary-tag bottom-boundary">
      <div class="boundary-info">
        <span class="boundary-badge success">✓ End of Assigned Reading</span>
        <span class="boundary-sub">You have reached the end of this study section.</span>
      </div>
      <button
        v-if="isTaskFlow"
        class="primary proceed-button"
        :disabled="disabled"
        @click="$emit('complete')"
      >
        {{ completing ? 'Completing Session...' : 'Complete & Proceed to Quiz →' }}
      </button>
    </div>

    <!-- Subsequent chapters continuing seamlessly below the banner -->
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-if="renderedAfter" class="markdown-viewport after-viewport" v-html="renderedAfter"></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { renderMarkdown } from '../services/markdown'

const props = defineProps({
  content: {
    type: String,
    default: '',
  },
  topicTitle: {
    type: String,
    default: 'Reading Session',
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

function splitMarkdownIntoSections(content) {
  if (!content) return []
  const lines = content.split('\n')
  const sections = []
  let currentHeading = ''
  let currentLines = []

  for (const line of lines) {
    const trimmed = line.trim()
    if (trimmed.startsWith('# ') && !trimmed.startsWith('##')) {
      if (currentHeading || currentLines.length > 0) {
        sections.push({
          heading: currentHeading,
          text: currentLines.join('\n').trim(),
        })
      }
      currentHeading = trimmed.replace(/^#\s+/, '').trim()
      currentLines = []
    } else {
      currentLines.push(line)
    }
  }

  if (currentHeading || currentLines.length > 0) {
    sections.push({
      heading: currentHeading,
      text: currentLines.join('\n').trim(),
    })
  }

  return sections
}

const validStartPage = computed(() => Math.max(1, Number(props.startPage) || 1))
const validEndPage = computed(() => Math.max(validStartPage.value, Number(props.endPage) || validStartPage.value))

const parsedSections = computed(() => {
  return splitMarkdownIntoSections(props.content || '')
})

const renderedBefore = computed(() => {
  const sections = parsedSections.value
  if (sections.length <= 1) return ''
  const start = validStartPage.value
  const before = sections.slice(0, start - 1)
  if (before.length === 0) return ''
  const mdText = before
    .map((s) => (s.heading ? `# ${s.heading}\n\n${s.text}` : s.text))
    .join('\n\n---\n\n')
  return renderMarkdown(mdText)
})

const renderedAssigned = computed(() => {
  const sections = parsedSections.value
  if (sections.length <= 1) {
    return renderMarkdown(props.content || '')
  }
  const start = validStartPage.value
  const end = validEndPage.value
  const assigned = sections.slice(start - 1, end)
  if (assigned.length === 0) {
    return renderMarkdown(props.content || '')
  }
  const mdText = assigned
    .map((s) => (s.heading ? `# ${s.heading}\n\n${s.text}` : s.text))
    .join('\n\n---\n\n')
  return renderMarkdown(mdText)
})

const renderedAfter = computed(() => {
  const sections = parsedSections.value
  if (sections.length <= 1) return ''
  const end = validEndPage.value
  const after = sections.slice(end)
  if (after.length === 0) return ''
  const mdText = after
    .map((s) => (s.heading ? `# ${s.heading}\n\n${s.text}` : s.text))
    .join('\n\n---\n\n')
  return renderMarkdown(mdText)
})
</script>

<style scoped>
.markdown-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
}

.markdown-boundary-tag {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  font-size: 14px;
  color: var(--on-surface);
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.markdown-boundary-tag.top-boundary {
  border-top: 3px solid var(--primary);
  background: color-mix(in srgb, var(--surface-container) 94%, var(--primary) 6%);
}

.markdown-boundary-tag.bottom-boundary {
  border-bottom: 3px solid #10b981;
  background: color-mix(in srgb, var(--surface-container) 90%, #10b981 10%);
  padding: 18px 24px;
  flex-wrap: wrap;
}

.boundary-badge {
  font-weight: 700;
  font-size: 14px;
  color: var(--on-surface);
  display: flex;
  align-items: center;
  gap: 8px;
}

.boundary-badge.success {
  color: #059669;
  font-size: 15px;
}

.boundary-sub {
  font-size: 13px;
  color: var(--muted-text);
}

.boundary-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.proceed-button {
  padding: 10px 22px;
  font-weight: 600;
  font-size: 14px;
  border-radius: 10px;
  cursor: pointer;
  margin-left: auto;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: transform 0.15s ease, opacity 0.15s ease;
}

.proceed-button:hover:not(:disabled) {
  transform: translateY(-1px);
}

.markdown-viewport {
  padding: 32px 40px;
  background: var(--surface-container-lowest);
  border-radius: 12px;
  border: 1px solid var(--outline-variant);
  line-height: 1.7;
  color: var(--on-surface);
  font-size: 15px;
}

.markdown-viewport :deep(> *:first-child) {
  margin-top: 0;
}

.markdown-viewport :deep(> *:last-child) {
  margin-bottom: 0;
}

.markdown-viewport :deep(h1),
.markdown-viewport :deep(h2),
.markdown-viewport :deep(h3),
.markdown-viewport :deep(h4) {
  font-family: 'Manrope', sans-serif;
  color: var(--on-surface);
  margin: 24px 0 12px;
  line-height: 1.3;
}

.markdown-viewport :deep(h1) {
  font-size: 1.6rem;
  border-bottom: 1px solid var(--outline-variant);
  padding-bottom: 8px;
}

.markdown-viewport :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 18px 0;
  font-size: 14px;
  border: 1px solid var(--outline-variant);
}

.markdown-viewport :deep(th),
.markdown-viewport :deep(td) {
  border: 1px solid var(--outline-variant);
  padding: 8px 12px;
  text-align: left;
}

.markdown-viewport :deep(th) {
  background: var(--surface-container);
  font-weight: 600;
  color: var(--on-surface);
}

.markdown-viewport :deep(tbody tr:nth-child(even)) {
  background: var(--surface-container-lowest);
}

.markdown-viewport :deep(blockquote) {
  margin: 16px 0;
  padding: 8px 16px;
  border-left: 4px solid var(--primary);
  background: var(--surface-container-low);
  border-radius: 0 8px 8px 0;
  color: var(--on-surface-variant);
}

.markdown-viewport :deep(ul),
.markdown-viewport :deep(ol) {
  padding-left: 24px;
  margin: 12px 0;
}

.markdown-viewport :deep(li) {
  margin: 4px 0;
}

.markdown-viewport :deep(hr) {
  border: none;
  border-top: 1px solid var(--outline-variant);
  margin: 24px 0;
}

.markdown-viewport :deep(pre) {
  background: var(--surface-container-high);
  border: 1px solid var(--outline-variant);
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.markdown-viewport :deep(code) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9em;
  background: var(--surface-container-high);
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid var(--outline-variant);
}

.markdown-viewport :deep(pre code) {
  background: none;
  padding: 0;
  border: none;
  border-radius: 0;
}

.markdown-viewport :deep(.hljs-keyword),
.markdown-viewport :deep(.hljs-selector-tag) {
  color: #d73a49;
}

.markdown-viewport :deep(.hljs-string),
.markdown-viewport :deep(.hljs-attribute) {
  color: #032f62;
}

.markdown-viewport :deep(.hljs-number),
.markdown-viewport :deep(.hljs-literal) {
  color: #005cc5;
}

.markdown-viewport :deep(.hljs-title),
.markdown-viewport :deep(.hljs-section) {
  color: #6f42c1;
}

.markdown-viewport :deep(.hljs-comment) {
  color: #6a737d;
  font-style: italic;
}

.markdown-viewport :deep(.task-list-item) {
  list-style-type: none;
  margin-left: -20px;
}

.markdown-viewport :deep(.task-list-item input[type='checkbox']) {
  margin-right: 8px;
}

.markdown-viewport :deep(.markdown-alert) {
  margin: 16px 0;
  padding: 12px 16px;
  border-left: 4px solid var(--outline);
  background: var(--surface-container-low);
  border-radius: 0 8px 8px 0;
}

.markdown-viewport :deep(.markdown-alert-title) {
  font-weight: 600;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.markdown-viewport :deep(.markdown-alert-note) {
  border-left-color: #2196f3;
}

.markdown-viewport :deep(.markdown-alert-note .markdown-alert-title) {
  color: #2196f3;
}

.markdown-viewport :deep(.markdown-alert-tip) {
  border-left-color: #4caf50;
}

.markdown-viewport :deep(.markdown-alert-tip .markdown-alert-title) {
  color: #4caf50;
}

.markdown-viewport :deep(.markdown-alert-important) {
  border-left-color: #9c27b0;
}

.markdown-viewport :deep(.markdown-alert-important .markdown-alert-title) {
  color: #9c27b0;
}

.markdown-viewport :deep(.markdown-alert-warning) {
  border-left-color: #ff9800;
}

.markdown-viewport :deep(.markdown-alert-warning .markdown-alert-title) {
  color: #ff9800;
}

.markdown-viewport :deep(.markdown-alert-caution) {
  border-left-color: #f44336;
}

.markdown-viewport :deep(.markdown-alert-caution .markdown-alert-title) {
  color: #f44336;
}

.markdown-viewport :deep(.footnotes) {
  margin-top: 32px;
  border-top: 1px solid var(--outline-variant);
  padding-top: 16px;
  font-size: 13px;
}
</style>
