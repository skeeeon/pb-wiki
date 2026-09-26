package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/skeeeon/pb-wiki/internal/hooks"
)

// Uploaded images now follow the access of the document that embeds them
// (see internal/hooks/assets.go). Before this, every asset file was public
// to anyone with the URL, and any logged-in user could list all assets.
//
// Existing assets are claimed by the first document that embeds them.
// Asset records are only for editors and admins now; viewers load the files
// through the download hook, which does its own check.
func init() {
	m.Register(func(app core.App) error {
		assets, err := app.FindCollectionByNameOrId("assets")
		if err != nil {
			return err
		}
		assets.ListRule = ptr(writerRule)
		assets.ViewRule = ptr(writerRule)
		if err := app.Save(assets); err != nil {
			return err
		}

		var docs []struct {
			ID   string `db:"id"`
			Body string `db:"body"`
		}
		if err := app.DB().NewQuery("SELECT [[id]], [[body]] FROM {{documents}} ORDER BY [[created]] ASC").All(&docs); err != nil {
			return err
		}
		for _, d := range docs {
			if err := hooks.ClaimAssets(app, d.ID, d.Body); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		assets, err := app.FindCollectionByNameOrId("assets")
		if err != nil {
			return err
		}
		loggedIn := `@request.auth.id != ""`
		assets.ListRule = &loggedIn
		assets.ViewRule = ptr("")
		return app.Save(assets)
	})
}
