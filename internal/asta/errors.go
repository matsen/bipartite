package asta

import (
	"errors"
	"fmt"
)

// Common errors returned by the ASTA client.
var (
	// ErrNotFound indicates the paper was not found in Semantic Scholar.
	ErrNotFound = errors.New("paper not found in Semantic Scholar")

	// ErrRateLimited indicates the rate limit has been exceeded.
	ErrRateLimited = errors.New("Semantic Scholar rate limit exceeded")

	// ErrNetworkError indicates a network connectivity issue.
	ErrNetworkError = errors.New("network error communicating with Semantic Scholar")

	// ErrInvalidResponse indicates an unexpected API response.
	ErrInvalidResponse = errors.New("invalid response from Semantic Scholar")
)

// APIError represents an error from the Semantic Scholar API.
type APIError struct {
	StatusCode int
	Message    string
	RetryAfter int // Seconds to wait before retrying (for rate limits)
}

func (e *APIError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("S2 API error (status %d): %s (retry after %ds)", e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("S2 API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error indicates a paper was not found.
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

// IsRateLimited returns true if the error indicates rate limiting.
func IsRateLimited(err error) bool {
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 429
	}
	return false
}
