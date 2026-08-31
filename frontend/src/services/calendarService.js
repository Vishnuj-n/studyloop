/**
 * calendarService.js
 * Utilities for syncing study routine to external calendars (Google Calendar, Outlook Web, Apple/Windows .ics)
 * and synthesizing in-app audio chimes using the Web Audio API.
 */

/**
 * Synthesizes a gentle, pleasant 2-tone chime using Web Audio API.
 * Zero external MP3/audio files required.
 */
let retainedAudioCtx = null

export function initAudioContext() {
  if (!retainedAudioCtx) {
    const AudioContextClass = typeof window !== 'undefined' && (window.AudioContext || window.webkitAudioContext)
    if (AudioContextClass) {
      retainedAudioCtx = new AudioContextClass()
    }
  }
  if (retainedAudioCtx && retainedAudioCtx.state === 'suspended') {
    void retainedAudioCtx.resume()
  }
  return retainedAudioCtx
}

/**
 * Synthesizes a gentle, pleasant 2-tone chime using Web Audio API.
 * Zero external MP3/audio files required.
 */
export async function playStudyChime() {
  try {
    const ctx = initAudioContext()
    if (!ctx) return

    if (ctx.state === 'suspended') {
      await ctx.resume()
    }

    const now = ctx.currentTime

    // First tone (D5 - 587.33 Hz)
    const osc1 = ctx.createOscillator()
    const gain1 = ctx.createGain()
    osc1.type = 'sine'
    osc1.frequency.setValueAtTime(587.33, now)
    gain1.gain.setValueAtTime(0.12, now)
    gain1.gain.exponentialRampToValueAtTime(0.001, now + 0.35)
    osc1.connect(gain1)
    gain1.connect(ctx.destination)
    osc1.start(now)
    osc1.stop(now + 0.35)

    // Second tone (A5 - 880.00 Hz)
    const osc2 = ctx.createOscillator()
    const gain2 = ctx.createGain()
    osc2.type = 'sine'
    osc2.frequency.setValueAtTime(880.0, now + 0.15)
    gain2.gain.setValueAtTime(0.15, now + 0.15)
    gain2.gain.exponentialRampToValueAtTime(0.001, now + 0.65)
    osc2.connect(gain2)
    gain2.connect(ctx.destination)
    osc2.start(now + 0.15)
    osc2.stop(now + 0.65)
  } catch (err) {
    console.warn('Unable to play audio chime:', err)
  }
}

/**
 * Formats start and end times into ISO strings and UTC date components.
 */
function getEventDates(startTime = '17:00', endTime = '19:00') {
  const [startH, startM] = (startTime || '17:00').split(':').map((s) => s.padStart(2, '0'))
  const [endH, endM] = (endTime || '19:00').split(':').map((s) => s.padStart(2, '0'))

  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')

  const dtStart = `${y}${m}${d}T${startH}${startM}00`
  const dtEnd = `${y}${m}${d}T${endH}${endM}00`

  // ISO formats for Outlook
  const isoStart = `${y}-${m}-${d}T${startH}:${startM}:00`
  const isoEnd = `${y}-${m}-${d}T${endH}:${endM}:00`

  return { dtStart, dtEnd, isoStart, isoEnd, y, m, d }
}

/**
 * Escapes text for RFC 5545 iCalendar TEXT values in strict order:
 * 1. Backslashes
 * 2. Commas
 * 3. Semicolons
 * 4. Line breaks
 */
export function escapeICSText(str = '') {
  return str
    .replace(/\\/g, '\\\\')
    .replace(/,/g, '\\,')
    .replace(/;/g, '\\;')
    .replace(/\r\n|\r|\n/g, '\\n')
}

const DEFAULT_STUDY_DESCRIPTION = 'Time for your daily StudyLoop study session! Clear your flashcard queue and complete reading tasks.'

/**
 * Returns a direct URL to create a recurring daily event on Google Calendar.
 */
export function getGoogleCalendarUrl(startTime = '17:00', endTime = '19:00') {
  const { dtStart, dtEnd } = getEventDates(startTime, endTime)
  const title = '📖 StudyLoop Daily Study Session'

  const params = new URLSearchParams({
    action: 'TEMPLATE',
    text: title,
    dates: `${dtStart}/${dtEnd}`,
    recur: 'RRULE:FREQ=DAILY',
    details: DEFAULT_STUDY_DESCRIPTION,
  })

  return `https://calendar.google.com/calendar/render?${params.toString()}`
}

/**
 * Generates raw iCalendar (.ics) string with daily recurrence and alarms.
 */
export function generateRoutineICS(startTime = '17:00', endTime = '19:00') {
  const { dtStart, dtEnd, y, m, d } = getEventDates(startTime, endTime)
  const title = '📖 StudyLoop Daily Study Session'
  const details = escapeICSText(DEFAULT_STUDY_DESCRIPTION)

  const icsLines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//StudyLoop//Daily Study Routine//EN',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
    'BEGIN:VEVENT',
    `UID:studyloop-daily-${Date.now()}@studyloop.app`,
    `DTSTAMP:${y}${m}${d}T000000Z`,
    `DTSTART:${dtStart}`,
    `DTEND:${dtEnd}`,
    'RRULE:FREQ=DAILY',
    `SUMMARY:${title}`,
    `DESCRIPTION:${details}`,
    'STATUS:CONFIRMED',
    'BEGIN:VALARM',
    'TRIGGER:-PT10M',
    'ACTION:DISPLAY',
    'DESCRIPTION:StudyLoop reminder: 10 minutes until study session begins!',
    'END:VALARM',
    'BEGIN:VALARM',
    'TRIGGER:PT0M',
    'ACTION:DISPLAY',
    'DESCRIPTION:StudyLoop: Time to start your daily study routine!',
    'END:VALARM',
    'END:VEVENT',
    'END:VCALENDAR',
  ]

  return icsLines.join('\r\n')
}

/**
 * Returns a direct URL to compose an event on Outlook Web / Live.
 */
export function getOutlookCalendarUrl(startTime = '17:00', endTime = '19:00') {
  const { isoStart, isoEnd } = getEventDates(startTime, endTime)
  const title = '📖 StudyLoop Daily Study Session'

  const params = new URLSearchParams({
    path: '/calendar/action/compose',
    rru: 'addevent',
    subject: title,
    body: DEFAULT_STUDY_DESCRIPTION,
    startdt: isoStart,
    enddt: isoEnd,
  })

  return `https://outlook.live.com/calendar/0/deeplink/compose?${params.toString()}`
}

/**
 * Generates and triggers download of a standard RFC 5545 .ics file
 * with daily recurrence (RRULE:FREQ=DAILY) and 10-minute + 0-minute audio alarms.
 */
export function downloadRoutineICS(startTime = '17:00', endTime = '19:00') {
  const icsContent = generateRoutineICS(startTime, endTime)
  const icsBlob = new Blob([icsContent], { type: 'text/calendar;charset=utf-8' })
  const url = URL.createObjectURL(icsBlob)
  const link = document.createElement('a')
  link.href = url
  link.setAttribute('download', 'studyloop-study-routine.ics')
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
