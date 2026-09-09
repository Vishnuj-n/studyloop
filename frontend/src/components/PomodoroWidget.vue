<template>
  <div
    v-if="enabled"
    class="pomodoro-capsule"
    :class="{ 'is-running': isRunning, 'is-break': sessionType === 'break' }"
    :title="sessionType === 'break' ? 'Break Session' : 'Focus Session'"
  >
    <!-- Left: Status indicator dot & Tabular Countdown -->
    <div class="capsule-left" @click="handlePlayPause">
      <span class="pulse-dot" :class="{ active: isRunning }"></span>
      <span class="capsule-time">{{ formattedTime }}</span>
      <span v-if="trackName && isAudioPlaying" class="audio-mini-waves" :title="trackName">
        <span></span><span></span><span></span>
      </span>
    </div>

    <!-- Right: Minimalist Action Glyph Row -->
    <div class="capsule-right">
      <!-- Mute / Audio Toggle -->
      <button
        type="button"
        class="glyph-btn"
        :class="{ active: !isMuted && isAudioPlaying }"
        :title="isMuted ? 'Unmute Focus Audio' : 'Mute Focus Audio'"
        aria-label="Toggle audio mute"
        @click.stop="toggleMute"
      >
        <svg v-if="isMuted" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
          <line x1="23" y1="9" x2="17" y2="15"></line>
          <line x1="17" y1="9" x2="23" y2="15"></line>
        </svg>
        <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 5"></polygon>
          <path d="M15.54 8.46a5 5 0 0 1 0 7.07"></path>
        </svg>
      </button>

      <!-- Play / Pause Button -->
      <button
        type="button"
        class="glyph-btn primary-glyph"
        :title="isRunning ? 'Pause' : isPaused ? 'Resume' : 'Start Focus'"
        :aria-label="isRunning ? 'Pause timer' : 'Start timer'"
        @click.stop="handlePlayPause"
      >
        <svg v-if="isRunning" width="11" height="11" viewBox="0 0 24 24" fill="currentColor">
          <rect x="6" y="4" width="4" height="16" rx="1.5"></rect>
          <rect x="14" y="4" width="4" height="16" rx="1.5"></rect>
        </svg>
        <svg v-else width="11" height="11" viewBox="0 0 24 24" fill="currentColor">
          <polygon points="6 4 20 12 6 20 6 4"></polygon>
        </svg>
      </button>

      <!-- Skip / Next Session Button -->
      <button
        v-if="isRunning || isPaused"
        type="button"
        class="glyph-btn"
        :title="sessionType === 'work' ? 'Skip to Break' : 'Skip to Focus'"
        aria-label="Skip session"
        @click.stop="handleSkip"
      >
        <svg width="11" height="11" viewBox="0 0 24 24" fill="currentColor">
          <polygon points="5 4 15 12 5 20 5 4"></polygon>
          <line x1="19" y1="5" x2="19" y2="19" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"></line>
        </svg>
      </button>

      <!-- Reset / Stop Button -->
      <button
        v-if="isRunning || isPaused"
        type="button"
        class="glyph-btn"
        title="Reset Timer"
        aria-label="Stop timer"
        @click.stop="handleStop"
      >
        <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
          <rect x="5" y="5" width="14" height="14" rx="2"></rect>
        </svg>
      </button>
    </div>

    <!-- Integrated Bottom Hairline Progress Glow -->
    <div class="capsule-progress-track">
      <div
        class="capsule-progress-fill"
        :style="{ width: progressPercent + '%' }"
      ></div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  startPomodoroTimer,
  pausePomodoroTimer,
  resumePomodoroTimer,
  stopPomodoroTimer,
  getPomodoroSettings,
  loadPomodoroProfiles,
  playPomodoroLooping,
  playPomodoroShuffleFolder,
  stopPomodoroAudio,
  setPomodoroVolume,
  recordPomodoroSessionComplete,
} from '../services/pomodoroApi'
import { playStudyChime } from '../services/calendarService'
import { EventsOn } from '../../wailsjs/runtime/runtime'

const enabled = ref(true)
const isRunning = ref(false)
const isPaused = ref(false)
const sessionType = ref('work') // 'work' | 'break'
const totalSec = ref(25 * 60)
const remainSec = ref(25 * 60)
const isMuted = ref(false)
const isAudioPlaying = ref(false)
const trackName = ref('')
const pomoSettings = ref({})
const currentProfile = ref(null)

const formattedTime = computed(() => {
  const m = Math.floor(remainSec.value / 60).toString().padStart(2, '0')
  const s = (remainSec.value % 60).toString().padStart(2, '0')
  return `${m}:${s}`
})

const progressPercent = computed(() => {
  if (totalSec.value <= 0) return 0
  return Math.min(100, Math.max(0, ((totalSec.value - remainSec.value) / totalSec.value) * 100))
})

async function loadConfiguration() {
  try {
    const s = await getPomodoroSettings()
    if (s) {
      pomoSettings.value = s
      if (typeof s.enabled !== 'undefined') {
        enabled.value = !!s.enabled
      }
    }
    const profiles = await loadPomodoroProfiles()
    if (Array.isArray(profiles) && profiles.length > 0) {
      const def = profiles.find((p) => p.isDefault) || profiles[0]
      currentProfile.value = def
      if (!isRunning.value && !isPaused.value) {
        totalSec.value = def.durationSec || 25 * 60
        remainSec.value = totalSec.value
      }
    }
  } catch (err) {
    console.warn('[POMODORO] Configuration load error:', err)
  }
}

async function startAudioForCurrentSession() {
  if (isMuted.value || !currentProfile.value) return
  const prof = currentProfile.value

  const musicPath = sessionType.value === 'break'
    ? (prof.breakMusicPath === '__none__' ? '' : prof.breakMusicPath || prof.musicPath)
    : prof.musicPath
  const shuffle = sessionType.value === 'break'
    ? (prof.breakMusicPath ? prof.breakShuffle : prof.shuffle)
    : prof.shuffle

  if (!musicPath) {
    await stopPomodoroAudio().catch(() => {})
    return
  }

  const vol = pomoSettings.value.defaultVolume ?? 70
  await setPomodoroVolume(vol).catch(() => {})

  if (shuffle) {
    await playPomodoroShuffleFolder(musicPath).catch(() => {})
  } else {
    await playPomodoroLooping(musicPath).catch(() => {})
  }
}

async function handlePlayPause() {
  if (isRunning.value) {
    isPaused.value = true
    isRunning.value = false
    await pausePomodoroTimer().catch(() => {})
    await stopPomodoroAudio().catch(() => {})
  } else if (isPaused.value) {
    isPaused.value = false
    isRunning.value = true
    await resumePomodoroTimer({
      profileId: currentProfile.value?.id || 'default',
      totalSec: totalSec.value,
      remainingSec: remainSec.value,
      savedAt: Math.floor(Date.now() / 1000),
    }).catch(() => {})
    await startAudioForCurrentSession()
  } else {
    isPaused.value = false
    isRunning.value = true
    sessionType.value = 'work'
    totalSec.value = currentProfile.value?.durationSec || 25 * 60
    remainSec.value = totalSec.value
    await startPomodoroTimer(currentProfile.value?.id || 'default', totalSec.value).catch(() => {})
    await startAudioForCurrentSession()
  }
}

async function handleStop() {
  isPaused.value = false
  isRunning.value = false
  sessionType.value = 'work'
  totalSec.value = currentProfile.value?.durationSec || 25 * 60
  remainSec.value = totalSec.value
  await stopPomodoroTimer().catch(() => {})
  await stopPomodoroAudio().catch(() => {})
}

async function handleSkip() {
  isPaused.value = false
  isRunning.value = false
  await stopPomodoroTimer().catch(() => {})
  await stopPomodoroAudio().catch(() => {})

  if (sessionType.value === 'work') {
    sessionType.value = 'break'
    totalSec.value = currentProfile.value?.breakDurationSec || 5 * 60
    remainSec.value = totalSec.value
    if (totalSec.value > 0) {
      isRunning.value = true
      await startPomodoroTimer((currentProfile.value?.id || 'default') + '-break', totalSec.value).catch(() => {})
      await startAudioForCurrentSession()
    }
  } else {
    sessionType.value = 'work'
    totalSec.value = currentProfile.value?.durationSec || 25 * 60
    remainSec.value = totalSec.value
  }
}

function toggleMute() {
  isMuted.value = !isMuted.value
  if (isMuted.value) {
    stopPomodoroAudio().catch(() => {})
  } else if (isRunning.value) {
    startAudioForCurrentSession()
  }
}

let unsubs = []

onMounted(async () => {
  await loadConfiguration()

  try {
    const unsubTick = EventsOn('timerTicked', (data) => {
      if (data && typeof data.remainingSec === 'number') {
        remainSec.value = data.remainingSec
      }
    })
    if (typeof unsubTick === 'function') unsubs.push(unsubTick)

    const unsubDone = EventsOn('timerCompleted', async () => {
      isPaused.value = false
      isRunning.value = false
      playStudyChime()

      if (sessionType.value === 'work') {
        try {
          await recordPomodoroSessionComplete()
        } catch (_) {}

        if (currentProfile.value && currentProfile.value.breakDurationSec > 0) {
          sessionType.value = 'break'
          totalSec.value = currentProfile.value.breakDurationSec
          remainSec.value = totalSec.value
          isRunning.value = true
          await startPomodoroTimer((currentProfile.value.id || 'default') + '-break', totalSec.value).catch(() => {})
          await startAudioForCurrentSession()
          return
        }
      } else {
        sessionType.value = 'work'
        totalSec.value = currentProfile.value ? currentProfile.value.durationSec : 25 * 60
        remainSec.value = totalSec.value
      }
    })
    if (typeof unsubDone === 'function') unsubs.push(unsubDone)

    const unsubAudio = EventsOn('audioStateChanged', (data) => {
      if (!data) return
      isAudioPlaying.value = data.state === 'playing'
      trackName.value = data.trackName || ''
    })
    if (typeof unsubAudio === 'function') unsubs.push(unsubAudio)
  } catch (err) {
    console.warn('[POMODORO] EventsOn subscription error:', err)
  }

  window.addEventListener('pomodoro-settings-updated', loadConfiguration)
})

onUnmounted(() => {
  unsubs.forEach((fn) => {
    try { fn() } catch (_) {}
  })
  window.removeEventListener('pomodoro-settings-updated', loadConfiguration)
})
</script>

<style scoped>
.pomodoro-capsule {
  position: relative;
  width: 100%;
  box-sizing: border-box;
  padding: 8px 12px 10px;
  background: color-mix(in srgb, var(--surface-container-low) 50%, transparent);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  overflow: hidden;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.pomodoro-capsule:hover {
  background: var(--surface-container-low);
  border-color: color-mix(in srgb, var(--primary) 30%, var(--outline-variant));
}

.pomodoro-capsule.is-running {
  border-color: color-mix(in srgb, var(--primary) 40%, transparent);
}

.pomodoro-capsule.is-break.is-running {
  border-color: rgba(16, 185, 129, 0.4);
}

/* Left side (time & pulse) */
.capsule-left {
  display: flex;
  align-items: center;
  gap: 7px;
  cursor: pointer;
  user-select: none;
}

.pulse-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--muted-text);
  opacity: 0.5;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.pulse-dot.active {
  background: var(--primary);
  opacity: 1;
  box-shadow: 0 0 6px var(--primary);
  animation: mini-pulse 2s infinite ease-in-out;
}

.is-break .pulse-dot.active {
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
}

@keyframes mini-pulse {
  0%, 100% { transform: scale(1); opacity: 0.8; }
  50% { transform: scale(1.35); opacity: 1; }
}

.capsule-time {
  font-family: 'JetBrains Mono', monospace;
  font-size: 15px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--on-surface);
  font-feature-settings: 'tnum' 1, 'zero' 1;
  line-height: 1;
}

.is-running .capsule-time {
  color: var(--primary);
}

.is-break.is-running .capsule-time {
  color: #10b981;
}

/* Mini Audio Waves */
.audio-mini-waves {
  display: inline-flex;
  align-items: flex-end;
  gap: 1.5px;
  height: 8px;
}

.audio-mini-waves span {
  width: 1.5px;
  height: 3px;
  background: var(--primary);
  border-radius: 1px;
  animation: wave 0.8s infinite alternate ease-in-out;
}

.audio-mini-waves span:nth-child(1) { animation-delay: 0s; }
.audio-mini-waves span:nth-child(2) { animation-delay: 0.2s; }
.audio-mini-waves span:nth-child(3) { animation-delay: 0.4s; }

@keyframes wave {
  0% { height: 2px; }
  100% { height: 8px; }
}

/* Right side action glyphs */
.capsule-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.glyph-btn {
  background: transparent;
  border: 1px solid transparent;
  color: var(--muted-text);
  border-radius: 6px;
  padding: 4px 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  line-height: 1;
}

.glyph-btn:hover {
  color: var(--on-surface);
  background: var(--surface-container);
  border-color: var(--outline-variant);
  transform: translateY(-1px);
}

.glyph-btn:active {
  transform: scale(0.92);
}

.glyph-btn.active {
  color: var(--primary);
}

.primary-glyph {
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border-color: color-mix(in srgb, var(--primary) 20%, transparent);
}

.primary-glyph:hover {
  background: var(--primary);
  color: var(--on-primary);
  border-color: var(--primary);
}

/* Bottom Hairline Progress Glow */
.capsule-progress-track {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: color-mix(in srgb, var(--outline-variant) 40%, transparent);
  overflow: hidden;
}

.capsule-progress-fill {
  height: 100%;
  background: var(--primary);
  transition: width 0.3s ease;
  box-shadow: 0 0 6px var(--primary);
}

.is-break .capsule-progress-fill {
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
}
</style>
