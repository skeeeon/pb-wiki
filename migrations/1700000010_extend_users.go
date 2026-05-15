package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Extend the built-in `users` auth collection with the pb-wiki role + groups
// fields, and widen the default API rules so admins can manage all users
// (PB's stock rules restrict every user to their own record).
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		users.Fields.Add(&core.SelectField{
			Name:      "role",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"admin", "editor", "viewer"},
		})
		users.Fields.Add(&core.JSONField{
			Name: "groups",
		})

		ownOrAdmin := "id = @request.auth.id || @request.auth.role = \"admin\""
		adminOnly := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
		users.ListRule = &ownOrAdmin
		users.ViewRule = &ownOrAdmin
		users.UpdateRule = &ownOrAdmin
		users.DeleteRule = &adminOnly
		// CreateRule intentionally left as configured by PB (open) — keeping
		// the door open for OAuth sign-up and admin-driven seed creation.

		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.RemoveByName("role")
		users.Fields.RemoveByName("groups")
		// Down-migration leaves the API rules at whatever the up-migration
		// changed them to; reverting them back to PB stock would require
		// hard-coding the originals here, which is more brittle than useful.
		return app.Save(users)
	})
}
