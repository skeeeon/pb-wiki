package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/skeeeon/pb-wiki/internal/access"
	"github.com/skeeeon/pb-wiki/internal/hooks"
)

// RegisterHistory wires GET /api/wiki/history onto the app router. The handler
// resolves a document by path, enforces pb-wiki's path-based access rules, and
// returns a curated list of revisions sourced from pb-audit's `audit_logs`
// collection. We funnel through this endpoint (rather than letting the
// frontend query audit_logs directly) so:
//
//   - audit_logs keeps its admin-only PB rules intact;
//   - the response shape is curated (no IP/auth_method/etc. leaked to viewers);
//   - access denials hide existence (404, not 403) to match documents.go.
func RegisterHistory(app *pocketbase.PocketBase) {
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
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	EventType string         `json:"event_type"`
	User      *historyUser   `json:"user"`
	Before    types.JSONRaw  `json:"before"`
	After     types.JSONRaw  `json:"after"`
}

type historyResponse struct {
	Revisions []historyRevision `json:"revisions"`
}

func handleHistory(e *core.RequestEvent) error {
	path := normalizePath(e.Request.URL.Query().Get("path"))

	doc, err := e.App.FindFirstRecordByFilter(
		"documents",
		"path = {:p}",
		dbx.Params{"p": path},
	)
	if err != nil || doc == nil {
		return e.NotFoundError("", nil)
	}

	rules, err := hooks.LoadRules(e.App)
	if err != nil {
		return e.InternalServerError("Failed to load access rules.", err)
	}
	cfg, err := hooks.LoadConfigFlags(e.App)
	if err != nil {
		return e.InternalServerError("Failed to load wiki config.", err)
	}
	user := hooks.RecordToUser(e.Auth)
	if !access.CanAccess(path, user, rules, cfg.PrivateDefault, cfg.RequireLogin) {
		// 404 (not 403) to hide existence, matching documents.go.
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
	// the response without a round-trip per row.
	if len(records) > 0 {
		e.App.ExpandRecords(records, []string{"user"}, nil)
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
