<template>
  <div
    v-if="petState.enabled"
    ref="petContainer"
    class="floating-pet-root"
    :style="containerStyle"
    @pointerdown="onPointerDown"
  >
    <!-- Speech Bubble -->
    <transition name="bubble-fade">
      <div
        v-if="bubbleText"
        class="pet-speech-bubble"
        @click.stop="hideBubble"
        @pointerdown.stop
      >
        <span class="bubble-content">{{ bubbleText }}</span>
        <div class="bubble-arrow"></div>
      </div>
    </transition>

    <!-- Pet Body & Animation Container -->
    <div
      class="pet-avatar-wrapper"
      :class="[
        `action-${currentAction}`,
        { 'is-dragging': isDragging }
      ]"
      @click="onPetClick"
      title="Click Mochi for encouragement or drag to move"
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
import { usePet } from '../composables/usePet'
import { PET_MESSAGES } from '../config/pets'

const { petState, currentSkin, setPosition } = usePet()

const petContainer = ref(null)
const currentAction = ref('idle')
const bubbleText = ref('')
const isDragging = ref(false)

let dragStart = { x: 0, y: 0 }
let initialPos = { x: 0, y: 0 }
let actionTimer = null
let actionResetTimer = null
let bubbleTimer = null
let hasMoved = false

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

function clearActionResetTimer() {
  if (actionResetTimer) {
    clearTimeout(actionResetTimer)
    actionResetTimer = null
  }
}

function triggerRandomAction() {
  clearActionResetTimer()
  const actions = ['idle', 'blink', 'wiggle', 'coffee', 'cheer', 'sleep']
  const next = actions[Math.floor(Math.random() * actions.length)]
  currentAction.value = next

  const duration = next === 'sleep' ? 6000 : next === 'coffee' ? 4000 : 2000
  actionResetTimer = setTimeout(() => {
    if (currentAction.value === next) {
      currentAction.value = 'idle'
    }
  }, duration)
}

function showSpeechBubble(msg = null) {
  if (bubbleTimer) clearTimeout(bubbleTimer)
  const text = msg || PET_MESSAGES[Math.floor(Math.random() * PET_MESSAGES.length)]
  bubbleText.value = text
  bubbleTimer = setTimeout(() => {
    bubbleText.value = ''
  }, 4500)
}

function hideBubble() {
  bubbleText.value = ''
  if (bubbleTimer) clearTimeout(bubbleTimer)
}

function onPetClick() {
  if (hasMoved) return
  clearActionResetTimer()
  currentAction.value = 'cheer'
  showSpeechBubble()
  actionResetTimer = setTimeout(() => {
    if (currentAction.value === 'cheer') {
      currentAction.value = 'idle'
    }
  }, 2200)
}

function onPointerDown(e) {
  if (e.button !== 0) return
  isDragging.value = true
  hasMoved = false
  dragStart = { x: e.clientX, y: e.clientY }

  const rect = petContainer.value.getBoundingClientRect()
  initialPos = { x: rect.left, y: rect.top }

  petContainer.value.setPointerCapture(e.pointerId)
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
}

function onPointerMove(e) {
  if (!isDragging.value) return
  const dx = e.clientX - dragStart.x
  const dy = e.clientY - dragStart.y

  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
    hasMoved = true
  }

  const maxX = window.innerWidth - 85
  const maxY = window.innerHeight - 85
  const newX = Math.max(10, Math.min(maxX, initialPos.x + dx))
  const newY = Math.max(10, Math.min(maxY, initialPos.y + dy))

  setPosition(newX, newY)
}

function onPointerUp(e) {
  if (!isDragging.value) return
  isDragging.value = false
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
}

function handleWindowResize() {
  if (petState.value.position.x !== null && petState.value.position.y !== null) {
    const maxX = window.innerWidth - 85
    const maxY = window.innerHeight - 85
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
  if (actionTimer) clearInterval(actionTimer)
  if (bubbleTimer) clearTimeout(bubbleTimer)
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
}

.pet-avatar-wrapper {
  width: 72px;
  height: 72px;
  cursor: grab;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  filter: drop-shadow(0 4px 10px rgba(0, 0, 0, 0.25));
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
  fill: rgba(0, 0, 0, 0.18);
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
  animation: petBounce 0.4s ease-in-out 4;
}

.action-cheer .pet-tail {
  animation: tailWag 0.2s ease-in-out infinite;
}

.steam-line {
  animation: steamRise 1.2s ease-out infinite;
}

@keyframes petBreathe {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-1.5px); }
}

@keyframes petBounce {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
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

/* Speech Bubble */
.pet-speech-bubble {
  position: absolute;
  bottom: 80px;
  right: -10px;
  min-width: 150px;
  max-width: 220px;
  background: var(--bg-surface, #1e293b);
  color: var(--text-primary, #f8fafc);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  padding: 8px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.35;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
  pointer-events: auto;
  cursor: pointer;
}

.bubble-arrow {
  position: absolute;
  bottom: -6px;
  right: 28px;
  width: 10px;
  height: 10px;
  background: var(--bg-surface, #1e293b);
  border-right: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  transform: rotate(45deg);
}

.bubble-fade-enter-active,
.bubble-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.bubble-fade-enter-from,
.bubble-fade-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.95);
}
</style>
