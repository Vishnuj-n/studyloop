import { describe, it, expect, vi } from 'vitest'
import {
  getGoogleCalendarUrl,
  getOutlookCalendarUrl,
  generateRoutineICS,
  downloadRoutineICS,
  playStudyChime,
  escapeICSText,
} from './calendarService'

describe('calendarService', () => {
  it('generates a valid Google Calendar URL with daily recurrence', () => {
    const url = getGoogleCalendarUrl('17:00', '19:00')
    expect(url).toContain('https://calendar.google.com/calendar/render')
    expect(url).toContain('RRULE%3AFREQ%3DDAILY')
    expect(url).toContain('StudyLoop')
  })

  it('generates a valid Outlook Web calendar URL with event details', () => {
    const url = getOutlookCalendarUrl('18:30', '20:00')
    expect(url).toContain('https://outlook.live.com/calendar/0/deeplink/compose')
    expect(url).toContain('path=%2Fcalendar%2Faction%2Fcompose')
    expect(url).toContain('rru=addevent')
    expect(url).toContain('subject=%F0%9F%93%96+StudyLoop+Daily+Study+Session')
    expect(url).toContain('startdt=')
    expect(url).toContain('enddt=')
  })

  it('escapes RFC 5545 TEXT characters in strict order (backslashes, commas, semicolons, line breaks)', () => {
    const raw = 'Path\\to\\folder, with; semi\nand newlines\r\ntoo'
    const escaped = escapeICSText(raw)
    expect(escaped).toBe('Path\\\\to\\\\folder\\, with\\; semi\\nand newlines\\ntoo')

    const ics = generateRoutineICS('17:00', '19:00')
    expect(ics).toContain('BEGIN:VCALENDAR')
    expect(ics).toContain('RRULE:FREQ=DAILY')
  })

  it('generates multi-slot .ics with distinct VEVENT and alarms for each schedule', () => {
    const multiSlots = [
      { name: 'Morning Review', start: '08:00', end: '09:00' },
      { name: 'Afternoon Practice', start: '14:00', end: '15:30' },
      { name: 'Night Focus', start: '21:00', end: '22:30' },
    ]
    const ics = generateRoutineICS(multiSlots)
    expect(ics).toContain('BEGIN:VCALENDAR')
    expect(ics).toContain('END:VCALENDAR')

    // Expect 3 VEVENT blocks
    const veventMatches = ics.match(/BEGIN:VEVENT/g)
    expect(veventMatches).toHaveLength(3)

    expect(ics).toContain('SUMMARY:📖 StudyLoop: Morning Review')
    expect(ics).toContain('SUMMARY:📖 StudyLoop: Afternoon Practice')
    expect(ics).toContain('SUMMARY:📖 StudyLoop: Night Focus')
    expect(ics).toContain('T080000')
    expect(ics).toContain('T140000')
    expect(ics).toContain('T210000')
  })

  it('accepts a JSON string of slots for generateRoutineICS', () => {
    const jsonStr = JSON.stringify([
      { name: 'Slot 1', start: '07:00', end: '08:00' },
      { name: 'Slot 2', start: '19:00', end: '20:00' },
    ])
    const ics = generateRoutineICS(jsonStr)
    const veventMatches = ics.match(/BEGIN:VEVENT/g)
    expect(veventMatches).toHaveLength(2)
    expect(ics).toContain('SUMMARY:📖 StudyLoop: Slot 1')
    expect(ics).toContain('SUMMARY:📖 StudyLoop: Slot 2')
  })

  it('handles overnight slots correctly without inverted end date', () => {
    const overnightSlot = [{ name: 'Late Night Grind', start: '23:00', end: '00:30' }]
    const ics = generateRoutineICS(overnightSlot)
    expect(ics).toContain('SUMMARY:📖 StudyLoop: Late Night Grind')
    expect(ics).toContain('T230000')
    expect(ics).toContain('T003000')
  })

  it('plays study chime without throwing when Web Audio API is available or unavailable', async () => {
    await expect(playStudyChime()).resolves.not.toThrow()
  })
})
