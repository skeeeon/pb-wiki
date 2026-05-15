package hooks

import "testing"

func TestEmailAllowed(t *testing.T) {
	cases := []struct {
		name      string
		email     string
		allowlist []string
		want      bool
	}{
		// Bare-domain entries match anything ending in @<domain>.
		{"domain match", "alice@example.com", []string{"example.com"}, true},
		{"domain mismatch", "alice@other.com", []string{"example.com"}, false},
		{"domain match, case-insensitive", "Alice@Example.COM", []string{"example.com"}, true},
		{"domain entry case-insensitive", "alice@example.com", []string{"EXAMPLE.com"}, true},

		// Full-email entries match exactly that address.
		{"full email match", "alice@example.com", []string{"alice@example.com"}, true},
		{"full email mismatch", "bob@example.com", []string{"alice@example.com"}, false},

		// Mixed entries — any matches counts.
		{"mixed: domain hits", "carol@partners.io", []string{"alice@example.com", "partners.io"}, true},
		{"mixed: full-email hits", "alice@example.com", []string{"alice@example.com", "partners.io"}, true},
		{"mixed: neither hits", "eve@other.com", []string{"alice@example.com", "partners.io"}, false},

		// Empty/whitespace entries are ignored, not treated as wildcards.
		{"empty entry ignored", "alice@example.com", []string{"", "  ", "example.com"}, true},
		{"only-empty-entries denies", "alice@example.com", []string{"", "  "}, false},

		// Substring-on-domain doesn't accidentally match — bare-domain entry
		// must match the suffix after "@", not any substring of the email.
		{"domain suffix-only, no substring", "alice@notexample.com", []string{"example.com"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := emailAllowed(tc.email, tc.allowlist); got != tc.want {
				t.Errorf("emailAllowed(%q, %v) = %v, want %v", tc.email, tc.allowlist, got, tc.want)
			}
		})
	}
}
