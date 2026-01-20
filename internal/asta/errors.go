package asta

import (
	"errors"
	"fmt"
)

// Common errors returned by the ASTA client.
var (
	// ErrNotFound indicates the resource was not found.
	ErrNotFound = errors.New("not found in ASTA")

	// ErrAuthError indicates an authentication error (missing/invalid API key).
	ErrAuthError = errors.New("ASTA authentication error")

	// ErrRateLimited indicates the rate limit has been exceeded.
	ErrRateLimited = errors.New("ASTA rate limit exceeded")

	// ErrAPIError indicates a general API error.
	ErrAPIError = errors.New("ASTA API error")

	// ErrNetworkError indicates a network connectivity issue.
	ErrNetworkError = errors.New("network error communicating with ASTA")

	// ErrInvalidResponse indicates an unexpected API response.
	ErrInvalidResponse = errors.New("invalid response from ASTA")
)

// APIError represents an error from the ASTA MCP API.
type APIError struct {
	StatusCode int
	Code       string // Error code from API (e.g., "not_found", "auth_error")
	Message    string
	PaperID    string // For context in paper-related errors
}

func (e *APIError) Error() string {
	if e.PaperID != "" {
		return fmt.Sprintf("ASTA API error (status %d, code %s): %s (paper: %s)", e.StatusCode, e.Code, e.Message, e.PaperID)
	}
	return fmt.Sprintf("ASTA API error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
}

// IsNotFound returns true if the error indicates a resource was not found.
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404 || apiErr.Code == "not_found"
	}
	return false
}

// IsAuthError returns true if the error indicates an authentication problem.
func IsAuthError(err error) bool {
	if errors.Is(err, ErrAuthError) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403 || apiErr.Code == "auth_error"
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
		return apiErr.StatusCode == 429 || apiErr.Code == "rate_limited"
	}
	return false
}
