// Command mkdocs-convert copies an MkDocs docs directory into pb-wiki's
// import format, ready for `pb-wiki import`:
//
//	go run ./scripts/mkdocs-convert -site ../platform-docs -prefix platform -out ./import
//	pb-wiki import ./import
//
// What it does:
//   - adds frontmatter: `path` (index.md becomes the folder's own page) and
//     `nav_order` from the order of the `nav:` entries in mkdocs.yml;
//   - rewrites `!!! type "Title"` (and collapsible `??? type`) admonitions
//     as `::: type Title` callouts, mapping MkDocs types pb-wiki lacks onto
//     note/tip/warning/danger;
//   - writes a section page (a list of its pages) for any folder in the nav
//     that has no page of its own, so the section has a title and a place in
//     the sidebar;
//   - reports #anchor links that won't resolve under pb-wiki's heading ids.
//
// Relative .md links are left alone; `pb-wiki import` rewrites them.
// The source directory is never modified.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	navFile    = regexp.MustCompile(`^\s*- (?:(.+): )?(\S+\.md)\s*$`)
	navSection = regexp.MustCompile(`^\s*- (.+):\s*$`)
	admonition = regexp.MustCompile(`^(?:!!!|\?\?\?\+?) (\w+)(?: "(.*)")?\s*$`)
	heading    = regexp.MustCompile(`^#{1,6}\s+(.*?)\s*#*\s*$`)
	linkTarget = regexp.MustCompile(`\]\(([^)\s]*)\)`)
	linkText   = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// calloutType maps MkDocs admonition types onto pb-wiki's four callouts.
var calloutType = map[string]string{
	"note": "note", "abstract": "note", "summary": "note", "info": "note", "todo": "note",
	"question": "note", "help": "note", "faq": "note", "example": "note", "quote": "note", "cite": "note",
	"tip": "tip", "hint": "tip", "important": "tip", "success": "tip", "check": "tip", "done": "tip",
	"warning": "warning", "caution": "warning", "attention": "warning",
	"danger": "danger", "error": "danger", "failure": "danger", "fail": "danger", "missing": "danger", "bug": "danger",
}

func main() {
	site := flag.String("site", ".", "MkDocs project directory (contains mkdocs.yml)")
	docsDir := flag.String("docs", "docs", "docs directory, relative to -site")
	prefix := flag.String("prefix", "", `wiki path the docs are imported under, e.g. "platform"`)
	out := flag.String("out", "", "output directory (required; must not exist)")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "mkdocs-convert: -out is required")
		os.Exit(2)
	}
	if err := convert(*site, filepath.Join(*site, *docsDir), strings.Trim(*prefix, "/"), *out); err != nil {
		fmt.Fprintln(os.Stderr, "mkdocs-convert:", err)
		os.Exit(1)
	}
}

func convert(site, src, prefix, out string) error {
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s already exists", out)
	}
	nav, err := readNav(filepath.Join(site, "mkdocs.yml"))
	if err != nil {
		return err
	}

	pages := map[string]string{} // docs-relative file (slash-separated) → converted body
	var files []string
	err = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		body := strings.ReplaceAll(string(raw), "\r\n", "\n")
		if strings.HasPrefix(body, "---\n") {
			return fmt.Errorf("%s already has frontmatter; merge it by hand", rel)
		}
		rel = filepath.ToSlash(rel)
		files = append(files, rel)
		pages[rel] = convertAdmonitions(body)
		return nil
	})
	if err != nil {
		return err
	}

	// Section pages for nav folders without a page of their own.
	for _, s := range nav.sections {
		if _, ok := pages[s.dir+".md"]; ok {
			continue
		}
		if _, ok := pages[s.dir+"/index.md"]; ok {
			continue
		}
		body := "# " + s.title + "\n\n"
		for _, f := range s.files {
			body += "- [" + titleOf(pages[f], f) + "](" + path.Base(s.dir) + "/" + strings.TrimPrefix(f, s.dir+"/") + ")\n"
		}
		pages[s.dir+".md"] = body
		files = append(files, s.dir+".md")
	}
	sort.Strings(files)

	misses := checkAnchors(files, pages)
	for _, m := range misses {
		fmt.Println("anchor will not resolve:", m)
	}

	for _, rel := range files {
		fm := "---\npath: " + wikiPath(prefix, rel) + "\n"
		if n, ok := nav.order[rel]; ok {
			fm += fmt.Sprintf("nav_order: %d\n", n)
		}
		dst := filepath.Join(out, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, []byte(fm+"---\n"+pages[rel]), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("%d pages written to %s, %d anchor problems\n", len(files), out, len(misses))
	return nil
}

type section struct {
	title string
	dir   string   // docs-relative folder its pages live in
	files []string // in nav order
}

type navInfo struct {
	order    map[string]int // docs-relative file → nav_order
	sections []*section
}

// readNav reads the nav: list from mkdocs.yml with a line scanner (the file
// often contains !!python tags a YAML parser would reject). Pages are
// numbered 10, 20, … within their folder, in nav order. A nav section whose
// pages share a subfolder gets that folder's section page numbered in its
// parent folder at the point the section appears.
func readNav(file string) (navInfo, error) {
	nav := navInfo{order: map[string]int{}}
	f, err := os.Open(file)
	if err != nil {
		return nav, err
	}
	defer f.Close()

	count := map[string]int{} // folder → pages numbered so far
	byDir := map[string]*section{}
	current := ""
	inNav := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "-") {
			inNav = strings.HasPrefix(line, "nav:")
			continue
		}
		if !inNav {
			continue
		}
		if m := navFile.FindStringSubmatch(line); m != nil {
			file, dir := m[2], path.Dir(m[2])
			if dir != "." && byDir[dir] == nil {
				s := &section{title: current, dir: dir}
				if s.title == "" {
					s.title = path.Base(dir)
				}
				byDir[dir] = s
				nav.sections = append(nav.sections, s)
				parent := path.Dir(dir)
				count[parent]++
				nav.order[dir+".md"] = count[parent] * 10
			}
			if dir != "." {
				byDir[dir].files = append(byDir[dir].files, file)
			}
			count[dir]++
			nav.order[file] = count[dir] * 10
		} else if m := navSection.FindStringSubmatch(line); m != nil {
			current = m[1]
		}
	}
	if err := sc.Err(); err != nil {
		return nav, err
	}
	if len(nav.order) == 0 {
		return nav, errors.New("no nav entries found in mkdocs.yml")
	}
	return nav, nil
}

func wikiPath(prefix, rel string) string {
	p := strings.TrimSuffix(rel, ".md")
	if p == "index" {
		p = ""
	}
	p = strings.TrimSuffix(p, "/index")
	switch {
	case prefix == "":
		return p
	case p == "":
		return prefix
	default:
		return prefix + "/" + p
	}
}

// convertAdmonitions rewrites MkDocs admonitions (a marker line followed by a
// block indented four spaces) as ::: callouts. Fenced code is left alone.
func convertAdmonitions(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inFence := false
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
		}
		m := admonition.FindStringSubmatch(l)
		if inFence || m == nil {
			out = append(out, l)
			continue
		}
		kind := calloutType[m[1]]
		if kind == "" {
			kind = "note"
		}
		out = append(out, strings.TrimSpace("::: "+kind+" "+m[2]))
		j := i + 1
		for ; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) != "" && !strings.HasPrefix(lines[j], "    ") {
				break
			}
			out = append(out, strings.TrimPrefix(lines[j], "    "))
		}
		for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
			out = out[:len(out)-1]
		}
		out = append(out, ":::", "")
		i = j - 1
	}
	return strings.Join(out, "\n")
}

func titleOf(body, file string) string {
	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "# ") {
			return strings.TrimPrefix(l, "# ")
		}
	}
	return strings.TrimSuffix(path.Base(file), ".md")
}

// checkAnchors returns "file -> link" for every #anchor link, on the same
// page or to another converted page, that matches no heading id there.
func checkAnchors(files []string, pages map[string]string) []string {
	ids := map[string]map[string]bool{}
	for _, rel := range files {
		ids[rel] = map[string]bool{}
		inFence := false
		for _, l := range strings.Split(pages[rel], "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), "```") {
				inFence = !inFence
			}
			if m := heading.FindStringSubmatch(l); m != nil && !inFence {
				ids[rel][slugify(headingText(m[1]))] = true
			}
		}
	}
	var misses []string
	for _, rel := range files {
		for _, m := range linkTarget.FindAllStringSubmatch(pages[rel], -1) {
			file, anchor, ok := strings.Cut(m[1], "#")
			if !ok || strings.Contains(file, "://") {
				continue
			}
			target := rel
			if file != "" {
				target = path.Clean(path.Join(path.Dir(rel), file))
			}
			if headings, ok := ids[target]; ok && !headings[anchor] {
				misses = append(misses, rel+" -> "+m[1])
			}
		}
	}
	return misses
}

// headingText approximates the plain text markdown-it gives the slugger:
// link text without the URL, no code or emphasis markers.
func headingText(s string) string {
	s = linkText.ReplaceAllString(s, "$1")
	return strings.NewReplacer("`", "", "**", "", "*", "").Replace(s)
}

// slugify must match slugify in frontend/src/lib/markdown.ts.
var (
	slugStrip = regexp.MustCompile(`[^\p{L}\p{N}\s_-]`)
	slugSpace = regexp.MustCompile(`\s+`)
	slugDash  = regexp.MustCompile(`-+`)
)

func slugify(s string) string {
	s = slugStrip.ReplaceAllString(strings.ToLower(s), "")
	s = slugSpace.ReplaceAllString(strings.TrimSpace(s), "-")
	return slugDash.ReplaceAllString(s, "-")
}
