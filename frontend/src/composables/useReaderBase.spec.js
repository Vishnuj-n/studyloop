import { describe, it, expect } from 'vitest'
import { cleanTopicTitle } from './useReaderBase'

describe('cleanTopicTitle', () => {
  it('formats standard raw topic titles properly', () => {
    expect(cleanTopicTitle('nb-123-ch-01-intro-to-coding')).toBe('Chapter 1: Intro To Coding')
    expect(cleanTopicTitle('nb-abc-ch-10-advanced-math')).toBe('Chapter 10: Advanced Math')
  })

  it('preserves subsequent -ch- tokens in the suffix', () => {
    // ponytail: test case covering a chapter suffix containing "-ch-"
    expect(cleanTopicTitle('nb-uuid-ch-01-intro-to-ch-models')).toBe('Chapter 1: Intro To Ch Models')
    expect(cleanTopicTitle('nb-uuid-ch-02-ch-algorithms-discussion')).toBe('Chapter 2: Ch Algorithms Discussion')
  })

  it('returns raw value if prefix is not nb- or -ch- is missing', () => {
    expect(cleanTopicTitle('just-some-random-title')).toBe('just-some-random-title')
    expect(cleanTopicTitle('nb-without-chapter')).toBe('nb-without-chapter')
  })
})
