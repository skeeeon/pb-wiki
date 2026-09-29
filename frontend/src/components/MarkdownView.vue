<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import { useTheme } from '@/composables/useTheme'

const props = defineProps<{ html: string }>()
const container = ref<HTMLElement | null>(null)
const { theme } = useTheme()
const router = useRouter()

// Every code block gets a Copy button. The block is wrapped so the button
// stays in the corner when the code scrolls sideways. v-html replaces the
// whole DOM on each change, so this runs on fresh blocks every time.
async function addCopyButtons() {
  await nextTick()
  for (const pre of container.value?.querySelectorAll('pre:not(.mermaid)') ?? []) {
    const wrap = document.createElement('div')
    wrap.className = 'code-block'
    pre.replaceWith(wrap)
    const btn = document.createElement('button')
    btn.type = 'button'
    btn.className = 'copy-btn'
    btn.textContent = 'Copy'
    wrap.append(pre, btn)
  }
}

async function copyCode(btn: HTMLElement) {
  const code = btn.parentElement?.querySelector('pre')?.textContent ?? ''
  try {
    await navigator.clipboard.writeText(code)
    btn.textContent = 'Copied'
  } catch {
    btn.textContent = 'Copy failed'
  }
  setTimeout(() => (btn.textContent = 'Copy'), 1500)
}

// Links to other wiki pages (/doc/…) navigate inside the SPA instead of
// reloading it. Modified clicks (new tab etc.), other origins, API URLs and
// links that open in a new tab keep the browser's default behaviour.
function onClick(ev: MouseEvent) {
  const btn = (ev.target as HTMLElement | null)?.closest<HTMLElement>('.copy-btn')
  if (btn) return void copyCode(btn)
  if (ev.defaultPrevented || ev.button !== 0 || ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return
  const a = (ev.target as HTMLElement | null)?.closest('a')
  if (!a || a.target === '_blank' || !a.href) return
  const url = new URL(a.href)
  if (url.origin !== window.location.origin || url.pathname.startsWith('/api/')) return
  ev.preventDefault()
  void router.push(url.pathname + url.search + url.hash)
}

// Lazy-load mermaid only when a doc actually contains a diagram. The fence
// plugin (lib/markdown-mermaid.ts) emits <pre class="mermaid" data-mermaid-src>
// placeholders; we walk them after v-html has settled, fetch mermaid on first
// use, and run it against the unprocessed nodes. The original source is kept
// in `data-mermaid-src` so a light/dark toggle can re-render in the new
// theme by clearing rendered SVGs and replaying mermaid.run().
async function renderMermaid() {
  await nextTick()
  const root = container.value
  if (!root) return

  const all = Array.from(root.querySelectorAll<HTMLElement>('pre.mermaid'))
  if (all.length === 0) return

  // After a theme change, previously rendered nodes have data-processed="true"
  // and contain SVG instead of source. Restore source so mermaid.run() will
  // re-process them.
  for (const node of all) {
    if (node.dataset.processed === 'true' && node.dataset.mermaidSrc) {
      node.removeAttribute('data-processed')
      node.innerHTML = ''
      node.textContent = node.dataset.mermaidSrc
    }
  }

  try {
    const { default: mermaid } = await import('mermaid')
    mermaid.initialize({
      startOnLoad: false,
      theme: theme.value === 'dark' ? 'dark' : 'default',
      securityLevel: 'strict',
    })
    await mermaid.run({ nodes: all })
  } catch (err) {
    console.error('mermaid render failed:', err)
  }
}

watch([() => props.html, theme], renderMermaid, { immediate: true })
watch(() => props.html, addCopyButtons, { immediate: true })
</script>

<template>
  <div ref="container" class="markdown-body space-y-3 leading-relaxed" v-html="html" @click="onClick" />
</template>

<style>
/* Lightweight typography for rendered markdown. Plain CSS instead of @apply
   because Tailwind v4's @apply only works in the main entry stylesheet (or
   behind an @reference directive) — not in per-component scoped blocks.

   Colours come from the theme variables in style.css (surface, line, muted,
   and the semantic colours), which switch in dark mode by themselves. Filled
   blocks follow the platform console's card idiom: the surface colour with a
   line-coloured border, on the page ground. */
/* Long unbroken words (paths, URLs, identifiers in inline code) wrap instead
   of widening the page on a phone. break-word, not anywhere: it leaves
   min-content sizing alone, so wide tables still scroll rather than squeeze. */
.markdown-body { overflow-wrap: break-word; }
.markdown-body h1 { font-size: 1.5rem; line-height: 2rem; font-weight: 600; margin: 1.5rem 0 0.75rem; padding-bottom: 0.3rem; border-bottom: 1px solid var(--color-line); scroll-margin-top: 5rem; }
.markdown-body h2 { font-size: 1.25rem; line-height: 1.75rem; font-weight: 600; margin: 1.25rem 0 0.5rem; padding-bottom: 0.25rem; border-bottom: 1px solid var(--color-line); scroll-margin-top: 5rem; }
.markdown-body h3 { font-size: 1.125rem; line-height: 1.75rem; font-weight: 600; margin: 1rem 0 0.5rem; scroll-margin-top: 5rem; }
.markdown-body h4 { font-size: 1rem; line-height: 1.5rem; font-weight: 600; margin: 0.75rem 0 0.5rem; scroll-margin-top: 5rem; }
.markdown-body p  { margin-bottom: 0.75rem; }
.markdown-body ul { list-style: disc; padding-left: 1.5rem; margin-bottom: 0.75rem; }
.markdown-body ol { list-style: decimal; padding-left: 1.5rem; margin-bottom: 0.75rem; }
.markdown-body li { margin-bottom: 0.25rem; }
.markdown-body a  { color: var(--color-primary); text-decoration: underline; }
.markdown-body code { padding: 0.125rem 0.25rem; border-radius: 0.25rem; background: var(--color-surface); border: 1px solid var(--color-line); font-size: 0.875rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.markdown-body pre  { padding: 0.75rem; border-radius: 0.5rem; background: var(--color-surface); border: 1px solid var(--color-line); overflow-x: auto; margin-bottom: 0.75rem; }
.markdown-body pre code { background: transparent; border: 0; padding: 0; }

/* Heading text links to its own section (lib/markdown.ts). It keeps the
   heading's colour; the underline on hover shows it can be clicked. */
.markdown-body a.header-anchor { color: inherit; text-decoration: none; }
.markdown-body a.header-anchor:hover { text-decoration: underline; }

/* Copy button on code blocks, added by addCopyButtons(). It shows on hover
   or keyboard focus, and always on touch screens, which have no hover. */
.markdown-body .code-block { position: relative; }
.markdown-body .copy-btn { position: absolute; top: 0.375rem; right: 0.375rem; padding: 0.125rem 0.5rem; font-size: 0.75rem; border: 1px solid var(--color-line); border-radius: 0.25rem; background: var(--color-surface); color: var(--color-muted); opacity: 0; transition: opacity 150ms; }
.markdown-body .code-block:hover .copy-btn, .markdown-body .copy-btn:focus-visible { opacity: 1; }
.markdown-body .copy-btn:hover { color: inherit; }
@media (hover: none) { .markdown-body .copy-btn { opacity: 1; } }
.markdown-body blockquote { border-left: 4px solid var(--color-line); padding: 0.5rem 1rem; background: var(--color-surface); border-radius: 0 0.5rem 0.5rem 0; font-style: italic; color: var(--color-muted); margin-bottom: 0.75rem; }
.markdown-body blockquote > p:last-child { margin-bottom: 0; }
.markdown-body table { border-collapse: collapse; margin-bottom: 0.75rem; display: block; overflow-x: auto; max-width: 100%; }
.markdown-body th, .markdown-body td { border: 1px solid var(--color-line); padding: 0.25rem 0.5rem; text-align: left; }
.markdown-body th { background: var(--color-surface); }
.markdown-body img { max-width: 100%; border-radius: 0.5rem; margin: 0.75rem 0; }
.markdown-body hr  { margin: 1.5rem 0; border-color: var(--color-line); }
.markdown-body mark { background: color-mix(in srgb, var(--color-warning) 30%, transparent); color: inherit; padding: 0 0.15rem; border-radius: 0.125rem; }

/* Image captions from `![alt](url "caption")` — markdown-it-implicit-figures
   wraps a standalone image in <figure>; we move the margin from <img> to
   <figure> so spacing matches a bare image, and style the caption as a
   small, muted line beneath. */
.markdown-body figure { margin: 0.75rem 0; }
.markdown-body figure img { margin: 0; }
.markdown-body figcaption { margin-top: 0.375rem; font-size: 0.875rem; color: var(--color-muted); font-style: italic; text-align: center; }

/* Frontmatter table rendered by lib/markdown-frontmatter.ts. Override the
   generic `.markdown-body table { display: block }` rule (which is there to
   scroll wide tables) so the frontmatter table sizes to its content. */
.markdown-body table.frontmatter { display: table; width: auto; font-size: 0.875rem; margin: 0 0 1rem 0; }
.markdown-body table.frontmatter th { font-weight: 600; text-align: right; vertical-align: top; color: var(--color-muted); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }

/* Task lists — no bullet; the checkbox sits inline at the start of the
   item's first line. Plain block layout on purpose: a flex item puts an
   item's paragraphs and nested lists side by side, which made a long task
   list hundreds of pixels wider than a phone screen. */
.markdown-body ul.contains-task-list { list-style: none; padding-left: 0.5rem; }
.markdown-body li.task-list-item input[type="checkbox"] { margin-right: 0.5rem; transform: translateY(0.1rem); }

/* Mermaid placeholder before the runtime takes over: hide the source so it
   doesn't flash as raw text in the brief window before mermaid loads. Once
   mermaid runs, it sets data-processed="true" and renders the SVG inline.
   The placeholder is overflow: hidden because its long source lines would
   otherwise widen the page until mermaid (a large download on a phone)
   arrives; a rendered diagram too wide for the screen scrolls in its box. */
.markdown-body pre.mermaid { background: transparent; border: 0; padding: 0; overflow: hidden; min-height: 1.5rem; color: transparent; }
.markdown-body pre.mermaid[data-processed="true"] { color: inherit; text-align: center; overflow-x: auto; }

/* YouTube embeds — unscoped so the editor preview renders them the same
   way. Aspect ratio keeps the iframe responsive without JS. */
.youtube-embed { aspect-ratio: 16 / 9; max-width: 720px; margin: 0.75rem 0; border-radius: 0.5rem; overflow: hidden; background: rgb(0 0 0); }
.youtube-embed iframe { width: 100%; height: 100%; border: 0; display: block; }

/* Callouts — ::: note / tip / warning / danger ::: — unscoped so they
   also render correctly inside the editor preview pane (.md-editor-preview),
   not just inside .markdown-body. The four types use the platform's state
   colours (info, success, warning, error) on the left border. Titles stay in
   the text colour: the light success and warning colours are under WCAG AA
   contrast as small text on white. */
.callout { border: 1px solid var(--color-line); border-left: 4px solid; padding: 0.75rem 1rem; border-radius: 0.5rem; margin-bottom: 0.75rem; background: var(--color-surface); }
.callout > p:last-child { margin-bottom: 0; }
.callout-title { font-weight: 600; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.25rem; }
.callout-note    { border-left-color: var(--color-info); }
.callout-tip     { border-left-color: var(--color-success); }
.callout-warning { border-left-color: var(--color-warning); }
.callout-danger  { border-left-color: var(--color-error); }
</style>
