package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Add documents.nav_order: the sidebar sorts siblings by nav_order, then by
// name. Existing documents get 0, so they keep their alphabetical order.
func init() {
	m.Register(func(app core.App) error {
		docs, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		docs.Fields.Add(&core.NumberField{Name: "nav_order", OnlyInt: true})
		return app.Save(docs)
	}, func(app core.App) error {
		docs, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		docs.Fields.RemoveByName("nav_order")
		return app.Save(docs)
	})
}
