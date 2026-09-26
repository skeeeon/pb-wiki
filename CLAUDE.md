# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go 1.27+)
```bash
go run . serve            # run PocketBase on :8090; Automigrate is ON in `go run` mode
go run . migrate up       # explicit migration (prod binaries don't auto-migrate)
go run . superuser upsert <email> <password>   # seed a PB superuser
go run . import <markdown-dir>                 # import markdown files with YAML frontmatter (see internal/importer)
go test ./...             # all unit tests
go test ./internal/hooks -run TestSearch      # single test
```

### Frontend (Vue 3 + Vite, in `frontend/`)
```bash
npm install
npm run dev          # Vite on :5173, proxies /api and /_ to :8090 — needs `go run . serve` running too
npm run build        # vue-tsc -b && vite build → frontend/dist (embedded into Go binary)
npm run type-check   # vue-tsc --noEmit
```

### Single-binary release build
The Vue build must exist under `frontend/dist/` before `go build` because `//go:embed all:frontend/dist` runs at compile time:
```bash
cd frontend && npm install && npm run build && cd ..
go build -o pb-wiki .
```

The Dockerfile already does this in two stages — prefer `docker build -t pb-wiki .` for reproducible builds.

## Architecture

This is a single-binary wiki: a Go program built on **PocketBase as a framework** (not a separate service) with a Vue 3 SPA embedded into the binary. Three things are wired in `main.go`: PB migrations register themselves, `hooks.Register` installs model hooks, and `static.Register` mounts the embedded SPA with a catch-all priority-999 handler so any earlier API route wins.

### The permission model (load-bearing)

All access is enforced by **native PocketBase collection rules** (migration `1700000090`). Each document has `access` (public/private/restricted) and `groups`; the `documents` ListRule/ViewRule allow admins, `public` pages (unless `wiki_config.require_login` and anonymous), `private` pages for any logged-in user, and `restricted` pages via `groups.id ?= @request.auth.groups.id` (at least one shared group). Update/Delete require editor/admin **and** read access.

Keep it in the rules. Filtering records in Go after the query leaks through `totalItems`, page boundaries and realtime broadcasts, which only check collection rules. Custom endpoints that query documents directly (`internal/api/search.go`, `history.go`) must check each row with `canView` (`internal/api/access.go`), which evaluates the same ViewRule.

Hooks only fill in data: `hooks/documents.go` copies access from the nearest ancestor page on create (falling back to `private_default`), and `hooks/assets.go` lets documents claim the assets they embed and checks asset downloads against the owning document. Asset downloads read the auth token from the `pbwiki_auth` cookie, because `<img>` requests can't send the Authorization header.

### Path conventions

`documents.path` is the slash-separated slug **without a leading slash**. The empty string is the homepage. The tree is *implicit* in the paths — there is no parent/child table. A "move subtree" is a prefix-update operation. The unique index on `path` enforces one homepage.

### Collections (see `migrations/`)

- `users` (auth) — extends PB's stock collection with `role` (admin/editor/viewer) and `groups` (relation to `groups`). Default role for new users is `viewer` (`hooks/auth.go`). The Create/Update rules reject `role` and `groups` from anyone but an admin, so sign-up and self-edit cannot escalate.
- `groups` — `name` unique. The UI edits groups as comma-separated names; `frontend/src/lib/groups.ts` maps names to ids.
- `documents` — markdown content; `path` unique, `access`, `groups`, `updated_by` relation to users.
- `assets` — uploaded images for markdown embeds; `document` is the owning page (set by the claim hook). Records are editor/admin-only; files are served through the download hook.
- `wiki_config` — singleton row (seeded by its migration). Holds `private_default`, `require_login`, `default_landing_path`. CreateRule is `nil` to keep it singleton.

### OAuth allowlist

Email-domain gating is deliberately *not* a pb-wiki feature — PocketBase's `EmailField.OnlyDomains` validator on the `users.email` field handles it natively for both password sign-up and OAuth. Admins set the list in the PB admin UI under Collections → users → `email` → Only domains. The only `users`-collection hook pb-wiki installs is `hooks/auth.go`'s default-role assigner (newly-created users default to `viewer` since OAuth sign-up doesn't supply a role).

### Frontend

Vue 3 + Vite + TS + Tailwind v4 + Reka UI + Pinia.

- `lib/pb.ts` — singleton PocketBase SDK client. Uses same-origin in both dev (via Vite proxy) and prod (the SPA is served by the same Go binary), so no backend host gets baked into the bundle.
- `stores/auth.ts` — wraps `pb.authStore` and re-broadcasts via Pinia; the SDK persists to localStorage so refresh keeps the session.
- `composables/useDoc.ts` — fetches the doc at a reactive path; exposes `loading / notFound / error / doc` refs.
- `composables/useSearch.ts` — **hybrid search**: synchronous in-memory filter for title/path matches, plus a debounced (200ms) server-side `body ~ {q}` filter for content matches. Results are merged with title/path winning on dedupe.
- `router/index.ts` — `:path(.*)*` catches arbitrarily deep slugs; the props mapper joins the segments into a single string before handing it to `DocView` / `DocEdit`. `requiresRole` meta is enforced in `beforeEach`.

### Migrations

Each file in `migrations/` self-registers via `init()` against PB's migration registry. `main.go` enables `Automigrate` only when running via `go run` (detected by inspecting `os.Args[0]`); production binaries must run `pb-wiki migrate up` explicitly. New migrations should follow the existing `1700000NNN_*.go` numbering.

### Where the SPA is served from

`frontend.go` at the repo root holds the `//go:embed all:frontend/dist` directive (Go disallows `..` in embed paths, which is why it can't live under `internal/static`). `internal/static/static.go` mounts it onto the PB router with `Priority: 999`, so API/realtime routes registered earlier always win.
