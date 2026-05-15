package access

import "testing"

// --- CanAccess: end-to-end behavior ---------------------------------------

func TestCanAccess_AdminBypassesEverything(t *testing.T) {
	rules := []Rule{
		{Pattern: "/finance/**", Access: AccessRestricted, Groups: []string{"finance"}},
	}
	admin := &User{Role: RoleAdmin}

	// Admin must reach a restricted path even without the required group.
	if !CanAccess("/finance/q4-report", admin, rules, true) {
		t.Error("admin should bypass restricted rule")
	}
	// And any other path, even with privateDefault=true.
	if !CanAccess("/anything", admin, nil, true) {
		t.Error("admin should bypass private wiki default")
	}
}

func TestCanAccess_NoRule_PublicWiki(t *testing.T) {
	if !CanAccess("/anything", nil, nil, false) {
		t.Error("anonymous should access an unprotected public wiki")
	}
	if !CanAccess("/anything", &User{Role: RoleViewer}, nil, false) {
		t.Error("viewer should access an unprotected public wiki")
	}
}

func TestCanAccess_NoRule_PrivateWiki(t *testing.T) {
	if CanAccess("/anything", nil, nil, true) {
		t.Error("anonymous must be denied when wiki is private")
	}
	if !CanAccess("/anything", &User{Role: RoleViewer}, nil, true) {
		t.Error("authenticated viewer should access a private wiki without explicit rules")
	}
}

func TestCanAccess_PublicRuleOverridesPrivateWiki(t *testing.T) {
	rules := []Rule{
		{Pattern: "/help/**", Access: AccessPublic},
	}
	if !CanAccess("/help/getting-started", nil, rules, true) {
		t.Error("public rule should let anonymous through even when wiki is private")
	}
}

func TestCanAccess_RestrictedRequiresGroupMembership(t *testing.T) {
	rules := []Rule{
		{Pattern: "/finance/**", Access: AccessRestricted, Groups: []string{"finance", "execs"}},
	}

	cases := []struct {
		name string
		user *User
		want bool
	}{
		{"anonymous denied", nil, false},
		{"wrong group denied", &User{Role: RoleViewer, Groups: []string{"engineering"}}, false},
		{"no groups denied", &User{Role: RoleViewer}, false},
		{"matching group allowed", &User{Role: RoleViewer, Groups: []string{"finance"}}, true},
		{"second-listed group allowed", &User{Role: RoleViewer, Groups: []string{"execs"}}, true},
		{"multiple groups including match", &User{Role: RoleViewer, Groups: []string{"engineering", "finance"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanAccess("/finance/q4-report", tc.user, rules, false)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCanAccess_PrivateRuleNeedsAuthOnly(t *testing.T) {
	rules := []Rule{
		{Pattern: "/internal/**", Access: AccessPrivate},
	}
	if CanAccess("/internal/runbook", nil, rules, false) {
		t.Error("private rule must reject anonymous")
	}
	if !CanAccess("/internal/runbook", &User{Role: RoleViewer}, rules, false) {
		t.Error("private rule should accept any authenticated user")
	}
}

func TestCanAccess_UnknownAccessLevelDeniesByDefault(t *testing.T) {
	// If a typo lands in config (`acces: pubic`), the safe behavior is to deny.
	// Admin still bypasses via the early-return, so use non-admin users here to
	// actually exercise checkRule's default branch.
	rules := []Rule{
		{Pattern: "/foo", Access: "pubic"}, // intentional typo
	}
	if CanAccess("/foo", &User{Role: RoleEditor}, rules, false) {
		t.Error("unknown access level must deny authenticated non-admin user")
	}
	if CanAccess("/foo", nil, rules, false) {
		t.Error("unknown access level must deny anonymous")
	}
}

func TestCanAccess_FirstMatchingRuleWins(t *testing.T) {
	// Order matters — the first rule whose pattern matches is the one applied,
	// even if a later rule would also match. Load-bearing for the "specific
	// rules above general rules" admin workflow.
	rules := []Rule{
		{Pattern: "/docs/secret/**", Access: AccessRestricted, Groups: []string{"sec"}},
		{Pattern: "/docs/**", Access: AccessPublic},
	}

	if CanAccess("/docs/secret/keys", nil, rules, false) {
		t.Error("specific restricted rule should win over later public rule")
	}
	if !CanAccess("/docs/secret/keys", &User{Role: RoleViewer, Groups: []string{"sec"}}, rules, false) {
		t.Error("user in required group should pass specific rule")
	}
	if !CanAccess("/docs/getting-started", nil, rules, false) {
		t.Error("non-matching specific rule should fall through to public")
	}
}

// --- matchPattern: glob translation ---------------------------------------

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		// Exact matches
		{"exact match", "/foo", "/foo", true},
		{"exact mismatch", "/foo", "/bar", false},
		{"exact does not match prefix", "/foo", "/foo/bar", false},

		// Leading-slash normalization
		{"pattern missing leading slash", "foo", "/foo", true},
		{"path missing leading slash", "/foo", "foo", true},
		{"both missing leading slash", "foo", "foo", true},

		// Single wildcard (does not cross /)
		{"single star matches segment", "/users/*", "/users/alice", true},
		{"single star does not cross slash", "/users/*", "/users/alice/profile", false},
		{"single star at end matches empty", "/users/*", "/users/", true},

		// Double wildcard (crosses /)
		{"double star matches across slashes", "/docs/**", "/docs/a/b/c", true},
		{"double star matches single segment", "/docs/**", "/docs/intro", true},
		{"double star middle of pattern", "/a/**/z", "/a/b/c/z", true},

		// Trailing /** — special "matches parent or any child" form
		{"trailing /** matches parent itself", "/finance/**", "/finance", true},
		{"trailing /** matches trailing slash", "/finance/**", "/finance/", true},
		{"trailing /** matches deep child", "/finance/**", "/finance/2025/q4", true},
		{"trailing /** does not match sibling", "/finance/**", "/financial", false},

		// Question mark — single char, no slash
		{"question mark single char", "/file?.md", "/file1.md", true},
		{"question mark does not cross slash", "/file?", "/file/a", false},

		// /** edge case — only matches root, never deeper paths
		// (prevents a recursive homepage rule from matching the entire wiki)
		{"slash-doublestar matches root", "/**", "/", true},
		{"slash-doublestar does NOT match deeper paths", "/**", "/anything", false},
		{"slash-doublestar does NOT match nested", "/**", "/a/b", false},

		// Regex-special characters in pattern are quoted
		{"dots are literal", "/docs.html", "/docsxhtml", false},
		{"dots are literal exact", "/docs.html", "/docs.html", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchPattern(tc.pattern, tc.path); got != tc.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
			}
		})
	}
}

// --- findMatchingRule -----------------------------------------------------

func TestFindMatchingRule_NilWhenNoMatch(t *testing.T) {
	rules := []Rule{
		{Pattern: "/foo", Access: AccessPublic},
	}
	if findMatchingRule("/bar", rules) != nil {
		t.Error("expected nil for non-matching path")
	}
}

func TestFindMatchingRule_ReturnsFirstMatch(t *testing.T) {
	// Use Groups to identify which rule was returned (Description isn't on the
	// new Rule struct — we only model the fields the evaluator needs).
	rules := []Rule{
		{Pattern: "/foo/**", Access: AccessRestricted, Groups: []string{"first"}},
		{Pattern: "/foo/bar", Access: AccessPublic, Groups: []string{"second"}},
	}
	got := findMatchingRule("/foo/bar", rules)
	if got == nil {
		t.Fatal("expected a match")
	}
	if len(got.Groups) != 1 || got.Groups[0] != "first" {
		t.Errorf("expected first rule, got groups %v", got.Groups)
	}
}
