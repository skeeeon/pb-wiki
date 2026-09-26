package api

import "github.com/pocketbase/pocketbase/core"

// canView reports whether the request may read doc. It evaluates the
// documents collection's ViewRule, the same check the record API and
// realtime use, so custom endpoints that query documents directly (search,
// history) cannot drift from it. Superusers always pass.
func canView(e *core.RequestEvent, doc *core.Record) (bool, error) {
	info, err := e.RequestInfo()
	if err != nil {
		return false, err
	}
	return e.App.CanAccessRecord(doc, info, doc.Collection().ViewRule)
}
