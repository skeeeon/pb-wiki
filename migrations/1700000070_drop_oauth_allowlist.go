package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Drop the wiki_config.oauth_email_allowlist field. OAuth-domain gating is
// now handled natively by PocketBase's EmailField.OnlyDomains validator on
// the users collection's `email` field — set it via the PB admin UI under
// Collections → users → `email` field options.
//
// Operators upgrading from a version that used the custom allowlist must
// migrate their domain entries to OnlyDomains before applying this migration;
// full-email entries are not portable (OnlyDomains only takes domains) and
// should be replaced by individual user records or a different gating layer.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("wiki_config")
		if err != nil {
			return err
		}
		c.Fields.RemoveByName("oauth_email_allowlist")
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("wiki_config")
		if err != nil {
			return err
		}
		c.Fields.Add(&core.JSONField{
			Name: "oauth_email_allowlist",
		})
		return app.Save(c)
	})
}
