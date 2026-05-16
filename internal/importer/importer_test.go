package importer

import (
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	strp := func(s string) *string { return &s }

	cases := []struct {
		name      string
		in        string
		wantPath  *string
		wantTitle string
		wantBody  string
		wantErr   bool
	}{
		{
			name:      "basic",
			in:        "---\npath: foo/bar\ntitle: Foo Bar\n---\n# Foo Bar\n\nbody\n",
			wantPath:  strp("foo/bar"),
			wantTitle: "Foo Bar",
			wantBody:  "# Foo Bar\n\nbody\n",
		},
		{
			name:     "homepage empty path",
			in:       "---\npath: \"\"\n---\nhome body\n",
			wantPath: strp(""),
			wantBody: "home body\n",
		},
		{
			name:     "title omitted",
			in:       "---\npath: a\n---\nbody\n",
			wantPath: strp("a"),
			wantBody: "body\n",
		},
		{
			name:     "no frontmatter",
			in:       "# just a heading\n\nbody\n",
			wantPath: nil,
			wantBody: "# just a heading\n\nbody\n",
		},
		{
			name:     "crlf line endings",
			in:       "---\r\npath: a\r\ntitle: T\r\n---\r\nbody\r\n",
			wantPath: strp("a"),
			wantTitle: "T",
			wantBody: "body\r\n",
		},
		{
			name:     "bom prefix",
			in:       "\xEF\xBB\xBF---\npath: a\n---\nbody\n",
			wantPath: strp("a"),
			wantBody: "body\n",
		},
		{
			name:    "unterminated frontmatter",
			in:      "---\npath: a\nno closing delim here\n",
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			in:      "---\npath: [unbalanced\n---\nbody\n",
			wantErr: true,
		},
		{
			name:     "empty frontmatter block",
			in:       "---\n---\nbody\n",
			wantPath: nil,
			wantBody: "body\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fm, body, err := parseFrontmatter([]byte(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got fm=%+v body=%q", fm, body)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got, want := fm.Path, tc.wantPath; !samePtrString(got, want) {
				t.Errorf("path: got %v want %v", strOrNil(got), strOrNil(want))
			}
			if fm.Title != tc.wantTitle {
				t.Errorf("title: got %q want %q", fm.Title, tc.wantTitle)
			}
			if string(body) != tc.wantBody {
				t.Errorf("body: got %q want %q", string(body), tc.wantBody)
			}
		})
	}
}

func samePtrString(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func strOrNil(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func TestSplitTitle(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantTitle string
		wantBody  string
	}{
		{"h1 with blank line", "# Hello\n\nbody\n", "Hello", "body\n"},
		{"h1 with no blank line", "# Hello\nbody\n", "Hello", "body\n"},
		{"no h1", "no heading here\n", "", "no heading here\n"},
		{"h2 not stripped", "## Sub\nbody\n", "", "## Sub\nbody\n"},
		{"empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, body := splitTitle(tc.in)
			if title != tc.wantTitle {
				t.Errorf("title: got %q want %q", title, tc.wantTitle)
			}
			if body != tc.wantBody {
				t.Errorf("body: got %q want %q", body, tc.wantBody)
			}
		})
	}
}
