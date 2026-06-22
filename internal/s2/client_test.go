package s2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// makeResponse builds a minimal *http.Response for checkResponse tests.
func makeResponse(status int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     header,
	}
}

func TestCheckResponse(t *testing.T) {
	t.Run("2xx is nil", func(t *testing.T) {
		if err := checkResponse(makeResponse(200, "ok", nil)); err != nil {
			t.Errorf("checkResponse(200) = %v, want nil", err)
		}
		if err := checkResponse(makeResponse(204, "", nil)); err != nil {
			t.Errorf("checkResponse(204) = %v, want nil", err)
		}
	})

	t.Run("404 is APIError", func(t *testing.T) {
		err := checkResponse(makeResponse(404, "missing", nil))
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("checkResponse(404) = %v, want *APIError", err)
		}
		if apiErr.StatusCode != 404 {
			t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
		}
		if apiErr.Message != "missing" {
			t.Errorf("Message = %q, want %q", apiErr.Message, "missing")
		}
	})

	t.Run("429 wraps ErrRateLimited with Retry-After", func(t *testing.T) {
		h := http.Header{}
		h.Set("Retry-After", "42")
		err := checkResponse(makeResponse(429, "slow down", h))
		if !IsRateLimited(err) {
			t.Fatalf("checkResponse(429): IsRateLimited = false, want true (err=%v)", err)
		}
		// The 429 path wraps ErrRateLimited (with %w) and embeds the
		// formatted *APIError (with %v), so the parsed Retry-After surfaces
		// in the message rather than as a retrievable *APIError.
		if !strings.Contains(err.Error(), "retry after 42s") {
			t.Errorf("error message = %q, want it to mention retry after 42s", err.Error())
		}
	})

	t.Run("empty body falls back to status", func(t *testing.T) {
		err := checkResponse(makeResponse(500, "", nil))
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("checkResponse(500) = %v, want *APIError", err)
		}
		if apiErr.Message == "" {
			t.Error("Message should fall back to status text, got empty")
		}
	})
}

func TestGetPaperHappyPath(t *testing.T) {
	const body = `{
		"paperId": "abc123",
		"title": "A Great Paper",
		"year": 2018,
		"venue": "Nature",
		"externalIds": {"DOI": "10.1038/nature12373"},
		"authors": [{"name": "Cheng Zhang"}]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/paper/") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("fields") == "" {
			t.Error("fields query param missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	paper, err := c.GetPaper(context.Background(), "DOI:10.1038/nature12373")
	if err != nil {
		t.Fatalf("GetPaper: %v", err)
	}
	if paper.PaperID != "abc123" {
		t.Errorf("PaperID = %q, want abc123", paper.PaperID)
	}
	if paper.Title != "A Great Paper" {
		t.Errorf("Title = %q", paper.Title)
	}
	if paper.ExternalIDs.DOI != "10.1038/nature12373" {
		t.Errorf("DOI = %q", paper.ExternalIDs.DOI)
	}
	if len(paper.Authors) != 1 || paper.Authors[0].Name != "Cheng Zhang" {
		t.Errorf("Authors = %+v", paper.Authors)
	}
}

func TestGetPaperNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	_, err := c.GetPaper(context.Background(), "DOI:10.0/missing")
	if !IsNotFound(err) {
		t.Errorf("GetPaper: IsNotFound = false, want true (err=%v)", err)
	}
}
