// pomodoroApi.js
// Frontend bridge for Pomodoro timer, audio playback, profiles, and settings.

function appBridge() {
  const bridge = window?.go?.app?.App || window?.go?.main?.App
  if (!bridge) {
    throw new Error('Wails backend bridge unavailable')
  }
  return bridge
}

// ── Timer ───────────────────────────────────────────────────────────────────

export function startPomodoroTimer(profileID, durationSec) {
  return appBridge().PomodoroStartTimer(profileID || 'default', durationSec || 25 * 60)
}

export function resumePomodoroTimer(state) {
  return appBridge().PomodoroResumeTimer(state)
}

export function pausePomodoroTimer() {
  return appBridge().PomodoroPauseTimer()
}

export function stopPomodoroTimer() {
  return appBridge().PomodoroStopTimer()
}

export function getPomodoroTimerState() {
  return appBridge().PomodoroGetTimerState()
}

export function checkResumePomodoroSession() {
  return appBridge().PomodoroCheckResumeSession()
}

// ── Audio ───────────────────────────────────────────────────────────────────

export function playPomodoroLooping(filePath) {
  return appBridge().PomodoroPlayLooping(filePath)
}

export function playPomodoroShuffleFolder(folder) {
  return appBridge().PomodoroPlayShuffleFolder(folder)
}

export function stopPomodoroAudio() {
  return appBridge().PomodoroStopAudio()
}

export function setPomodoroVolume(volume) {
  return appBridge().PomodoroSetVolume(volume)
}

export function getPomodoroAudioState() {
  return appBridge().PomodoroGetAudioState()
}

export function playPomodoroChime() {
  return appBridge().PomodoroPlayChime()
}

// ── Profiles & Settings (Client + Bridge Fallback) ──────────────────────────

const LOCAL_STORAGE_PROFILES_KEY = 'studyloop_pomodoro_profiles'
const LOCAL_STORAGE_SETTINGS_KEY = 'studyloop_pomodoro_settings'

const DEFAULT_POMO_SETTINGS = {
  enabled: true,
  defaultVolume: 70,
  autoStartAudio: true,
  notifyOnComplete: true,
  autoStartNextTimer: false,
  playSoundOnComplete: true,
  theme: 'dark',
}

const DEFAULT_POMO_PROFILES = [
  {
    id: 'default',
    name: 'Standard Focus',
    durationSec: 25 * 60,
    breakDurationSec: 5 * 60,
    musicPath: '',
    shuffle: false,
    breakMusicPath: '',
    breakShuffle: false,
    isDefault: true,
  },
]

export function loadPomodoroProfiles() {
  try {
    const raw = localStorage.getItem(LOCAL_STORAGE_PROFILES_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length > 0) {
        return Promise.resolve(parsed)
      }
    }
  } catch (err) {
    console.warn('[POMODORO_API] Failed to parse stored profiles:', err)
  }
  return Promise.resolve(DEFAULT_POMO_PROFILES)
}

export function savePomodoroProfile(profile) {
  try {
    let profiles = []
    const raw = localStorage.getItem(LOCAL_STORAGE_PROFILES_KEY)
    if (raw) {
      profiles = JSON.parse(raw) || []
    }
    const idx = profiles.findIndex((p) => p.id === profile.id)
    if (idx >= 0) {
      profiles[idx] = { ...profiles[idx], ...profile }
    } else {
      profiles.push(profile)
    }
    localStorage.setItem(LOCAL_STORAGE_PROFILES_KEY, JSON.stringify(profiles))
  } catch (err) {
    console.error('[POMODORO_API] Failed to save profile:', err)
  }
  return Promise.resolve(null)
}

export function deletePomodoroProfile(id) {
  try {
    let profiles = []
    const raw = localStorage.getItem(LOCAL_STORAGE_PROFILES_KEY)
    if (raw) {
      profiles = JSON.parse(raw) || []
    }
    profiles = profiles.filter((p) => p.id !== id)
    localStorage.setItem(LOCAL_STORAGE_PROFILES_KEY, JSON.stringify(profiles))
  } catch (err) {
    console.error('[POMODORO_API] Failed to delete profile:', err)
  }
  return Promise.resolve(null)
}

export function getPomodoroSettings() {
  try {
    const raw = localStorage.getItem(LOCAL_STORAGE_SETTINGS_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (parsed && typeof parsed === 'object') {
        return Promise.resolve({ ...DEFAULT_POMO_SETTINGS, ...parsed })
      }
    }
  } catch (err) {
    console.warn('[POMODORO_API] Failed to parse stored settings:', err)
  }
  return Promise.resolve(DEFAULT_POMO_SETTINGS)
}

export function savePomodoroSettings(settings) {
  try {
    const current = localStorage.getItem(LOCAL_STORAGE_SETTINGS_KEY)
    const parsed = current ? JSON.parse(current) : {}
    const updated = { ...DEFAULT_POMO_SETTINGS, ...parsed, ...settings }
    localStorage.setItem(LOCAL_STORAGE_SETTINGS_KEY, JSON.stringify(updated))
  } catch (err) {
    console.error('[POMODORO_API] Failed to save settings:', err)
  }
  return Promise.resolve(null)
}


export function getPomodoroStats() {
  return appBridge().PomodoroGetStats()
}

export function recordPomodoroSessionComplete() {
  return appBridge().PomodoroRecordSessionComplete()
}

export function pickPomodoroMusicFile() {
  return appBridge().PomodoroPickMusicFile()
}

export function pickPomodoroMusicFolder() {
  return appBridge().PomodoroPickMusicFolder()
}
