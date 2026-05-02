package asta

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads a JSON fixture file from testdata/.
func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return strings.TrimSpace(string(data))
}

func TestCombineStreamingResults(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := combineStreamingResults(nil)
		if !errors.Is(err, ErrInvalidResponse) {
			t.Fatalf("expected ErrInvalidResponse, got %v", err)
		}
	})

	t.Run("single chunk passthrough", func(t *testing.T) {
		chunk := loadFixture(t, "paper_single_chunk.json")
		got, err := combineStreamingResults([]string{chunk})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != chunk {
			t.Errorf("single chunk should pass through unchanged.\n got: %s\nwant: %s", got, chunk)
		}
	})

	t.Run("multi chunk wraps as result array", func(t *testing.T) {
		chunk := loadFixture(t, "paper_single_chunk.json")
		got, err := combineStreamingResults([]string{chunk, chunk})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"result":[` + chunk + `,` + chunk + `]}`
		if string(got) != want {
			t.Errorf("multi-chunk wrapping mismatch.\n got: %s\nwant: %s", got, want)
		}
	})
}

func TestParseSearchPapersResult(t *testing.T) {
	paperChunk := loadFixture(t, "paper_single_chunk.json")

	t.Run("wrapped multi-chunk array", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{paperChunk, paperChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseSearchPapersResult(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if resp.Total != 2 || len(resp.Papers) != 2 {
			t.Fatalf("expected 2 papers, got total=%d len=%d", resp.Total, len(resp.Papers))
		}
		if resp.Papers[0].PaperID != "abc123" {
			t.Errorf("expected paperId abc123, got %q", resp.Papers[0].PaperID)
		}
	})

	t.Run("bare array", func(t *testing.T) {
		raw := []byte(`[` + paperChunk + `]`)
		resp, err := parseSearchPapersResult(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if resp.Total != 1 || len(resp.Papers) != 1 {
			t.Fatalf("expected 1 paper, got total=%d len=%d", resp.Total, len(resp.Papers))
		}
	})

	t.Run("single chunk bare paper object (issue #134)", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{paperChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseSearchPapersResult(raw)
		if err != nil {
			t.Fatalf("parse should succeed for bare single object, got: %v", err)
		}
		if resp.Total != 1 || len(resp.Papers) != 1 {
			t.Fatalf("expected 1 paper, got total=%d len=%d", resp.Total, len(resp.Papers))
		}
		if resp.Papers[0].Title != "Single Paper" {
			t.Errorf("unexpected title: %q", resp.Papers[0].Title)
		}
	})

	t.Run("malformed object returns array-parse error", func(t *testing.T) {
		// Object with no "result" key, no paperId, and not a valid array.
		raw := []byte(`{"foo":"bar"}`)
		_, err := parseSearchPapersResult(raw)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse wrapping, got %v", err)
		}
		if !strings.Contains(err.Error(), "as array") {
			t.Errorf("expected array-parse error, got: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parseSearchPapersResult([]byte(`not json`))
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse, got %v", err)
		}
	})

	t.Run("empty wrapped result", func(t *testing.T) {
		resp, err := parseSearchPapersResult([]byte(`{"result":[]}`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if resp.Total != 0 || len(resp.Papers) != 0 {
			t.Errorf("expected empty result, got total=%d len=%d", resp.Total, len(resp.Papers))
		}
	})

	t.Run("bare null returns error", func(t *testing.T) {
		// JSON null parses cleanly into a nil slice. The parser must not silently
		// treat that as a successful empty list.
		_, err := parseSearchPapersResult([]byte(`null`))
		if err == nil {
			t.Fatal("expected error for bare null, got nil")
		}
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse, got %v", err)
		}
	})
}

func TestParseSearchAuthorsResult(t *testing.T) {
	authorChunk := loadFixture(t, "author_single_chunk.json")

	t.Run("wrapped multi-chunk array", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{authorChunk, authorChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseSearchAuthorsResult(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if len(resp.Authors) != 2 {
			t.Fatalf("expected 2 authors, got %d", len(resp.Authors))
		}
		if resp.Authors[0].Name != "Jane Smith" {
			t.Errorf("unexpected name: %q", resp.Authors[0].Name)
		}
	})

	t.Run("single chunk bare author object (issue #134)", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{authorChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseSearchAuthorsResult(raw)
		if err != nil {
			t.Fatalf("parse should succeed for bare single author, got: %v", err)
		}
		if len(resp.Authors) != 1 {
			t.Fatalf("expected 1 author, got %d", len(resp.Authors))
		}
		if resp.Authors[0].AuthorID != "a1" {
			t.Errorf("unexpected authorId: %q", resp.Authors[0].AuthorID)
		}
	})

	t.Run("bare array", func(t *testing.T) {
		raw := []byte(`[` + authorChunk + `]`)
		resp, err := parseSearchAuthorsResult(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if len(resp.Authors) != 1 {
			t.Fatalf("expected 1 author, got %d", len(resp.Authors))
		}
	})

	t.Run("malformed object returns array-parse error", func(t *testing.T) {
		raw := []byte(`{"foo":"bar"}`)
		_, err := parseSearchAuthorsResult(raw)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse wrapping, got %v", err)
		}
		if !strings.Contains(err.Error(), "as array") {
			t.Errorf("expected array-parse error, got: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parseSearchAuthorsResult([]byte(`not json`))
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse, got %v", err)
		}
	})
}

func TestParseAuthorPapersResult(t *testing.T) {
	paperChunk := loadFixture(t, "paper_single_chunk.json")
	const authorID = "author-xyz"

	t.Run("wrapped multi-chunk array", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{paperChunk, paperChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseAuthorPapersResult(raw, authorID)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if resp.AuthorID != authorID {
			t.Errorf("authorID not propagated: got %q", resp.AuthorID)
		}
		if len(resp.Papers) != 2 {
			t.Fatalf("expected 2 papers, got %d", len(resp.Papers))
		}
	})

	t.Run("single chunk bare paper object (issue #134)", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{paperChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseAuthorPapersResult(raw, authorID)
		if err != nil {
			t.Fatalf("parse should succeed for bare single paper, got: %v", err)
		}
		if len(resp.Papers) != 1 {
			t.Fatalf("expected 1 paper, got %d", len(resp.Papers))
		}
		if resp.AuthorID != authorID {
			t.Errorf("authorID not propagated: got %q", resp.AuthorID)
		}
	})

	t.Run("bare array", func(t *testing.T) {
		raw := []byte(`[` + paperChunk + `]`)
		resp, err := parseAuthorPapersResult(raw, authorID)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if len(resp.Papers) != 1 {
			t.Fatalf("expected 1 paper, got %d", len(resp.Papers))
		}
		if resp.AuthorID != authorID {
			t.Errorf("authorID not propagated: got %q", resp.AuthorID)
		}
	})

	t.Run("malformed object returns array-parse error", func(t *testing.T) {
		raw := []byte(`{"foo":"bar"}`)
		_, err := parseAuthorPapersResult(raw, authorID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "as array") {
			t.Errorf("expected array-parse error, got: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parseAuthorPapersResult([]byte(`not json`), authorID)
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse, got %v", err)
		}
	})
}

func TestParseCitationsResult(t *testing.T) {
	citationChunk := loadFixture(t, "citation_single_chunk.json")
	const paperID = "cited-paper"

	t.Run("wrapped multi-chunk array", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{citationChunk, citationChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseCitationsResult(raw, paperID)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if resp.PaperID != paperID {
			t.Errorf("paperID not propagated: got %q", resp.PaperID)
		}
		if resp.CitationCount != 2 || len(resp.Citations) != 2 {
			t.Fatalf("expected 2 citations, got count=%d len=%d", resp.CitationCount, len(resp.Citations))
		}
		if resp.Citations[0].PaperID != "cite-1" {
			t.Errorf("expected citing paperId cite-1, got %q", resp.Citations[0].PaperID)
		}
	})

	t.Run("single chunk bare citation object (issue #134)", func(t *testing.T) {
		raw, err := combineStreamingResults([]string{citationChunk})
		if err != nil {
			t.Fatalf("combine: %v", err)
		}
		resp, err := parseCitationsResult(raw, paperID)
		if err != nil {
			t.Fatalf("parse should succeed for bare single citation, got: %v", err)
		}
		if resp.CitationCount != 1 || len(resp.Citations) != 1 {
			t.Fatalf("expected 1 citation, got count=%d len=%d", resp.CitationCount, len(resp.Citations))
		}
		if resp.Citations[0].Title != "A Citing Paper" {
			t.Errorf("unexpected citing paper title: %q", resp.Citations[0].Title)
		}
	})

	t.Run("bare array", func(t *testing.T) {
		raw := []byte(`[` + citationChunk + `]`)
		resp, err := parseCitationsResult(raw, paperID)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if resp.CitationCount != 1 || len(resp.Citations) != 1 {
			t.Fatalf("expected 1 citation, got count=%d len=%d", resp.CitationCount, len(resp.Citations))
		}
		if resp.PaperID != paperID {
			t.Errorf("paperID not propagated: got %q", resp.PaperID)
		}
	})

	t.Run("malformed object returns array-parse error", func(t *testing.T) {
		// Object that has neither "result" nor "citingPaper" with a paperId.
		raw := []byte(`{"foo":"bar"}`)
		_, err := parseCitationsResult(raw, paperID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "as array") {
			t.Errorf("expected array-parse error, got: %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := parseCitationsResult([]byte(`not json`), paperID)
		if !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("expected ErrInvalidResponse, got %v", err)
		}
	})
}
