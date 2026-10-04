<template>
  <div v-if="stats && stats.is_compressed" ref="containerRef" class="compression-badge-wrapper">
    <button
      type="button"
      class="compression-pill-btn"
      :class="{ 'is-open': isOpen }"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      title="Click to view LLM context compression details"
      @click.stop="toggleOpen"
    >
      <span class="pill-icon">
        <BaseIcon name="zap" size="13" />
      </span>
      <span class="pill-text">Compressed</span>
      <span v-if="savingsPercent > 0" class="pill-savings-tag">
        -{{ savingsPercent }}%
      </span>
      <span class="pill-chevron" :class="{ 'rotate': isOpen }">
        <BaseIcon name="chevron-down" size="11" />
      </span>
    </button>

    <!-- Expandable Detailed Popover -->
    <Transition name="compression-popover-fade">
      <div
        v-if="isOpen"
        class="compression-popover-card"
        role="dialog"
        aria-label="Prompt Compression Breakdown"
        @click.stop
      >
        <div class="popover-header">
          <div class="header-title-group">
            <div class="header-icon-badge">
              <BaseIcon name="sparkles" size="14" />
            </div>
            <div>
              <div class="header-title">Prompt Optimization</div>
              <div class="header-subtitle">Extractive token reduction active</div>
            </div>
          </div>
          <button
            type="button"
            class="popover-close-btn"
            aria-label="Close details"
            @click.stop="isOpen = false"
          >
            &times;
          </button>
        </div>

        <div class="popover-metrics-grid">
          <div class="metric-item">
            <span class="metric-label">Original Prompt</span>
            <span class="metric-val">{{ (stats.raw_tokens || 0).toLocaleString() }} <small>tokens</small></span>
          </div>
          <div class="metric-item">
            <span class="metric-label">Optimized Prompt</span>
            <span class="metric-val highlight">{{ (stats.compressed_tokens || 0).toLocaleString() }} <small>tokens</small></span>
          </div>
          <div class="metric-item full-width">
            <span class="metric-label">Token Savings</span>
            <div class="savings-val-row">
              <span class="saved-count">{{ (stats.tokens_saved || 0).toLocaleString() }} tokens pruned</span>
              <span class="saved-badge">{{ savingsPercent }}% reduced</span>
            </div>
          </div>
        </div>

        <!-- Visual Bar Track -->
        <div class="savings-bar-container">
          <div class="savings-bar-track">
            <div
              class="savings-bar-fill"
              :style="{ width: `${Math.max(10, 100 - savingsPercent)}%` }"
              title="Compressed context size relative to original"
            ></div>
          </div>
          <div class="savings-bar-legend">
            <span>{{ Math.max(10, 100 - savingsPercent) }}% Sent to AI</span>
            <span>{{ savingsPercent }}% Trimmed</span>
          </div>
        </div>

        <div class="popover-footer-note">
          <BaseIcon name="info" size="12" />
          <span v-if="stats.chunk_count && stats.compressed_chunk_count && stats.compressed_chunk_count < stats.chunk_count">
            Batch progress: {{ stats.compressed_chunk_count }} / {{ stats.chunk_count }} chunks compressed ({{ activeSavingsPercent }}% pruned on compressed chunks).
          </span>
          <span v-else>
            Removes redundant context &amp; filler words to lower inference latency while preserving 100% key concepts and formulas.
          </span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'

const props = defineProps({
  stats: {
    type: Object,
    default: () => ({
      is_compressed: false,
      raw_tokens: 0,
      compressed_tokens: 0,
      tokens_saved: 0,
      saved_percentage: 0,
    }),
  },
})

const isOpen = ref(false)
const containerRef = ref(null)

const savingsPercent = computed(() => {
  if (!props.stats || !props.stats.saved_percentage) return 0
  return Math.round(props.stats.saved_percentage)
})

const activeSavingsPercent = computed(() => {
  if (!props.stats || !props.stats.active_saved_percentage) return savingsPercent.value
  return Math.round(props.stats.active_saved_percentage)
})

function toggleOpen() {
  isOpen.value = !isOpen.value
}

function handleOutsideClick(event) {
  if (containerRef.value && !containerRef.value.contains(event.target)) {
    isOpen.value = false
  }
}

function handleKeydown(event) {
  if (event.key === 'Escape' && isOpen.value) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleOutsideClick)
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('click', handleOutsideClick)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.compression-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  user-select: none;
  font-family: inherit;
}

/* Main Pill Trigger Button */
.compression-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px 3px 8px;
  background: var(--surface-container-high, rgba(142, 180, 70, 0.12));
  border: 1px solid var(--outline-variant, rgba(142, 180, 70, 0.25));
  border-radius: 9999px;
  color: var(--on-surface, #3c4820);
  font-size: 11.5px;
  font-weight: 600;
  line-height: 1.2;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
}

.compression-pill-btn:hover {
  background: var(--surface-container-highest, rgba(142, 180, 70, 0.2));
  border-color: var(--primary, #6b8e23);
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
}

.compression-pill-btn.is-open {
  background: var(--surface-container-highest, #f0f4e8);
  border-color: var(--primary, #6b8e23);
  box-shadow: 0 0 0 2px rgba(107, 142, 35, 0.2);
}

.pill-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--primary, #6b8e23);
}

.pill-text {
  letter-spacing: 0.01em;
}

.pill-savings-tag {
  background: rgba(107, 142, 35, 0.16);
  color: var(--primary, #4a6b10);
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.pill-chevron {
  display: inline-flex;
  align-items: center;
  color: var(--muted-text, #707b66);
  transition: transform 0.2s ease;
  margin-left: -2px;
}

.pill-chevron.rotate {
  transform: rotate(180deg);
}

/* Detailed Popover Card */
.compression-popover-card {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 100;
  width: 290px;
  padding: 14px;
  background: var(--surface-container-lowest, #ffffff);
  border: 1px solid var(--outline-variant, rgba(0, 0, 0, 0.12));
  border-radius: 12px;
  box-shadow: 0 10px 25px -4px rgba(0, 0, 0, 0.14), 0 4px 8px -2px rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(12px);
  cursor: default;
}

/* Header */
.popover-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--outline-variant, rgba(0, 0, 0, 0.06));
}

.header-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 7px;
  background: rgba(107, 142, 35, 0.12);
  color: var(--primary, #6b8e23);
}

.header-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--on-surface, #1e2518);
  line-height: 1.2;
}

.header-subtitle {
  font-size: 10.5px;
  color: var(--muted-text, #687360);
}

.popover-close-btn {
  background: transparent;
  border: none;
  font-size: 18px;
  line-height: 1;
  color: var(--muted-text, #889280);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.popover-close-btn:hover {
  background: var(--surface-container-high, rgba(0, 0, 0, 0.05));
  color: var(--on-surface, #1e2518);
}

/* Metrics Grid */
.popover-metrics-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-bottom: 12px;
}

.metric-item {
  background: var(--surface-container-low, rgba(0, 0, 0, 0.03));
  border: 1px solid var(--outline-variant, rgba(0, 0, 0, 0.05));
  border-radius: 8px;
  padding: 7px 9px;
  display: flex;
  flex-direction: column;
}

.metric-item.full-width {
  grid-column: span 2;
}

.metric-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-weight: 600;
  color: var(--muted-text, #687360);
  margin-bottom: 2px;
}

.metric-val {
  font-size: 13.5px;
  font-weight: 700;
  color: var(--on-surface, #1e2518);
}

.metric-val small {
  font-size: 10px;
  font-weight: 500;
  color: var(--muted-text, #889280);
}

.metric-val.highlight {
  color: var(--primary, #4a6b10);
}

.savings-val-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.saved-count {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--on-surface, #1e2518);
}

.saved-badge {
  background: rgba(46, 125, 50, 0.14);
  color: var(--primary, #2e7d32);
  padding: 1px 6px;
  border-radius: 6px;
  font-size: 10.5px;
  font-weight: 700;
}

/* Savings Track */
.savings-bar-container {
  margin-bottom: 12px;
}

.savings-bar-track {
  width: 100%;
  height: 6px;
  background: rgba(107, 142, 35, 0.18);
  border-radius: 999px;
  overflow: hidden;
  display: flex;
}

.savings-bar-fill {
  height: 100%;
  background: var(--primary, #6b8e23);
  border-radius: 999px;
  transition: width 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.savings-bar-legend {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 10px;
  font-weight: 600;
  color: var(--muted-text, #687360);
}

/* Note Footer */
.popover-footer-note {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 10.5px;
  line-height: 1.35;
  color: var(--muted-text, #687360);
  background: var(--surface-container-lowest, rgba(0, 0, 0, 0.02));
  padding: 6px 8px;
  border-radius: 6px;
  border-left: 2px solid var(--primary, #6b8e23);
}

.popover-footer-note svg {
  flex-shrink: 0;
  margin-top: 1px;
}

/* Animations */
.compression-popover-fade-enter-active,
.compression-popover-fade-leave-active {
  transition: all 0.18s cubic-bezier(0.16, 1, 0.3, 1);
}

.compression-popover-fade-enter-from,
.compression-popover-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.97);
}
</style>
