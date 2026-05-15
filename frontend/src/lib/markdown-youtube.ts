import type MarkdownIt from 'markdown-it'

// Matches youtube.com/watch?v=, youtu.be/, youtube.com/embed/, youtube.com/shorts/.
// The 11-char ID is the well-defined YouTube video-ID format. Everything
// after the ID is treated as ignorable query/fragment, which lets us parse
// the start-time separately.
const YOUTUBE_RE =
  /^https?:\/\/(?:www\.|m\.)?(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([A-Za-z0-9_-]{11})(?:[?&#][^\s]*)?$/i

const TIME_RE = /[?&](?:t|start)=([0-9hms]+)/i

function parseStartSeconds(url: string): number {
  const m = TIME_RE.exec(url)
  if (!m) return 0
  const raw = m[1]
  if (/^\d+$/.test(raw)) return parseInt(raw, 10)
  const hms = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/i.exec(raw)
  if (!hms) return 0
  return Number(hms[1] || 0) * 3600 + Number(hms[2] || 0) * 60 + Number(hms[3] || 0)
}

// Drop-in plugin: registers a block rule that consumes a line containing
// only a YouTube URL and renders it as an embed. Keeps the renderer's
// `html: false` posture intact — we own the iframe markup, the user only
// supplies a video ID via the matched URL.
export function applyYoutubePlugin(md: MarkdownIt) {
  md.block.ruler.before(
    'paragraph',
    'youtube',
    (state, startLine, _endLine, silent) => {
      const pos = state.bMarks[startLine] + state.tShift[startLine]
      const max = state.eMarks[startLine]
      const line = state.src.slice(pos, max).trim()
      const match = YOUTUBE_RE.exec(line)
      if (!match) return false
      if (silent) return true

      const id = match[1]
      const start = parseStartSeconds(line)
      const src = `https://www.youtube-nocookie.com/embed/${id}${start ? `?start=${start}` : ''}`

      const token = state.push('youtube_embed', '', 0)
      token.markup = line
      token.content = src
      token.block = true
      token.map = [startLine, startLine + 1]

      state.line = startLine + 1
      return true
    },
    // alt allows our rule to terminate a paragraph, so a URL on the next
    // line ends the paragraph above instead of being slurped into it.
    { alt: ['paragraph', 'reference', 'blockquote', 'list'] },
  )

  md.renderer.rules.youtube_embed = (tokens, idx) => {
    const src = tokens[idx].content
    return (
      `<div class="youtube-embed">` +
      `<iframe src="${src}" title="YouTube video" ` +
      `frameborder="0" loading="lazy" ` +
      `referrerpolicy="strict-origin-when-cross-origin" ` +
      `allow="accelerometer; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" ` +
      `allowfullscreen></iframe></div>\n`
    )
  }
}
