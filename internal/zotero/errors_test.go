package zotero

import (
	"fmt"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(ErrNotFound) {
		t.Error("IsNotFound(ErrNotFound) = false")
	}
	if !IsNotFound(fmt.Errorf("wrapped: %w", ErrNotFound)) {
		t.Error("IsNotFound(wrapped ErrNotFound) = false")
	}
	if !IsNotFound(&APIError{StatusCode: 404, Message: "not found"}) {
		t.Error("IsNotFound(APIError 404) = false")
	}
	if IsNotFound(ErrRateLimited) {
		t.Error("IsNotFound(ErrRateLimited) = true")
	}
}

func TestIsRateLimited(t *testing.T) {
	if !IsRateLimited(ErrRateLimited) {
		t.Error("IsRateLimited(ErrRateLimited) = false")
	}
	if !IsRateLimited(&APIError{StatusCode: 429, Message: "too many"}) {
		t.Error("IsRateLimited(APIError 429) = false")
	}
	if IsRateLimited(ErrNotFound) {
		t.Error("IsRateLimited(ErrNotFound) = true")
	}
}

func TestIsConflict(t *testing.T) {
	if !IsConflict(ErrConflict) {
		t.Error("IsConflict(ErrConflict) = false")
	}
	if !IsConflict(&APIError{StatusCode: 412, Message: "precondition"}) {
		t.Error("IsConflict(APIError 412) = false")
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 429, Message: "slow down", RetryAfter: 30}
	got := err.Error()
	if got != "Zotero API error (status 429): slow down (retry after 30s)" {
		t.Errorf("Error() = %q", got)
	}

	err2 := &APIError{StatusCode: 500, Message: "internal"}
	got2 := err2.Error()
	if got2 != "Zotero API error (status 500): internal" {
		t.Errorf("Error() = %q", got2)
	}
}
