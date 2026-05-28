package flow

import "testing"

func TestSlugifyTitle(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain", "Fix the parser", "fix-the-parser"},
		{"uppercase only", "ABC", "abc"},
		{"punctuation runs", "Hello,  world!!!", "hello-world"},
		{"leading and trailing punctuation", "  !hello!  ", "hello"},
		{"underscore is non-alphanumeric", "snake_case_thing", "snake-case-thing"},
		{"unicode replaced", "café münch", "caf-m-nch"},
		{"emoji replaced", "ship it 🚀 today", "ship-it-today"},
		{"digits kept", "v2 update 3", "v2-update-3"},
		{"length cap", "this is a very long title that should be truncated at the configured slug limit", "this-is-a-very-long-title-that-should-be"},
		// "abc-def-ghi-jkl-mno-pqr-stu-vwx-yz1-234-" lands exactly at length 40
		// with a trailing dash; the trim removes it, leaving 39 chars.
		{"length cap trims trailing dash", "abc def ghi jkl mno pqr stu vwx yz1 234 567", "abc-def-ghi-jkl-mno-pqr-stu-vwx-yz1-234"},
		{"only punctuation collapses to empty", "!!! --- ???", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SlugifyTitle(c.in)
			if got != c.want {
				t.Errorf("SlugifyTitle(%q) = %q, want %q", c.in, got, c.want)
			}
			if len(got) > slugMaxLen {
				t.Errorf("slug exceeds max length: len=%d max=%d", len(got), slugMaxLen)
			}
		})
	}
}
