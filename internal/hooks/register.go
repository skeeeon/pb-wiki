// Package hooks wires pb-wiki's model behaviours onto a PocketBase app:
// default access for new documents, asset ownership and download checks,
// and the default role for new users.
// Access itself is enforced by collection API rules (see migrations).
// main.go calls Register exactly once during boot.
package hooks

import "github.com/pocketbase/pocketbase/core"

// Register installs every pb-wiki hook on app. Safe to call once at startup.
func Register(app core.App) {
	registerDocumentHooks(app)
	registerAuthHooks(app)
	registerAssetHooks(app)
}
