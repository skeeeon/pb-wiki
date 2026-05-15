package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Add the `require_login` flag to wiki_config. When true, every document read
// requires an authenticated user — overriding even rules explicitly marked
// `public`. The frontend reads the same flag at boot and redirects anonymous
// requests to /login from any route.
//
// wiki_config itself stays world-readable (see migration 1700000050) so the
// login page can render the wiki title and the redirect can read the flag
// before the user has a session.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("wiki_config")
		if err != nil {
			return err
		}

		c.Fields.Add(&core.BoolField{
			Name: "require_login",
		})

		if err := app.Save(c); err != nil {
			return err
		}

		// Backfill the singleton row's new field to false so existing wikis
		// keep their current behavior on upgrade.
		row, err := app.FindFirstRecordByFilter("wiki_config", "")
		if err != nil {
			return err
		}
		row.Set("require_login", false)
		return app.Save(row)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("wiki_config")
		if err != nil {
			return err
		}
		c.Fields.RemoveByName("require_login")
		return app.Save(c)
	})
}
