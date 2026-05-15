# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go 1.25+)
```bash
go run . serve            # run PocketBase on :8090; Automigrate is ON in `go run` mode
go run . migrate up       # explicit migration (prod binaries don't auto-migrate)
go run . superuser upsert <email> <password>   # seed a PB superuser
go run . import <wiki-go-data-dir>             # one-shot wiki-go importer
go test ./...             # all unit tests
go test ./internal/access -run TestCanAccess   # single test
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

This is a single-binary wiki: a Go program built on **PocketBase as a framework** (not a separate service) with a Vue 3 SPA embedded into the binary. Three things are wired in `main.go`: PB migrations register themselves, `hooks.Register` installs request-time guards, and `static.Register` mounts the embedded SPA with a catch-all priority-999 handler so any earlier API route wins.

### The two-layer permission model (load-bearing)

Permissions are enforced at **two orthogonal layers**, ported from wiki-go's "RequireRole gates actions, AccessRules gate paths":

1. **Role gates actions** — collection-level API rules in migrations check `@request.auth.role` (admin/editor can create/update/delete; everyone can read).
2. **Path gates content** — `internal/hooks/documents.go` adds hooks on top of the `documents` collection that consult `internal/access` to decide whether the caller can see/touch a particular path. List requests filter rows in-memory; view returns **404 on deny** (to hide existence); create/update/delete return 403.

`internal/access` is intentionally PocketBase-free so it can be unit-tested in isolation — the `hooks` package is the seam that loads rules + the current user from PB records and calls into it. If you change the evaluator, update the tests there too; they cover wiki-go's exact glob semantics (`*`, `**`, `?`, trailing `/**` for "parent or any descendant", and the special case where bare `/**` matches only `/`).

### Path conventions

`documents.path` is the slash-separated slug **without a leading slash**. The empty string is the homepage. The tree is *implicit* in the paths — there is no parent/child table. A "move subtree" is a prefix-update operation. The unique index on `path` enforces one homepage.

The access evaluator normalizes both pattern and path to have a leading `/` before matching, so glob rules in `access_rules.pattern` can be written with or without the leading slash.

### Collections (see `migrations/`)

- `users` (auth) — extends PB's stock collection with `role` (admin/editor/viewer) and `groups` (json array). Default role for OAuth-created users is forced to `viewer` in `hooks/auth.go`.
- `documents` — markdown content; `path` unique, `updated_by` relation to users.
- `assets` — uploaded images for markdown embeds (public file URLs).
- `access_rules` — `pattern`, `access` (public/private/restricted), `groups`, `priority`. Loaded `ORDER BY priority ASC`; **first match wins**.
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
