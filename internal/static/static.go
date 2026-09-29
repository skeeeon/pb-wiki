// Package static mounts the embedded Vue frontend onto a PocketBase router
// with SPA-style fallback: any GET path that doesn't resolve to a real file
// falls back to index.html so Vue Router can take over client-side. API and
// realtime routes registered earlier (priority < 999) win over this handler.
package static

import (
	"io/fs"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// Register attaches the frontend routes:
//
//   - /assets/* holds Vite's content-hashed build output. A file's name
//     changes whenever its content does, so browsers may keep it forever.
//     There is no index.html fallback here: a stale tab asking for a chunk
//     from an old build gets a 404 instead of HTML with a JavaScript name.
//   - Everything else (index.html, sw.js, manifest, icons) keeps its name
//     across deploys, so browsers must check back on every load.
//
// The embedded files have no modification time, so without these headers
// browsers cannot cache anything and download the whole app on every visit.
func Register(app core.App, frontend fs.FS) {
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(se *core.ServeEvent) error {
			assets, err := fs.Sub(frontend, "assets")
			if err != nil {
				return err
			}
			se.Router.GET("/assets/{path...}", apis.Static(assets, false)).
				BindFunc(cacheControl("public, max-age=31536000, immutable"))

			// Only mount if no upstream handler claimed the catch-all already.
			if !se.Router.HasRoute(http.MethodGet, "/{path...}") {
				se.Router.GET("/{path...}", apis.Static(frontend, true)).
					BindFunc(cacheControl("no-cache"))
			}
			return se.Next()
		},
		Priority: 999, // run last so any user-defined routes win
	})
}

// cacheControl sets the Cache-Control header on successful responses. It is
// removed again when the handler fails, so a 404 is never cached for a year.
// PocketBase writes the error response after this returns.
func cacheControl(value string) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		e.Response.Header().Set("Cache-Control", value)
		err := e.Next()
		if err != nil {
			e.Response.Header().Del("Cache-Control")
		}
		return err
	}
}
