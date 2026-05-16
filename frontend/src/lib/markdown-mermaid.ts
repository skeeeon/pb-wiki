import type MarkdownIt from 'markdown-it'

// Translate ` ```mermaid `…``` fenced blocks into a placeholder element that
// MarkdownView's post-render hook lazy-loads mermaid against. The renderer
// itself stays pure string-in/string-out — mermaid runs client-side on the
// rendered DOM after Vue's v-html settles. See components/MarkdownView.vue.
//
// We intentionally do NOT register this on md-editor-v3's preview MD
// instance — md-editor-v3 ships its own mermaid integration for the preview
// pane. Adding ours would either be shadowed by theirs (fence renderer is
// last-write-wins) or conflict with the placeholder shape their runtime
// expects. The split is fine: view = ours, preview = theirs.

function esc(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function applyMermaidPlugin(md: MarkdownIt) {
  const defaultFence =
    md.renderer.rules.fence ??
    ((tokens, idx, opts, _env, self) => self.renderToken(tokens, idx, opts))
  md.renderer.rules.fence = (tokens, idx, opts, env, self) => {
    const info = tokens[idx].info.trim().toLowerCase()
    if (info === 'mermaid') {
      // The element receives an SVG replacement after mermaid.run() in the
      // post-render hook. `data-mermaid-src` survives that replacement so we
      // can re-render with a new theme on light/dark toggle without re-fetching
      // the original markdown.
      const src = tokens[idx].content
      return `<pre class="mermaid" data-mermaid-src="${esc(src)}">${esc(src)}</pre>\n`
    }
    return defaultFence(tokens, idx, opts, env, self)
  }
}
