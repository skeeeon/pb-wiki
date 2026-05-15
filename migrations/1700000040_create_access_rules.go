package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// access_rules drives the path-based access enforcement evaluator
// (internal/access). Rules are walked in ascending `priority` order and the
// first matching pattern wins — admins managing this collection must order
// specific rules ahead of general ones.
//
// All API rules are admin-only.
func init() {
	m.Register(func(app core.App) error {
		c := core.NewBaseCollection("access_rules")

		c.Fields.Add(&core.TextField{
			Name:     "pattern",
			Required: true,
			Max:      500,
		})
		c.Fields.Add(&core.SelectField{
			Name:      "access",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"public", "private", "restricted"},
		})
		c.Fields.Add(&core.JSONField{
			Name: "groups",
		})
		c.Fields.Add(&core.NumberField{
			Name: "priority",
		})
		c.Fields.Add(&core.TextField{
			Name: "description",
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

		c.Indexes = []string{
			"CREATE INDEX `idx_access_rules_priority` ON `access_rules` (`priority`)",
		}

		admin := "@request.auth.id != \"\" && @request.auth.role = \"admin\""
		c.ListRule = &admin
		c.ViewRule = &admin
		c.CreateRule = &admin
		c.UpdateRule = &admin
		c.DeleteRule = &admin

		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("access_rules")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
}
