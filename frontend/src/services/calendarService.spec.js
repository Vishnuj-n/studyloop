import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getGoogleCalendarUrl,
  getOutlookCalendarUrl,
  downloadRoutineICS,
  playStudyChime,
} from './calendarService'

describe('calendarService', () => {
  it('generates a valid Google Calendar URL with daily recurrence and custom URL', () => {
    const url = getGoogleCalendarUrl('17:00', '19:00', 'https://my-school.edu/study')
    expect(url).toContain('https://calendar.google.com/calendar/render')
    expect(url).toContain('RRULE%3AFREQ%3DDAILY')
    expect(url).toContain('StudyLoop')
    expect(url).toContain('https%3A%2F%2Fmy-school.edu%2Fstudy')
  })

  it('generates a valid Outlook Web Calendar URL with start/end datetimes', () => {
    const url = getOutlookCalendarUrl('18:30', '20:00', 'https://custom-notes.com')
    expect(url).toContain('https://outlook.live.com/calendar/0/deeplink/compose')
    expect(url).toContain('18%3A30%3A00')
    expect(url).toContain('20%3A00%3A00')
    expect(url).toContain('StudyLoop')
    expect(url).toContain('https%3A%2F%2Fcustom-notes.com')
  })

  it('triggers an .ics download without crashing', () => {
    const clickSpy = vi.fn()
    const appendSpy = vi.spyOn(document.body, 'appendChild').mockImplementation(() => {})
    const removeSpy = vi.spyOn(document.body, 'removeChild').mockImplementation(() => {})

    global.URL.createObjectURL = vi.fn(() => 'blob:mock-ics-url')
    global.URL.revokeObjectURL = vi.fn()

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
    expect(global.URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock-ics-url')
  })

  it('plays study chime without throwing when Web Audio API is available or unavailable', () => {
    expect(() => playStudyChime()).not.toThrow()
  })
})
