package api

import (
	"encoding/json"
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

// notTimestampOnly leaves out saves that changed nothing but the `updated`
// timestamp. pb-audit writes changed_fields exactly as `["updated"]` for
// those. Rows from before pb-audit v0.2 have no changed_fields (NULL), and
// IS NOT keeps them.
const notTimestampOnly = `changed_fields IS NOT '["updated"]'`

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

	q := e.App.RecordQuery("audit_logs").
		AndWhere(dbx.HashExp{"collection_name": "documents", "record_id": doc.Id}).
		AndWhere(dbx.In("event_type", "create", "update")).
		AndWhere(dbx.NewExp(notTimestampOnly)).
		OrderBy("timestamp DESC").
		Limit(int64(limit))
	if before := strings.TrimSpace(e.Request.URL.Query().Get("before")); before != "" {
		q = q.AndWhere(dbx.NewExp("timestamp < {:before}", dbx.Params{"before": before}))
	}

	var records []*core.Record
	if err := q.All(&records); err != nil {
		return e.InternalServerError("Failed to load history.", err)
	}

	// The editor of each revision is the page's `updated_by` in that
	// revision's snapshot. audit_logs.user can't be used: pb-audit fills it
	// only for request events, and these create/update events are written
	// after the save, with no request attached.
	revisions := make([]historyRevision, 0, len(records))
	var userIDs []string
	for _, r := range records {
		rev := historyRevision{
			ID:        r.Id,
			Timestamp: r.GetDateTime("timestamp").String(),
			EventType: r.GetString("event_type"),
			Before:    jsonRaw(r, "before_changes"),
			After:     jsonRaw(r, "after_changes"),
		}
		if id := updatedBy(rev.After); id != "" {
			userIDs = append(userIDs, id)
		}
		revisions = append(revisions, rev)
	}

	// One query for all editors. A deleted user is simply not found, and
	// that revision shows no editor.
	users := map[string]*core.Record{}
	if len(userIDs) > 0 {
		found, err := e.App.FindRecordsByIds("users", userIDs)
		if err != nil {
			return e.InternalServerError("Failed to load history.", err)
		}
		for _, u := range found {
			users[u.Id] = u
		}
	}
	for i := range revisions {
		if u := users[updatedBy(revisions[i].After)]; u != nil {
			revisions[i].User = &historyUser{
				ID:    u.Id,
				Email: u.GetString("email"),
				Name:  u.GetString("name"),
			}
		}
	}

	return e.JSON(http.StatusOK, historyResponse{Revisions: revisions})
}

// updatedBy returns the `updated_by` user id from a documents snapshot, or ""
// when there is none (an import, a superuser edit, or a missing snapshot).
func updatedBy(snapshot types.JSONRaw) string {
	var s struct {
		UpdatedBy string `json:"updated_by"`
	}
	if json.Unmarshal(snapshot, &s) != nil {
		return ""
	}
	return s.UpdatedBy
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
