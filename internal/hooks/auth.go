package hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// registerAuthHooks installs the user-creation defaults pb-wiki needs on top
// of PocketBase's stock auth collection.
//
// Email-domain gating is intentionally not handled here: PocketBase's
// EmailField.OnlyDomains setting on the users collection applies natively to
// both password signup and OAuth, so admins should configure that via the PB
// admin UI rather than via a custom hook + Settings field.
func registerAuthHooks(app *pocketbase.PocketBase) {
	// Default role for any newly-created user, since `role` is Required on
	// the collection but OAuth sign-up doesn't supply one. Admins can promote
	// later via the Users admin page.
	app.OnRecordCreate("users").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("role") == "" {
			e.Record.Set("role", "viewer")
		}
		return e.Next()
	})
}
