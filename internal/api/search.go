package api

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterSearch wires GET /api/wiki/search onto the app router. The handler
// runs a SQLite FTS5 MATCH against documents_fts (kept in sync by triggers; see
// migration 1700000080) and then checks each matched row against the documents
// ViewRule.
//
// The FTS join lives outside PocketBase's record API, so the collection rules
// do NOT apply to it. Checking every row is therefore load-bearing: without
// it, restricted document bodies would be searchable (and snippet-leaked) by
// anyone. Denied rows are silently dropped.
func RegisterSearch(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/api/wiki/search", handleSearch)
		return se.Next()
	})
}

// We over-fetch FTS candidates because the access check below may drop some,
// then trim the survivors to resultLimit for the response.
const matchCandidateLimit = 200
const searchResultLimit = 50

// searchRow is the raw FTS join result. `id` UNINDEXED in the FTS table lets us
// join back to documents; snippet() builds a short body excerpt around the hit.
type searchRow struct {
	ID      string `db:"id"`
	Path    string `db:"path"`
	Title   string `db:"title"`
	Snippet string `db:"snippet"`
}

type searchResult struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

func handleSearch(e *core.RequestEvent) error {
	match := buildMatchQuery(e.Request.URL.Query().Get("q"))
	if match == "" {
		return e.JSON(http.StatusOK, searchResponse{Results: []searchResult{}})
	}

	// snippet() column index 2 = body (0=id UNINDEXED, 1=title, 2=body). Empty
	// open/close markers: the frontend escapes the snippet and inserts its own
	// <mark> tags, so returning HTML here would be double-escaped and unsafe.
	var rows []searchRow
	err := e.App.DB().NewQuery(`
		SELECT d.id AS id, d.path AS path, d.title AS title,
		       snippet(documents_fts, 2, '', '', '…', 12) AS snippet
		FROM documents_fts
		JOIN documents d ON d.id = documents_fts.id
		WHERE documents_fts MATCH {:q}
		ORDER BY rank
		LIMIT {:lim}`).
		Bind(dbx.Params{"q": match, "lim": matchCandidateLimit}).
		All(&rows)
	if err != nil {
		return e.InternalServerError("Search failed.", err)
	}

	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	records, err := e.App.FindRecordsByIds("documents", ids)
	if err != nil {
		return e.InternalServerError("Search failed.", err)
	}
	byID := make(map[string]*core.Record, len(records))
	for _, rec := range records {
		byID[rec.Id] = rec
	}

	results := make([]searchResult, 0, len(rows))
	for _, r := range rows {
		rec := byID[r.ID]
		if rec == nil {
			continue
		}
		ok, err := canView(e, rec)
		if err != nil {
			return e.InternalServerError("Failed to check access.", err)
		}
		if !ok {
			continue
		}
		results = append(results, searchResult{
			ID:      r.ID,
			Path:    r.Path,
			Title:   r.Title,
			Snippet: strings.TrimSpace(r.Snippet),
		})
		if len(results) >= searchResultLimit {
			break
		}
	}

	return e.JSON(http.StatusOK, searchResponse{Results: results})
}

// buildMatchQuery turns a raw user query into a safe FTS5 MATCH expression.
//
// We strip every character that isn't a letter or digit, so FTS5 query
// operators a user might type (", *, :, -, AND, NEAR, parentheses) can never
// reach the MATCH parser and trigger a "fts5: syntax error". Each surviving
// term becomes a prefix token, AND-ed together — that gives search-as-you-type
// behaviour ("auth" matches "authentication") without exposing FTS5 query
// syntax to the search box. Returns "" when nothing searchable remains.
func buildMatchQuery(raw string) string {
	terms := strings.FieldsFunc(raw, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(terms) == 0 {
		return ""
	}
	for i, t := range terms {
		terms[i] = t + "*"
	}
	return strings.Join(terms, " ")
}
