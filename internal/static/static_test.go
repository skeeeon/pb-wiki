package static_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/skeeeon/pb-wiki/internal/static"
)

func TestCacheHeaders(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	static.Register(app, fstest.MapFS{
		"index.html":        {Data: []byte("<!doctype html>")},
		"sw.js":             {Data: []byte("// sw")},
		"assets/app-abc.js": {Data: []byte("console.log(1)")},
	})

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	var mux http.Handler
	err = app.OnServe().Trigger(&core.ServeEvent{App: app, Router: router}, func(e *core.ServeEvent) error {
		mux, err = e.Router.BuildMux()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		target, cache string
		status        int
	}{
		{"/assets/app-abc.js", "public, max-age=31536000, immutable", http.StatusOK},
		{"/assets/app-old.js", "", http.StatusNotFound}, // no index.html fallback, not cached
		{"/", "no-cache", http.StatusOK},
		{"/doc/some/page", "no-cache", http.StatusOK}, // SPA fallback
		{"/sw.js", "no-cache", http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.target, nil))
		if rec.Code != tc.status {
			t.Errorf("%s: status %d, want %d", tc.target, rec.Code, tc.status)
		}
		if got := rec.Header().Get("Cache-Control"); got != tc.cache {
			t.Errorf("%s: Cache-Control %q, want %q", tc.target, got, tc.cache)
		}
	}
}
