package hooks

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// registerDocumentHooks fills in a new document's access when the creator
// did not choose one. Enforcement itself lives in the documents collection's
// API rules (migration 1700000090), which read the `access` and `groups`
// fields set here.
//
// This is a model hook, so it covers every way a document is created: the
// API, the importer, bulk move and the PB admin UI.
func registerDocumentHooks(app core.App) {
	app.OnRecordCreate("documents").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("access") == "" {
			if err := inheritAccess(e.App, e.Record); err != nil {
				return err
			}
		}
		return e.Next()
	})
}

// inheritAccess copies access and groups from the nearest existing ancestor
// document (for "a/b/c": "a/b", then "a"). A top-level document, or one with
// no ancestors, gets private when wiki_config.private_default is set and
// public otherwise. The homepage is not treated as everyone's parent.
func inheritAccess(app core.App, doc *core.Record) error {
	for p := parentPath(doc.GetString("path")); p != ""; p = parentPath(p) {
		parent, err := app.FindFirstRecordByData("documents", "path", p)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		doc.Set("access", parent.GetString("access"))
		doc.Set("groups", parent.GetStringSlice("groups"))
		return nil
	}

	cfg, err := app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		return err
	}
	if cfg.GetBool("private_default") {
		doc.Set("access", "private")
	} else {
		doc.Set("access", "public")
	}
	return nil
}

// parentPath returns "a/b" for "a/b/c" and "" for a top-level path.
func parentPath(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return ""
	}
	return p[:i]
}
