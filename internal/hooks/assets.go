package hooks

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Uploaded images follow the access of the document they belong to.
//
// An asset belongs to the first document whose body embeds it: the editor
// uploads before the document is saved, so the document claims its assets
// when it is saved. Until then only editors and admins can fetch the file.
//
// PocketBase serves files without any access check unless a file field is
// Protected, and protected files need a short-lived ?token= in the URL. An
// <img> tag cannot send the Authorization header either, so instead the
// frontend mirrors the auth token into the fileAuthCookie cookie (path
// /api/files/, see frontend/src/lib/pb.ts) and the download hook below reads
// it. Stored markdown keeps plain PocketBase file URLs.

const fileAuthCookie = "pbwiki_auth"

// assetURL matches the record id in a PocketBase file URL:
// /api/files/<collection>/<record id>/<file name>.
var assetURL = regexp.MustCompile(`/api/files/[^/\s]+/([a-z0-9]+)/`)

func registerAssetHooks(app core.App) {
	claim := func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return ClaimAssets(e.App, e.Record.Id, e.Record.GetString("body"))
	}
	app.OnRecordCreate("documents").BindFunc(claim)
	app.OnRecordUpdate("documents").BindFunc(claim)

	app.OnFileDownloadRequest("assets").BindFunc(func(e *core.FileDownloadRequestEvent) error {
		auth := e.Auth
		if auth == nil {
			auth = fileRequestAuth(e.App, e.Request)
		}
		ok, err := canDownloadAsset(e, auth)
		if err != nil {
			return err
		}
		if !ok {
			return e.NotFoundError("", nil)
		}

		// PocketBase lets browsers and shared caches (a CDN in front of the
		// wiki) keep files for 30 days. That is only safe for files an
		// anonymous visitor could fetch anyway; PocketBase keeps a
		// Cache-Control header that is already set.
		public, err := canDownloadAsset(e, nil)
		if err != nil {
			return err
		}
		if !public {
			e.Response.Header().Set("Cache-Control", "private, no-store")
		}
		return e.Next()
	})
}

// ClaimAssets sets `document` on every unclaimed asset that body embeds.
// Ids that are not assets (an avatar URL, say) match no row and are ignored.
func ClaimAssets(app core.App, docID, body string) error {
	ids := []any{}
	for _, m := range assetURL.FindAllStringSubmatch(body, -1) {
		ids = append(ids, m[1])
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := app.DB().Update("assets",
		dbx.Params{"document": docID},
		dbx.And(dbx.In("id", ids...), dbx.HashExp{"document": ""}),
	).Execute()
	return err
}

// canDownloadAsset reports whether auth (nil = anonymous) may fetch the
// asset: editors and admins for an unclaimed asset, otherwise whoever may
// read the owning document.
func canDownloadAsset(e *core.FileDownloadRequestEvent, auth *core.Record) (bool, error) {
	info, err := e.RequestInfo()
	if err != nil {
		return false, err
	}
	if auth != nil && auth.IsSuperuser() {
		return true, nil
	}

	docID := e.Record.GetString("document")
	if docID == "" {
		role := ""
		if auth != nil {
			role = auth.GetString("role")
		}
		return role == "admin" || role == "editor", nil
	}
	doc, err := e.App.FindRecordById("documents", docID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	check := *info
	check.Auth = auth
	return e.App.CanAccessRecord(doc, &check, doc.Collection().ViewRule)
}

// fileRequestAuth finds the caller of a file request that has no
// Authorization header: the pb-wiki auth cookie from the SPA, or the
// ?token= file token the PocketBase admin UI adds.
func fileRequestAuth(app core.App, r *http.Request) *core.Record {
	if c, err := r.Cookie(fileAuthCookie); err == nil {
		if rec, err := app.FindAuthRecordByToken(c.Value, core.TokenTypeAuth); err == nil {
			return rec
		}
	}
	if t := r.URL.Query().Get("token"); t != "" {
		if rec, err := app.FindAuthRecordByToken(t, core.TokenTypeFile); err == nil {
			return rec
		}
	}
	return nil
}
