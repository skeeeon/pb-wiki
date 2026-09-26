package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/dbx"
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
