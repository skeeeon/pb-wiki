// Package access implements pb-wiki's path-based access rules.
//
// The semantics are a direct port of wiki-go's internal/auth/access.go: each
// rule has a glob pattern + access level + optional groups list; the first
// matching rule wins; admins bypass everything; unknown access levels fail
// closed. See matchPattern in glob.go for the supported pattern syntax.
//
// This package is intentionally free of any pocketbase imports so it can be
// unit-tested in isolation. Callers (the hooks package) load rules and the
// current user from PocketBase records and pass them in as the typed Rule and
// User values defined here.
package access

// Role constants for users.
const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
	RoleViewer = "viewer"
)

// Access-level constants for rules.
const (
	AccessPublic     = "public"
	AccessPrivate    = "private"
	AccessRestricted = "restricted"
)

// Rule mirrors a row of the access_rules collection. The caller is expected to
// load rules ORDER BY priority ASC before passing them to CanAccess (first
// match wins, so ordering is load-bearing).
type Rule struct {
	Pattern string
	Access  string
	Groups  []string
}

// User is the minimal view of the requester needed to evaluate a rule.
// A nil *User represents an anonymous request.
type User struct {
	Role   string
	Groups []string
}

// CanAccess reports whether user may read path under the given rules.
//
//  1. Admins bypass all rules.
//  2. The first rule whose pattern matches path is the one applied.
//  3. If no rule matches, the wiki's privateDefault decides: when true, only
//     authenticated users are allowed through.
//  4. Restricted rules require at least one overlap between the rule's groups
//     and the user's groups.
//  5. Unknown access levels (e.g. typos in config) deny — fail-closed.
func CanAccess(path string, user *User, rules []Rule, privateDefault bool) bool {
	if user != nil && user.Role == RoleAdmin {
		return true
	}
	if rule := findMatchingRule(path, rules); rule != nil {
		return checkRule(rule, user)
	}
	if privateDefault {
		return user != nil
	}
	return true
}

func findMatchingRule(path string, rules []Rule) *Rule {
	for i := range rules {
		if matchPattern(rules[i].Pattern, path) {
			return &rules[i]
		}
	}
	return nil
}

func checkRule(rule *Rule, user *User) bool {
	switch rule.Access {
	case AccessPublic:
		return true
	case AccessPrivate:
		return user != nil
	case AccessRestricted:
		if user == nil {
			return false
		}
		for _, g := range rule.Groups {
			for _, ug := range user.Groups {
				if g == ug {
					return true
				}
			}
		}
		return false
	default:
		// Unknown access level — fail closed.
		return false
	}
}
