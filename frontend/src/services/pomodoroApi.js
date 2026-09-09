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

// ── Profiles & Settings ─────────────────────────────────────────────────────

export function loadPomodoroProfiles() {
  return appBridge().PomodoroLoadProfiles()
}

export function savePomodoroProfile(profile) {
  return appBridge().PomodoroSaveProfile(profile)
}

export function deletePomodoroProfile(id) {
  return appBridge().PomodoroDeleteProfile(id)
}

export function getPomodoroSettings() {
  return appBridge().PomodoroGetSettings()
}

export function savePomodoroSettings(settings) {
  return appBridge().PomodoroSaveSettings(settings)
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
