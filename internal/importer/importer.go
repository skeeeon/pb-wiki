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
//	access: private                   # optional: public | private | restricted
//	groups: [finance]                 # optional; group names, for restricted
//	nav_order: 20                     # optional; sidebar position among siblings
//	---
//	# Installation Guide
//	...body...
//
// Without `access`, a new page inherits its parent page's access and an
// existing page keeps what it has; the same goes for `nav_order`.
//
// Relative links to other .md files in the tree ([Deploy](../ops/deploy.md#rollback))
// are rewritten to wiki URLs (/doc/ops/deploy#rollback), so the source files
// still link correctly when browsed on disk or in a git host.
//
// Files without frontmatter, or with frontmatter missing `path`, are skipped
// (logged, not fatal) so a partial input tree doesn't abort the whole run.
//
// The command is idempotent: records are matched by `path`, so re-running
// updates rather than duplicates.
package importer

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
optionally declare "title", "access", "groups" and "nav_order":

  ---
  path: getting-started/install
  title: Installation Guide
  access: private
  nav_order: 20
  ---
  # Installation Guide
  ...

If "title" is omitted, the first H1 in the body is used and stripped. Without
"access", new pages inherit their parent page's access. Relative links to
other .md files in the tree become wiki links. Records are matched by path,
so re-running is safe.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return run(app, args[0])
		},
	}
	return cmd
}

type frontmatter struct {
	// Pointer so we can distinguish "field absent" from "path: \"\"" (homepage).
	Path     *string  `yaml:"path"`
	Title    string   `yaml:"title"`
	Access   string   `yaml:"access"`
	Groups   []string `yaml:"groups"`
	NavOrder *int     `yaml:"nav_order"`
}

// page is one parsed input file.
type page struct {
	file  string // absolute path on disk
	fm    frontmatter
	title string
	body  string
}

func run(app core.App, root string) error {
	docs, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return fmt.Errorf("find documents collection: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}

	// Pass 1: parse every file, so links can be resolved to any page.
	var pages []page
	var skipped int
	pathOf := map[string]string{} // absolute file → wiki path
	fileOf := map[string]string{} // wiki path → file, to report duplicates

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
		if prev, dup := fileOf[slug]; dup {
			return fmt.Errorf("duplicate path %q in input: %s and %s", slug, prev, p)
		}
		fileOf[slug] = p
		pathOf[p] = slug

		title := fm.Title
		bodyStr := string(body)
		if title == "" {
			title, bodyStr = splitTitle(bodyStr)
		}
		pages = append(pages, page{file: p, fm: fm, title: title, body: bodyStr})
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, fs.ErrNotExist) {
		return walkErr
	}

	// Pass 2: write parents before children ("a" sorts before "a/b"), so a
	// new page without `access` can inherit from a parent created in this run.
	sort.Slice(pages, func(i, j int) bool { return *pages[i].fm.Path < *pages[j].fm.Path })

	var created, updated int
	for _, pg := range pages {
		body, missing := rewriteLinks(pg.body, filepath.Dir(pg.file), pathOf)
		for _, target := range missing {
			fmt.Printf("  warn   %s: link to %s does not match an imported file\n", pg.file, target)
		}
		pg.body = body

		isUpdate, err := upsert(app, docs, pg)
		if err != nil {
			return fmt.Errorf("upsert %q (from %s): %w", *pg.fm.Path, pg.file, err)
		}
		report(*pg.fm.Path, isUpdate)
		if isUpdate {
			updated++
		} else {
			created++
		}
	}

	fmt.Printf("\nDone. %d created, %d updated, %d skipped.\n", created, updated, skipped)
	return nil
}

// mdLink matches an inline markdown link whose target is a .md file,
// optionally with an #anchor: [text](../ops/deploy.md#rollback).
var mdLink = regexp.MustCompile(`\]\(([^)\s#]+\.md)(#[^)\s]*)?\)`)

// rewriteLinks turns relative links to imported .md files into wiki URLs.
// dir is the directory of the file being imported; pathOf maps absolute file
// paths to wiki paths. Absolute and external links are left alone. Relative
// .md links that match no imported file are left alone and returned, so the
// caller can warn about them.
func rewriteLinks(body, dir string, pathOf map[string]string) (string, []string) {
	var missing []string
	out := mdLink.ReplaceAllStringFunc(body, func(m string) string {
		sub := mdLink.FindStringSubmatch(m)
		target, anchor := sub[1], sub[2]
		if strings.HasPrefix(target, "/") || strings.Contains(target, "://") {
			return m
		}
		slug, ok := pathOf[filepath.Join(dir, filepath.FromSlash(target))]
		if !ok {
			missing = append(missing, target)
			return m
		}
		url := "/doc/" + slug
		if slug == "" {
			url = "/"
		}
		return "](" + url + anchor + ")"
	})
	return out, missing
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

func upsert(app core.App, coll *core.Collection, pg page) (bool, error) {
	// dbx.HashExp goes directly to parameterized SQL — we deliberately avoid
	// FindFirstRecordByFilter here because PB's filter parser JSON-encodes
	// empty-string params into a literal `""` value (filter.go:71-77 in
	// pocketbase@v0.38), which would prevent the homepage (path="") from
	// matching itself on a re-import.
	existing, err := app.FindAllRecords("documents", dbx.HashExp{"path": *pg.fm.Path})
	if err != nil {
		return false, err
	}
	isUpdate := len(existing) > 0
	rec := core.NewRecord(coll)
	if isUpdate {
		rec = existing[0]
	}
	rec.Set("path", *pg.fm.Path)
	rec.Set("title", pg.title)
	rec.Set("body", pg.body)
	if pg.fm.Access != "" {
		ids, err := groupIDs(app, pg.fm.Groups)
		if err != nil {
			return false, err
		}
		rec.Set("access", pg.fm.Access)
		rec.Set("groups", ids)
	}
	if pg.fm.NavOrder != nil {
		rec.Set("nav_order", *pg.fm.NavOrder)
	}
	return isUpdate, app.Save(rec)
}

// groupIDs returns the ids of the named groups, creating any that don't
// exist yet (the importer runs with full database access).
func groupIDs(app core.App, names []string) ([]string, error) {
	coll, err := app.FindCollectionByNameOrId("groups")
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, name := range names {
		g, err := app.FindFirstRecordByData(coll, "name", name)
		if errors.Is(err, sql.ErrNoRows) {
			g = core.NewRecord(coll)
			g.Set("name", name)
			err = app.Save(g)
		}
		if err != nil {
			return nil, err
		}
		ids = append(ids, g.Id)
	}
	return ids, nil
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
