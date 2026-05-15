import MarkdownIt from 'markdown-it'
import type Token from 'markdown-it/lib/token.mjs'
import anchor from 'markdown-it-anchor'
import taskLists from 'markdown-it-task-lists'
import container from 'markdown-it-container'

export interface Heading {
  level: number
  text: string
  slug: string
}

export interface RenderedDoc {
  html: string
  headings: Heading[]
  showToc: boolean
}

// Slug: lowercase, strip punctuation, collapse whitespace to '-'. Trimmed of
// leading/trailing dashes. Matches what the TOC links to and what
// markdown-it-anchor stamps on each heading_open token.
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\s-]/gu, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
}

// html: false — never trust raw HTML in user-supplied markdown.
// linkify   — auto-link bare URLs.
// breaks    — single newlines don't become <br>; require blank lines.
const md = new MarkdownIt({ html: false, linkify: true, breaks: false })

md.use(anchor, {
  slugify,
  permalink: anchor.permalink.linkInsideHeader({
    symbol: '#',
    placement: 'before',
    ariaHidden: true,
    class: 'heading-anchor',
  }),
})

md.use(taskLists, { enabled: true, label: true })

for (const name of ['note', 'tip', 'warning', 'danger'] as const) {
  md.use(container, name, {
    render(tokens: Token[], idx: number) {
      if (tokens[idx].nesting === 1) {
        return `<div class="callout callout-${name}"><div class="callout-title">${name}</div>\n`
      }
      return `</div>\n`
    },
  })
}

// `<!-- toc: false -->` or `<!-- no-toc -->` at the top of a doc suppresses
// the auto-sidebar TOC. Stripped before rendering so it doesn't appear as text.
const TOC_OFF_RE = /^\s*<!--\s*(?:toc:\s*false|no-toc)\s*-->\s*\r?\n?/i

function parseDirectives(src: string): { src: string; showToc: boolean } {
  if (TOC_OFF_RE.test(src)) {
    return { src: src.replace(TOC_OFF_RE, ''), showToc: false }
  }
  return { src, showToc: true }
}

function collectHeadings(tokens: ReturnType<typeof md.parse>): Heading[] {
  const headings: Heading[] = []
  for (let i = 0; i < tokens.length; i++) {
    const t = tokens[i]
    if (t.type !== 'heading_open') continue
    const level = Number(t.tag.slice(1))
    const inline = tokens[i + 1]
    const text = inline?.content ?? ''
    const slug = t.attrGet('id') ?? slugify(text)
    headings.push({ level, text, slug })
  }
  return headings
}

export function renderDoc(src: string): RenderedDoc {
  const { src: cleaned, showToc } = parseDirectives(src ?? '')
  const env = {}
  const tokens = md.parse(cleaned, env)
  const headings = collectHeadings(tokens)
  const html = md.renderer.render(tokens, md.options, env)
  return { html, headings, showToc }
}
