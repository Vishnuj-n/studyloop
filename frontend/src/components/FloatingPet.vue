<template>
  <div
    v-if="petState.enabled"
    ref="petContainer"
    class="floating-pet-root"
    :style="containerStyle"
    @pointerdown="onPointerDown"
  >
    <!-- DEV MODE toggle button — only in dev builds -->
    <button
      v-if="isDev"
      class="dev-toggle-btn"
      :class="{ active: showDevPanel }"
      :title="showDevPanel ? 'Hide dev panel' : 'Show dev panel'"
      @click.stop="showDevPanel = !showDevPanel"
      @pointerdown.stop
    >
      <BaseIcon name="settings" size="11" />
    </button>

    <!-- Close / Dismiss button (hides Mochi) -->
    <button
      class="pet-dismiss-btn"
      title="Close Mochi (can re-enable in Settings)"
      @click.stop="togglePet(false)"
      @pointerdown.stop
    >
      <BaseIcon name="x" size="10" />
    </button>

    <!-- DEV MODE PANEL — hidden until toggle is clicked -->
    <transition name="dev-panel-fade">
      <div
        v-if="isDev && showDevPanel"
        class="pet-dev-panel"
        @pointerdown.stop
      >
        <div class="dev-panel-header">
          <span class="dev-badge">DEV</span>
          <span class="dev-action-label">{{ currentAction }}</span>
          <button class="dev-close-btn" @click.stop="showDevPanel = false">
            <BaseIcon name="x" size="10" />
          </button>
        </div>
        <div class="dev-action-buttons">
          <button
            v-for="action in allActions"
            :key="action"
            class="dev-action-btn"
            :class="{ active: currentAction === action }"
            @click.stop="forceAction(action)"
          >
            {{ action }}
          </button>
          <button
            class="dev-action-btn dev-msg-btn"
            @click.stop="triggerSpeechBubble()"
          >
            💬 Say
          </button>
        </div>
      </div>
    </transition>

    <!-- Speech / Thought Bubble -->
    <transition name="pet-bubble-fade">
      <div
        v-if="activeMessage"
        class="pet-speech-bubble"
        @click.stop="dismissBubble"
        @pointerdown.stop
      >
        <span class="bubble-text">{{ activeMessage }}</span>
        <div class="bubble-tail"></div>
      </div>
    </transition>

    <!-- Pet Body & Animation Container -->
    <div
      class="pet-avatar-wrapper"
      :class="[
        `action-${currentAction}`,
        { 'is-dragging': isDragging }
      ]"
      title="Click to interact · Drag to move"
      @click="onPetClick"
    >
      <PetAvatar
        :pet-id="petState.activePetId"
        :skin="currentSkin"
        :action="currentAction"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import PetAvatar from './PetAvatar.vue'
import { usePet } from '../composables/usePet'
import { PET_REGISTRY, PET_MESSAGES } from '../config/pets'

const { petState, currentSkin, setPosition, togglePet } = usePet()

// Dev mode: auto-detected from Vite's build mode
const isDev = import.meta.env.DEV
const allActions = computed(() => {
  return PET_REGISTRY.find((p) => p.id === petState.value.activePetId)?.actions ?? ['idle', 'blink', 'wiggle', 'coffee', 'cheer', 'sleep']
})
const showDevPanel = ref(false)

const petContainer = ref(null)
const currentAction = ref('idle')
const isDragging = ref(false)

// Speech bubble state
const activeMessage = ref(null)
let bubbleTimer = null
let lastMessageIdx = -1

let dragStart = { x: 0, y: 0 }
let initialPos = { x: 0, y: 0 }
let actionTimer = null
let actionResetTimer = null
// Raised from 3→8px: micro hand-tremor was suppressing clicks
let hasMoved = false
const DRAG_THRESHOLD_PX = 8

const containerStyle = computed(() => {
  if (petState.value.position.x !== null && petState.value.position.y !== null) {
    return {
      left: `${petState.value.position.x}px`,
      top: `${petState.value.position.y}px`,
      right: 'auto',
      bottom: 'auto',
    }
  }
  return {
    right: '24px',
    bottom: '24px',
  }
})

// Petting click reactions: only happy petting responses (bounce/wiggle)
const PETTING_ACTIONS = ['cheer', 'wiggle']
// Ambient idle actions: behaviors performed spontaneously while studying
const AMBIENT_ACTIONS = ['idle', 'blink', 'wiggle', 'coffee', 'sleep']

const ACTION_DURATIONS = {
  sleep: 6000,
  coffee: 4000,
  wiggle: 1800,
  blink: 1200,
  cheer: 1600,
}
const DEV_ACTION_DURATIONS = {
  sleep: 8000,
  coffee: 5000,
}
let lastAction = ''

function clearActionResetTimer() {
  if (actionResetTimer) {
    clearTimeout(actionResetTimer)
    actionResetTimer = null
  }
}

function clearBubbleTimer() {
  if (bubbleTimer) {
    clearTimeout(bubbleTimer)
    bubbleTimer = null
  }
}

function triggerSpeechBubble(customText = null) {
  clearBubbleTimer()
  if (customText) {
    activeMessage.value = customText
  } else if (PET_MESSAGES && PET_MESSAGES.length > 0) {
    let nextIdx = Math.floor(Math.random() * PET_MESSAGES.length)
    if (PET_MESSAGES.length > 1 && nextIdx === lastMessageIdx) {
      nextIdx = (nextIdx + 1) % PET_MESSAGES.length
    }
    lastMessageIdx = nextIdx
    activeMessage.value = PET_MESSAGES[nextIdx]
  }

  // Auto-dismiss after 4 seconds
  bubbleTimer = setTimeout(() => {
    activeMessage.value = null
    bubbleTimer = null
  }, 4000)
}

function dismissBubble() {
  clearBubbleTimer()
  activeMessage.value = null
}

function triggerRandomAction() {
  clearActionResetTimer()
  const next = AMBIENT_ACTIONS[Math.floor(Math.random() * AMBIENT_ACTIONS.length)]
  currentAction.value = next

  const duration = ACTION_DURATIONS[next] || 2000
  actionResetTimer = setTimeout(() => {
    if (currentAction.value === next) {
      currentAction.value = 'idle'
    }
  }, duration)
}

// Interactive petting reaction (Bounce / Wiggle)
function triggerPettingReaction() {
  clearActionResetTimer()
  const available = PETTING_ACTIONS.filter((a) => a !== lastAction)
  const next = available[Math.floor(Math.random() * available.length)] || PETTING_ACTIONS[0]
  lastAction = next
  currentAction.value = next

  // ~20% chance on click to show a friendly study message bubble if none active
  if (!activeMessage.value && Math.random() < 0.22) {
    triggerSpeechBubble()
  }

  const duration = ACTION_DURATIONS[next] || 1600
  actionResetTimer = setTimeout(() => {
    if (currentAction.value === next) {
      currentAction.value = 'idle'
    }
  }, duration)
}

// Click handler
function onPetClick() {
  if (hasMoved) {
    return
  }
  triggerPettingReaction()
}

// DEV: force any action directly from the panel
function forceAction(action) {
  clearActionResetTimer()
  currentAction.value = action
  const duration = DEV_ACTION_DURATIONS[action] || 3000
  actionResetTimer = setTimeout(() => {
    if (currentAction.value === action) {
      currentAction.value = 'idle'
    }
  }, duration)
}

function onPointerDown(e) {
  if (e.button !== 0) return
  isDragging.value = true
  hasMoved = false
  dragStart = { x: e.clientX, y: e.clientY }

  const rect = petContainer.value.getBoundingClientRect()
  initialPos = { x: rect.left, y: rect.top }

  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
}

function onPointerMove(e) {
  if (!isDragging.value) return
  const dx = e.clientX - dragStart.x
  const dy = e.clientY - dragStart.y

  if (Math.abs(dx) > DRAG_THRESHOLD_PX || Math.abs(dy) > DRAG_THRESHOLD_PX) {
    hasMoved = true
    try {
      if (petContainer.value && !petContainer.value.hasPointerCapture(e.pointerId)) {
        petContainer.value.setPointerCapture(e.pointerId)
      }
    } catch {
      // pointer capture unavailable
    }
  }

  if (!hasMoved) return

  const maxX = window.innerWidth - 110
  const maxY = window.innerHeight - 110
  const newX = Math.max(10, Math.min(maxX, initialPos.x + dx))
  const newY = Math.max(10, Math.min(maxY, initialPos.y + dy))

  setPosition(newX, newY)
}

function onPointerUp(e) {
  if (!isDragging.value) return
  isDragging.value = false
  try {
    if (petContainer.value && petContainer.value.hasPointerCapture(e.pointerId)) {
      petContainer.value.releasePointerCapture(e.pointerId)
    }
  } catch {
    // pointer release unavailable
  }
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
}

function handleWindowResize() {
  if (petState.value.position.x !== null && petState.value.position.y !== null) {
    const maxX = window.innerWidth - 110
    const maxY = window.innerHeight - 110
    const clampedX = Math.max(10, Math.min(maxX, petState.value.position.x))
    const clampedY = Math.max(10, Math.min(maxY, petState.value.position.y))
    if (clampedX !== petState.value.position.x || clampedY !== petState.value.position.y) {
      setPosition(clampedX, clampedY)
    }
  }
}

onMounted(() => {
  window.addEventListener('resize', handleWindowResize)
  actionTimer = setInterval(() => {
    if (Math.random() > 0.35) {
      triggerRandomAction()
    }
  }, 12000)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleWindowResize)
  clearActionResetTimer()
  clearBubbleTimer()
  if (actionTimer) clearInterval(actionTimer)
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
})
</script>

<style scoped>
.floating-pet-root {
  position: fixed;
  z-index: 9999;
  user-select: none;
  touch-action: none;
  /* needed for dev-toggle-btn and dev-panel absolute positioning */
  isolation: isolate;
}

.pet-avatar-wrapper {
  width: 100px;
  height: 100px;
  cursor: grab;
  position: relative;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  filter: drop-shadow(0 4px 10px rgba(0, 0, 0, 0.12));
}

.pet-avatar-wrapper:hover {
  transform: scale(1.08);
}

.pet-avatar-wrapper.is-dragging {
  cursor: grabbing;
  transform: scale(1.12);
}

/* Animations */
.action-idle .pet-body {
  animation: petBreathe 3s ease-in-out infinite;
}

.action-wiggle .pet-ear.ear-left {
  animation: earTwitchLeft 0.5s ease-in-out 3;
}

.action-wiggle .pet-ear.ear-right {
  animation: earTwitchRight 0.5s ease-in-out 3;
}

.action-cheer {
  animation: petBounce 0.35s ease-in-out 4;
}

.action-cheer .pet-tail {
  animation: tailWag 0.2s ease-in-out infinite;
}

@keyframes petBreathe {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-2px); }
}

@keyframes petBounce {
  0%, 100% { transform: translateY(0) scale(1); }
  40% { transform: translateY(-14px) scale(1.05); }
  70% { transform: translateY(-6px) scale(0.98); }
}

@keyframes earTwitchLeft {
  0%, 100% { transform: rotate(0deg); transform-origin: 38px 25px; }
  50% { transform: rotate(-12deg); transform-origin: 38px 25px; }
}

@keyframes earTwitchRight {
  0%, 100% { transform: rotate(0deg); transform-origin: 62px 25px; }
  50% { transform: rotate(12deg); transform-origin: 62px 25px; }
}

@keyframes tailWag {
  0%, 100% { transform: rotate(0deg); transform-origin: 26px 66px; }
  50% { transform: rotate(-20deg); transform-origin: 26px 66px; }
}

/* ── Dismiss / Close Button ────────────────────────────── */
/* Small close button on top-right corner of pet */
.pet-dismiss-btn {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-lowest);
  color: var(--muted-text);
  font-size: 10px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  pointer-events: auto;
  z-index: 2;
  opacity: 0;
  transition: opacity 0.2s, background 0.15s, color 0.15s, border-color 0.15s;
}

.floating-pet-root:hover .pet-dismiss-btn {
  opacity: 1;
}

.pet-dismiss-btn:hover {
  background: var(--danger, #ef4444);
  border-color: var(--danger, #ef4444);
  color: #ffffff;
}

/* ── Dev Panel ─────────────────────────────────────────── */

/* Toggle button: settings pill, top-left corner of the pet */
.dev-toggle-btn {
  position: absolute;
  top: -8px;
  left: -8px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-lowest);
  color: var(--muted-text);
  font-size: 10px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  pointer-events: auto;
  z-index: 2;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.dev-toggle-btn:hover {
  background: var(--surface-container-high);
  color: var(--on-surface);
  border-color: var(--primary);
}

.dev-toggle-btn.active {
  background: var(--primary);
  border-color: var(--primary);
  color: var(--on-primary);
}

/* Panel slide-fade */
.dev-panel-fade-enter-active,
.dev-panel-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.dev-panel-fade-enter-from,
.dev-panel-fade-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.97);
}

.pet-dev-panel {
  position: absolute;
  bottom: 108px;
  right: 0;
  width: 160px;
  background: var(--surface-container);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  padding: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  font-family: ui-monospace, monospace;
  pointer-events: auto;
  z-index: 1;
}

.dev-panel-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.dev-close-btn {
  margin-left: auto;
  background: none;
  border: none;
  color: var(--muted-text);
  font-size: 10px;
  cursor: pointer;
  padding: 0 2px;
  line-height: 1;
}

.dev-close-btn:hover {
  color: var(--on-surface);
}

.dev-badge {
  font-size: 9px;
  font-weight: 700;
  background: var(--primary);
  color: var(--on-primary);
  padding: 1px 5px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}

.dev-action-label {
  font-size: 11px;
  color: var(--muted-text);
  flex: 1;
  text-align: right;
}

.dev-action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.dev-action-btn {
  font-size: 10px;
  font-family: inherit;
  padding: 3px 6px;
  border-radius: 5px;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface-variant);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.dev-action-btn:hover {
  background: var(--surface-container-lowest);
  color: var(--on-surface);
}

.dev-action-btn.active {
  background: var(--primary);
  border-color: var(--primary);
  color: var(--on-primary);
}

.dev-msg-btn {
  background: color-mix(in srgb, #0f766e 25%, var(--surface-container-low));
  border-color: color-mix(in srgb, #14b8a6 40%, transparent);
  color: #0d9488;
}

.dev-msg-btn:hover {
  background: color-mix(in srgb, #0f766e 40%, var(--surface-container-low));
}

/* ── Speech / Thought Bubble ────────────────────────────── */
.pet-speech-bubble {
  position: absolute;
  bottom: 104px;
  left: 50%;
  transform: translateX(-50%);
  width: max-content;
  max-width: 170px;
  background: var(--surface-container-highest);
  border: 1px solid var(--outline-variant);
  color: var(--on-surface);
  padding: 6px 10px;
  border-radius: 12px;
  font-size: 11px;
  line-height: 1.35;
  font-weight: 500;
  text-align: center;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.12);
  cursor: pointer;
  pointer-events: auto;
  z-index: 10;
  white-space: normal;
  word-break: break-word;
  user-select: none;
}

.pet-speech-bubble:hover {
  filter: brightness(1.05);
}

.bubble-text {
  display: block;
}

.bubble-tail {
  position: absolute;
  bottom: -6px;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-top: 6px solid var(--surface-container-highest);
}

/* Bubble pop-fade transitions */
.pet-bubble-fade-enter-active {
  animation: bubblePopIn 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.pet-bubble-fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.pet-bubble-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(4px) scale(0.95);
}

@keyframes bubblePopIn {
  0% {
    opacity: 0;
    transform: translateX(-50%) translateY(8px) scale(0.85);
  }
  100% {
    opacity: 1;
    transform: translateX(-50%) translateY(0) scale(1);
  }
}
</style>
