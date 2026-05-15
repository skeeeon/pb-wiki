<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{ body: string }>()

// html: false — never trust raw HTML in user-supplied markdown.
// linkify   — auto-link bare URLs.
// breaks    — single newlines don't become <br>; require blank lines.
const md = new MarkdownIt({ html: false, linkify: true, breaks: false })

const html = computed(() => md.render(props.body ?? ''))
</script>

<template>
  <div class="markdown-body space-y-3 leading-relaxed" v-html="html" />
</template>

<style>
/* Lightweight typography for rendered markdown. Plain CSS instead of @apply
   because Tailwind v4's @apply only works in the main entry stylesheet (or
   behind an @reference directive) — not in per-component scoped blocks. */
.markdown-body h1 { font-size: 1.5rem; line-height: 2rem; font-weight: 600; margin: 1.5rem 0 0.75rem; }
.markdown-body h2 { font-size: 1.25rem; line-height: 1.75rem; font-weight: 600; margin: 1.25rem 0 0.5rem; }
.markdown-body h3 { font-size: 1.125rem; line-height: 1.75rem; font-weight: 600; margin: 1rem 0 0.5rem; }
.markdown-body h4 { font-size: 1rem; line-height: 1.5rem; font-weight: 600; margin: 0.75rem 0 0.5rem; }
.markdown-body p  { margin-bottom: 0.75rem; }
.markdown-body ul { list-style: disc; padding-left: 1.5rem; margin-bottom: 0.75rem; }
.markdown-body ol { list-style: decimal; padding-left: 1.5rem; margin-bottom: 0.75rem; }
.markdown-body li { margin-bottom: 0.25rem; }
.markdown-body a  { color: #2563eb; text-decoration: underline; }
.markdown-body code { padding: 0.125rem 0.25rem; border-radius: 0.25rem; background: rgb(244 244 245); font-size: 0.875rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.markdown-body pre  { padding: 0.75rem; border-radius: 0.375rem; background: rgb(244 244 245); overflow-x: auto; margin-bottom: 0.75rem; }
.markdown-body pre code { background: transparent; padding: 0; }
.markdown-body blockquote { border-left: 4px solid rgb(212 212 216); padding-left: 1rem; font-style: italic; color: rgb(63 63 70); margin-bottom: 0.75rem; }
.markdown-body table { border-collapse: collapse; margin-bottom: 0.75rem; }
.markdown-body th, .markdown-body td { border: 1px solid rgb(212 212 216); padding: 0.25rem 0.5rem; text-align: left; }
.markdown-body img { max-width: 100%; border-radius: 0.375rem; margin: 0.75rem 0; }
.markdown-body hr  { margin: 1.5rem 0; border-color: rgb(228 228 231); }

@media (prefers-color-scheme: dark) {
  .markdown-body a { color: #60a5fa; }
  .markdown-body code, .markdown-body pre { background: rgb(39 39 42); }
  .markdown-body blockquote { border-left-color: rgb(63 63 70); color: rgb(212 212 216); }
  .markdown-body th, .markdown-body td { border-color: rgb(63 63 70); }
  .markdown-body hr { border-color: rgb(39 39 42); }
}
</style>
