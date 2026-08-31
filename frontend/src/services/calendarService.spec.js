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
  it('generates a valid Google Calendar URL with daily recurrence and custom URL', () => {
    const url = getGoogleCalendarUrl('17:00', '19:00', 'https://my-school.edu/study')
    expect(url).toContain('https://calendar.google.com/calendar/render')
    expect(url).toContain('RRULE%3AFREQ%3DDAILY')
    expect(url).toContain('StudyLoop')
    expect(url).toContain('https%3A%2F%2Fmy-school.edu%2Fstudy')
  })

  it('generates a valid Outlook Web calendar URL with event details', () => {
    const url = getOutlookCalendarUrl('18:30', '20:00', 'https://custom-notes.com')
    expect(url).toContain('https://outlook.live.com/calendar/0/deeplink/compose')
    expect(url).toContain('path=%2Fcalendar%2Faction%2Fcompose')
    expect(url).toContain('rru=addevent')
    expect(url).toContain('subject=%F0%9F%93%96+StudyLoop+Daily+Study+Session')
    expect(url).toContain('startdt=')
    expect(url).toContain('enddt=')
    expect(url).toContain('https%3A%2F%2Fcustom-notes.com')
  })

  it('escapes RFC 5545 TEXT characters in strict order (backslashes, commas, semicolons, line breaks)', () => {
    const raw = 'Path\\to\\folder, with; semi\nand newlines\r\ntoo'
    const escaped = escapeICSText(raw)
    expect(escaped).toBe('Path\\\\to\\\\folder\\, with\\; semi\\nand newlines\\ntoo')

    const ics = generateRoutineICS('17:00', '19:00', 'https://foo.com/a\\b,c;d\ne')
    expect(ics).toContain('https://foo.com/a\\\\b\\,c\\;d\\ne')
  })

  it('triggers an .ics download without crashing', () => {
    const clickSpy = vi.fn()
    const appendSpy = vi.spyOn(document.body, 'appendChild').mockImplementation(() => {})
    const removeSpy = vi.spyOn(document.body, 'removeChild').mockImplementation(() => {})

    window.URL.createObjectURL = vi.fn(() => 'blob:mock-ics-url')
    window.URL.revokeObjectURL = vi.fn()

    const createElementOriginal = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag) => {
      const el = createElementOriginal(tag)
      if (tag === 'a') {
        el.click = clickSpy
      }
      return el
    })

    downloadRoutineICS('16:00', '17:30', 'https://portal.study.edu')

    expect(clickSpy).toHaveBeenCalled()
    expect(appendSpy).toHaveBeenCalled()
    expect(removeSpy).toHaveBeenCalled()
    expect(window.URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock-ics-url')
  })

  it('plays study chime without throwing when Web Audio API is available or unavailable', async () => {
    await expect(playStudyChime()).resolves.not.toThrow()
  })
})
