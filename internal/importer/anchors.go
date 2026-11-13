package importer

import (
	"fmt"
	"html"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// jsSpace is the set of characters JavaScript's \s matches. Go's \s is
// ASCII only, so slugify spells the JavaScript set out.
const jsSpace = `\t\n\v\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`

var (
	slugStrip = regexp.MustCompile(`[^\p{L}\p{N}` + jsSpace + `_-]`)
	slugSpace = regexp.MustCompile(`[` + jsSpace + `]+`)
	slugDash  = regexp.MustCompile(`-+`)
)

// slugify turns heading text into the id the wiki gives the heading.
// It must stay in step with slugify in frontend/src/lib/markdown.ts.
// (scripts/mkdocs-convert/main.go has a third copy.)
func slugify(s string) string {
	s = slugStrip.ReplaceAllString(strings.ToLower(s), "")
	s = strings.TrimFunc(s, func(r rune) bool { return slugSpace.MatchString(string(r)) })
	s = slugSpace.ReplaceAllString(s, "-")
	return slugDash.ReplaceAllString(s, "-")
}

var (
	atxHeading    = regexp.MustCompile(`^ {0,3}#{1,6}(?:[ \t]+(.*))?$`)
	closingHashes = regexp.MustCompile(`(?:^|[ \t])#+[ \t]*$`)
	fence         = regexp.MustCompile("^[ \t]*(`{3,}|~{3,})")
	imageMarkup   = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	linkMarkup    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// headingIDs returns the ids the wiki gives the headings in a page body.
// The wiki renders with markdown-it-anchor (frontend/src/lib/markdown.ts),
// which slugs every heading's text and adds -1, -2, ... to repeated ids.
//
// Only ATX headings (# to ######) outside fenced code blocks are found.
// Setext headings (underlined with === or ---) and headings inside block
// quotes or list items are not, so a link to one of those is reported
// although it works.
//
// The page title that splitTitle takes out of the body is shown above the
// page without an id, so a link to it does not resolve. Pass the body after
// splitTitle to get the same result.
func headingIDs(body string) map[string]bool {
	ids := map[string]bool{}
	var open string // the fence that opened the current code block, or ""
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		if m := fence.FindStringSubmatch(line); m != nil {
			switch {
			case open == "":
				open = m[1]
			case m[1][0] == open[0] && len(m[1]) >= len(open) && strings.TrimSpace(line) == m[1]:
				open = ""
			}
			continue
		}
		if open != "" {
			continue
		}
		m := atxHeading.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := closingHashes.ReplaceAllString(m[1], "")
		uniqueSlug(slugify(headingText(text)), ids)
	}
	return ids
}

// uniqueSlug adds slug to used, with -1, -2, ... added when used already
// has it, the same as markdown-it-anchor with uniqueSlugStartIndex 1 (its
// default; frontend/src/lib/markdown.ts does not change it).
func uniqueSlug(slug string, used map[string]bool) {
	s := slug
	for i := 1; used[s]; i++ {
		s = fmt.Sprintf("%s-%d", slug, i)
	}
	used[s] = true
}

// headingText approximates the text markdown-it-anchor slugs: the text and
// inline code of the heading. A link gives its text, an image gives
// nothing, and code spans keep what is in them. Outside code, emphasis
// markers, backslash escapes and entities are removed. A '_' between two
// letters stays (snake_case is not emphasis); a '_' at the edge of a word
// is taken as emphasis, even when it has no partner.
func headingText(s string) string {
	s = imageMarkup.ReplaceAllString(s, "")
	s = linkMarkup.ReplaceAllString(s, "$1")
	var out strings.Builder
	for s != "" {
		i := strings.IndexByte(s, '`')
		if i < 0 {
			out.WriteString(plainText(s))
			break
		}
		out.WriteString(plainText(s[:i]))
		s = s[i:]
		n := len(s) - len(strings.TrimLeft(s, "`"))
		end := closingTicks(s[n:], n)
		if end < 0 {
			out.WriteString(s[:n]) // no closing run: the backticks are text
			s = s[n:]
			continue
		}
		code := s[n : n+end]
		if len(code) > 1 && code[0] == ' ' && code[len(code)-1] == ' ' && strings.Trim(code, " ") != "" {
			code = code[1 : len(code)-1]
		}
		out.WriteString(code)
		s = s[n+end+n:]
	}
	return out.String()
}

// closingTicks returns the index in s of the first backtick run that is
// exactly n long, or -1.
func closingTicks(s string, n int) int {
	for i := 0; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(s) && s[j] == '`' {
			j++
		}
		if j-i == n {
			return i
		}
		i = j
	}
	return -1
}

// asciiPunct is what a backslash can escape in CommonMark.
const asciiPunct = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// plainText removes emphasis '_', backslash escapes and entities from
// heading text outside code spans. slugify drops '*' itself.
func plainText(s string) string {
	rs := []rune(s)
	var out []rune
	word := func(i int) bool {
		return i >= 0 && i < len(rs) && (unicode.IsLetter(rs[i]) || unicode.IsDigit(rs[i]))
	}
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\\' && i+1 < len(rs) && strings.ContainsRune(asciiPunct, rs[i+1]):
			out = append(out, rs[i+1])
			i++
		case r == '_':
			j := i
			for j < len(rs) && rs[j] == '_' {
				j++
			}
			if word(i-1) == word(j) {
				out = append(out, rs[i:j]...) // inside a word, or alone: text
			}
			i = j - 1
		default:
			out = append(out, r)
		}
	}
	return html.UnescapeString(string(out))
}

// anchorLink matches an inline link with an #anchor, to a .md file or to
// the same page: [text](../ops/deploy.md#rollback), [text](#rollback).
var anchorLink = regexp.MustCompile(`\]\(((?:[^)\s#]+\.md)?)#([^)\s]+)\)`)

// brokenAnchors returns the #anchor links in body that match no heading in
// their target page. dir is the directory of the file being imported,
// pathOf maps absolute file paths to wiki paths, and idsOf maps wiki paths
// to heading ids. self is the wiki path of the page itself. Links to files
// that were not imported are skipped; rewriteLinks reports those. Like
// rewriteLinks, it also looks at links inside code blocks.
func brokenAnchors(body, dir, self string, pathOf map[string]string, idsOf map[string]map[string]bool) []string {
	var broken []string
	for _, m := range anchorLink.FindAllStringSubmatch(body, -1) {
		target, anchor := m[1], m[2]
		slug := self
		if target != "" {
			if strings.HasPrefix(target, "/") || strings.Contains(target, "://") {
				continue
			}
			var ok bool
			if slug, ok = pathOf[filepath.Join(dir, filepath.FromSlash(target))]; !ok {
				continue
			}
		}
		id := anchor
		if u, err := url.PathUnescape(anchor); err == nil {
			id = u
		}
		if !idsOf[slug][id] {
			broken = append(broken, target+"#"+anchor)
		}
	}
	return broken
}
