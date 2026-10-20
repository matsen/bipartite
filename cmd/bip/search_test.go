package main

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestParseYearRange(t *testing.T) {
	tests := []struct {
		spec     string
		wantFrom int
		wantTo   int
		wantErr  bool
	}{
		// Exact year
		{"2024", 2024, 2024, false},
		{"2020", 2020, 2020, false},

		// Full range
		{"2020:2024", 2020, 2024, false},
		{"2020:2020", 2020, 2020, false},

		// Open-ended ranges
		{"2020:", 2020, 0, false},
		{":2024", 0, 2024, false},

		// Edge cases
		{"", 0, 0, false},
		{"  2024  ", 2024, 2024, false}, // Whitespace trimmed
		{" 2020:2024 ", 2020, 2024, false},

		// Errors
		{"abc", 0, 0, true},
		{"20:24", 20, 24, false}, // Not an error, just weird input
		{"abc:2024", 0, 0, true},
		{"2020:abc", 0, 0, true},
		{":", 0, 0, false}, // Both empty - no filter
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			from, to, err := parseYearRange(tt.spec)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseYearRange(%q) error = %v, wantErr %v", tt.spec, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if from != tt.wantFrom {
					t.Errorf("parseYearRange(%q) from = %d, want %d", tt.spec, from, tt.wantFrom)
				}
				if to != tt.wantTo {
					t.Errorf("parseYearRange(%q) to = %d, want %d", tt.spec, to, tt.wantTo)
				}
			}
		})
	}
}

func TestSortByDateDesc(t *testing.T) {
	refs := []reference.Reference{
		{ID: "old", Published: reference.PublicationDate{Year: 2020, Month: 5}},
		{ID: "newMonth", Published: reference.PublicationDate{Year: 2026, Month: 9}},
		{ID: "noMonth", Published: reference.PublicationDate{Year: 2026}},
		{ID: "newDay", Published: reference.PublicationDate{Year: 2026, Month: 9, Day: 12}},
		{ID: "mid", Published: reference.PublicationDate{Year: 2026, Month: 1}},
	}
	sortByDateDesc(refs)
	want := []string{"newDay", "newMonth", "mid", "noMonth", "old"}
	for i, id := range want {
		if refs[i].ID != id {
			t.Fatalf("position %d: got %s, want %s (order %v)", i, refs[i].ID, id, refs)
		}
	}
}
