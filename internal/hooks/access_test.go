package hooks_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/skeeeon/pb-wiki/internal/api"
	"github.com/skeeeon/pb-wiki/internal/hooks"
	_ "github.com/skeeeon/pb-wiki/migrations"
)

// These tests run the real API against a fresh database with pb-wiki's
// migrations, hooks and search endpoint.
//
// Fixture pages: pub/a public, team/a private, fin/a restricted to finance,
// multi/a restricted to finance or ops. Users: admin, alice (finance),
// olivia (ops), bob (no groups), erin (editor, no groups), fred (editor,
// finance).

type fixture struct {
	app    *tests.TestApp
	mux    http.Handler
	groups map[string]string // name → id
	users  map[string]*core.Record
	docs   map[string]*core.Record // keyed by path
}

func newApp(t *testing.T) *fixture {
	t.Helper()
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	hooks.Register(app)
	api.RegisterSearch(app)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{app: app, groups: map[string]string{}, users: map[string]*core.Record{}, docs: map[string]*core.Record{}}
	se := &core.ServeEvent{App: app, Router: router}
	err = app.OnServe().Trigger(se, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		f.mux = mux
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := newApp(t)
	for _, name := range []string{"finance", "ops"} {
		f.groups[name] = f.save(t, "groups", map[string]any{"name": name}).Id
	}
	for name, u := range map[string]struct {
		role   string
		groups []string
	}{
		"admin":  {"admin", nil},
		"alice":  {"viewer", []string{"finance"}},
		"olivia": {"viewer", []string{"ops"}},
		"bob":    {"viewer", nil},
		"erin":   {"editor", nil},
		"fred":   {"editor", []string{"finance"}},
	} {
		f.users[name] = f.save(t, "users", map[string]any{
			"email": name + "@example.com", "password": "Passw0rd!Passw0rd",
			"role": u.role, "groups": f.ids(u.groups...),
		})
	}
	for path, d := range map[string]struct {
		access string
		groups []string
		body   string
	}{
		"pub/a":   {"public", nil, "public words"},
		"team/a":  {"private", nil, "teamsecret"},
		"fin/a":   {"restricted", []string{"finance"}, "finsecret"},
		"multi/a": {"restricted", []string{"finance", "ops"}, "multisecret"},
	} {
		f.docs[path] = f.save(t, "documents", map[string]any{
			"path": path, "title": path, "body": d.body, "access": d.access, "groups": f.ids(d.groups...),
		})
	}
	return f
}

func (f *fixture) ids(names ...string) []string {
	out := []string{}
	for _, n := range names {
		out = append(out, f.groups[n])
	}
	return out
}

func (f *fixture) save(t *testing.T, collection string, data map[string]any) *core.Record {
	t.Helper()
	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(data)
	if pw, ok := data["password"].(string); ok {
		r.SetPassword(pw)
	}
	if err := f.app.Save(r); err != nil {
		t.Fatalf("save %s %v: %v", collection, data, err)
	}
	return r
}

// do sends a request as the named user ("" = anonymous) and returns the
// status code and decoded JSON body.
func (f *fixture) do(t *testing.T, as, method, target string, body any) (int, map[string]any) {
	t.Helper()
	var payload string
	if body != nil {
		b, _ := json.Marshal(body)
		payload = string(b)
	}
	req := httptest.NewRequest(method, target, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if as != "" {
		token, err := f.users[as].NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// listTotal returns totalItems for a documents list request. Before access
// moved into the collection rules, this count leaked unreadable rows.
func (f *fixture) listTotal(t *testing.T, as, filter string) int {
	t.Helper()
	target := "/api/collections/documents/records"
	if filter != "" {
		target += "?filter=" + url.QueryEscape(filter)
	}
	code, out := f.do(t, as, http.MethodGet, target, nil)
	if code != http.StatusOK {
		t.Fatalf("list as %q filter %q: status %d %v", as, filter, code, out)
	}
	total, _ := out["totalItems"].(float64)
	return int(total)
}

func (f *fixture) reload(t *testing.T, collection, id string) *core.Record {
	t.Helper()
	r, err := f.app.FindRecordById(collection, id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSignupCannotChooseRoleOrGroups(t *testing.T) {
	f := newFixture(t)
	signup := func(extra map[string]any) (int, map[string]any) {
		body := map[string]any{"email": "new@example.com", "password": "Passw0rd!Passw0rd", "passwordConfirm": "Passw0rd!Passw0rd"}
		for k, v := range extra {
			body[k] = v
		}
		return f.do(t, "", http.MethodPost, "/api/collections/users/records", body)
	}

	if code, _ := signup(map[string]any{"role": "admin"}); code == http.StatusOK {
		t.Error("anonymous sign-up with role=admin was accepted")
	}
	if code, _ := signup(map[string]any{"groups": f.ids("finance")}); code == http.StatusOK {
		t.Error("anonymous sign-up with groups was accepted")
	}
	code, out := signup(nil)
	if code != http.StatusOK || out["role"] != "viewer" {
		t.Errorf("plain sign-up: status %d role %v, want 200 viewer", code, out["role"])
	}

	code, out = f.do(t, "admin", http.MethodPost, "/api/collections/users/records", map[string]any{
		"email": "ed@example.com", "password": "Passw0rd!Passw0rd", "passwordConfirm": "Passw0rd!Passw0rd", "role": "editor",
	})
	if code != http.StatusOK || out["role"] != "editor" {
		t.Errorf("admin creating an editor: status %d role %v, want 200 editor", code, out["role"])
	}
}

func TestUserCannotChangeOwnRoleOrGroups(t *testing.T) {
	f := newFixture(t)
	self := "/api/collections/users/records/" + f.users["bob"].Id

	if code, _ := f.do(t, "bob", http.MethodPatch, self, map[string]any{"role": "admin"}); code == http.StatusOK {
		t.Error("user set their own role")
	}
	if code, _ := f.do(t, "bob", http.MethodPatch, self, map[string]any{"groups": f.ids("finance")}); code == http.StatusOK {
		t.Error("user set their own groups")
	}
	bob := f.reload(t, "users", f.users["bob"].Id)
	if bob.GetString("role") != "viewer" || len(bob.GetStringSlice("groups")) != 0 {
		t.Fatalf("bob is now role=%q groups=%v", bob.GetString("role"), bob.GetStringSlice("groups"))
	}

	// Unchanged values (a client that sends the whole record back) are fine.
	if code, out := f.do(t, "bob", http.MethodPatch, self, map[string]any{"name": "Bob", "role": "viewer"}); code != http.StatusOK {
		t.Errorf("profile update: status %d %v, want 200", code, out)
	}
	if code, out := f.do(t, "admin", http.MethodPatch, self, map[string]any{"role": "editor", "groups": f.ids("ops")}); code != http.StatusOK {
		t.Errorf("admin update: status %d %v, want 200", code, out)
	}
	bob = f.reload(t, "users", f.users["bob"].Id)
	if bob.GetString("role") != "editor" || len(bob.GetStringSlice("groups")) != 1 {
		t.Errorf("after admin update bob is role=%q groups=%v", bob.GetString("role"), bob.GetStringSlice("groups"))
	}
}

func TestDocumentListCountsOnlyReadableRows(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		as, filter string
		want       int
	}{
		{"", "", 1},
		{"", "body~'teamsecret'", 0},
		{"", "body~'finsecret'", 0},
		{"", "path='team/a'", 0},
		{"bob", "", 2},
		{"bob", "body~'finsecret'", 0},
		{"alice", "", 4},  // finance: fin/a and multi/a
		{"olivia", "", 3}, // ops: multi/a only
		{"olivia", "body~'finsecret'", 0},
		{"admin", "", 4},
	}
	for _, c := range cases {
		if got := f.listTotal(t, c.as, c.filter); got != c.want {
			t.Errorf("as %q filter %q: totalItems %d, want %d", c.as, c.filter, got, c.want)
		}
	}
}

func TestDocumentViewHidesUnreadableRows(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		as, path string
		want     int
	}{
		{"", "pub/a", http.StatusOK},
		{"", "team/a", http.StatusNotFound},
		{"bob", "team/a", http.StatusOK},
		{"bob", "fin/a", http.StatusNotFound},
		{"alice", "fin/a", http.StatusOK},
		{"olivia", "fin/a", http.StatusNotFound},
		{"olivia", "multi/a", http.StatusOK},
	}
	for _, c := range cases {
		code, _ := f.do(t, c.as, http.MethodGet, "/api/collections/documents/records/"+f.docs[c.path].Id, nil)
		if code != c.want {
			t.Errorf("view %s as %q: status %d, want %d", c.path, c.as, code, c.want)
		}
	}
}

// Realtime broadcasts call app.CanAccessRecord with the collection ListRule
// (or ViewRule for a single-record topic). Checking that call directly covers
// what a subscriber receives.
func TestRealtimeRuleCheck(t *testing.T) {
	f := newFixture(t)
	docs, err := f.app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		as, path string
		want     bool
	}{
		{"", "pub/a", true},
		{"", "team/a", false},
		{"", "fin/a", false},
		{"bob", "team/a", true},
		{"bob", "fin/a", false},
		{"alice", "fin/a", true},
		{"olivia", "multi/a", true},
		{"olivia", "fin/a", false},
	}
	for _, c := range cases {
		info := &core.RequestInfo{Context: core.RequestInfoContextRealtime}
		if c.as != "" {
			info.Auth = f.users[c.as]
		}
		for name, rule := range map[string]*string{"ListRule": docs.ListRule, "ViewRule": docs.ViewRule} {
			got, err := f.app.CanAccessRecord(f.docs[c.path], info, rule)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("%s: %s as %q = %v, want %v", name, c.path, c.as, got, c.want)
			}
		}
	}
}

func TestRequireLoginHidesPublicPages(t *testing.T) {
	f := newFixture(t)
	cfg, err := f.app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("require_login", true)
	if err := f.app.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if got := f.listTotal(t, "", ""); got != 0 {
		t.Errorf("with require_login: anonymous totalItems %d, want 0", got)
	}
	if got := f.listTotal(t, "bob", ""); got != 2 {
		t.Errorf("with require_login: bob totalItems %d, want 2", got)
	}
}

func TestNewDocumentInheritsAccess(t *testing.T) {
	f := newFixture(t)
	create := func(path string, extra map[string]any) *core.Record {
		data := map[string]any{"path": path, "title": path}
		for k, v := range extra {
			data[k] = v
		}
		return f.save(t, "documents", data)
	}

	// Nearest ancestor wins, groups included.
	child := create("multi/a/b/c", nil)
	if child.GetString("access") != "restricted" || len(child.GetStringSlice("groups")) != 2 {
		t.Errorf("multi/a/b/c: access %q groups %v, want restricted with 2 groups", child.GetString("access"), child.GetStringSlice("groups"))
	}
	// A sibling is not an ancestor: team/b has no parent document.
	if got := create("team/b", nil).GetString("access"); got != "public" {
		t.Errorf("team/b: access %q, want public (no ancestor, private_default off)", got)
	}
	// An explicit choice is kept.
	if got := create("fin/a/open", map[string]any{"access": "public"}).GetString("access"); got != "public" {
		t.Errorf("fin/a/open with explicit access: got %q, want public", got)
	}

	cfg, err := f.app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("private_default", true)
	if err := f.app.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if got := create("top", nil).GetString("access"); got != "private" {
		t.Errorf("top with private_default: access %q, want private", got)
	}
}

func TestEditorsWriteOnlyReadablePages(t *testing.T) {
	f := newFixture(t)
	fin := "/api/collections/documents/records/" + f.docs["fin/a"].Id

	if code, _ := f.do(t, "erin", http.MethodPatch, fin, map[string]any{"access": "public"}); code == http.StatusOK {
		t.Error("editor outside finance updated a finance page")
	}
	if code, _ := f.do(t, "erin", http.MethodDelete, fin, nil); code == http.StatusNoContent {
		t.Error("editor outside finance deleted a finance page")
	}
	if code, out := f.do(t, "fred", http.MethodPatch, fin, map[string]any{"title": "Finance"}); code != http.StatusOK {
		t.Errorf("finance editor update: status %d %v, want 200", code, out)
	}
	if code, _ := f.do(t, "alice", http.MethodPatch, fin, map[string]any{"title": "x"}); code == http.StatusOK {
		t.Error("viewer updated a page")
	}
	if code, out := f.do(t, "erin", http.MethodPost, "/api/collections/documents/records", map[string]any{"path": "erin/new", "title": "n"}); code != http.StatusOK {
		t.Errorf("editor create: status %d %v, want 200", code, out)
	}
}

// A PocketBase superuser has no role field; the collection rules let it
// through regardless.
func TestSuperuserSeesEverything(t *testing.T) {
	f := newFixture(t)
	f.users["su"] = f.save(t, core.CollectionNameSuperusers, map[string]any{"email": "su@example.com", "password": "Passw0rd!Passw0rd"})

	code, out := f.do(t, "su", http.MethodGet, "/api/collections/documents/records", nil)
	items, _ := out["items"].([]any)
	if code != http.StatusOK || out["totalItems"] != float64(4) || len(items) != 4 {
		t.Errorf("superuser list: status %d totalItems %v items %d, want 200 4 4", code, out["totalItems"], len(items))
	}
	if code, out := f.do(t, "su", http.MethodGet, "/api/wiki/search?q=finsecret", nil); code != http.StatusOK || len(out["results"].([]any)) != 1 {
		t.Errorf("superuser search: status %d %v, want 1 result", code, out)
	}
}

func TestSearchReturnsOnlyReadablePages(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		as, q string
		want  int
	}{
		{"", "public", 1},
		{"", "teamsecret", 0},
		{"", "finsecret", 0},
		{"bob", "teamsecret", 1},
		{"bob", "finsecret", 0},
		{"alice", "finsecret", 1},
		{"olivia", "multisecret", 1},
		{"olivia", "finsecret", 0},
	}
	for _, c := range cases {
		code, out := f.do(t, c.as, http.MethodGet, "/api/wiki/search?q="+url.QueryEscape(c.q), nil)
		results, _ := out["results"].([]any)
		if code != http.StatusOK || len(results) != c.want {
			t.Errorf("search %q as %q: status %d results %d, want %d", c.q, c.as, code, len(results), c.want)
		}
	}
}
