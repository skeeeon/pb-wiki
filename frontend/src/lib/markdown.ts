import MarkdownIt from 'markdown-it'
import type Token from 'markdown-it/lib/token.mjs'
import anchor from 'markdown-it-anchor'
import taskLists from 'markdown-it-task-lists'
import container from 'markdown-it-container'
import sub from 'markdown-it-sub'
import sup from 'markdown-it-sup'
import mark from 'markdown-it-mark'
import implicitFigures from 'markdown-it-implicit-figures'

import { applyYoutubePlugin } from './markdown-youtube'
import { applyFrontmatterPlugin } from './markdown-frontmatter'
import { applyMermaidPlugin } from './markdown-mermaid'

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

// No visible permalink — we just need `id` attributes so the TOC can scroll
// to headings and so #fragment URLs work. The TOC sidebar carries the
// "jump to section" affordance; an inline # would push heading text right
// of the body's left edge whenever it appeared.
md.use(anchor, { slugify })

md.use(taskLists, { enabled: true, label: true })

// External links open in a new tab with safe rel attributes. "External" =
// any http(s):// URL — relative links and #fragments stay in-app so router
// navigation isn't broken.
const defaultLinkOpen = md.renderer.rules.link_open ?? ((tokens, idx, opts, _env, self) =>
  self.renderToken(tokens, idx, opts))
md.renderer.rules.link_open = (tokens, idx, opts, env, self) => {
  const href = tokens[idx].attrGet('href') ?? ''
  if (/^https?:\/\//i.test(href)) {
    tokens[idx].attrSet('target', '_blank')
    tokens[idx].attrSet('rel', 'noopener noreferrer')
  }
  return defaultLinkOpen(tokens, idx, opts, env, self)
}

// Installs our ::: note / tip / warning / danger ::: callout containers.
// Exported so the editor preview (md-editor-v3) can use the same syntax via
// its `markdownItConfig` hook — it ships its own !!! admonition, heading
// IDs, and task lists, but not our :::-style callouts.
export function applyCalloutContainers(md: MarkdownIt) {
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
}

// Inline formatting and figure captions used by both the view renderer and
// the editor preview. Keeping them in one function ensures the two stay in
// lockstep — adding a plugin here picks it up in both places.
// - sub: `~text~` → <sub>
// - sup: `^text^` → <sup>
// - mark: `==text==` → <mark>
// - implicit-figures: a standalone `![alt](url "caption")` becomes
//   <figure><img><figcaption>caption</figcaption></figure>. `figcaption: 'title'`
//   sources the caption from the title attribute (third arg), leaving the
//   alt text on the <img> for screen readers. (`true`/`'alt'` would instead
//   use alt as the caption and wipe it from the img — not what we want.)
export function applyMarkdownExtras(md: MarkdownIt) {
  md.use(sub)
  md.use(sup)
  md.use(mark)
  md.use(implicitFigures, { figcaption: 'title' })
}

applyCalloutContainers(md)
applyYoutubePlugin(md)
applyMarkdownExtras(md)
applyFrontmatterPlugin(md)
applyMermaidPlugin(md)

// Re-export so the editor preview (md-editor-v3) can wire the same syntax.
// Note: mermaid is intentionally NOT re-exported — md-editor-v3 ships its own
// mermaid integration for the preview pane (see markdown-mermaid.ts).
export { applyYoutubePlugin, applyFrontmatterPlugin }

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
