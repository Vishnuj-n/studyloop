<template>
  <div
    :data-page="pageNum"
    class="pdf-page-render"
    :style="{ width: width + 'px' }"
  >
    <vue-pdf-embed
      :key="`${source}-p${pageNum}-w${width}`"
      :source="source"
      :page="pageNum"
      :width="width"
      :text-layer="true"
      :annotation-layer="false"
      @rendered="$emit('rendered', pageNum)"
      @loading-failed="onFailed"
      @rendering-failed="onFailed"
    />
  </div>
</template>

<script setup>
import VuePdfEmbed from 'vue-pdf-embed'
import 'vue-pdf-embed/dist/styles/textLayer.css'

const props = defineProps({
  /** PDF source URL (all instances share the same pdfjs-dist document cache) */
  source: { type: String, required: true },
  /** Page number to render */
  pageNum: { type: Number, required: true },
  /** Exact CSS pixel width — vue-pdf-embed calculates scale + text layer from this */
  width: { type: Number, required: true },
})

const emit = defineEmits(['rendered', 'load-error'])

function onFailed(err) {
  // If a render was aborted or cancelled because a new scale/page was requested, do not emit as fatal error
  const msg = err?.message || String(err || '')
  if (msg.includes('cancelled') || msg.includes('aborted') || msg.includes('multiple render()')) {
    console.warn(`[PdfPage] Render cancelled/superseded for page ${props.pageNum}`)
    return
  }
  emit('load-error', err)
}
</script>

<style scoped>
.pdf-page-render {
  margin: 0 auto;
}

/* ponytail: no CSS overrides on vue-pdf-embed internals — the :width prop
   drives correct canvas + text layer coordinate sync out of the box */
</style>
