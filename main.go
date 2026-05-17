package main

import (
	"log"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	pbaudit "github.com/skeeeon/pb-audit"

	"github.com/skeeeon/pb-wiki/internal/api"
	"github.com/skeeeon/pb-wiki/internal/hooks"
	"github.com/skeeeon/pb-wiki/internal/importer"
	"github.com/skeeeon/pb-wiki/internal/static"
	_ "github.com/skeeeon/pb-wiki/migrations"
)

func main() {
	app := pocketbase.New()

	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  isGoRun,
	})

	hooks.Register(app)
	if err := pbaudit.Setup(app, pbaudit.DefaultOptions()); err != nil {
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
