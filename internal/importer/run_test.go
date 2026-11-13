package importer

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/skeeeon/pb-wiki/internal/hooks"
	_ "github.com/skeeeon/pb-wiki/migrations"
)

func TestRewriteLinks(t *testing.T) {
	dir := filepath.FromSlash("/src/docs")
	pathOf := map[string]string{
		filepath.FromSlash("/src/docs/index.md"):          "platform",
		filepath.FromSlash("/src/docs/auth.md"):           "platform/auth",
		filepath.FromSlash("/src/docs/decisions/0001.md"): "platform/decisions/0001",
		filepath.FromSlash("/src/home.md"):                "",
	}
	in := "[a](./auth.md) [b](auth.md#roles) [c](decisions/0001.md) [d](../home.md) " +
		"[e](./gone.md) [f](https://x.test/a.md) [g](/doc/x) [h](#local) [i](./index.md)"
	want := "[a](/doc/platform/auth) [b](/doc/platform/auth#roles) [c](/doc/platform/decisions/0001) [d](/) " +
		"[e](./gone.md) [f](https://x.test/a.md) [g](/doc/x) [h](#local) [i](/doc/platform)"
	got, missing := rewriteLinks(in, dir, pathOf)
	if got != want {
		t.Errorf("rewriteLinks:\n got %s\nwant %s", got, want)
	}
	if len(missing) != 1 || missing[0] != "./gone.md" {
		t.Errorf("missing = %v, want [./gone.md]", missing)
	}
}

func TestRunImportsTree(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	hooks.Register(app)

	src := t.TempDir()
	write := func(name, content string) {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// WalkDir visits fin/q4.md before fin.md (the directory "fin" sorts first),
	// so this also checks that parents are written before their children.
	write("fin.md", "---\npath: fin\naccess: restricted\ngroups: [finance, ops]\nnav_order: 20\n---\n# Finance\n\nSee [Q4](fin/q4.md#totals).\n")
	write("fin/q4.md", "---\npath: fin/q4\n---\n# Q4\n\nBack to [finance](../fin.md).\n")
	write("about.md", "---\npath: about\nnav_order: 10\n---\n# About\n")

	if err := run(app, src); err != nil {
		t.Fatal(err)
	}

	get := func(path string) (access string, groups int, navOrder int, body string) {
		t.Helper()
		recs, err := app.FindAllRecords("documents", dbx.HashExp{"path": path})
		if err != nil || len(recs) != 1 {
			t.Fatalf("find %q: %v (%d records)", path, err, len(recs))
		}
		r := recs[0]
		return r.GetString("access"), len(r.GetStringSlice("groups")), r.GetInt("nav_order"), r.GetString("body")
	}

	if access, groups, order, body := get("fin"); access != "restricted" || groups != 2 || order != 20 || body != "See [Q4](/doc/fin/q4#totals).\n" {
		t.Errorf("fin: access %q groups %d nav_order %d body %q", access, groups, order, body)
	}
	if access, groups, _, body := get("fin/q4"); access != "restricted" || groups != 2 || body != "Back to [finance](/doc/fin).\n" {
		t.Errorf("fin/q4 should inherit from fin: access %q groups %d body %q", access, groups, body)
	}
	if access, _, order, _ := get("about"); access != "public" || order != 10 {
		t.Errorf("about: access %q nav_order %d, want public 10", access, order)
	}
	if n, _ := app.CountRecords("groups"); n != 2 {
		t.Errorf("groups created: %d, want 2", n)
	}

	// Re-import is an update, and a page without `access` keeps its access.
	doc, _ := app.FindFirstRecordByData("documents", "path", "about")
	doc.Set("access", "private")
	if err := app.Save(doc); err != nil {
		t.Fatal(err)
	}
	if err := run(app, src); err != nil {
		t.Fatal(err)
	}
	if access, _, _, _ := get("about"); access != "private" {
		t.Errorf("re-import changed about's access to %q; it has no access in frontmatter", access)
	}
	if n, _ := app.CountRecords("documents"); n != 3 {
		t.Errorf("documents after re-import: %d, want 3", n)
	}
}

// A repeat import must not save pages whose content is unchanged: every save
// bumps `updated` and adds a revision to the page history.
func TestRunSkipsUnchangedPages(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	hooks.Register(app)

	src := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(src, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("home.md", "---\npath: \"\"\n---\n# Home\n\nSee [ops](ops.md).\n")
	write("ops.md", "---\npath: ops\naccess: restricted\ngroups: [ops]\nnav_order: 5\n---\n# Ops\n\nRunbooks.\n")

	if err := run(app, src); err != nil {
		t.Fatal(err)
	}
	updated := func(path string) string {
		t.Helper()
		recs, err := app.FindAllRecords("documents", dbx.HashExp{"path": path})
		if err != nil || len(recs) != 1 {
			t.Fatalf("find %q: %v (%d records)", path, err, len(recs))
		}
		return recs[0].GetString("updated") + " " + recs[0].GetString("body")
	}
	home, ops := updated(""), updated("ops")

	time.Sleep(5 * time.Millisecond) // a save now would get a later `updated`
	if err := run(app, src); err != nil {
		t.Fatal(err)
	}
	if got := updated(""); got != home {
		t.Errorf("unchanged homepage was saved again: %q, was %q", got, home)
	}
	if got := updated("ops"); got != ops {
		t.Errorf("unchanged ops page was saved again: %q, was %q", got, ops)
	}

	// Someone last saved ops in the wiki; an import that changes it must
	// clear updated_by, so the page doesn't say they made the import's edit.
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	alice := core.NewRecord(users)
	alice.SetEmail("alice@example.test")
	alice.SetPassword("password123456")
	if err := app.Save(alice); err != nil {
		t.Fatal(err)
	}
	opsRec, err := app.FindFirstRecordByData("documents", "path", "ops")
	if err != nil {
		t.Fatal(err)
	}
	opsRec.Set("updated_by", alice.Id)
	if err := app.Save(opsRec); err != nil {
		t.Fatal(err)
	}
	ops = updated("ops")

	write("ops.md", "---\npath: ops\naccess: restricted\ngroups: [ops]\nnav_order: 5\n---\n# Ops\n\nNew runbooks.\n")
	if err := run(app, src); err != nil {
		t.Fatal(err)
	}
	if got := updated("ops"); got == ops {
		t.Errorf("changed ops page was not saved: %q", got)
	}
	if opsRec, _ = app.FindFirstRecordByData("documents", "path", "ops"); opsRec.GetString("updated_by") != "" {
		t.Errorf("import left updated_by = %q, want empty", opsRec.GetString("updated_by"))
	}
	if got := updated(""); got != home {
		t.Errorf("unchanged homepage was saved again: %q, was %q", got, home)
	}
}

// An #anchor link that matches no heading in its target page is reported;
// one that matches is not. The import still succeeds.
func TestRunWarnsBrokenAnchors(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	hooks.Register(app)

	src := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(src, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("guide.md", "---\npath: guide\n---\n# Guide\n\n## 2.1 `server_url` and `websocket_urls`\n\n"+
		"[good](ops.md#restart-the-service) [bad](ops.md#restart) "+
		"[here](#21-server_url-and-websocket_urls) [stripped](#21-serverurl-and-websocketurls) "+
		"[title](#guide)\n")
	write("ops.md", "---\npath: ops\n---\n# Ops\n\n## Restart the service\n")

	out := captureStdout(t, func() {
		if err := run(app, src); err != nil {
			t.Fatal(err)
		}
	})
	guide := filepath.Join(src, "guide.md")
	for _, want := range []string{
		"  warn   " + guide + ": link to ops.md#restart matches no heading\n",
		"  warn   " + guide + ": link to #21-serverurl-and-websocketurls matches no heading\n",
		// splitTitle took the H1 out of the body, so it has no id.
		"  warn   " + guide + ": link to #guide matches no heading\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	for _, good := range []string{"#restart-the-service", "#21-server_url-and-websocket_urls "} {
		if strings.Contains(out, good) {
			t.Errorf("good anchor %s was reported:\n%s", good, out)
		}
	}
}

// captureStdout returns what f prints to stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = stdout }()
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	f()
	w.Close()
	return <-done
}
