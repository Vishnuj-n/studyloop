<template>
  <Teleport to="body">
    <transition name="toast-slide">
      <div
        v-if="!setupModalState?.isOpen && activeSetup?.visibleToast && activeSetup?.extensionId && activeSetup?.status !== 'idle'"
        ref="toastContainer"
        class="ext-setup-toast"
        :class="{
          'is-running': activeSetup.status === 'running',
          'is-success': activeSetup.status === 'success',
          'is-error': activeSetup.status === 'error',
          'is-dragging': isDragging
        }"
        :style="containerStyle"
        @pointerdown="onPointerDown"
      >
        <div class="toast-drag-handle" title="Drag to reposition">
          <BaseIcon name="grid" size="13" />
        </div>

        <div class="toast-indicator">
          <div v-if="activeSetup.status === 'running'" class="toast-spinner"></div>
          <span v-else-if="activeSetup.status === 'success'" class="toast-icon-badge success">
            <BaseIcon name="check" size="14" />
          </span>
          <span v-else class="toast-icon-badge error">!</span>
        </div>

        <div class="toast-content">
          <div class="toast-header-row">
            <span class="toast-title">
              {{ activeSetup.extensionName }}
            </span>
            <span v-if="activeSetup.status === 'running'" class="toast-step-pill">
              Step {{ activeSetup.step }}/3
            </span>
          </div>

          <div class="toast-status-msg">
            {{ getStatusText() }}
          </div>
        </div>

        <div class="toast-actions" @pointerdown.stop>
          <button
            v-if="activeSetup.status === 'running' || activeSetup.status === 'error'"
            class="view-logs-btn"
            title="Open detailed logs"
            @click="handleViewLogs"
          >
            Logs
          </button>
          <button
            class="dismiss-btn"
            title="Dismiss notification"
            @click="dismissSetupToast"
          >
            <BaseIcon name="x" size="12" />
          </button>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { useExtensions } from '../composables/useExtensions'

const toastContainer = ref(null)
const isDragging = ref(false)
const position = ref({ x: null, y: null })

let dragStart = { x: 0, y: 0 }
let initialPos = { x: 0, y: 0 }
let hasMoved = false
const DRAG_THRESHOLD_PX = 4

const containerStyle = computed(() => {
  if (position.value.x !== null && position.value.y !== null) {
    return {
      left: `${position.value.x}px`,
      top: `${position.value.y}px`,
      right: 'auto',
      bottom: 'auto',
    }
  }
  return {
    right: '24px',
    bottom: '84px',
  }
})

function onPointerDown(e) {
  if (e.button !== 0) return
  isDragging.value = true
  hasMoved = false
  dragStart = { x: e.clientX, y: e.clientY }

  if (toastContainer.value) {
    const rect = toastContainer.value.getBoundingClientRect()
    initialPos = { x: rect.left, y: rect.top }
  }

  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
}

function onPointerMove(e) {
  if (!isDragging.value) return
  const dx = e.clientX - dragStart.x
  const dy = e.clientY - dragStart.y

  if (Math.abs(dx) > DRAG_THRESHOLD_PX || Math.abs(dy) > DRAG_THRESHOLD_PX) {
    hasMoved = true
  }

  if (!hasMoved) return

  const maxX = window.innerWidth - 340
  const maxY = window.innerHeight - 70
  const newX = Math.max(16, Math.min(maxX, initialPos.x + dx))
  const newY = Math.max(16, Math.min(maxY, initialPos.y + dy))

  position.value = { x: newX, y: newY }
}

function onPointerUp(e) {
  if (!isDragging.value) return
  isDragging.value = false
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
}

function handleResize() {
  if (position.value.x !== null && position.value.y !== null) {
    const maxX = window.innerWidth - 340
    const maxY = window.innerHeight - 70
    position.value.x = Math.max(16, Math.min(maxX, position.value.x))
    position.value.y = Math.max(16, Math.min(maxY, position.value.y))
  }
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
})

const {
  activeSetup,
  setupModalState,
  dismissSetupToast,
  openSetupModal,
  extensionsMetadata,
} = useExtensions()

function getStatusText() {
  if (activeSetup.value.status === 'running') {
    if (activeSetup.value.step === 1) return 'Setting up Python runtime...'
    if (activeSetup.value.step === 2) return 'Downloading dependencies...'
    if (activeSetup.value.step === 3) return 'Testing self-probe...'
    return 'Setting up in background...'
  }
  if (activeSetup.value.status === 'success') {
    return 'Extension is ready to use!'
  }
  if (activeSetup.value.status === 'error') {
    return activeSetup.value.errorMessage || 'Setup failed'
  }
  return ''
}

function handleViewLogs() {
  const ext = extensionsMetadata.value.find((e) => e.id === activeSetup.value.extensionId) || {
    id: activeSetup.value.extensionId,
    name: activeSetup.value.extensionName,
  }
  openSetupModal(ext)
}

// Auto-dismiss success toast after 6 seconds
watch(
  () => activeSetup.value.status,
  (newStatus) => {
    if (newStatus === 'success') {
      setTimeout(() => {
        if (activeSetup.value.status === 'success') {
          dismissSetupToast()
        }
      }, 6000)
    }
  }
)
</script>

<style scoped>
.ext-setup-toast {
  position: fixed;
  bottom: 84px; /* Default spawn position */
  right: 24px;
  z-index: 9999;
  background: var(--surface-container-lowest, #1e1e24);
  border: 1px solid var(--outline-variant, rgba(255, 255, 255, 0.12));
  border-radius: 14px;
  padding: 10px 14px;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 320px;
  max-width: 420px;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.18);
  backdrop-filter: blur(12px);
  pointer-events: auto;
  cursor: grab;
  user-select: none;
  touch-action: none;
}

.ext-setup-toast.is-dragging {
  cursor: grabbing;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.12);
  transform: scale(1.02);
}

.toast-drag-handle {
  color: var(--muted-text);
  opacity: 0.45;
  display: grid;
  place-items: center;
  margin-right: -2px;
  transition: opacity 0.15s ease;
}

.ext-setup-toast:hover .toast-drag-handle {
  opacity: 0.85;
}

.ext-setup-toast.is-running {
  border-color: rgba(59, 130, 246, 0.4);
}

.ext-setup-toast.is-success {
  border-color: rgba(46, 125, 50, 0.5);
  background: linear-gradient(135deg, var(--surface-container-lowest) 0%, rgba(46, 125, 50, 0.1) 100%);
}

.ext-setup-toast.is-error {
  border-color: rgba(239, 68, 68, 0.5);
  background: linear-gradient(135deg, var(--surface-container-lowest) 0%, rgba(239, 68, 68, 0.1) 100%);
}

.toast-indicator {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.toast-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--outline-variant);
  border-top-color: var(--primary, #3b82f6);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.toast-icon-badge {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 11px;
  font-weight: bold;
}

.toast-icon-badge.success {
  background: #2e7d32;
  color: #ffffff;
}

.toast-icon-badge.error {
  background: #ef4444;
  color: #ffffff;
}

.toast-content {
  flex: 1;
  min-width: 0;
}

.toast-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.toast-title {
  font-family: 'Manrope', sans-serif;
  font-size: 13px;
  font-weight: 600;
  color: var(--on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.toast-step-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 6px;
  background: var(--surface-container-high, rgba(255, 255, 255, 0.08));
  color: var(--primary, #3b82f6);
}

.toast-status-msg {
  font-size: 12px;
  color: var(--muted-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.toast-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.view-logs-btn {
  background: var(--surface-container-low);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  font-size: 11px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.view-logs-btn:hover {
  background: var(--surface-container);
  border-color: var(--primary);
}

.dismiss-btn {
  background: transparent;
  border: none;
  color: var(--muted-text);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  display: grid;
  place-items: center;
  transition: color 0.15s ease, background 0.15s ease;
}

.dismiss-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container-high);
}

.toast-slide-enter-active,
.toast-slide-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-slide-enter-from,
.toast-slide-leave-to {
  opacity: 0;
  transform: translateY(16px) scale(0.95);
}
</style>
