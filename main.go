package main

import (
	"log"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/skeeeon/pb-wiki/internal/hooks"
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
	static.Register(app, frontendDist())

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
