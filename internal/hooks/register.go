// Package hooks wires pb-wiki's request-time behaviors onto a PocketBase app:
// path-based access enforcement for documents and the OAuth email-domain
// allowlist. main.go calls Register exactly once during boot.
package hooks

import "github.com/pocketbase/pocketbase"

// Register installs every pb-wiki hook on app. Safe to call once at startup.
func Register(app *pocketbase.PocketBase) {
	registerDocumentHooks(app)
	registerAuthHooks(app)
}
