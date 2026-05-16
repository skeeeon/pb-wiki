package api

import "testing"

// rewritePath is the path-rewriting kernel of the bulk-move handler. It
// gets unit-tested in isolation since the surrounding handler needs a live
// PocketBase app and we cover the SQL/transaction side by hand.
func TestRewritePath(t *testing.T) {
	cases := []struct {
		name string
		old  string
		from string
		to   string
		want string
	}{
		{"exact match maps to to", "finance", "finance", "fin", "fin"},
		{"child keeps suffix", "finance/q4", "finance", "fin", "fin/q4"},
		{"deep child keeps suffix", "finance/q4/notes", "finance", "fin", "fin/q4/notes"},
		{"empty to renames to homepage", "finance", "finance", "", ""},
		{"empty to drops prefix on children", "finance/q4", "finance", "", "q4"},
		{"to extends from", "eng", "eng", "engineering", "engineering"},
		{"to extends from with child", "eng/docs", "eng", "engineering", "engineering/docs"},
		{"to nests inside from", "eng", "eng", "eng/sub", "eng/sub"},
		{"to nests inside from with child", "eng/x", "eng", "eng/sub", "eng/sub/x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rewritePath(tc.old, tc.from, tc.to); got != tc.want {
				t.Errorf("rewritePath(%q, %q, %q) = %q, want %q", tc.old, tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  ", ""},
		{"foo", "foo"},
		{"/foo", "foo"},
		{"foo/", "foo"},
		{"/foo/", "foo"},
		{"  /foo/  ", "foo"},
		{"foo/bar", "foo/bar"},
		{"///foo///", "foo"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizePath(tc.in); got != tc.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
