package hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/skeeeon/pb-wiki/internal/access"
)

// registerDocumentHooks wires path-based access enforcement onto the
// `documents` collection. Role/auth gating already lives in the collection's
// API rules (set in migration 1700000020); this layer adds the orthogonal
// per-path check from access rules — the two together reproduce wiki-go's
// "RequireRole gates actions, AccessRules gate paths" model.
func registerDocumentHooks(app *pocketbase.PocketBase) {
	// View — return 404 (not 403) on deny to avoid revealing existence.
	app.OnRecordViewRequest("documents").BindFunc(func(e *core.RecordRequestEvent) error {
		ok, err := canAccessRecord(e.App, e.Auth, e.Record)
		if err != nil {
			return err
		}
		if !ok {
			return e.NotFoundError("", nil)
		}
		return e.Next()
	})

	// List — filter the in-memory result set; records the user can't see
	// silently disappear from listings.
	app.OnRecordsListRequest("documents").BindFunc(func(e *core.RecordsListRequestEvent) error {
		rules, err := loadRules(e.App)
		if err != nil {
			return err
		}
		privateDefault, err := loadPrivateDefault(e.App)
		if err != nil {
			return err
		}
		user := recordToUser(e.Auth)

		filtered := make([]*core.Record, 0, len(e.Records))
		for _, r := range e.Records {
			if access.CanAccess(r.GetString("path"), user, rules, privateDefault) {
				filtered = append(filtered, r)
			}
		}
		e.Records = filtered
		if e.Result != nil {
			e.Result.Items = filtered
		}
		return e.Next()
	})

	// Create/Update/Delete — must have access to the target path (in addition
	// to the role gate the collection's API rule applies). Use 403 here since
	// the caller already knew the path; the failure mode is "you can't write
	// here," not "this doesn't exist."
	writeGuard := func(e *core.RecordRequestEvent) error {
		ok, err := canAccessRecord(e.App, e.Auth, e.Record)
		if err != nil {
			return err
		}
		if !ok {
			return e.ForbiddenError("", nil)
		}
		return e.Next()
	}
	app.OnRecordCreateRequest("documents").BindFunc(writeGuard)
	app.OnRecordUpdateRequest("documents").BindFunc(writeGuard)
	app.OnRecordDeleteRequest("documents").BindFunc(writeGuard)
}

// canAccessRecord loads rules + config and evaluates access for the given
// auth record against the given document record's path.
func canAccessRecord(app core.App, auth *core.Record, doc *core.Record) (bool, error) {
	rules, err := loadRules(app)
	if err != nil {
		return false, err
	}
	privateDefault, err := loadPrivateDefault(app)
	if err != nil {
		return false, err
	}
	return access.CanAccess(doc.GetString("path"), recordToUser(auth), rules, privateDefault), nil
}
