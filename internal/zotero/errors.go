package zotero

import (
	"errors"
	"fmt"
)

// Common errors returned by the Zotero client.
var (
	// ErrNotFound indicates the item was not found.
	ErrNotFound = errors.New("item not found in Zotero")

	// ErrRateLimited indicates the rate limit has been exceeded.
	ErrRateLimited = errors.New("Zotero API rate limit exceeded")

	// ErrNetworkError indicates a network connectivity issue.
	ErrNetworkError = errors.New("network error communicating with Zotero API")

	// ErrConflict indicates a version conflict (412 Precondition Failed).
	ErrConflict = errors.New("version conflict: library was modified since last sync")

	// ErrForbidden indicates an authentication or permission error.
	ErrForbidden = errors.New("Zotero API key invalid or insufficient permissions")

	// ErrNotConfigured indicates missing API credentials.
	ErrNotConfigured = errors.New("Zotero API key or user ID not configured")
)

// APIError represents an error from the Zotero API.
type APIError struct {
	StatusCode int
	Message    string
	RetryAfter int // Seconds to wait before retrying (for rate limits)
}

func (e *APIError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("Zotero API error (status %d): %s (retry after %ds)", e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("Zotero API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error indicates an item was not found.
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

// IsConflict returns true if the error indicates a version conflict.
func IsConflict(err error) bool {
	if errors.Is(err, ErrConflict) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 412
	}
	return false
}
