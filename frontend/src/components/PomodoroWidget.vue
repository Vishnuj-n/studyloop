<template>
  <div v-if="enabled" class="pomodoro-sidebar-widget">
    <div class="pomodoro-header">
      <span class="pomodoro-mode-badge" :class="{ 'is-break': sessionType === 'break' }">
        {{ sessionType === 'break' ? '☕ Break' : '🎯 Focus' }}
      </span>
      <button
        type="button"
        class="pomodoro-mute-btn"
        :title="isMuted ? 'Unmute Focus Audio' : 'Mute Focus Audio'"
        @click="toggleMute"
      >
        {{ isMuted ? '🔇' : '🔊' }}
      </button>
    </div>

    <!-- Timer Countdown and Progress -->
    <div class="pomodoro-main">
      <span class="pomodoro-time">{{ formattedTime }}</span>
      <div class="pomodoro-bar-bg">
        <div class="pomodoro-bar-fill" :style="{ width: progressPercent + '%' }"></div>
      </div>
    </div>

    <!-- Actions: Play/Pause, Stop/Reset, Skip -->
    <div class="pomodoro-controls">
      <button
        type="button"
        class="pomo-btn primary-pomo-btn"
        :title="isRunning ? 'Pause' : isPaused ? 'Resume' : 'Start Focus'"
        @click="handlePlayPause"
      >
        {{ isRunning ? '⏸' : '▶' }}
      </button>
      <button
        v-if="isRunning || isPaused"
        type="button"
        class="pomo-btn secondary-pomo-btn"
        title="Stop Session"
        @click="handleStop"
      >
        ⏹
      </button>
      <button
        v-if="isRunning || isPaused"
        type="button"
        class="pomo-btn secondary-pomo-btn"
        title="Skip to Next"
        @click="handleSkip"
      >
        ⏭
      </button>
    </div>

    <!-- Music Track Info -->
    <div v-if="trackName" class="pomodoro-track" :title="trackName">
      <span class="track-dot" :class="{ active: isAudioPlaying }"></span>
      <span class="track-title">{{ trackName }}</span>
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
  getPomodoroTimerState,
  checkResumePomodoroSession,
  playPomodoroLooping,
  playPomodoroShuffleFolder,
  stopPomodoroAudio,
  setPomodoroVolume,
  getPomodoroSettings,
  loadPomodoroProfiles,
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
    // Pause
    isPaused.value = true
    isRunning.value = false
    await pausePomodoroTimer().catch(() => {})
    await stopPomodoroAudio().catch(() => {})
  } else if (isPaused.value) {
    // Resume
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
    // Start fresh
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
    // Switch to break
    sessionType.value = 'break'
    totalSec.value = currentProfile.value?.breakDurationSec || 5 * 60
    remainSec.value = totalSec.value
    if (totalSec.value > 0) {
      isRunning.value = true
      await startPomodoroTimer((currentProfile.value?.id || 'default') + '-break', totalSec.value).catch(() => {})
      await startAudioForCurrentSession()
    }
  } else {
    // Switch back to work
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

  // Listen for backend ticker events
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

        // Transition to break if configured
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
        // Break is over, reset to work
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
.pomodoro-sidebar-widget {
  margin-top: 10px;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--on-surface) 4%, transparent);
  border: 1px solid var(--outline-variant);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

.pomodoro-sidebar-widget:hover {
  background: var(--surface-container-low);
  border-color: color-mix(in srgb, var(--primary) 35%, var(--outline-variant));
}

.pomodoro-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.pomodoro-mode-badge {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  padding: 2px 7px;
  border-radius: 6px;
}

.pomodoro-mode-badge.is-break {
  color: #10b981;
  background: rgba(16, 185, 129, 0.12);
}

.pomodoro-mute-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 12px;
  padding: 2px 4px;
  opacity: 0.7;
  transition: opacity 0.2s ease;
}

.pomodoro-mute-btn:hover {
  opacity: 1;
}

.pomodoro-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pomodoro-time {
  font-family: 'JetBrains Mono', monospace, sans-serif;
  font-size: 18px;
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: -0.02em;
  text-align: center;
}

.pomodoro-bar-bg {
  height: 4px;
  background: var(--outline-variant);
  border-radius: 999px;
  overflow: hidden;
}

.pomodoro-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--primary-dim), var(--primary));
  transition: width 0.3s ease;
}

.pomodoro-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.pomo-btn {
  border: 1px solid var(--outline-variant);
  background: var(--surface-container-low);
  color: var(--on-surface);
  border-radius: 8px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.pomo-btn:hover {
  background: var(--surface-container-highest);
  color: var(--primary);
  border-color: var(--primary);
  transform: translateY(-1px);
}

.primary-pomo-btn {
  background: var(--primary);
  color: var(--on-primary);
  border-color: var(--primary);
  font-weight: 700;
  flex: 1;
}

.primary-pomo-btn:hover {
  background: var(--primary-dim, var(--primary));
  color: var(--on-primary);
}

.secondary-pomo-btn {
  padding: 4px 8px;
}

.pomodoro-track {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--muted-text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.track-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--outline-variant);
}

.track-dot.active {
  background: #10b981;
  box-shadow: 0 0 6px #10b981;
}

.track-title {
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
