import { describe, expect, it } from 'vitest'
import { renderMarkdown } from './markdown'

describe('renderMarkdown', () => {
  it('renders standard markdown paragraphs and headings', () => {
    const html = renderMarkdown('# Title\n\nHello world')
    expect(html).toContain('<h1>Title</h1>')
    expect(html).toContain('<p>Hello world</p>')
  })

  it('renders inline and display math via KaTeX', () => {
    const inline = renderMarkdown('Energy $E = mc^2$ equation')
    expect(inline).toContain('class="katex"')

    const block = renderMarkdown('$$\\sum_{i=1}^n x_i$$')
    expect(block).toContain('class="katex-display"')
  })

  it('highlights code blocks with highlight.js classes', () => {
    const code = renderMarkdown('```javascript\nconst x = 42;\n```')
    expect(code).toContain('hljs-keyword')
    expect(code).toContain('const')
  })

  it('renders task list checkboxes', () => {
    const tasks = renderMarkdown('- [ ] Todo item\n- [x] Done item')
    expect(tasks).toContain('task-list-item')
    expect(tasks).toContain('type="checkbox"')
    expect(tasks).toContain('checked')
  })

  it('renders GitHub callout alerts', () => {
    const alert = renderMarkdown('> [!NOTE]\n> Important context here')
    expect(alert).toContain('markdown-alert')
    expect(alert).toContain('markdown-alert-note')
  })

  it('renders footnotes', () => {
    const fn = renderMarkdown('Reference[^1]\n\n[^1]: Footnote content')
    expect(fn).toContain('class="footnote-ref"')
    expect(fn).toContain('class="footnotes"')
    expect(fn).toContain('Footnote content')
  })

  it('sanitizes dangerous XSS scripts', () => {
    const dirty = renderMarkdown('<script>alert("xss")</script>Safe text')
    expect(dirty).not.toContain('<script>')
    expect(dirty).toContain('Safe text')
  })
})
