---
name: wiki
description: Query pb-wiki — a PocketBase-backed knowledge base — via the `pb` CLI. Use whenever the user references "the wiki", asks to look up a documented page or procedure, or asks a question that is likely answered by a stored wiki document rather than by the local codebase.
---

# Wiki skill

Access to the `documents` collection in the pb-wiki PocketBase instance via the `pb` CLI. Read-by-default; writes go through a draft-and-confirm flow (see [Writing](#writing)).

## Bootstrap — run this first

Before issuing any wiki query, confirm there is an active, authenticated context that exposes the `documents` collection:

```bash
pb context show
```

Inspect the output for all three:

- **`Active context:` line is present** — if the command errors with "no active context", tell the user to run `pb context list` to see options, then `pb context select <name>`.
- **`Authentication: Valid (expires …)`** — if it says expired/missing, the user must run `pb auth` themselves (it's interactive). Tell them to type `! pb auth` in this session so the output lands in the conversation.
- **`Available Collections` includes `documents`** — if not, this context isn't a wiki; ask the user to switch contexts.

If all three pass, proceed. Don't repeat the bootstrap check within the same conversation unless a later command fails with an auth/context error.

## Schema

Each document has: `title`, `path`, `body` (markdown), `access`, `groups`, `nav_order`.

- `path` is the canonical slug — a forward-slash-delimited hierarchy (e.g. `section/page`, `section/subsection/page`). The root landing page has `path=""`.
- `access` is who can read the page: `public` (anyone), `private` (any logged-in user) or `restricted` (users who share at least one group with the page). `groups` holds **group record ids**, not names; it only matters for `restricted`.
- `nav_order` is the page's position in the sidebar among its siblings (then by name; `0` is the default).

The server only ever returns pages the authenticated user can read — list results and `totalItems` already exclude the rest. If a page the user expects is missing, it may exist but be hidden from their account; say so rather than concluding it doesn't exist.

Links between pages in a body are written as `/doc/<path>` (the homepage is `/`), with an optional `#heading-id`.

## Workflow: index → fetch

Don't pull `body` in bulk — it can blow out your context. Two-step it:

1. **Index** — get titles + paths to find candidate documents.
2. **Fetch** — pull `body` for the specific path(s) you need.

### 1. Index (always start here unless you already know the path)

List all titles + paths:

```bash
pb collections list documents --fields=title,path -o yaml --limit 100
```

Substring search across title or body (PocketBase `~` operator):

```bash
pb collections list documents --filter='title~"KEYWORD" || body~"KEYWORD"' --fields=title,path -o yaml --limit 100
```

Scope to a section by path prefix:

```bash
pb collections list documents --filter='path~"section-name/"' --fields=title,path -o yaml --limit 100
```

### 2. Fetch a single document by path

```bash
pb collections list documents --filter='path="section-name/page-slug"' --fields=title,path,body -o yaml
```

(Use `--filter` with `path=`, not `get <id>` — the user thinks in paths, not record IDs.)

### Fetch multiple specific documents

```bash
pb collections list documents --filter='path="a" || path="b" || path="c"' --fields=title,path,body -o yaml
```

## Conventions

- **Always pass `-o yaml`** — it renders markdown bodies as readable block scalars instead of JSON-escaped strings.
- **Always pass `--fields`** — never fetch every column; `body` is the expensive one, only request it when you need it.
- **Default `--limit 100`** on index queries; bump higher (or paginate) if `totalPages > 1` in the response.
- **`||` and `&&`** are the PocketBase filter operators (not `or`/`and`).
- **Quote string values with double quotes** inside the filter, single-quote the whole `--filter` arg.

## Writing

Writes (create / update) MUST go through a draft-and-confirm flow. Never call `pb collections create|update documents …` without an explicit "push it" / "ship it" / "yes" from the user on a draft they have seen. Wiki edits are visible to the team — treat them like a `git push`.

Deletes are out of scope for this skill. If the user asks to delete a page, stop and ask them to confirm by running `pb collections delete documents <id>` themselves.

### Draft file format

Drafts live at `/tmp/pb-wiki-drafts/<slugified-path>.md` (create the directory if missing). Each draft uses YAML frontmatter so the title, path, and body travel together as one reviewable artifact:

```markdown
---
path: section/page-slug
title: Page Title
# optional — include only when the user asked to set or change them:
# access: private          # public | private | restricted
# groups: [finance]        # group NAMES, for restricted
# nav_order: 20
---

<markdown body here>
```

Leave `access` out of a new page's draft unless the user chose one: the server then copies access and groups from the nearest parent page (or the wiki default for top-level pages). Leave it out of an update's draft to keep the page's current access.

### Updating an existing page

1. **Resolve the record** — look up `id`, `title`, current `body` and access by path:

   ```bash
   pb collections list documents --filter='path="section/page-slug"' --fields=id,path,title,body,access,groups,nav_order -o yaml
   ```

   Capture the `id` — you'll need it at push time. If `totalItems` is 0, the page doesn't exist; switch to the create flow below.

2. **Write the draft** to `/tmp/pb-wiki-drafts/<slugified-path>.md` with the current state prefilled, then apply your edits.

3. **Tell the user** the draft path and a one-sentence summary of what's changing (e.g. "Added a Troubleshooting section and tightened the intro"). Invite them to open it, edit in place, and reply when ready.

4. **Wait for explicit confirmation.** Don't push speculatively.

5. **Push.** Re-read the draft (frontmatter may have been edited too), build the JSON payload with `yq` + `jq`, and update by ID:

   ```bash
   DRAFT=/tmp/pb-wiki-drafts/<slug>.md
   TITLE=$(yq --front-matter=extract '.title' "$DRAFT")
   PATH_VAL=$(yq --front-matter=extract '.path' "$DRAFT")
   BODY=$(awk 'BEGIN{n=0} /^---$/{n++; next} n>=2{print}' "$DRAFT")
   jq -n --arg t "$TITLE" --arg p "$PATH_VAL" --arg b "$BODY" \
     '{title:$t, path:$p, body:$b}' > /tmp/pb-wiki-drafts/<slug>.json

   pb collections update documents <id> --file /tmp/pb-wiki-drafts/<slug>.json -o yaml
   ```

   This pipeline round-trips body markdown cleanly through quotes, backticks, fenced code blocks, and colons. `yq` here is mikefarah's Go yq v4+ (not the Python `yq`).

   If the draft sets `access`, `groups` or `nav_order`, add them to the payload. Groups must be sent as ids, so look each name up first (only editors and admins can list groups; a missing group means an admin has to create it — stop and tell the user):

   ```bash
   pb collections list groups --filter='name="finance"' --fields=id,name -o yaml
   jq --arg a restricted --argjson g '["<group id>"]' --argjson n 20 \
     '. + {access:$a, groups:$g, nav_order:$n}' /tmp/pb-wiki-drafts/<slug>.json > /tmp/pb-wiki-drafts/<slug>.tmp \
     && mv /tmp/pb-wiki-drafts/<slug>.tmp /tmp/pb-wiki-drafts/<slug>.json
   ```

   For `public` or `private`, send `groups: []`.

6. **Confirm.** Show the user the returned record's `updated` timestamp from the YAML output as proof.

### Creating a new page

Same flow, with two differences:

- Skip step 1 (no record to look up). Pick the new `path` with the user before drafting — it's the URL and is hard to change later.
- At push time, use `create` instead of `update` (no `<id>` arg):

  ```bash
  pb collections create documents --file /tmp/pb-wiki-drafts/<slug>.json -o yaml
  ```

### Guardrails

- **Show, don't tell.** Always write the draft to disk so the user can see the full text. Don't summarize what you're about to push.
- **Don't auto-push after edits.** Each round of changes is a new draft; re-confirm.
- **One page per draft.** Multi-page edits = multiple drafts, each confirmed independently. It's tempting to batch, but each push is independently visible.
- **Don't invent paths or titles.** If the user hasn't specified them, ask.
- **Access changes are visibility changes.** Never change `access` or `groups` unless the user asked. When a draft does, say plainly who gains or loses the ability to read the page (e.g. "this makes the page readable by anyone on the internet") and get a separate yes for that.

## What this skill does NOT do

- **No semantic search.** Filters are exact/substring only. If a keyword search returns nothing useful, broaden the term or fall back to listing titles and letting the user pick.
- **No deletes.** See note above.

## Common gotchas

- The root document has `path=""` (empty string), not `path="/"` or `path="root"`. Filter with `--filter='path=""'`.
- Paths use forward slashes and are case-sensitive.
- If `totalItems` exceeds `perPage` in the result, paginate with `--page 2`, `--page 3`, etc., or raise `--limit`.
