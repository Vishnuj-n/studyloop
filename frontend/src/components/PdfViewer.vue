<template>
  <div
    ref="viewportRef"
    class="pdf-viewer-viewport"
    tabindex="0"
    :style="{ opacity: ready || loadError ? 1 : 0, transition: 'opacity 0.2s ease' }"
  >
    <div v-if="loadError" class="pdf-viewer-error">{{ loadError }}</div>
    <div
      v-else
      class="pdf-viewer-scroll-container"
      :style="{ width: containerWidth + 'px', margin: '0 auto' }"
    >
      <div
        v-for="pg in pageCount"
        :key="pg"
        class="pdf-page-slot"
        :style="{ height: pageSlotHeight + 'px', marginBottom: PAGE_GAP + 'px' }"
      >
        <PdfPage
          v-if="isPageActive(pg)"
          :source="source"
          :page-num="pg"
          :width="containerWidth"
          @rendered="onPageRendered(pg)"
          @load-error="onLoadError"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import PdfPage from './PdfPage.vue'

const props = defineProps({
  /** PDF source URL */
  source: { type: String, required: true },
  /** Total number of pages in the document */
  pageCount: { type: Number, required: true },
  /** Page to open on initial mount (1-indexed) */
  initialPage: { type: Number, default: 1 },
  /** Zoom scale (1.0 = 100%) */
  zoomScale: { type: Number, default: 0.7 },
})

const emit = defineEmits(['update:currentPage', 'load-error', 'rendered'])

// ─── Constants ──────────────────────────────────────────────────────────────────
const BASE_WIDTH = 800
const PAGE_GAP = 20
const BUFFER = 3 // render activePage ± BUFFER pages (3 above, current, 3 below = 7 active pages)

// ─── Refs ───────────────────────────────────────────────────────────────────────
const viewportRef = ref(null)
const ready = ref(false)
const loadError = ref('')
const activePage = ref(props.initialPage)
// ponytail: default aspect ratio assumes US Letter; overwritten after first page renders
const pageAspectRatio = ref(11 / 8.5)
let scrollListenerActive = false
let resizeObserver = null
let parentWidth = ref(BASE_WIDTH)

// ─── DEBUG LOGGING ──────────────────────────────────────────────────────────────
console.log('[PdfViewer] INIT props:', {
  source: props.source,
  pageCount: props.pageCount,
  initialPage: props.initialPage,
  zoomScale: props.zoomScale,
})

// ─── Derived ────────────────────────────────────────────────────────────────────
const containerWidth = computed(() =>
  Math.round(Math.max(200, parentWidth.value * props.zoomScale))
)

const pageSlotHeight = computed(() =>
  Math.round(containerWidth.value * pageAspectRatio.value)
)

const totalSlotHeight = computed(() =>
  pageSlotHeight.value + PAGE_GAP
)

// ─── Virtualization ─────────────────────────────────────────────────────────────
function isPageActive(pg) {
  const active = Math.abs(pg - activePage.value) <= BUFFER
  if (active) {
    console.log('[PdfViewer] isPageActive:', pg, '→ ACTIVE (activePage:', activePage.value, ')')
  }
  return active
}

// ─── O(1) Scroll Tracking ───────────────────────────────────────────────────────
function pageFromScrollTop(scrollTop, viewportHeight) {
  // Which page's center is closest to the viewport center?
  const centerY = scrollTop + viewportHeight / 2
  const page = Math.floor(centerY / totalSlotHeight.value) + 1
  return Math.max(1, Math.min(page, props.pageCount))
}

function scrollTopForPage(page) {
  return (page - 1) * totalSlotHeight.value
}

let scrollRafId = null

function handleScroll() {
  if (scrollRafId) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = null
    const vp = viewportRef.value
    if (!vp) return
    const newPage = pageFromScrollTop(vp.scrollTop, vp.clientHeight)
    if (newPage !== activePage.value) {
      activePage.value = newPage
      emit('update:currentPage', newPage)
    }
  })
}

// ─── Instant Jump (no scrollIntoView) ───────────────────────────────────────────
function jumpToPage(page) {
  const vp = viewportRef.value
  if (!vp) return
  const target = scrollTopForPage(page)
  // Temporarily disable scroll listener to prevent feedback loop
  scrollListenerActive = false
  vp.scrollTop = target
  activePage.value = page
  // Re-enable after the browser settles
  requestAnimationFrame(() => {
    scrollListenerActive = true
  })
}

// ─── Page Rendered Callback ─────────────────────────────────────────────────────
function onPageRendered(pg) {
  emit('rendered', pg)
  if (!ready.value) {
    ready.value = true
  }
  // On first render of page 1, measure the actual aspect ratio from the canvas
  if (pg <= 2) {
    nextTick(() => {
      const vp = viewportRef.value
      if (!vp) return
      const canvas = vp.querySelector(`[data-page="${pg}"] canvas`)
      if (canvas && canvas.clientWidth > 0 && canvas.clientHeight > 0) {
        const measured = canvas.clientHeight / canvas.clientWidth
        // Only update if significantly different from default (avoids infinite re-render)
        if (Math.abs(measured - pageAspectRatio.value) > 0.01) {
          pageAspectRatio.value = measured
          // Re-jump to maintain position after aspect ratio update
          nextTick(() => jumpToPage(activePage.value))
        }
      }
    })
  }
}

function onLoadError(err) {
  const msg =
    typeof err === 'string'
      ? err
      : err?.message || (err && JSON.stringify(err)) || 'Failed to load PDF document.'
  loadError.value = msg
  emit('load-error', msg)
}

// ─── Scroll Event Binding ───────────────────────────────────────────────────────
function onViewportScroll() {
  if (!scrollListenerActive) return
  handleScroll()
}

// ─── Watch zoom changes → re-jump to maintain current page ──────────────────────
watch(
  () => props.zoomScale,
  () => {
    nextTick(() => jumpToPage(activePage.value))
  }
)

// ─── Watch source change → reset state ──────────────────────────────────────────
watch(
  () => props.source,
  () => {
    loadError.value = ''
    ready.value = false
    activePage.value = props.initialPage
    nextTick(() => jumpToPage(props.initialPage))
  }
)

// ─── Lifecycle ──────────────────────────────────────────────────────────────────
onMounted(() => {
  const vp = viewportRef.value
  console.log('[PdfViewer] onMounted:', {
    viewportExists: !!vp,
    clientWidth: vp?.clientWidth,
    clientHeight: vp?.clientHeight,
    source: props.source,
    pageCount: props.pageCount,
    initialPage: props.initialPage,
    activePage: activePage.value,
    containerWidth: containerWidth.value,
    pageSlotHeight: pageSlotHeight.value,
  })
  if (!vp) return

  // Measure available parent width for responsive sizing
  parentWidth.value = vp.clientWidth || BASE_WIDTH
  console.log('[PdfViewer] parentWidth set to:', parentWidth.value)

  // ResizeObserver for container width tracking
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      parentWidth.value = entry.contentRect.width || BASE_WIDTH
    }
  })
  resizeObserver.observe(vp)

  // Attach scroll listener
  vp.addEventListener('scroll', onViewportScroll, { passive: true })
  scrollListenerActive = true

  // Instant jump to initial page (synchronous scrollTop set, no animation)
  nextTick(() => jumpToPage(props.initialPage))
})

onUnmounted(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  if (scrollRafId) {
    cancelAnimationFrame(scrollRafId)
    scrollRafId = null
  }
  const vp = viewportRef.value
  if (vp) {
    vp.removeEventListener('scroll', onViewportScroll)
  }
})

// ─── Expose for parent to call programmatic navigation ──────────────────────────
defineExpose({ jumpToPage })
</script>

<style scoped>
.pdf-viewer-viewport {
  width: 100%;
  height: calc(100vh - 160px);
  overflow-y: auto;
  overflow-x: auto;
  background: var(--background);
  border-radius: 10px;
}

.pdf-viewer-scroll-container {
  /* Width set inline via containerWidth */
}

.pdf-page-slot {
  /* Height set inline via pageSlotHeight */
  width: 100%;
  background: var(--surface-container-lowest, #ffffff);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid var(--outline-variant);
  border-radius: 4px;
  overflow: hidden;
}

.pdf-viewer-error {
  color: #b42318;
  background: color-mix(in srgb, #b42318 12%, var(--surface-container-lowest));
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 10px;
  font-size: 13px;
}
</style>
