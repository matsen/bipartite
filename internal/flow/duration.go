package flow

import (
	"errors"
	"fmt"
	"time"
)

// Duration parsing errors.
var (
	ErrInvalidDuration = errors.New("invalid duration format")
	ErrUnknownUnit     = errors.New("unknown duration unit")
)

// ParseDuration parses a duration string like "2d", "12h", "1w".
// Supported units: d (days), h (hours), w (weeks).
func ParseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, ErrInvalidDuration
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	// Parse the numeric value
	value, err := parsePositiveInt(valueStr)
	if err != nil {
		return 0, ErrInvalidDuration
	}

	switch unit {
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("%w: %c", ErrUnknownUnit, unit)
	}
}

// FormatDateRange formats a date range for display (e.g., "Jan 12-18").
func FormatDateRange(since, until time.Time) string {
	if since.Month() == until.Month() {
		return fmt.Sprintf("%s %d-%d", since.Format("Jan"), since.Day(), until.Day())
	}
	return fmt.Sprintf("%s-%s", since.Format("Jan 2"), until.Format("Jan 2"))
}
