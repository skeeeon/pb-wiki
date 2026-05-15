// Package static mounts the embedded Vue frontend onto a PocketBase router
// with SPA-style fallback: any GET path that doesn't resolve to a real file
// falls back to index.html so Vue Router can take over client-side. API and
// realtime routes registered earlier (priority < 999) win over this handler.
package static

import (
	"io/fs"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// Register attaches a catch-all GET handler that serves files from frontend
// with SPA fallback to index.html.
func Register(app *pocketbase.PocketBase, frontend fs.FS) {
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(se *core.ServeEvent) error {
			// Only mount if no upstream handler claimed the catch-all already.
			if !se.Router.HasRoute(http.MethodGet, "/{path...}") {
				se.Router.GET("/{path...}", apis.Static(frontend, true))
			}
			return se.Next()
		},
		Priority: 999, // run last so any user-defined routes win
	})
}
