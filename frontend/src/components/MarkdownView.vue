<script setup lang="ts">
defineProps<{ html: string }>()
</script>

<template>
  <div class="markdown-body space-y-3 leading-relaxed" v-html="html" />
</template>

<style>
/* Lightweight typography for rendered markdown. Plain CSS instead of @apply
   because Tailwind v4's @apply only works in the main entry stylesheet (or
   behind an @reference directive) — not in per-component scoped blocks. */
.markdown-body h1 { font-size: 1.5rem; line-height: 2rem; font-weight: 600; margin: 1.5rem 0 0.75rem; padding-bottom: 0.3rem; border-bottom: 1px solid rgb(228 228 231); scroll-margin-top: 5rem; }
.markdown-body h2 { font-size: 1.25rem; line-height: 1.75rem; font-weight: 600; margin: 1.25rem 0 0.5rem; padding-bottom: 0.25rem; border-bottom: 1px solid rgb(228 228 231); scroll-margin-top: 5rem; }
.markdown-body h3 { font-size: 1.125rem; line-height: 1.75rem; font-weight: 600; margin: 1rem 0 0.5rem; scroll-margin-top: 5rem; }
.markdown-body h4 { font-size: 1rem; line-height: 1.5rem; font-weight: 600; margin: 0.75rem 0 0.5rem; scroll-margin-top: 5rem; }
.markdown-body p  { margin-bottom: 0.75rem; }
.markdown-body ul { list-style: disc; padding-left: 1.5rem; margin-bottom: 0.75rem; }
.markdown-body ol { list-style: decimal; padding-left: 1.5rem; margin-bottom: 0.75rem; }
.markdown-body li { margin-bottom: 0.25rem; }
.markdown-body a  { color: var(--color-brand-blue); text-decoration: underline; }
.markdown-body code { padding: 0.125rem 0.25rem; border-radius: 0.25rem; background: rgb(244 244 245); font-size: 0.875rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.markdown-body pre  { padding: 0.75rem; border-radius: 0.375rem; background: rgb(244 244 245); overflow-x: auto; margin-bottom: 0.75rem; }
.markdown-body pre code { background: transparent; padding: 0; }
.markdown-body blockquote { border-left: 4px solid rgb(212 212 216); padding: 0.5rem 1rem; background: rgb(244 244 245); border-radius: 0 0.375rem 0.375rem 0; font-style: italic; color: rgb(63 63 70); margin-bottom: 0.75rem; }
.markdown-body blockquote > p:last-child { margin-bottom: 0; }
.markdown-body table { border-collapse: collapse; margin-bottom: 0.75rem; display: block; overflow-x: auto; max-width: 100%; }
.markdown-body th, .markdown-body td { border: 1px solid rgb(212 212 216); padding: 0.25rem 0.5rem; text-align: left; }
.markdown-body img { max-width: 100%; border-radius: 0.375rem; margin: 0.75rem 0; }
.markdown-body hr  { margin: 1.5rem 0; border-color: rgb(228 228 231); }
.markdown-body mark { background: var(--color-brand-yellow); color: rgb(24 24 27); padding: 0 0.15rem; border-radius: 0.125rem; }

/* Image captions from `![alt](url "caption")` — markdown-it-implicit-figures
   wraps a standalone image in <figure>; we move the margin from <img> to
   <figure> so spacing matches a bare image, and style the caption as a
   small, muted line beneath. */
.markdown-body figure { margin: 0.75rem 0; }
.markdown-body figure img { margin: 0; }
.markdown-body figcaption { margin-top: 0.375rem; font-size: 0.875rem; color: rgb(82 82 91); font-style: italic; text-align: center; }
.dark .markdown-body figcaption { color: rgb(161 161 170); }

/* Task lists — checkbox aligned with first line, no bullet. */
.markdown-body ul.contains-task-list { list-style: none; padding-left: 0.5rem; }
.markdown-body li.task-list-item { display: flex; align-items: baseline; gap: 0.5rem; }
.markdown-body li.task-list-item input[type="checkbox"] { transform: translateY(0.1rem); }

/* YouTube embeds — unscoped so the editor preview renders them the same
   way. Aspect ratio keeps the iframe responsive without JS. */
.youtube-embed { aspect-ratio: 16 / 9; max-width: 720px; margin: 0.75rem 0; border-radius: 0.375rem; overflow: hidden; background: rgb(0 0 0); }
.youtube-embed iframe { width: 100%; height: 100%; border: 0; display: block; }

/* Callouts — ::: note / tip / warning / danger ::: — unscoped so they
   also render correctly inside the editor preview pane (.md-editor-preview),
   not just inside .markdown-body. */
.callout { border-left: 4px solid; padding: 0.75rem 1rem; border-radius: 0.375rem; margin-bottom: 0.75rem; background: rgb(244 244 245); }
.callout > p:last-child { margin-bottom: 0; }
.callout-title { font-weight: 600; font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.25rem; }
.callout-note    { border-color: var(--color-brand-blue); }
.callout-tip     { border-color: rgb(34 197 94); }
.callout-warning { border-color: rgb(234 179 8); }
.callout-danger  { border-color: rgb(220 38 38); }
.callout-note    .callout-title { color: var(--color-brand-blue); }
.callout-tip     .callout-title { color: rgb(22 163 74); }
.callout-warning .callout-title { color: rgb(161 98 7); }
.callout-danger  .callout-title { color: rgb(185 28 28); }

.dark .markdown-body a { color: var(--color-brand-blue-dark); }
.dark .markdown-body code, .dark .markdown-body pre { background: var(--color-brand-navy-200); }
.dark .markdown-body blockquote { background: var(--color-brand-navy-200); border-left-color: var(--color-brand-navy-100); color: rgb(212 212 216); }
.dark .markdown-body th, .dark .markdown-body td { border-color: var(--color-brand-navy-100); }
.dark .markdown-body hr { border-color: var(--color-brand-navy-200); }
.dark .markdown-body h1, .dark .markdown-body h2 { border-bottom-color: var(--color-brand-navy-100); }
.dark .callout { background: var(--color-brand-navy-200); }
</style>
