// Package importer provides a one-shot CLI command for migrating a wiki-go
// (leomoon-studios) data directory into pb-wiki's documents collection.
//
// Mapping:
//
//	<data-dir>/pages/home/document.md      → path = ""        (homepage)
//	<data-dir>/documents/<slug>/document.md → path = "<slug>"  (directory chain joined by "/")
//
// The first level-1 heading is extracted into `title` and stripped from the
// stored body, since pb-wiki renders the title separately and including it in
// body would produce a duplicate header.
//
// The command is idempotent: re-running upserts on `path`, so a partial import
// can be safely retried.
package importer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

// New returns the `pb-wiki import` cobra command bound to the given app.
func New(app *pocketbase.PocketBase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <wiki-go-data-dir>",
		Short: "Import wiki-go documents into pb-wiki",
		Long: `Walks <wiki-go-data-dir>/pages/home/document.md and
<wiki-go-data-dir>/documents/**/document.md and upserts a record into the
documents collection for each one.

Mapping:
  pages/home/document.md         → path = ""
  documents/A/document.md        → path = "A"
  documents/A/B/document.md      → path = "A/B"

The first H1 in the markdown becomes the document title; everything below it
becomes the body. Re-running is safe — records are matched by path.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(app, args[0])
		},
	}
	return cmd
}

func run(app *pocketbase.PocketBase, dataDir string) error {
	docs, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return fmt.Errorf("find documents collection: %w", err)
	}

	var created, updated int

	// Homepage.
	homePath := filepath.Join(dataDir, "pages", "home", "document.md")
	if _, err := os.Stat(homePath); err == nil {
		isUpdate, err := upsert(app, docs, "", homePath)
		if err != nil {
			return fmt.Errorf("import home: %w", err)
		}
		report("", isUpdate)
		bump(&created, &updated, isUpdate)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat home: %w", err)
	}

	// documents/**/document.md
	docsRoot := filepath.Join(dataDir, "documents")
	walkErr := filepath.WalkDir(docsRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Base(p) != "document.md" {
			return nil
		}
		rel, err := filepath.Rel(docsRoot, filepath.Dir(p))
		if err != nil {
			return err
		}
		slug := filepath.ToSlash(rel)
		isUpdate, err := upsert(app, docs, slug, p)
		if err != nil {
			return fmt.Errorf("import %q: %w", slug, err)
		}
		report(slug, isUpdate)
		bump(&created, &updated, isUpdate)
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
		return walkErr
	}

	fmt.Printf("\nDone. %d created, %d updated.\n", created, updated)
	return nil
}

func upsert(app core.App, coll *core.Collection, slug, mdPath string) (bool, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return false, err
	}
	title, body := splitTitle(string(content))

	// dbx.HashExp goes directly to parameterized SQL — we deliberately avoid
	// FindFirstRecordByFilter here because PB's filter parser JSON-encodes
	// empty-string params into a literal `""` value (filter.go:71-77 in
	// pocketbase@v0.38), which would prevent the homepage (path="") from
	// matching itself on a re-import.
	existing, err := app.FindAllRecords("documents", dbx.HashExp{"path": slug})
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		rec := existing[0]
		rec.Set("title", title)
		rec.Set("body", body)
		if err := app.Save(rec); err != nil {
			return false, err
		}
		return true, nil
	}

	rec := core.NewRecord(coll)
	rec.Set("path", slug)
	rec.Set("title", title)
	rec.Set("body", body)
	return false, app.Save(rec)
}

// splitTitle pulls the first level-1 heading out of a markdown document and
// returns (title, body-without-that-heading). Falls back to ("", original)
// when no H1 is present so the import still succeeds.
func splitTitle(md string) (string, string) {
	lines := strings.SplitN(md, "\n", 2)
	first := ""
	if len(lines) > 0 {
		first = strings.TrimSpace(lines[0])
	}
	if !strings.HasPrefix(first, "# ") {
		return "", md
	}
	title := strings.TrimSpace(strings.TrimPrefix(first, "# "))
	rest := ""
	if len(lines) > 1 {
		// Skip a single blank line that conventionally follows the H1 so the
		// body doesn't start with awkward leading whitespace.
		rest = strings.TrimPrefix(lines[1], "\n")
	}
	return title, rest
}

func report(slug string, isUpdate bool) {
	verb := "create"
	if isUpdate {
		verb = "update"
	}
	display := slug
	if display == "" {
		display = "(home)"
	}
	fmt.Printf("  %s  %s\n", verb, display)
}

func bump(created, updated *int, isUpdate bool) {
	if isUpdate {
		*updated++
	} else {
		*created++
	}
}
