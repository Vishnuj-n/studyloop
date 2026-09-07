import { describe, it, expect } from 'vitest'
import { cleanTopicTitle } from './useReaderBase'

describe('cleanTopicTitle', () => {
  it('formats nb-uuid-ch-05-05 as Chapter 5 without trailing duplicate chapter number', () => {
    expect(cleanTopicTitle('nb-uuid-ch-05-05')).toBe('Chapter 5')
  })

  it('formats standard topic titles properly', () => {
    expect(cleanTopicTitle('nb-92c8f059-78e2-440c-81e8-62d5032d4330-ch-01-cn-final-revision-sh')).toBe('Chapter 1: Cn Final Revision Sh')
  })
})
