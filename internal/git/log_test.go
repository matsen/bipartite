package git

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestParseGitLogOneline(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []CommitInfo
	}{
		{
			name: "two commits",
			data: "abc123 Add paper X\ndef456 Remove paper Y\n",
			want: []CommitInfo{
				{SHA: "abc123", Message: "Add paper X"},
				{SHA: "def456", Message: "Remove paper Y"},
			},
		},
		{
			name: "blank lines skipped",
			data: "abc123 first\n\n\ndef456 second\n",
			want: []CommitInfo{
				{SHA: "abc123", Message: "first"},
				{SHA: "def456", Message: "second"},
			},
		},
		{
			name: "sha only, no message",
			data: "abc123\n",
			want: []CommitInfo{{SHA: "abc123", Message: ""}},
		},
		{
			name: "message with spaces preserved",
			data: "abc123 fix: handle multi word subject lines\n",
			want: []CommitInfo{{SHA: "abc123", Message: "fix: handle multi word subject lines"}},
		},
		{
			name: "empty input",
			data: "",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGitLogOneline([]byte(tt.data))
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestShortSHA(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"abcdef0123456789", "abcdef01"},
		{"abcdef01", "abcdef01"},
		{"abc", "abc"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := shortSHA(tt.in); got != tt.want {
				t.Errorf("shortSHA(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSortRefsAlphabetically(t *testing.T) {
	refs := []reference.Reference{
		{ID: "Zhang2018-vi"},
		{ID: "Adams2001-os"},
		{ID: "Madison1999-nn"},
	}
	SortRefsAlphabetically(refs)
	want := []string{"Adams2001-os", "Madison1999-nn", "Zhang2018-vi"}
	for i, w := range want {
		if refs[i].ID != w {
			t.Errorf("[%d] = %q, want %q", i, refs[i].ID, w)
		}
	}
}
