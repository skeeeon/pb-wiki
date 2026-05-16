// Package importer provides a one-shot CLI command for importing a directory
// of markdown files (with YAML frontmatter) into pb-wiki's documents
// collection. This is the input side of a git-ops style workflow: author
// content as plain markdown files in a git repo, then run `pb-wiki import` to
// upsert them into the wiki.
//
// File format:
//
//	---
//	path: getting-started/install     # required; use "" for the homepage
//	title: Installation Guide         # optional; falls back to the first H1
//	---
//	# Installation Guide
//	...body...
//
// Files without frontmatter, or with frontmatter missing `path`, are skipped
// (logged, not fatal) so a partial input tree doesn't abort the whole run.
//
// The command is idempotent: records are matched by `path`, so re-running
// updates rather than duplicates.
package importer

import (
	"bytes"
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
	"gopkg.in/yaml.v3"
)

// New returns the `pb-wiki import` cobra command bound to the given app.
func New(app *pocketbase.PocketBase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <markdown-dir>",
		Short: "Import markdown documents (with YAML frontmatter) into pb-wiki",
		Long: `Recursively walks <markdown-dir> for .md files. Each file must begin with
YAML frontmatter declaring a "path" (use path: "" for the homepage) and may
optionally declare a "title":

  ---
  path: getting-started/install
  title: Installation Guide
  ---
  # Installation Guide
  ...

If "title" is omitted, the first H1 in the body is used and stripped. Records
are matched by path, so re-running is safe.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(app, args[0])
		},
	}
	return cmd
}

type frontmatter struct {
	// Pointer so we can distinguish "field absent" from "path: \"\"" (homepage).
	Path  *string `yaml:"path"`
	Title string  `yaml:"title"`
}

func run(app *pocketbase.PocketBase, root string) error {
	docs, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return fmt.Errorf("find documents collection: %w", err)
	}

	var created, updated, skipped int
	// Detect duplicate paths within the input tree before we let the DB's
	// unique index reject the second one with a less helpful error.
	seen := map[string]string{}

	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		fm, body, err := parseFrontmatter(content)
		if err != nil {
			fmt.Printf("  skip   %s: %v\n", p, err)
			skipped++
			return nil
		}
		if fm.Path == nil {
			fmt.Printf("  skip   %s: missing required `path` in frontmatter\n", p)
			skipped++
			return nil
		}
		slug := *fm.Path
		if prev, dup := seen[slug]; dup {
			return fmt.Errorf("duplicate path %q in input: %s and %s", slug, prev, p)
		}
		seen[slug] = p

		title := fm.Title
		bodyStr := string(body)
		if title == "" {
			title, bodyStr = splitTitle(bodyStr)
		}

		isUpdate, err := upsert(app, docs, slug, title, bodyStr)
		if err != nil {
			return fmt.Errorf("upsert %q (from %s): %w", slug, p, err)
		}
		report(slug, isUpdate)
		if isUpdate {
			updated++
		} else {
			created++
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
		return walkErr
	}

	fmt.Printf("\nDone. %d created, %d updated, %d skipped.\n", created, updated, skipped)
	return nil
}

// parseFrontmatter pulls a YAML frontmatter block (delimited by `---` lines)
// off the front of a markdown document and returns the parsed metadata plus
// the remaining body. A file without frontmatter is fine: returns a zero
// frontmatter and the original bytes. An opened-but-unclosed block is an
// error so authors notice typos rather than silently importing the whole file
// body as YAML.
func parseFrontmatter(b []byte) (frontmatter, []byte, error) {
	var fm frontmatter
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})

	if !bytes.HasPrefix(b, []byte("---\n")) && !bytes.HasPrefix(b, []byte("---\r\n")) {
		return fm, b, nil
	}

	// Skip past the opening delimiter line.
	nl := bytes.IndexByte(b, '\n')
	if nl < 0 {
		return fm, nil, errors.New("frontmatter opened but not closed")
	}
	rest := b[nl+1:]

	// Walk lines looking for a closing line containing only "---".
	off := 0
	for off <= len(rest) {
		nlIdx := bytes.IndexByte(rest[off:], '\n')
		var line []byte
		var lineEnd int
		if nlIdx < 0 {
			line = rest[off:]
			lineEnd = len(rest)
		} else {
			line = rest[off : off+nlIdx]
			lineEnd = off + nlIdx + 1
		}
		if bytes.Equal(bytes.TrimRight(line, "\r"), []byte("---")) {
			if err := yaml.Unmarshal(rest[:off], &fm); err != nil {
				return frontmatter{}, nil, fmt.Errorf("parse frontmatter yaml: %w", err)
			}
			return fm, rest[lineEnd:], nil
		}
		if nlIdx < 0 {
			break
		}
		off = lineEnd
	}
	return fm, nil, errors.New("frontmatter opened but not closed")
}

func upsert(app core.App, coll *core.Collection, slug, title, body string) (bool, error) {
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
