package flow

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		// Valid formats
		{"2d", 2 * 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"12h", 12 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},

		// Invalid formats
		{"5m", 0, true},   // Unknown unit
		{"", 0, true},     // Empty
		{"d", 0, true},    // Too short (no number)
		{"abcd", 0, true}, // Non-numeric
		{"-5d", 0, true},  // Negative (via non-numeric parse)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatDateRange(t *testing.T) {
	tests := []struct {
		since    time.Time
		until    time.Time
		expected string
	}{
		{
			time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC),
			"Jan 12-18",
		},
		{
			time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			"Jan 25-Feb 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatDateRange(tt.since, tt.until)
			if got != tt.expected {
				t.Errorf("FormatDateRange() = %q, want %q", got, tt.expected)
			}
		})
	}
}
