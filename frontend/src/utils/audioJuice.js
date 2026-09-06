// Web Audio API Procedural Sound Synthesizer
// Zero external MP3/WAV assets needed. 100% lightweight procedural sound synthesis.

let audioCtx = null

function getAudioContext() {
  if (typeof window === 'undefined') return null
  const AudioContextClass = window.AudioContext || window.webkitAudioContext
  if (!AudioContextClass) return null
  if (!audioCtx) {
    audioCtx = new AudioContextClass()
  }
  if (audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {})
  }
  return audioCtx
}

export function isAudioMuted() {
  try {
    return localStorage.getItem('studyloop_audio_muted') === 'true'
  } catch {
    return false
  }
}

export function setAudioMuted(muted) {
  try {
    localStorage.setItem('studyloop_audio_muted', muted ? 'true' : 'false')
  } catch {}
}

/**
 * Ascending chime chord (C5 -> E5 -> G5) for correct answers and positive reinforcement.
 */
export function playCorrectChime() {
  if (isAudioMuted()) return
  const ctx = getAudioContext()
  if (!ctx) return

  const notes = [523.25, 659.25, 783.99] // C5, E5, G5
  const now = ctx.currentTime

  notes.forEach((freq, idx) => {
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'sine'
    osc.frequency.setValueAtTime(freq, now + idx * 0.08)

    gain.gain.setValueAtTime(0.18, now + idx * 0.08)
    gain.gain.exponentialRampToValueAtTime(0.001, now + idx * 0.08 + 0.35)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start(now + idx * 0.08)
    osc.stop(now + idx * 0.08 + 0.4)
  })
}

/**
 * Low damped sine thud (130Hz -> 50Hz) for mistakes without harshness.
 */
export function playIncorrectThud() {
  if (isAudioMuted()) return
  const ctx = getAudioContext()
  if (!ctx) return

  const now = ctx.currentTime
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()

  osc.type = 'triangle'
  osc.frequency.setValueAtTime(130, now)
  osc.frequency.exponentialRampToValueAtTime(50, now + 0.25)

  gain.gain.setValueAtTime(0.2, now)
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.25)

  osc.connect(gain)
  gain.connect(ctx.destination)

  osc.start(now)
  osc.stop(now + 0.28)
}

/**
 * Rhythmic pitch-scaling tick when rating cards (Easy = high pitch, Hard = mid pitch).
 */
export function playCardRatingTick(rating) {
  if (isAudioMuted()) return
  const ctx = getAudioContext()
  if (!ctx) return

  const now = ctx.currentTime
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()

  let freq = 440
  if (rating === 'again' || rating === 1) freq = 260
  if (rating === 'hard' || rating === 2) freq = 370
  if (rating === 'good' || rating === 3) freq = 520
  if (rating === 'easy' || rating === 4) freq = 680

  osc.type = 'sine'
  osc.frequency.setValueAtTime(freq, now)

  gain.gain.setValueAtTime(0.15, now)
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.15)

  osc.connect(gain)
  gain.connect(ctx.destination)

  osc.start(now)
  osc.stop(now + 0.18)
}

/**
 * Wooden chest wobble / rattle click sound.
 */
export function playChestRattle() {
  if (isAudioMuted()) return
  const ctx = getAudioContext()
  if (!ctx) return

  const now = ctx.currentTime
  for (let i = 0; i < 3; i++) {
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'square'
    osc.frequency.setValueAtTime(180 + Math.random() * 40, now + i * 0.05)

    gain.gain.setValueAtTime(0.08, now + i * 0.05)
    gain.gain.exponentialRampToValueAtTime(0.001, now + i * 0.05 + 0.04)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start(now + i * 0.05)
    osc.stop(now + i * 0.05 + 0.05)
  }
}

/**
 * Celebratory triumphant fanfare (major arpeggio C5 -> G5 -> C6) on mystery chest reveal.
 */
export function playChestOpenFanfare() {
  if (isAudioMuted()) return
  const ctx = getAudioContext()
  if (!ctx) return

  const notes = [523.25, 659.25, 783.99, 1046.5] // C5, E5, G5, C6
  const now = ctx.currentTime

  notes.forEach((freq, idx) => {
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = idx === notes.length - 1 ? 'triangle' : 'sine'
    osc.frequency.setValueAtTime(freq, now + idx * 0.09)

    const duration = idx === notes.length - 1 ? 0.7 : 0.25
    gain.gain.setValueAtTime(0.22, now + idx * 0.09)
    gain.gain.exponentialRampToValueAtTime(0.001, now + idx * 0.09 + duration)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start(now + idx * 0.09)
    osc.stop(now + idx * 0.09 + duration + 0.05)
  })
}
