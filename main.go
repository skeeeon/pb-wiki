package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	pbaudit "github.com/skeeeon/pb-audit"

	"github.com/skeeeon/pb-wiki/internal/api"
	"github.com/skeeeon/pb-wiki/internal/hooks"
	"github.com/skeeeon/pb-wiki/internal/importer"
	"github.com/skeeeon/pb-wiki/internal/static"
	_ "github.com/skeeeon/pb-wiki/migrations"
)

// --- pb-audit configuration --------------------------------------------------
// These options are baked into the binary at build time. Adjust the values
// below and rebuild to change audit behaviour for a deployment. The defaults
// match pb-audit's own DefaultOptions() with a conservative retention policy
// layered on so audit_logs doesn't grow unbounded.
//
// Knobs:
//   - LogRequestEvents — record pre-commit API request events (user intent + IP).
//   - LogSuccessEvents — record post-commit DB events (drives the History view).
//   - LogAuthEvents    — record sign-in / sign-up / token refresh events.
//   - LogToConsole     — mirror events to stdout (useful in dev, noisy in prod).
//   - Retention.MaxAge / MaxRecords — bounds on how much audit history is kept;
//     either constraint alone is enough, both together apply as an AND. Set
//     MaxAge to 0 to disable age-based pruning; MaxRecords to 0 to disable the
//     row cap. Interval is a cron expression for when the cleanup job runs.
//
// The History view (/api/wiki/history) relies on LogSuccessEvents — turning
// that off makes per-doc history disappear from the UI.
var auditOptions = func() pbaudit.Options {
	o := pbaudit.DefaultOptions()
	o.Retention = &pbaudit.RetentionPolicy{
		MaxAge:     365 * 24 * time.Hour, // keep one year of audit history
		MaxRecords: 500_000,              // hard cap as a safety net
		Interval:   "0 2 * * *",          // nightly at 02:00
	}
	return o
}()

func main() {
	app := pocketbase.New()

	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  isGoRun,
	})

	hooks.Register(app)
	if err := pbaudit.Setup(app, auditOptions); err != nil {
		log.Fatalf("Failed to set up pb-audit: %v", err)
	}
	api.RegisterBulkMove(app)
	api.RegisterHistory(app)
	static.Register(app, frontendDist())

	app.RootCmd.AddCommand(importer.New(app))

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
