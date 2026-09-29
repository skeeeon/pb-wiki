package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// RegisterHistory wires GET /api/wiki/history onto the app router. The handler
// resolves a document by path, checks it against the documents ViewRule, and
// returns a curated list of revisions sourced from pb-audit's `audit_logs`
// collection. We funnel through this endpoint (rather than letting the
// frontend query audit_logs directly) so:
//
//   - audit_logs keeps its admin-only PB rules intact;
//   - the response shape is curated (no IP/auth_method/etc. leaked to viewers);
//   - access denials hide existence (404, not 403) to match the record API.
func RegisterHistory(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/wiki/history", handleHistory)
		return se.Next()
	})
}

// defaultHistoryLimit caps a single page. Frontend can paginate with ?before.
const defaultHistoryLimit = 50
const maxHistoryLimit = 200

type historyUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type historyRevision struct {
	ID        string        `json:"id"`
	Timestamp string        `json:"timestamp"`
	EventType string        `json:"event_type"`
	User      *historyUser  `json:"user"`
	Before    types.JSONRaw `json:"before"`
	After     types.JSONRaw `json:"after"`
}

type historyResponse struct {
	Revisions []historyRevision `json:"revisions"`
}

func handleHistory(e *core.RequestEvent) error {
	path := normalizePath(e.Request.URL.Query().Get("path"))

	// dbx.HashExp goes directly to parameterized SQL — we deliberately avoid
	// FindFirstRecordByFilter here because PB's filter parser JSON-encodes
	// empty-string params into a literal `""` value, which would prevent the
	// homepage (path="") from matching itself. Same workaround as the
	// importer's upsert (internal/importer/importer.go).
	docs, err := e.App.FindAllRecords("documents", dbx.HashExp{"path": path})
	if err != nil {
		return e.InternalServerError("Failed to load document.", err)
	}
	if len(docs) == 0 {
		return e.NotFoundError("", nil)
	}
	doc := docs[0]

	ok, err := canView(e, doc)
	if err != nil {
		return e.InternalServerError("Failed to check access.", err)
	}
	if !ok {
		// 404 (not 403) to hide existence, matching the record API.
		return e.NotFoundError("", nil)
	}

	limit := defaultHistoryLimit
	if raw := e.Request.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > maxHistoryLimit {
				n = maxHistoryLimit
			}
			limit = n
		}
	}

	filter := "collection_name = {:cn} && record_id = {:rid} && (event_type = 'create' || event_type = 'update')"
	params := dbx.Params{"cn": "documents", "rid": doc.Id}
	if before := strings.TrimSpace(e.Request.URL.Query().Get("before")); before != "" {
		filter += " && timestamp < {:before}"
		params["before"] = before
	}

	records, err := e.App.FindRecordsByFilter("audit_logs", filter, "-timestamp", limit, 0, params)
	if err != nil {
		return e.InternalServerError("Failed to load history.", err)
	}

	// Expand the user relation in-place so we can render editor identity in
	// the response without a round-trip per row. A deleted user is simply
	// left out, so an error here is a real database or schema failure.
	if len(records) > 0 {
		if failed := e.App.ExpandRecords(records, []string{"user"}, nil); len(failed) > 0 {
			return e.InternalServerError("Failed to load history.", failed["user"])
		}
	}

	revisions := make([]historyRevision, 0, len(records))
	for _, r := range records {
		rev := historyRevision{
			ID:        r.Id,
			Timestamp: r.GetDateTime("timestamp").String(),
			EventType: r.GetString("event_type"),
			Before:    jsonRaw(r, "before_changes"),
			After:     jsonRaw(r, "after_changes"),
		}
		if u := r.ExpandedOne("user"); u != nil {
			rev.User = &historyUser{
				ID:    u.Id,
				Email: u.GetString("email"),
				Name:  u.GetString("name"),
			}
		}
		revisions = append(revisions, rev)
	}

	return e.JSON(http.StatusOK, historyResponse{Revisions: revisions})
}

// jsonRaw safely extracts a JSON field as types.JSONRaw. If the field is
// missing or stored as some other type, return the empty value (`null`) so the
// frontend always sees valid JSON.
func jsonRaw(r *core.Record, field string) types.JSONRaw {
	switch v := r.Get(field).(type) {
	case types.JSONRaw:
		return v
	case []byte:
		return types.JSONRaw(v)
	case string:
		return types.JSONRaw(v)
	default:
		return types.JSONRaw("null")
	}
}
