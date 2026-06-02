package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Create the documents_fts SQLite FTS5 virtual table plus the triggers that
// keep it in sync with the documents collection. Full-text search runs as a
// MATCH against this table (see internal/api/search.go).
//
// Sync is done entirely in SQLite via AFTER INSERT/UPDATE/DELETE triggers, so
// the index stays current no matter how a document changes — API writes, the
// importer, bulk-move, or hand edits in the PB admin UI — with no
// application-level sync code to maintain.
//
// The index covers title + body with the `porter` tokenizer (English stemming:
// a search for "running" also matches "run"). `id` is stored UNINDEXED purely
// so matched rows can be joined back to the documents table; PocketBase record
// ids are TEXT, so they can't double as the FTS5 integer rowid.
func init() {
	m.Register(func(app core.App) error {
		stmts := []string{
			`CREATE VIRTUAL TABLE documents_fts USING fts5(
				id UNINDEXED, title, body, tokenize='porter'
			)`,
			// Backfill rows that already exist at migration time.
			`INSERT INTO documents_fts(id, title, body)
				SELECT id, title, body FROM documents`,
			`CREATE TRIGGER documents_fts_ai AFTER INSERT ON documents BEGIN
				INSERT INTO documents_fts(id, title, body)
					VALUES (new.id, new.title, new.body);
			END`,
			`CREATE TRIGGER documents_fts_au AFTER UPDATE ON documents BEGIN
				UPDATE documents_fts SET title = new.title, body = new.body
					WHERE id = new.id;
			END`,
			`CREATE TRIGGER documents_fts_ad AFTER DELETE ON documents BEGIN
				DELETE FROM documents_fts WHERE id = old.id;
			END`,
		}
		for _, s := range stmts {
			if _, err := app.DB().NewQuery(s).Execute(); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		stmts := []string{
			`DROP TRIGGER IF EXISTS documents_fts_ai`,
			`DROP TRIGGER IF EXISTS documents_fts_au`,
			`DROP TRIGGER IF EXISTS documents_fts_ad`,
			`DROP TABLE IF EXISTS documents_fts`,
		}
		for _, s := range stmts {
			if _, err := app.DB().NewQuery(s).Execute(); err != nil {
				return err
			}
		}
		return nil
	})
}
