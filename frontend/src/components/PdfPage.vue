<template>
  <div
    ref="rootEl"
    :data-page="pageNum"
    class="pdf-page-render"
    :style="{ width: width + 'px' }"
    @click="handleLayerClick"
  >
    <vue-pdf-embed
      :key="pageKey"
      :source="source"
      :page="pageNum"
      :width="width"
      :text-layer="true"
      :annotation-layer="true"
      @rendered="onRendered(pageNum)"
      @loading-failed="onFailed"
      @rendering-failed="onFailed"
      @internal-link-clicked="handleInternalLinkClicked"
    />
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import VuePdfEmbed from 'vue-pdf-embed'
import 'vue-pdf-embed/dist/styles/textLayer.css'
import 'vue-pdf-embed/dist/styles/annotationLayer.css'

const props = defineProps({
  /** PDF source URL or shared PDFDocumentProxy */
  source: { type: [String, Object], required: true },
  /** Page number to render */
  pageNum: { type: Number, required: true },
  /** Exact CSS pixel width - vue-pdf-embed calculates scale + text layer from this */
  width: { type: Number, required: true },
})

const emit = defineEmits(['rendered', 'load-error', 'navigate'])
const rootEl = ref(null)

const pageKey = computed(() => {
  const docId = typeof props.source === 'string'
    ? props.source
    : props.source?.fingerprint || props.source?.fingerprints?.[0] || 'shared-doc'
  return `${docId}-p${props.pageNum}`
})

function onRendered(pg) {
  emit('rendered', pg)
  const total = document.querySelectorAll('canvas').length
  console.log(`[GPU/Memory Diagnostics] Page ${props.pageNum} rendered. Total DOM canvases: ${total}`)
}

onUnmounted(() => {
  if (rootEl.value) {
    const canvases = rootEl.value.querySelectorAll('canvas')
    canvases.forEach((canvas) => {
      // Releasing the backing store forces Chromium/WebView2 to free the GPU texture immediately
      canvas.width = 0
      canvas.height = 0
    })
    const remaining = document.querySelectorAll('canvas').length
    console.log(
      `[GPU/Memory Diagnostics] Page ${props.pageNum} unmounted. Cleared ${canvases.length} canvas GPU texture(s). Remaining canvases in DOM: ${remaining}`
    )
  }
})

function onFailed(err) {
  // If a render was aborted or cancelled because a new scale/page was requested, do not emit as fatal error
  const msg = err?.message || String(err || '')
  if (msg.includes('cancelled') || msg.includes('aborted') || msg.includes('multiple render()')) {
    console.warn(`[PdfPage] Render cancelled/superseded for page ${props.pageNum}`)
    return
  }
  emit('load-error', err)
}

function handleInternalLinkClicked(targetPage) {
  const target = Number.parseInt(targetPage, 10)
  if (target > 0) {
    emit('navigate', { fromPage: props.pageNum, targetPage: target })
  }
}

function handleLayerClick(event) {
  const link = event.target.closest('a')
  if (!link) return

  const href = link.getAttribute('href') || ''
  if (!href) return

  // External links open in a new tab/window safely
  if (/^https?:\/\//i.test(href)) {
    link.setAttribute('target', '_blank')
    link.setAttribute('rel', 'noopener noreferrer')
  }
}
</script>

<style scoped>
.pdf-page-render {
  margin: 0 auto;
  position: relative;
}

/* ponytail: keep annotation links visible with pointer cursor */
:deep(.annotationLayer section.linkAnnotation a) {
  cursor: pointer;
}
</style>
