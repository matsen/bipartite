package ncbi

import (
	"errors"
	"fmt"
	"strings"
)

// Common errors returned by the NCBI client.
var (
	// ErrRateLimited indicates the NCBI rate limit (3 req/sec without API key,
	// 10 with) has been exceeded.
	ErrRateLimited = errors.New("NCBI rate limit exceeded")

	// ErrNetworkError indicates a network connectivity issue.
	ErrNetworkError = errors.New("network error communicating with NCBI")

	// ErrInvalidResponse indicates a malformed or unexpected response shape.
	ErrInvalidResponse = errors.New("invalid response from NCBI")
)

// APIError represents a request-wide error returned by the NCBI ID Converter
// (HTTP status 4xx/5xx, or a JSON envelope with `status: "error"`).
//
// BatchIDs records the IDs that were in the failing batch so callers can
// report or retry them. Per the issue's Test plan, HTTP errors must include
// the failing batch's IDs in their message.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	BatchIDs   []string
}

func (e *APIError) Error() string {
	if len(e.BatchIDs) > 0 {
		return fmt.Sprintf("NCBI API error (status %d, code %s): %s (batch: %s)",
			e.StatusCode, e.Code, e.Message, strings.Join(e.BatchIDs, ","))
	}
	return fmt.Sprintf("NCBI API error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
}

// IsRateLimited reports whether the error indicates rate limiting.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}
