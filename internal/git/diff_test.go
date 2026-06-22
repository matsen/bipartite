package git

import (
	"sort"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func ids(refs []reference.Reference) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.ID
	}
	sort.Strings(out)
	return out
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDiffRefs(t *testing.T) {
	ref := func(id string) reference.Reference { return reference.Reference{ID: id} }

	tests := []struct {
		name        string
		old         []reference.Reference
		current     []reference.Reference
		wantAdded   []string
		wantRemoved []string
	}{
		{
			name:        "added only",
			old:         []reference.Reference{ref("a")},
			current:     []reference.Reference{ref("a"), ref("b"), ref("c")},
			wantAdded:   []string{"b", "c"},
			wantRemoved: nil,
		},
		{
			name:        "removed only",
			old:         []reference.Reference{ref("a"), ref("b"), ref("c")},
			current:     []reference.Reference{ref("a")},
			wantAdded:   nil,
			wantRemoved: []string{"b", "c"},
		},
		{
			name:        "unchanged",
			old:         []reference.Reference{ref("a"), ref("b")},
			current:     []reference.Reference{ref("a"), ref("b")},
			wantAdded:   nil,
			wantRemoved: nil,
		},
		{
			name:        "added and removed",
			old:         []reference.Reference{ref("a"), ref("b")},
			current:     []reference.Reference{ref("b"), ref("c")},
			wantAdded:   []string{"c"},
			wantRemoved: []string{"a"},
		},
		{
			name:        "both empty",
			old:         nil,
			current:     nil,
			wantAdded:   nil,
			wantRemoved: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffRefs(tt.old, tt.current)
			if gotAdded := ids(diff.Added); !equalIDs(gotAdded, tt.wantAdded) {
				t.Errorf("Added = %v, want %v", gotAdded, tt.wantAdded)
			}
			if gotRemoved := ids(diff.Removed); !equalIDs(gotRemoved, tt.wantRemoved) {
				t.Errorf("Removed = %v, want %v", gotRemoved, tt.wantRemoved)
			}
		})
	}
}
