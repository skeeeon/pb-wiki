# pb-wiki

A flat-feeling markdown wiki built on [PocketBase](https://pocketbase.io) + Vue 3.

- **Single Go binary.** PocketBase is used as a Go framework; the Vue build is bundled into the binary via `//go:embed`.
- **Markdown documents** organized by slash-separated paths (`engineering/runbooks/deploy`). The tree is implicit in the path; moving a subtree is a prefix update.
- **Three roles**: `admin` / `editor` / `viewer`.
- **Path-based access rules** with first-match-wins glob matching (ported 1:1 from leomoon-studios/wiki-go's `internal/auth/access.go`).
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
# Backend — Go 1.25+, talks to PocketBase on :8090
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

Each file must begin with YAML frontmatter declaring `path` (use `path: ""` for the homepage); `title` is optional and falls back to the first H1 in the body.

```markdown
---
path: getting-started/install
title: Installation Guide
---
# Installation Guide
...
```

Records are matched by `path`, so re-running the import updates existing documents in place. Files without a `path` are logged and skipped; duplicate paths within the input tree are reported as an error before any writes happen.

```bash
go run . import ./wiki        # or ./pb-wiki import ./wiki for the built binary
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

The repo ships a Claude Code skill at [`.claude/skills/wiki/SKILL.md`](./.claude/skills/wiki/SKILL.md) that wraps [pb-cli](https://github.com/skeeeon/pb-cli) with an **index → fetch** pattern: list `title,path` first, pull `body` only for the page(s) you actually need, and route writes through a draft-and-confirm flow that respects the same path-based access rules the UI enforces. It activates automatically when an agent working in this repo is asked about "the wiki".

One-time setup per machine:

```bash
pb context add wiki --url https://your-wiki.example.com   # see pb-cli docs
pb context select wiki
pb auth
```

Reach for the export workflow above instead when you specifically need a flat tree of every page (snapshots, backups, bulk transforms) — not for routine AI lookup or edits.

## Schema and migrations

Migrations live in [`migrations/`](./migrations/) and self-register via `init()`. They run automatically on first boot (`Automigrate` is enabled when running via `go run`, and explicitly through `pb-wiki migrate up` for prod binaries).

| Collection | Purpose |
|---|---|
| `users` (auth) | Extends PB's stock collection with `role` (admin/editor/viewer) and `groups` (json array). |
| `documents` | `path` (unique), `title`, `body` (markdown), `updated_by`. Empty `path` is the homepage. |
| `assets` | Uploaded images embedded in markdown. Public file URLs. |
| `access_rules` | `pattern`, `access` (public/private/restricted), `groups`, `priority`, `description`. First-match-wins. |
| `wiki_config` | Singleton row: `title`, `private_default`, `require_login`, `default_landing_path`. |

## Permissions

- **Role hierarchy** gates *actions*: only admin/editor can create/update/delete documents.
- **Access rules** gate *paths*: rules are matched against `documents.path` (and the wiki's `private_default` flag is the fallback for unmatched paths). Admins bypass all rules.
- **OAuth domain allow-list**: configured natively in PocketBase under Collections → `users` → `email` field → "Only domains" (set the list of allowed domains there). Applies to both password sign-up and OAuth.

Path-glob syntax (matches wiki-go):

| Pattern | Meaning |
|---|---|
| `*` | any run of characters within one path segment |
| `**` | any run of characters across `/` |
| `/foo/**` | matches `/foo`, `/foo/`, and any descendant — the trailing-`/**` "parent or any child" form |
| `?` | one non-`/` character |
| `/**` (bare) | matches `/` only (does **not** swallow the whole wiki) |

See [`internal/access/`](./internal/access/) for the evaluator and its tests.

## Project layout

```
pb-wiki/
├── main.go                      # pocketbase.New() + hooks + static embed
├── frontend.go                  # //go:embed all:frontend/dist
├── internal/
│   ├── access/                  # path-rule evaluator (unit-tested, no PB import)
│   ├── hooks/                   # document access enforcement, OAuth allow-list, default role
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

- `internal/access` — covers glob translation, first-match precedence, admin bypass, group overlap, fail-closed unknown access levels, and the `require_login` lockdown flag.
- `internal/importer` — covers YAML frontmatter parsing (BOM, CRLF, unterminated blocks, invalid YAML) and the H1 title fallback.

## License

TBD
