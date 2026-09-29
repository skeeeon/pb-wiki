package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Add audit_logs.changed_fields for pb-audit v0.2. Every audit row names the
// fields that changed there. pb-audit creates the column only when it creates
// audit_logs, so a wiki made with an older pb-audit needs it added. Without
// it, PocketBase drops the value on save.
//
// Skips when audit_logs does not exist yet (a new wiki: pb-audit creates it
// with the column) or already has the column.
func init() {
	m.Register(func(app core.App) error {
		logs, err := app.FindCollectionByNameOrId("audit_logs")
		if err != nil || logs.Fields.GetByName("changed_fields") != nil {
			return nil
		}
		logs.Fields.Add(&core.JSONField{Name: "changed_fields", MaxSize: 100000})
		return app.Save(logs)
	}, func(app core.App) error {
		logs, err := app.FindCollectionByNameOrId("audit_logs")
		if err != nil {
			return nil
		}
		logs.Fields.RemoveByName("changed_fields")
		return app.Save(logs)
	})
}
