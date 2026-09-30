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
      <svg
        viewBox="0 0 100 100"
        class="pet-svg"
        xmlns="http://www.w3.org/2000/svg"
      >
        <!-- Shadow -->
        <ellipse cx="50" cy="88" rx="28" ry="6" class="pet-shadow" />

        <!-- Tail -->
        <path
          d="M 24 72 C 14 70 10 58 12 50 C 14 54 20 62 26 66"
          :fill="currentSkin.accentColor"
          class="pet-tail"
        />

        <!-- Main Body -->
        <path
          d="M 26 52 C 26 36 36 28 50 28 C 64 28 74 36 74 52 C 74 68 70 82 50 82 C 30 82 26 68 26 52 Z"
          :fill="currentSkin.primaryColor"
          class="pet-body"
        />

        <!-- Belly Patch -->
        <ellipse
          cx="50"
          cy="62"
          rx="15"
          ry="14"
          :fill="currentSkin.secondaryColor"
          class="pet-belly"
        />

        <!-- Left Ear -->
        <polygon
          points="30,34 38,14 48,30"
          :fill="currentSkin.primaryColor"
          class="pet-ear ear-left"
        />
        <polygon
          points="33,31 38,18 45,28"
          :fill="currentSkin.secondaryColor"
          class="pet-ear-inner"
        />

        <!-- Right Ear -->
        <polygon
          points="70,34 62,14 52,30"
          :fill="currentSkin.primaryColor"
          class="pet-ear ear-right"
        />
        <polygon
          points="67,31 62,18 55,28"
          :fill="currentSkin.secondaryColor"
          class="pet-ear-inner"
        />

        <!-- Eyes: Normal vs Blink vs Sleep -->
        <g v-if="currentAction === 'sleep'" class="eyes-sleeping">
          <path d="M 38 46 Q 42 50 46 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
          <path d="M 54 46 Q 58 50 62 46" fill="none" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
        </g>
        <g v-else-if="currentAction === 'blink'" class="eyes-blink">
          <line x1="38" y1="46" x2="46" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
          <line x1="54" y1="46" x2="62" y2="46" stroke="#1E293B" stroke-width="2.5" stroke-linecap="round" />
        </g>
        <g v-else class="eyes-open">
          <ellipse cx="42" cy="45" rx="3.5" ry="4.5" fill="#1E293B" />
          <circle cx="43.5" cy="43.5" r="1.5" fill="#FFFFFF" />
          <ellipse cx="58" cy="45" rx="3.5" ry="4.5" fill="#1E293B" />
          <circle cx="59.5" cy="43.5" r="1.5" fill="#FFFFFF" />
        </g>

        <!-- Nose -->
        <polygon points="48,51 52,51 50,54" fill="#F43F5E" />

        <!-- Mouth -->
        <path
          d="M 46 55 Q 50 58 50 55 Q 50 58 54 55"
          fill="none"
          stroke="#1E293B"
          stroke-width="1.5"
          stroke-linecap="round"
        />

        <!-- Whiskers -->
        <line x1="28" y1="48" x2="36" y2="50" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
        <line x1="28" y1="54" x2="36" y2="53" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
        <line x1="64" y1="50" x2="72" y2="48" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />
        <line x1="64" y1="53" x2="72" y2="54" stroke="#64748B" stroke-width="1.2" stroke-linecap="round" />

        <!-- Coffee Mug (Shown during 'coffee' action) -->
        <g v-if="currentAction === 'coffee'" class="pet-coffee-mug">
          <rect x="44" y="66" width="12" height="11" rx="2" fill="#E2E8F0" stroke="#475569" stroke-width="1" />
          <path d="M 56 68 Q 60 71 56 74" fill="none" stroke="#475569" stroke-width="1" />
          <path d="M 48 63 Q 50 60 50 58" fill="none" stroke="#94A3B8" stroke-width="1" stroke-linecap="round" class="steam-line" />
        </g>

        <!-- Front Paws -->
        <ellipse cx="40" cy="78" rx="5" ry="4" :fill="currentSkin.secondaryColor" />
        <ellipse cx="60" cy="78" rx="5" ry="4" :fill="currentSkin.secondaryColor" />
      </svg>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import BaseIcon from './BaseIcon.vue'
import { usePet } from '../composables/usePet'
import { PET_REGISTRY, PET_MESSAGES } from '../config/pets'

const { petState, currentSkin, setPosition, togglePet } = usePet()

// Dev mode: auto-detected from Vite's build mode
const isDev = import.meta.env.DEV
const allActions = PET_REGISTRY.find((p) => p.id === 'cat')?.actions ?? ['idle', 'blink', 'wiggle', 'coffee', 'cheer', 'sleep']
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

.pet-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
}

.pet-shadow {
  fill: rgba(0, 0, 0, 0.14);
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

.steam-line {
  animation: steamRise 1.2s ease-out infinite;
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

@keyframes steamRise {
  0% { opacity: 0; transform: translateY(0); }
  50% { opacity: 0.8; }
  100% { opacity: 0; transform: translateY(-4px); }
}

/* ── Dismiss / Close Button ────────────────────────────── */
/* Small close button on top-right corner of Mochi */
.pet-dismiss-btn {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--outline-variant);
  background: var(--surface-container, #1e293b);
  color: var(--muted-text, #94a3b8);
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
  background: #ef4444;
  border-color: #f87171;
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
  background: var(--surface-container, #1e293b);
  color: var(--muted-text, #64748b);
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
  background: #334155;
  color: #f1f5f9;
  border-color: #6366f1;
}

.dev-toggle-btn.active {
  background: #6366f1;
  border-color: #818cf8;
  color: #fff;
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
  background: var(--surface-container-high, #0f172a);
  border: 1px solid var(--outline-variant);
  border-radius: 10px;
  padding: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14);
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
  color: #64748b;
  font-size: 10px;
  cursor: pointer;
  padding: 0 2px;
  line-height: 1;
}

.dev-close-btn:hover {
  color: #f1f5f9;
}

.dev-badge {
  font-size: 9px;
  font-weight: 700;
  background: #f43f5e;
  color: #fff;
  padding: 1px 5px;
  border-radius: 4px;
  letter-spacing: 0.05em;
}

.dev-action-label {
  font-size: 11px;
  color: #94a3b8;
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
  background: var(--surface-container, #1e293b);
  color: #cbd5e1;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.dev-action-btn:hover {
  background: #334155;
  color: #f1f5f9;
}

.dev-action-btn.active {
  background: #6366f1;
  border-color: #818cf8;
  color: #fff;
}

.dev-msg-btn {
  background: #0f766e;
  border-color: #14b8a6;
  color: #ccfbf1;
}

.dev-msg-btn:hover {
  background: #115e59;
  color: #f0fdfa;
}

/* ── Speech / Thought Bubble ────────────────────────────── */
.pet-speech-bubble {
  position: absolute;
  bottom: 104px;
  left: 50%;
  transform: translateX(-50%);
  width: max-content;
  max-width: 170px;
  background: var(--surface-container-highest, #1e293b);
  border: 1px solid var(--outline-variant, #475569);
  color: var(--on-surface, #f8fafc);
  padding: 6px 10px;
  border-radius: 12px;
  font-size: 11px;
  line-height: 1.35;
  font-weight: 500;
  text-align: center;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18);
  cursor: pointer;
  pointer-events: auto;
  z-index: 10;
  white-space: normal;
  word-break: break-word;
  user-select: none;
}

.pet-speech-bubble:hover {
  filter: brightness(1.08);
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
  border-top: 6px solid var(--surface-container-highest, #1e293b);
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
