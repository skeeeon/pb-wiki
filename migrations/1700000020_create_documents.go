package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// documents holds the wiki's markdown content. The `path` field is the
// slash-separated slug (no leading slash). Empty string is the homepage. The
// tree is built client-side from a flat list of paths.
//
// API rules are permissive on read because path-based access enforcement
// happens in hooks (internal/hooks/documents.go). Writes are restricted to
// admin/editor at the rule level; the hook layer additionally requires the
// caller to have access to the target path.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		c := core.NewBaseCollection("documents")

		c.Fields.Add(&core.TextField{
			Name: "path",
			// Not Required: empty string is the homepage (see plan). The
			// unique index on path means there can still only be one of them.
			Max: 500,
		})
		c.Fields.Add(&core.TextField{
			Name: "title",
			Max:  200,
		})
		c.Fields.Add(&core.TextField{
			Name: "body",
			Max:  10_000_000,
		})
		c.Fields.Add(&core.RelationField{
			Name:         "updated_by",
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

		c.Indexes = []string{
			"CREATE UNIQUE INDEX `idx_documents_path` ON `documents` (`path`)",
		}

		anyone := ""
		writer := "@request.auth.id != \"\" && (@request.auth.role = \"admin\" || @request.auth.role = \"editor\")"
		c.ListRule = &anyone
		c.ViewRule = &anyone
		c.CreateRule = &writer
		c.UpdateRule = &writer
		c.DeleteRule = &writer

		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
}
