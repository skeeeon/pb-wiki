// Package api hosts pb-wiki's hand-rolled HTTP endpoints that don't fit the
// stock PocketBase record CRUD model — currently just the bulk-move route,
// which atomically rewrites the `path` prefix of every document under a
// subtree.
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

// RegisterBulkMove wires POST /api/wiki/bulk-move onto the app router.
// The handler requires an authenticated admin user and rewrites the `path`
// prefix of every document under `from` to `to` in a single transaction.
func RegisterBulkMove(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/wiki/bulk-move", handleBulkMove).
			Bind(apis.RequireAuth("users"))
		return se.Next()
	})
}

type bulkMoveRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type bulkMoveItem struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type bulkMoveResponse struct {
	Moved int            `json:"moved"`
	Items []bulkMoveItem `json:"items"`
}

func handleBulkMove(e *core.RequestEvent) error {
	if e.Auth == nil || e.Auth.GetString("role") != "admin" {
		return e.ForbiddenError("Admin role required.", nil)
	}

	var req bulkMoveRequest
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Invalid JSON body.", err)
	}

	from := normalizePath(req.From)
	to := normalizePath(req.To)

	if from == "" {
		return e.BadRequestError("'from' prefix is required.", nil)
	}
	if from == to {
		return e.BadRequestError("'from' and 'to' are equal.", nil)
	}

	// path == from OR path starts with `from + "/"`. The `~` operator wraps
	// literals with `%` by default, so `prefix` carries an explicit trailing
	// `%` to opt into "starts with" rather than "contains".
	affected, err := e.App.FindRecordsByFilter(
		"documents",
		"path = {:from} || path ~ {:prefix}",
		"+path",
		0, 0,
		dbx.Params{"from": from, "prefix": from + "/%"},
	)
	if err != nil {
		return e.InternalServerError("Failed to load documents.", err)
	}
	if len(affected) == 0 {
		return e.NotFoundError("No documents match that prefix.", nil)
	}

	affectedIDs := make(map[string]struct{}, len(affected))
	newPaths := make(map[string]string, len(affected))
	items := make([]bulkMoveItem, 0, len(affected))
	for _, r := range affected {
		oldPath := r.GetString("path")
		affectedIDs[r.Id] = struct{}{}
		newP := rewritePath(oldPath, from, to)
		newPaths[r.Id] = newP
		items = append(items, bulkMoveItem{ID: r.Id, From: oldPath, To: newP})
	}

	// Outside-collision check: if a doc that isn't in the affected set
	// already holds one of our target paths, refuse — the move would either
	// silently overwrite that doc or trip the unique index mid-transaction.
	// Uses dbx.HashExp (not FindFirstRecordByFilter) so a move *to* the
	// homepage (item.To == "") correctly collides with the existing
	// path="" row; PB's filter parser would otherwise JSON-encode the
	// empty string into a literal `""` and miss the match.
	for _, item := range items {
		matches, _ := e.App.FindAllRecords("documents", dbx.HashExp{"path": item.To})
		if len(matches) == 0 {
			continue
		}
		existing := matches[0]
		if _, inSet := affectedIDs[existing.Id]; !inSet {
			return e.BadRequestError(
				fmt.Sprintf("Target path %q is already held by a document outside the moved set.", item.To),
				nil,
			)
		}
	}

	// Two-phase update: park every affected doc on a unique temp path
	// before stamping the final ones. This sidesteps any transient
	// collision on `path`'s unique index when the new namespace overlaps
	// the old one (e.g. moving "a" → "a/sub" with "a/sub" already in the
	// affected set, or any reshuffle that swaps siblings).
	err = e.App.RunInTransaction(func(txApp core.App) error {
		tempPrefix := fmt.Sprintf("__pb-wiki-bulk-move-%s__/", security.RandomString(8))
		for _, r := range affected {
			r.Set("path", tempPrefix+r.Id)
			if err := txApp.Save(r); err != nil {
				return err
			}
		}
		for _, r := range affected {
			r.Set("path", newPaths[r.Id])
			if err := txApp.Save(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return e.InternalServerError("Bulk move failed.", err)
	}

	return e.JSON(http.StatusOK, bulkMoveResponse{
		Moved: len(items),
		Items: items,
	})
}

// normalizePath trims whitespace and surrounding slashes so the handler can
// treat "/foo/", "foo", and " foo " as equivalent. Documents store paths
// without a leading slash (the empty string is the homepage), so we follow
// the same convention here.
func normalizePath(p string) string {
	return strings.Trim(strings.TrimSpace(p), "/")
}

// rewritePath maps a document path under `from` to the equivalent path under
// `to`. Caller guarantees that `old == from || strings.HasPrefix(old, from+"/")`.
func rewritePath(old, from, to string) string {
	if old == from {
		return to
	}
	suffix := old[len(from)+1:]
	if to == "" {
		return suffix
	}
	return to + "/" + suffix
}
