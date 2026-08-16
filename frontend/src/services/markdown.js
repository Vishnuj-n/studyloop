// ponytail: single configured instance, DOMPurify allows MathML + task inputs
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import katex from '@iktakahiro/markdown-it-katex'
import taskLists from 'markdown-it-task-lists'
import githubAlerts from 'markdown-it-github-alerts'
import footnote from 'markdown-it-footnote'
import hljs from 'highlight.js'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
      } catch {}
    }
    return ''
  },
})
  .use(katex)
  .use(taskLists, { enabled: false })
  .use(githubAlerts)
  .use(footnote)

const SANITIZE_CONFIG = {
  ADD_TAGS: [
    'math',
    'annotation',
    'semantics',
    'mtext',
    'mn',
    'mo',
    'mi',
    'mspace',
    'mover',
    'munder',
    'mfrac',
    'mroot',
    'msqrt',
    'msub',
    'msup',
    'msubsup',
    'input',
  ],
  ADD_ATTR: ['aria-hidden', 'type', 'checked', 'disabled'],
}

export function renderMarkdown(input) {
  const source = typeof input === 'string' ? input : ''
  return DOMPurify.sanitize(md.render(source), SANITIZE_CONFIG)
}
