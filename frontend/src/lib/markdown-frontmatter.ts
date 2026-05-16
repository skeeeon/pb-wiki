import type MarkdownIt from 'markdown-it'

// A leading YAML frontmatter block (`---` ... `---` at line 0) is rendered as
// a small key/value table at the top of the doc — same look as GitHub's
// frontmatter preview. The importer strips frontmatter from `.md` files
// before saving them as documents, so an imported page won't trigger this;
// it fires when an author explicitly types frontmatter into the editor body
// or pastes a `.md` file that still has it.
//
// The parser is intentionally flat: it handles `key: value` lines only.
// Lists, nested maps, and folded scalars are not interpreted — they fall
// through unrecognized, which means a wiki author can't use the table to
// render anything richer than what GitHub itself shows in its preview pane.
// That's fine for path/title/tags-style metadata, which is the whole use case.

function unquote(v: string): string {
  if (v.length >= 2) {
    const f = v[0]
    if ((f === '"' || f === "'") && v[v.length - 1] === f) {
      return v.slice(1, -1)
    }
  }
  return v
}

function parsePairs(yaml: string): Array<[string, string]> {
  const pairs: Array<[string, string]> = []
  for (const raw of yaml.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line.startsWith('#')) continue
    const m = /^([A-Za-z_][A-Za-z0-9_-]*)\s*:\s*(.*)$/.exec(line)
    if (!m) continue
    pairs.push([m[1], unquote(m[2].trim())])
  }
  return pairs
}

function esc(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function applyFrontmatterPlugin(md: MarkdownIt) {
  md.block.ruler.before(
    'paragraph',
    'frontmatter',
    (state, startLine, endLine, silent) => {
      // Only valid as the very first block of the document.
      if (startLine !== 0) return false

      const line0 = state.src.slice(state.bMarks[startLine], state.eMarks[startLine])
      if (line0.trim() !== '---') return false

      // Walk forward looking for the closing `---` line.
      let closing = -1
      for (let l = startLine + 1; l < endLine; l++) {
        const text = state.src.slice(state.bMarks[l], state.eMarks[l]).trim()
        if (text === '---') {
          closing = l
          break
        }
      }
      if (closing === -1) return false

      let yaml = ''
      for (let l = startLine + 1; l < closing; l++) {
        yaml += state.src.slice(state.bMarks[l], state.eMarks[l]) + '\n'
      }
      const pairs = parsePairs(yaml)
      if (pairs.length === 0) return false

      if (silent) return true

      const token = state.push('frontmatter', '', 0)
      token.markup = '---'
      token.block = true
      token.map = [startLine, closing + 1]
      token.meta = { pairs }

      state.line = closing + 1
      return true
    },
    { alt: [] },
  )

  md.renderer.rules.frontmatter = (tokens, idx) => {
    const meta = tokens[idx].meta as { pairs?: Array<[string, string]> } | null
    const pairs = meta?.pairs ?? []
    if (pairs.length === 0) return ''
    const rows = pairs
      .map(([k, v]) => `<tr><th>${esc(k)}</th><td>${esc(v)}</td></tr>`)
      .join('')
    return `<table class="frontmatter"><tbody>${rows}</tbody></table>\n`
  }
}
