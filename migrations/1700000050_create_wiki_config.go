package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// wiki_config is a single-row collection that stores wiki-wide settings. The
// migration inserts the one row with an auto-generated id; callers fetch it
// with `getFirstListItem("")`. CreateRule is nil so the row count stays at
// one — only superusers can ever insert another.
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("wiki_config")

		c.Fields.Add(&core.TextField{
			Name: "title",
			Max:  200,
		})
		c.Fields.Add(&core.BoolField{
			Name: "private_default",
		})
		c.Fields.Add(&core.JSONField{
			Name: "oauth_email_allowlist",
		})
		c.Fields.Add(&core.TextField{
			Name: "default_landing_path",
			Max:  500,
		})
		c.Fields.Add(&core.AutodateField{
			Name:     "created",
			OnCreate: true,
		})
		c.Fields.Add(&core.AutodateField{
			Name:     "updated",
			OnCreate: true,
			OnUpdate: true,
		})

		anyone := ""
		admin := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
		c.ListRule = &anyone
		c.ViewRule = &anyone
		c.CreateRule = nil // only superuser; we insert the singleton row below
		c.UpdateRule = &admin
		c.DeleteRule = nil

		if err := app.Save(c); err != nil {
			return err
		}

		// Insert the singleton config row with an auto-generated id.
		row := core.NewRecord(c)
		row.Set("title", "pb-wiki")
		row.Set("private_default", false)
		row.Set("oauth_email_allowlist", []string{})
		row.Set("default_landing_path", "")
		return app.Save(row)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("wiki_config")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
}
