package importer

import (
	"reflect"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Hello World", "hello-world"},
		{"server_url", "server_url"},
		{"What's new? (v2.0)", "whats-new-v20"},
		{"a -- b", "a-b"},
		{"a - - b", "a-b"},
		{"  Trim me  ", "trim-me"},
		{"Café Ünïcode 日本語", "café-ünïcode-日本語"},
		{"tab\tand nbsp", "tab-and-nbsp"},
		{"2.1 `server_url` and `websocket_urls` are different addresses",
			"21-server_url-and-websocket_urls-are-different-addresses"},
	}
	for _, tc := range cases {
		if got := slugify(tc.in); got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHeadingText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2.1 `server_url` and `websocket_urls` are different addresses",
			"2.1 server_url and websocket_urls are different addresses"},
		{"**Bold** and *em*", "**Bold** and *em*"}, // slugify drops '*'
		{"_emphasis_ and snake_case", "emphasis and snake_case"},
		{"See [the guide](./guide.md#x)", "See the guide"},
		{"Logo ![logo](logo.png) here", "Logo  here"},
		{"Escaped \\_name\\_ &amp; more", "Escaped _name_ & more"},
		{"``a `tick` b``", "a `tick` b"},
		{"unclosed `tick", "unclosed `tick"},
	}
	for _, tc := range cases {
		if got := headingText(tc.in); got != tc.want {
			t.Errorf("headingText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHeadingIDs(t *testing.T) {
	body := "Intro\n\n" +
		"## Setup\n" +
		"### Setup\n" +
		"## Setup ##\n" +
		"```bash\n# not a heading\n```\n" +
		"~~~~\n```\n# still code\n~~~~\n" +
		"   # Indented three\n" +
		"    # Indented four is code\n" +
		"#NoSpace\n" +
		"## C#\r\n"
	want := map[string]bool{
		"setup": true, "setup-1": true, "setup-2": true,
		"indented-three": true, "c": true,
	}
	if got := headingIDs(body); !reflect.DeepEqual(got, want) {
		t.Errorf("headingIDs = %v, want %v", got, want)
	}
}
