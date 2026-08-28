// ponytail: single configured instance, DOMPurify allows MathML + task inputs
import DOMPurify from 'dompurify'
import MarkdownIt from 'markdown-it'
import katex from '@iktakahiro/markdown-it-katex'
import taskLists from 'markdown-it-task-lists'
import githubAlerts from 'markdown-it-github-alerts'
import footnote from 'markdown-it-footnote'
import hljs from 'highlight.js/lib/core'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import java from 'highlight.js/lib/languages/java'
import cpp from 'highlight.js/lib/languages/cpp'
import csharp from 'highlight.js/lib/languages/csharp'
import sql from 'highlight.js/lib/languages/sql'
import bash from 'highlight.js/lib/languages/bash'
import json from 'highlight.js/lib/languages/json'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'

hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('python', python)
hljs.registerLanguage('py', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('golang', go)
hljs.registerLanguage('java', java)
hljs.registerLanguage('cpp', cpp)
hljs.registerLanguage('c', cpp)
hljs.registerLanguage('csharp', csharp)
hljs.registerLanguage('cs', csharp)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('json', json)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
      } catch (e) {
        // Fall back to unhighlighted escaping on parse error
        return ''
      }
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
