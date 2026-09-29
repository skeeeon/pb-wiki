package hooks_test

import (
	"net/http"
	"testing"
)

// LIKE treats _ and % as wildcards. A source path that contains them must
// move only its own subtree, not look-alike paths.
func TestBulkMoveMatchesLiteralPrefix(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"my_docs", "my_docs/a", "my-docs/b", "myXdocs", "100%/c", "100x/d"} {
		f.docs[path] = f.save(t, "documents", map[string]any{"path": path, "title": path, "access": "public"})
	}

	for _, tc := range []struct {
		from, to string
		want     map[string]string // old path → new path
	}{
		{"my_docs", "moved", map[string]string{"my_docs": "moved", "my_docs/a": "moved/a", "my-docs/b": "my-docs/b", "myXdocs": "myXdocs"}},
		{"100%", "pct", map[string]string{"100%/c": "pct/c", "100x/d": "100x/d"}},
	} {
		code, out := f.do(t, "admin", http.MethodPost, "/api/wiki/bulk-move", map[string]string{"from": tc.from, "to": tc.to})
		if code != http.StatusOK {
			t.Fatalf("move %q: status %d %v", tc.from, code, out)
		}
		for old, want := range tc.want {
			if got := f.reload(t, "documents", f.docs[old].Id).GetString("path"); got != want {
				t.Errorf("move %q: %q is now %q, want %q", tc.from, old, got, want)
			}
		}
	}
}
