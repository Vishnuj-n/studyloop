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

const defaultImageRender = md.renderer.rules.image || function (tokens, idx, options, env, self) {
  return self.renderToken(tokens, idx, options)
}

export function resolveNoteAssetSrc(src, noteRelativeFolder = '') {
  if (!src || typeof src !== 'string') return src
  const raw = src.trim()
  if (
    raw.startsWith('http://') ||
    raw.startsWith('https://') ||
    raw.startsWith('data:') ||
    raw.startsWith('blob:')
  ) {
    return raw
  }

  // Normalize Windows slashes
  let normalized = raw.replace(/\\/g, '/')

  // Handle absolute notes paths (e.g. C:/Users/.../Studyloop/notes/<relativeFolder>/assets/pic.png or /notes/...)
  const notesMatch = normalized.match(/(?:^|\/)(?:notes|note-assets)\/(.+)$/i)
  if (notesMatch && notesMatch[1]) {
    const encoded = notesMatch[1]
      .split('/')
      .map((segment) => encodeURIComponent(decodeURIComponent(segment)))
      .join('/')
    return `/note-assets/${encoded}`
  }

  // Handle relative assets (e.g. assets/pic.png or ./assets/pic.png)
  if (normalized.startsWith('./')) {
    normalized = normalized.slice(2)
  }

  if (noteRelativeFolder) {
    const cleanFolder = noteRelativeFolder.replace(/\\/g, '/').replace(/^\/+|\/+$/g, '')
    let targetRelPath = normalized
    if (targetRelPath.startsWith('assets/')) {
      targetRelPath = `${cleanFolder}/${targetRelPath}`
    } else if (!targetRelPath.startsWith('/')) {
      targetRelPath = `${cleanFolder}/assets/${targetRelPath}`
    }
    const encoded = targetRelPath
      .split('/')
      .map((segment) => encodeURIComponent(decodeURIComponent(segment)))
      .join('/')
    return `/note-assets/${encoded}`
  }

  return raw
}

export function parseImageSize(rawAlt) {
  if (!rawAlt || !rawAlt.includes('|')) {
    return { alt: rawAlt, width: null, height: null, style: null }
  }
  const lastPipe = rawAlt.lastIndexOf('|')
  const alt = rawAlt.slice(0, lastPipe).trim()
  const sizePart = rawAlt.slice(lastPipe + 1).trim()

  const match = sizePart.match(/^(\d+(?:\.\d+)?%?)(?:x(\d+(?:\.\d+)?%?))?$/i)
  if (!match) {
    return { alt: rawAlt, width: null, height: null, style: null }
  }

  const rawWidth = match[1]
  const rawHeight = match[2]

  const width = rawWidth.endsWith('%') ? rawWidth : `${rawWidth}px`
  const height = rawHeight ? (rawHeight.endsWith('%') ? rawHeight : `${rawHeight}px`) : null

  let style = 'max-width: 100%;'
  if (width) {
    style += ` width: ${width};`
  }
  if (height) {
    style += ` height: ${height}; object-fit: contain;`
  } else {
    style += ' height: auto;'
  }

  return { alt, width: rawWidth, height: rawHeight, style }
}

md.renderer.rules.image = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  const srcIndex = token.attrIndex('src')
  if (srcIndex >= 0) {
    const src = token.attrs[srcIndex][1]
    const noteFolder = env && env.noteRelativeFolder ? env.noteRelativeFolder : ''
    token.attrs[srcIndex][1] = resolveNoteAssetSrc(src, noteFolder)
  }

  const rawAlt = token.content || ''
  const parsedSize = parseImageSize(rawAlt)
  if (parsedSize.style) {
    token.attrSet('style', parsedSize.style)
    token.content = parsedSize.alt
    if (token.children && token.children.length > 0) {
      token.children.forEach((child) => {
        if (child.type === 'text') {
          const childParsed = parseImageSize(child.content)
          child.content = childParsed.alt
        }
      })
    }
  }

  return defaultImageRender(tokens, idx, options, env, self)
}

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
    'img',
    'video',
    'source',
    'figure',
    'figcaption',
  ],
  ADD_ATTR: ['aria-hidden', 'type', 'checked', 'disabled', 'src', 'alt', 'title', 'width', 'height', 'style', 'controls', 'poster'],
}

export function renderMarkdown(input, options = {}) {
  let source = typeof input === 'string' ? input : ''
  const noteRelativeFolder =
    typeof options === 'string' ? options : options?.noteRelativeFolder || ''

  // Normalize LaTeX delimiters \( ... \) and \[ ... \] to $ ... $ and $$ ... $$ only in Markdown prose
  const parts = source.split(/(```[\s\S]*?```|`[^`\n]+`)/g)
  source = parts
    .map((part, index) => {
      if (index % 2 === 1) return part
      return part
        .replace(/\\\[([\s\S]*?)\\\]/g, (_, math) => `\n$$\n${math.trim()}\n$$\n`)
        .replace(/\\\(([\s\S]*?)\\\)/g, (_, math) => `$${math.trim()}$`)
    })
    .join('')
  return DOMPurify.sanitize(md.render(source, { noteRelativeFolder }), SANITIZE_CONFIG)
}
