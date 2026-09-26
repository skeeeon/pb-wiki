# pb-wiki

A flat-feeling markdown wiki built on [PocketBase](https://pocketbase.io) + Vue 3.

- **Single Go binary.** PocketBase is used as a Go framework; the Vue build is bundled into the binary via `//go:embed`.
- **Markdown documents** organized by slash-separated paths (`engineering/runbooks/deploy`). The tree is implicit in the path; moving a subtree is a prefix update.
- **Three roles**: `admin` / `editor` / `viewer`.
- **Per-page access** (public / private / restricted to groups), enforced by native PocketBase collection rules. New pages inherit from their parent page.
- **SSO via PocketBase OAuth providers** — combined with PocketBase's native `OnlyDomains` validator on the users `email` field, this replaces the need for oauth2-proxy in front of the app.
- **Split-pane markdown editing** via [md-editor-v3](https://github.com/imzbf/md-editor-v3).

## Quick start (Docker)

```bash
docker build -t pb-wiki .
docker run --rm -p 8090:8090 -v $(pwd)/pb_data:/home/pbwiki/pb_data pb-wiki
```

Visit:
- `http://localhost:8090/_/` — PocketBase admin UI (create a superuser on first run)
- `http://localhost:8090/` — the wiki frontend

To seed an initial admin user:

```bash
# Inside the running container — or with `go run . superuser upsert ...` locally
docker exec -it <container> /usr/local/bin/pb-wiki superuser upsert admin@example.com 'a-long-password'
```

Then, in the PocketBase admin UI, add a row to the `users` collection (`email`, `password`, `role=admin`) so you have an in-app admin to manage the wiki via the Vue UI.

## Development

```bash
# Backend — Go 1.27+, talks to PocketBase on :8090
go run . serve

# Frontend — Vite dev server on :5173, proxies /api and /_ to :8090
cd frontend
npm install
npm run dev
```

Both ports must run side-by-side during dev. Open `http://localhost:5173/`.

## Build the single binary locally

```bash
cd frontend && npm install && npm run build && cd ..
go build -o pb-wiki .
./pb-wiki serve
```

The Vue build under `frontend/dist/` is embedded into the binary; redistributing the binary alone is enough to serve the full app.

## Importing content

`pb-wiki import <markdown-dir>` recursively imports a directory of markdown files into the `documents` collection. This is the input side of a git-ops authoring workflow: keep your content in a git repo as plain markdown and re-run the importer to upsert into the wiki. Imports are one-way — the wiki does not write back to disk.

Each file must begin with YAML frontmatter declaring `path` (use `path: ""` for the homepage). Everything else is optional:

```markdown
---
path: getting-started/install
title: Installation Guide   # default: the first H1 in the body
access: private             # public | private | restricted; default: inherit from the parent page
groups: [ops]               # group names for restricted; missing groups are created
nav_order: 20               # sidebar position among siblings (then by name)
---
# Installation Guide
...
```

Relative links to other files in the tree, such as `[Deploy](../ops/deploy.md#rollback)`, are rewritten to wiki URLs (`/doc/ops/deploy#rollback`), so the source files still link correctly on disk and on a git host. A relative `.md` link that matches no imported file is left as it is and reported.

Records are matched by `path`, so re-running the import updates existing documents in place. On an update, `access`, `groups` and `nav_order` only change when the frontmatter sets them. Parents are written before their children, so new pages inherit access from a parent created in the same run. Files without a `path` are logged and skipped; duplicate paths within the input tree are reported as an error before any writes happen.

```bash
go run . import ./wiki        # or ./pb-wiki import ./wiki for the built binary
```

### Migrating from MkDocs

[`scripts/mkdocs-convert`](./scripts/mkdocs-convert/main.go) copies an MkDocs `docs/` directory into import format: it adds `path` and `nav_order` frontmatter from the `mkdocs.yml` nav, turns `!!!` / `???` admonitions into titled `:::` callouts, writes a page for each nav section folder that lacks one, and reports `#anchor` links that won't resolve. The source is never modified.

```bash
go run ./scripts/mkdocs-convert -site ../my-docs -prefix handbook -out ./import
go run . import ./import
```

### Exporting content

pb-wiki itself is import-only — there is no `export` subcommand and the wiki never writes back to disk. To get content out, use [pb-cli](https://github.com/skeeeon/pb-cli) plus `jq` to materialize each `documents` record as a frontmatter-prefixed `.md` file.

This is the right tool when you want a **flat snapshot on disk** — seeding a new wiki, periodic backups, or bulk text transforms you'd rather run with `rg`/`sed` than through PocketBase filters. For AI/agent access, don't export; see [Using the wiki with an AI agent](#using-the-wiki-with-an-ai-agent) below — a static dump goes stale as soon as anyone edits a page.

Because `path` lives in the frontmatter, the exported files can live in a single flat directory — name them by record ID for stability across renames:

```bash
mkdir -p wiki
pb c list documents --limit 500 --output json \
  | jq -c '.items[]' \
  | while read -r row; do
      id=$(jq -r '.id' <<<"$row")
      {
        printf -- '---\npath: %s\ntitle: %s\n---\n' \
          "$(jq -r '.path' <<<"$row")" \
          "$(jq -r '.title' <<<"$row")"
        jq -r '.body' <<<"$row"
      } > "wiki/${id}.md"
    done
```

The resulting files round-trip cleanly through `pb-wiki import`. Caveats:

- `--limit 500` is one page; for larger wikis paginate with `--page` or raise the limit.
- A `title` containing a YAML metacharacter (e.g. an unquoted colon) would produce invalid frontmatter; quote or sanitize titles up front if that's a concern.

## Using the wiki with an AI agent

For Claude Code or similar tooling, query the wiki live rather than working off an exported snapshot. A snapshot goes stale the moment someone edits a page, the record-ID filenames defeat name-based grep, and reading every body up-front burns context.

The repo ships a Claude Code skill at [`.claude/skills/wiki/SKILL.md`](./.claude/skills/wiki/SKILL.md) that wraps [pb-cli](https://github.com/skeeeon/pb-cli) with an **index → fetch** pattern: list `title,path` first, pull `body` only for the page(s) you actually need, and route writes through a draft-and-confirm flow that respects the same page access the UI enforces. It activates automatically when an agent working in this repo is asked about "the wiki".

One-time setup per machine:

```bash
pb context add wiki --url https://your-wiki.example.com   # see pb-cli docs
pb context select wiki
pb auth
```

Reach for the export workflow above instead when you specifically need a flat tree of every page (snapshots, backups, bulk transforms) — not for routine AI lookup or edits.

## Supported markdown

CommonMark plus a small, opinionated set of extensions:

| Feature | Syntax |
|---|---|
| Heading anchors | auto on every heading; lowercase, punctuation except `_` and `-` removed (same ids as GitHub and MkDocs) |
| Task lists | `- [ ]` / `- [x]` |
| Subscript | `~text~` |
| Superscript | `^text^` |
| Highlight | `==text==` |
| Image caption | `![alt](url "caption")` → `<figure>` + `<figcaption>` |
| Callouts | `::: note`, `::: tip`, `::: warning`, `::: danger`, optionally followed by a title (`::: warning Back up first`); close with `:::` |
| YouTube embeds | a line containing only a YouTube URL |
| Mermaid diagrams | ` ```mermaid ` fenced block (library lazy-loaded on first use) |
| Frontmatter table | leading `---` … `---` YAML block renders as a key/value table |
| Hide auto-TOC | `<!-- toc: false -->` or `<!-- no-toc -->` on the first line |

Raw HTML is stripped on both the saved view and the editor preview
(`html: false`), so `<script>` and inline event handlers can't slip through —
even from a compromised editor account.

See [`examples/markdown-reference.md`](./examples/markdown-reference.md) for a
copy-pasteable cheatsheet that exercises every feature. `go run . import
./examples` imports it into the wiki as `/doc/examples/markdown-reference`.

## Schema and migrations

Migrations live in [`migrations/`](./migrations/) and self-register via `init()`. They run automatically on first boot (`Automigrate` is enabled when running via `go run`, and explicitly through `pb-wiki migrate up` for prod binaries).

| Collection | Purpose |
|---|---|
| `users` (auth) | Extends PB's stock collection with `role` (admin/editor/viewer) and `groups` (relation to `groups`). |
| `groups` | `name` (unique). Editors and admins can list them; only admins create or change them. |
| `documents` | `path` (unique), `title`, `body` (markdown), `access` (public/private/restricted), `groups`, `updated_by`. Empty `path` is the homepage. |
| `assets` | Uploaded images embedded in markdown. Each belongs to the first document that embeds it and follows that document's access. |
| `wiki_config` | Singleton row: `title`, `private_default`, `require_login`, `default_landing_path`. |

## Permissions

Every rule below is a native PocketBase collection rule, so the record API, list counts, pagination and realtime subscriptions all enforce the same thing.

- **Each page has its own access.** `public`: anyone. `private`: any logged-in user. `restricted`: users who share at least one group with the page. Admins and PocketBase superusers read everything.
- **New pages inherit.** A page created without an access setting copies `access` and `groups` from its nearest ancestor page; a top-level page gets `private` when `private_default` is on, else `public`.
- **`require_login`** hides even public pages from anonymous visitors.
- **Roles gate writes.** Admins and editors create pages, and can update or delete any page they can read.
- **Only admins set `role` and `groups`.** Sign-up (password or OAuth) always creates a `viewer`; users can edit their own profile but not their role or groups.
- **Images follow their page.** A page claims the uploaded images it embeds when it is saved; an image not yet in any page is visible to editors and admins only. Because `<img>` requests can't carry the `Authorization` header, the SPA mirrors the auth token into a `pbwiki_auth` cookie scoped to `/api/files/`, which the download hook reads.
- **OAuth domain allow-list**: configured natively in PocketBase under Collections → `users` → `email` field → "Only domains" (set the list of allowed domains there). Applies to both password sign-up and OAuth.

Put the PocketBase admin UI (`/_/`) behind your reverse proxy's access control if the wiki is public.

## Project layout

```
pb-wiki/
├── main.go                      # pocketbase.New() + hooks + static embed
├── frontend.go                  # //go:embed all:frontend/dist
├── internal/
│   ├── api/                     # search, history and bulk-move endpoints
│   ├── hooks/                   # access inheritance, asset ownership + downloads, default role
│   └── static/                  # SPA fallback handler mounted on PB router
├── migrations/                  # Go-style PB migrations
└── frontend/                    # Vue 3 + Vite + TS + Tailwind v4 + Reka UI
    ├── src/
    │   ├── components/          # Sidebar, Breadcrumbs, MarkdownView, admin/*
    │   ├── composables/         # useDoc
    │   ├── lib/                 # pb (SDK singleton), types
    │   ├── stores/              # auth, config (Pinia)
    │   ├── views/               # DocView, DocEdit, Login, NotFound, admin/*
    │   └── router/
    └── public/
```

## Tests

```bash
go test ./...
```

- `internal/hooks` — runs the real API against a fresh database: page access through list, view, realtime and search; group overlap; inheritance; write access; sign-up and self-edit escalation; image downloads; and a round trip of the migration from the old path rules.
- `internal/importer` — covers YAML frontmatter parsing (BOM, CRLF, unterminated blocks, invalid YAML) and the H1 title fallback.

## License

TBD
