package main

import "testing"

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
