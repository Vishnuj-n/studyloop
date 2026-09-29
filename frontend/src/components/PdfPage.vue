<template>
  <div
    :data-page="pageNum"
    class="pdf-page-render"
    :style="{ width: width + 'px' }"
    @click="handleLayerClick"
  >
    <vue-pdf-embed
      :key="`${source}-p${pageNum}`"
      :source="source"
      :page="pageNum"
      :width="width"
      :text-layer="true"
      :annotation-layer="true"
      @rendered="$emit('rendered', pageNum)"
      @loading-failed="onFailed"
      @rendering-failed="onFailed"
    />
  </div>
</template>

<script setup>
import VuePdfEmbed from 'vue-pdf-embed'
import 'vue-pdf-embed/dist/styles/textLayer.css'
import 'vue-pdf-embed/dist/styles/annotationLayer.css'

const props = defineProps({
  /** PDF source URL (all instances share the same pdfjs-dist document cache) */
  source: { type: String, required: true },
  /** Page number to render */
  pageNum: { type: Number, required: true },
  /** Exact CSS pixel width — vue-pdf-embed calculates scale + text layer from this */
  width: { type: Number, required: true },
})

const emit = defineEmits(['rendered', 'load-error', 'navigate'])

function onFailed(err) {
  // If a render was aborted or cancelled because a new scale/page was requested, do not emit as fatal error
  const msg = err?.message || String(err || '')
  if (msg.includes('cancelled') || msg.includes('aborted') || msg.includes('multiple render()')) {
    console.warn(`[PdfPage] Render cancelled/superseded for page ${props.pageNum}`)
    return
  }
  emit('load-error', err)
}

function handleLayerClick(event) {
  const link = event.target.closest('a')
  if (!link) return

  const href = link.getAttribute('href') || ''
  if (!href) return

  // Check if it's an internal PDF page reference (e.g., #page=88, #nameddest=..., #[{num: 88, ...}])
  if (href.startsWith('#')) {
    event.preventDefault()
    event.stopPropagation()

    // Match "#page=88" or "#88" or explicit page parameters
    const pageMatch = href.match(/page=(\d+)/i) || href.match(/^#(\d+)$/)
    if (pageMatch) {
      const targetPage = Number.parseInt(pageMatch[1], 10)
      if (targetPage > 0) {
        emit('navigate', { fromPage: props.pageNum, targetPage })
      }
    }
  } else if (/^https?:\/\//i.test(href)) {
    // Ensure external links open safely in a new browser window
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
