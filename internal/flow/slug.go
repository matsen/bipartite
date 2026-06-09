package flow

import (
	"regexp"
	"strings"
)

const slugMaxLen = 40

var slugMultiDash = regexp.MustCompile(`-+`)

// SlugifyTitle converts a free-form title (e.g. a GitHub issue title) into a
// short, filesystem-safe slug suitable for branch names and worktree slot
// templates.
//
// Rules: ASCII letters are lowercased; ASCII digits are kept; everything else
// (whitespace, punctuation, non-ASCII) becomes "-"; runs of "-" collapse to a
// single dash; leading and trailing dashes are trimmed; the result is
// truncated to slugMaxLen bytes — the output is ASCII, so bytes equal
// characters (with any trailing dash trimmed again).
// Empty input returns "".
func SlugifyTitle(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		default:
			b.WriteByte('-')
		}
	}
	out := slugMultiDash.ReplaceAllString(b.String(), "-")
	out = strings.Trim(out, "-")
	if len(out) > slugMaxLen {
		out = out[:slugMaxLen]
		out = strings.TrimRight(out, "-")
	}
	return out
}
