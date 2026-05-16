---
path: examples/markdown-reference
title: Markdown reference
---

This page demonstrates every markdown feature pb-wiki renders. It doubles as
an import smoke test — `go run . import ./examples` should upsert it cleanly.

## Headings

`#` through `####` are the levels you'll typically use. Each heading gets an
auto-generated `id` so the TOC sidebar can scroll to it and `#fragment` URLs
work. Don't put an H1 at the top of the body — the page title (set in the
frontmatter or the editor's Title field) already serves that role.

### Third-level heading
#### Fourth-level heading

## Text formatting

**bold**, *italic*, ***bold italic***, ~~strikethrough~~, `inline code`.

Highlight a phrase with `==marker==` → ==highlighted text==.

Subscript `H~2~O` renders as H~2~O. Superscript `E=mc^2^` renders as E=mc^2^.

A blank line separates paragraphs. A single newline inside a paragraph stays a
space (`breaks: false`) — you have to leave a blank line to start a new one.

## Links

[Inline link](https://example.com) — bracketed text, URL in parens.

Bare URLs auto-link thanks to `linkify: true`: https://example.com.

External links (anything `http(s)://`) open in a new tab with safe
`rel="noopener noreferrer"`. Relative links and `#fragment` URLs stay in-app
so SPA navigation isn't broken — link to other wiki pages by path:
[Getting started](getting-started/install) or [the homepage](.).

## Lists

Unordered:

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

Ordered:

1. First
2. Second
3. Third

Task list — checkboxes are interactive in the rendered view:

- [x] Completed task
- [ ] Pending task
- [ ] Another pending task

## Blockquotes

> A single-line quote.

> A multi-paragraph quote with **inline formatting** and a [link](https://example.com).
>
> The second paragraph of the same quote.

## Code blocks

Inline `code` with backticks.

A fenced code block. The saved view doesn't ship syntax highlighting — the
editor preview highlights for readability while authoring, but the rendered
page shows code as plain monospace:

```go
func main() {
    fmt.Println("hello, pb-wiki")
}
```

## Tables

Pipe-delimited tables with optional column alignment (`:---`, `---:`, `:---:`):

| Column 1 | Right-aligned | Centered |
|----------|--------------:|:--------:|
| left     |             1 |    x     |
| left     |            42 |    y     |
| left     |           999 |    z     |

## Images

A bare image (no caption):

![A descriptive alt for screen readers](https://placehold.co/600x300/3780f4/ffffff?text=Bare+image)

With a caption — the third argument (the title attribute, in quotes) becomes a
`<figcaption>` beneath the image. Alt text stays on the `<img>` for screen
readers:

![Alt text describing the image](https://placehold.co/600x300/f9423a/ffffff?text=With+caption "Figure 1: caption shown beneath the image")

To embed your own images, drag-and-drop or paste them into the editor — they
upload to the `assets` collection and an `![](…)` reference is inserted at the
cursor.

## Horizontal rule

Three dashes on a line by themselves:

---

## Callouts

Four flavors: `note`, `tip`, `warning`, `danger`. The container syntax is
`::: name` to open and `:::` to close.

::: note
A neutral aside. Good for clarifying context or pointing at a related page.
:::

::: tip
A helpful recommendation or best practice.
:::

::: warning
Something the reader should be careful about — a possible footgun or a
surprising default.
:::

::: danger
A destructive or irreversible action — data loss, security implications, etc.
:::

## YouTube embeds

A line containing only a YouTube URL becomes an embedded player. `watch?v=`,
`youtu.be/`, `shorts/`, and `embed/` URLs all work, and a `?t=` / `?start=`
parameter is preserved.

https://www.youtube.com/watch?v=dQw4w9WgXcQ

## Suppressing the auto-TOC

By default the right-hand sidebar auto-builds a table of contents from the
H2/H3 headings on the page. To hide it for a particular doc, place this
HTML-comment directive on the very first line of the body (before any other
content):

    <!-- toc: false -->

`<!-- no-toc -->` works too. The directive is stripped from the rendered
output, so it never appears as visible text.

## Things to know

- **Raw HTML is stripped.** `html: false` is set on both the saved view and
  the editor preview, so `<script>` tags and inline event handlers like
  `onerror=` render as escaped text instead of executing — even when typed
  by an editor account.
- **Math and Mermaid don't render in the view.** md-editor-v3's preview pane
  will render KaTeX (`$$…$$`) and Mermaid (a `mermaid` fenced block), but
  pb-wiki's saved view doesn't ship those plugins. Content that depends on
  them will look right while editing and wrong on the rendered page.
- **Heading anchors are invisible.** There's no `#` permalink icon next to
  headings — the sidebar TOC is the navigation affordance, and `#fragment`
  URLs jump to the right heading because each carries an `id`.
