package access

import (
	"regexp"
	"strings"
)

// matchPattern reports whether path matches the wiki-go-style glob pattern.
//
// Both inputs are normalized to start with "/". The wildcard alphabet:
//
//   *   matches any run of characters within a single path segment (no "/")
//   ?   matches a single non-"/" character
//   **  matches any run of characters including "/"
//   /** at the very end of a pattern matches the parent path OR any descendant
//       (e.g. "/finance/**" matches "/finance", "/finance/", and "/finance/a/b")
//
// The bare pattern "/**" is treated specially: it matches only "/" itself. This
// prevents a "recursive homepage" rule from accidentally swallowing the whole
// wiki — see wiki-go internal/auth/access.go:50-59 for the original rationale.
//
// All other regex-special characters in the pattern are quoted literal.
func matchPattern(pattern, path string) bool {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}

	if pattern == "/**" {
		return path == "/"
	}

	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		// /** at end → optional "/anything"
		if i+3 <= len(pattern) && pattern[i:i+3] == "/**" && i+3 == len(pattern) {
			b.WriteString("(/.*)?")
			i += 2
			break
		}
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i++ // consume the second *
		} else if pattern[i] == '*' {
			b.WriteString("[^/]*")
		} else if pattern[i] == '?' {
			b.WriteString("[^/]")
		} else {
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")

	matched, _ := regexp.MatchString(b.String(), path)
	return matched
}
