package s2

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"sentinel", ErrNotFound, true},
		{"wrapped sentinel", fmt.Errorf("lookup: %w", ErrNotFound), true},
		{"APIError 404", &APIError{StatusCode: 404}, true},
		{"APIError 500", &APIError{StatusCode: 500}, false},
		{"rate limited", ErrRateLimited, false},
		{"unrelated", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"sentinel", ErrRateLimited, true},
		{"wrapped sentinel", fmt.Errorf("call: %w", ErrRateLimited), true},
		{"APIError 429", &APIError{StatusCode: 429}, true},
		{"APIError 404", &APIError{StatusCode: 404}, false},
		{"not found", ErrNotFound, false},
		{"unrelated", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimited(tt.err); got != tt.want {
				t.Errorf("IsRateLimited(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "without retry-after",
			err:  &APIError{StatusCode: 404, Message: "not found"},
			want: "S2 API error (status 404): not found",
		},
		{
			name: "with retry-after",
			err:  &APIError{StatusCode: 429, Message: "slow down", RetryAfter: 30},
			want: "S2 API error (status 429): slow down (retry after 30s)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
