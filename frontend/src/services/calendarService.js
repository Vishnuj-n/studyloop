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
/**
 * Formats start and end times into ISO strings and UTC date components.
 * Handles overnight schedules (where end time < start time) by rolling end date to next day.
 */
function getEventDates(startTime = '17:00', endTime = '19:00') {
  const [startH, startM] = (startTime || '17:00').split(':').map((s) => s.padStart(2, '0'))
  const [endH, endM] = (endTime || '19:00').split(':').map((s) => s.padStart(2, '0'))

  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')

  const dtStart = `${y}${m}${d}T${startH}${startM}00`

  // If end time is earlier than or equal to start time, it rolls over to the next day
  const isOvernight = Number(endH) * 60 + Number(endM) <= Number(startH) * 60 + Number(startM)
  let endY = y
  let endMStr = m
  let endD = d

  if (isOvernight) {
    const nextDay = new Date(now.getTime() + 86400000)
    endY = nextDay.getFullYear()
    endMStr = String(nextDay.getMonth() + 1).padStart(2, '0')
    endD = String(nextDay.getDate()).padStart(2, '0')
  }

  const dtEnd = `${endY}${endMStr}${endD}T${endH}${endM}00`

  // ISO formats for Outlook
  const isoStart = `${y}-${m}-${d}T${startH}:${startM}:00`
  const isoEnd = `${endY}-${endMStr}-${endD}T${endH}:${endM}:00`

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
 * Supports an array of study slots [{ name, start, end }], a JSON string, or single startTime/endTime fallback.
 */
export function generateRoutineICS(slotsOrStart = '17:00', maybeEnd = '18:00') {
  let slots = []
  if (Array.isArray(slotsOrStart) && slotsOrStart.length > 0) {
    slots = slotsOrStart
  } else if (typeof slotsOrStart === 'string') {
    if (slotsOrStart.trim().startsWith('[') || slotsOrStart.trim().startsWith('{')) {
      try {
        const parsed = JSON.parse(slotsOrStart)
        if (Array.isArray(parsed) && parsed.length > 0) {
          slots = parsed
        }
      } catch {
        slots = [{ name: 'Daily Study Session', start: slotsOrStart, end: maybeEnd }]
      }
    } else {
      slots = [{ name: 'Daily Study Session', start: slotsOrStart, end: maybeEnd }]
    }
  }

  if (slots.length === 0) {
    slots = [{ name: 'Daily Study Session', start: '17:00', end: '18:00' }]
  }

  const details = escapeICSText(DEFAULT_STUDY_DESCRIPTION)
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  const timestamp = `${y}${m}${d}T000000Z`
  const baseTime = Date.now()

  const icsLines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//StudyLoop//Daily Study Routine//EN',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
  ]

  slots.forEach((slot, idx) => {
    const slotStart = slot.start || '17:00'
    const slotEnd = slot.end || '18:00'
    const { dtStart, dtEnd } = getEventDates(slotStart, slotEnd)
    const slotName = slot.name ? `📖 StudyLoop: ${slot.name}` : `📖 StudyLoop Study Session ${idx + 1}`
    const uniqueUID = `studyloop-slot-${idx + 1}-${baseTime}-${Math.floor(Math.random() * 10000)}@studyloop.app`

    icsLines.push(
      'BEGIN:VEVENT',
      `UID:${uniqueUID}`,
      `DTSTAMP:${timestamp}`,
      `DTSTART:${dtStart}`,
      `DTEND:${dtEnd}`,
      'RRULE:FREQ=DAILY',
      `SUMMARY:${escapeICSText(slotName)}`,
      `DESCRIPTION:${details}`,
      'STATUS:CONFIRMED',
      'BEGIN:VALARM',
      'TRIGGER:-PT10M',
      'ACTION:DISPLAY',
      `DESCRIPTION:StudyLoop reminder: 10 minutes until ${escapeICSText(slot.name || 'study session')} begins!`,
      'END:VALARM',
      'BEGIN:VALARM',
      'TRIGGER:PT0M',
      'ACTION:DISPLAY',
      `DESCRIPTION:StudyLoop: Time for your ${escapeICSText(slot.name || 'study session')}!`,
      'END:VALARM',
      'END:VEVENT'
    )
  })

  icsLines.push('END:VCALENDAR')
  return icsLines.join('\r\n')
}

/**
 * Returns a direct URL to compose an event on Outlook Web / Live.
 */
export function getOutlookCalendarUrl(startTime = '17:00', endTime = '19:00', titleName = '') {
  const { isoStart, isoEnd } = getEventDates(startTime, endTime)
  const title = titleName ? `📖 StudyLoop: ${titleName}` : '📖 StudyLoop Daily Study Session'

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
 * with daily recurrence (RRULE:FREQ=DAILY) for all configured study slots.
 */
export function downloadRoutineICS(slotsOrStart = '17:00', maybeEnd = '19:00') {
  const icsContent = generateRoutineICS(slotsOrStart, maybeEnd)
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

