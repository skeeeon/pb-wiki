package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// assets holds uploaded images and attachments that get embedded in markdown
// bodies. The `document` relation is optional — the markdown body is the
// source of truth for "is this asset still in use," so cascade-on-delete is
// intentionally OFF. An orphan-cleanup hook can be added later.
//
// ViewRule is empty so the file URL can be fetched without auth (images in
// public docs need to load anonymously). File names already include a random
// suffix, so the URL is the secret.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		docs, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}

		c := core.NewBaseCollection("assets")

		c.Fields.Add(&core.FileField{
			Name:      "file",
			Required:  true,
			MaxSelect: 1,
			MaxSize:   10_000_000, // 10 MB
		})
		c.Fields.Add(&core.RelationField{
			Name:         "document",
			CollectionId: docs.Id,
			MaxSelect:    1,
		})
		c.Fields.Add(&core.RelationField{
			Name:         "uploaded_by",
			CollectionId: users.Id,
			MaxSelect:    1,
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
		loggedIn := "@request.auth.id != \"\""
		writer := "@request.auth.id != \"\" && (@request.auth.role = \"admin\" || @request.auth.role = \"editor\")"
		c.ListRule = &loggedIn
		c.ViewRule = &anyone
		c.CreateRule = &writer
		c.UpdateRule = &writer
		c.DeleteRule = &writer

		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("assets")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
}
