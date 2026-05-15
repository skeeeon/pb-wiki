# pb-wiki

A flat-feeling markdown wiki built on [PocketBase](https://pocketbase.io) + Vue 3.

- **Single Go binary.** PocketBase is used as a Go framework; the Vue build is bundled into the binary via `//go:embed`.
- **Markdown documents** organized by slash-separated paths (`engineering/runbooks/deploy`). The tree is implicit in the path; moving a subtree is a prefix update.
- **Three roles**: `admin` / `editor` / `viewer`.
- **Path-based access rules** with first-match-wins glob matching (ported 1:1 from leomoon-studios/wiki-go's `internal/auth/access.go`).
- **SSO via PocketBase OAuth providers**, with an email-domain allow-list — replaces the need for oauth2-proxy in front of the app.
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

## Schema and migrations

Migrations live in [`migrations/`](./migrations/) and self-register via `init()`. They run automatically on first boot (`Automigrate` is enabled when running via `go run`, and explicitly through `pb-wiki migrate up` for prod binaries).

| Collection | Purpose |
|---|---|
| `users` (auth) | Extends PB's stock collection with `role` (admin/editor/viewer) and `groups` (json array). |
| `documents` | `path` (unique), `title`, `body` (markdown), `updated_by`. Empty `path` is the homepage. |
| `assets` | Uploaded images embedded in markdown. Public file URLs. |
| `access_rules` | `pattern`, `access` (public/private/restricted), `groups`, `priority`, `description`. First-match-wins. |
| `wiki_config` | Singleton row: `title`, `private_default`, `oauth_email_allowlist`, `default_landing_path`. |

## Permissions

- **Role hierarchy** gates *actions*: only admin/editor can create/update/delete documents.
- **Access rules** gate *paths*: rules are matched against `documents.path` (and the wiki's `private_default` flag is the fallback for unmatched paths). Admins bypass all rules.
- **OAuth domain allow-list**: configured in the admin Settings page. Bare entries (`example.com`) match anything `@example.com`; full emails (`alice@example.com`) match exactly. Empty list disables the check.

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

- `internal/access` — 36 cases covering glob translation, first-match precedence, admin bypass, group overlap, fail-closed unknown access levels.
- `internal/hooks` — 12 cases covering the OAuth email-domain allow-list matching.

## License

TBD
