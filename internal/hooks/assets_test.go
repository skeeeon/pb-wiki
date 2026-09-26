package hooks_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func (f *fixture) upload(t *testing.T, name string) (*core.Record, string) {
	t.Helper()
	assets, err := f.app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes([]byte("GIF89a"), name)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(assets)
	r.Set("file", file)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	// The URL the editor inserts: pb.files.getURL uses the collection id.
	return r, "/api/files/" + assets.Id + "/" + r.Id + "/" + r.GetString("file")
}

// fetch downloads a file as the named user, sending the token the way the
// SPA does for <img> tags (cookie) or the way API clients do (header).
func (f *fixture) fetch(t *testing.T, as, via, target string) int {
	t.Helper()
	return f.fetchResponse(t, as, via, target).Code
}

func (f *fixture) fetchResponse(t *testing.T, as, via, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if as != "" {
		token, err := f.users[as].NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		if via == "cookie" {
			req.AddCookie(&http.Cookie{Name: "pbwiki_auth", Value: token})
		} else {
			req.Header.Set("Authorization", token)
		}
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) setBody(t *testing.T, path, body string) {
	t.Helper()
	doc := f.reload(t, "documents", f.docs[path].Id)
	doc.Set("body", body)
	if err := f.app.Save(doc); err != nil {
		t.Fatal(err)
	}
}

func TestAssetsFollowDocumentAccess(t *testing.T) {
	f := newFixture(t)
	secret, secretURL := f.upload(t, "chart.gif")
	_, publicURL := f.upload(t, "logo.gif")

	type check struct {
		as, via string
		want    int
	}
	expect := func(when, url string, checks ...check) {
		t.Helper()
		for _, c := range checks {
			if got := f.fetch(t, c.as, c.via, url); got != c.want {
				t.Errorf("%s: fetch as %q via %s: %d, want %d", when, c.as, c.via, got, c.want)
			}
		}
	}

	// Uploaded but not yet saved into a page: editors and admins only.
	expect("unclaimed", secretURL,
		check{"", "", http.StatusNotFound},
		check{"bob", "cookie", http.StatusNotFound},
		check{"erin", "cookie", http.StatusOK},
		check{"admin", "header", http.StatusOK},
	)

	f.setBody(t, "fin/a", "![chart]("+secretURL+")")
	f.setBody(t, "pub/a", "![logo]("+publicURL+")")
	if got := f.reload(t, "assets", secret.Id).GetString("document"); got != f.docs["fin/a"].Id {
		t.Fatalf("fin/a did not claim its image: document = %q", got)
	}

	expect("in a finance page", secretURL,
		check{"", "", http.StatusNotFound},
		check{"bob", "cookie", http.StatusNotFound},
		check{"erin", "cookie", http.StatusNotFound}, // editor outside finance
		check{"alice", "cookie", http.StatusOK},
		check{"alice", "header", http.StatusOK},
	)
	expect("in a public page", publicURL,
		check{"", "", http.StatusOK},
		check{"bob", "cookie", http.StatusOK},
	)

	// Only files an anonymous visitor could fetch may be cached by browsers
	// and shared caches.
	for _, c := range []struct{ as, url, want string }{
		{"alice", secretURL, "private, no-store"},
		{"bob", publicURL, "max-age=2592000, stale-while-revalidate=86400"},
		{"", publicURL, "max-age=2592000, stale-while-revalidate=86400"},
	} {
		if got := f.fetchResponse(t, c.as, "cookie", c.url).Header().Get("Cache-Control"); got != c.want {
			t.Errorf("Cache-Control for %s as %q: %q, want %q", c.url, c.as, got, c.want)
		}
	}

	// Embedding a claimed image elsewhere does not move it.
	f.setBody(t, "pub/a", "![chart]("+secretURL+") ![logo]("+publicURL+")")
	if got := f.reload(t, "assets", secret.Id).GetString("document"); got != f.docs["fin/a"].Id {
		t.Errorf("second page stole the image: document = %q", got)
	}
	expect("claimed image embedded in a public page", secretURL, check{"", "", http.StatusNotFound})

	// Asset records themselves are for editors and admins.
	if got := f.listTotalOf(t, "bob", "assets"); got != 0 {
		t.Errorf("viewer lists %d assets, want 0", got)
	}
	if got := f.listTotalOf(t, "erin", "assets"); got != 2 {
		t.Errorf("editor lists %d assets, want 2", got)
	}
}

func (f *fixture) listTotalOf(t *testing.T, as, collection string) int {
	t.Helper()
	code, out := f.do(t, as, http.MethodGet, "/api/collections/"+collection+"/records", nil)
	if code != http.StatusOK {
		t.Fatalf("list %s as %q: status %d %v", collection, as, code, out)
	}
	total, _ := out["totalItems"].(float64)
	return int(total)
}
