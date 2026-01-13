package export

import (
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestToBibTeX_BasicArticle(t *testing.T) {
	ref := reference.Reference{
		ID:    "Smith2026-ab",
		DOI:   "10.1234/test",
		Title: "Test Paper Title",
		Authors: []reference.Author{
			{First: "John", Last: "Smith"},
			{First: "Jane", Last: "Doe"},
		},
		Abstract:  "This is the abstract",
		Venue:     "Nature",
		Published: reference.PublicationDate{Year: 2026, Month: 3},
	}

	got := ToBibTeX(ref)

	// Check entry type and key
	if !strings.HasPrefix(got, "@article{Smith2026-ab,") {
		t.Errorf("ToBibTeX() should start with @article{Smith2026-ab, got:\n%s", got)
	}

	// Check author format
	if !strings.Contains(got, `author = {Smith, John and Doe, Jane}`) {
		t.Errorf("ToBibTeX() should contain properly formatted authors, got:\n%s", got)
	}

	// Check title
	if !strings.Contains(got, `title = {Test Paper Title}`) {
		t.Errorf("ToBibTeX() should contain title, got:\n%s", got)
	}

	// Check journal
	if !strings.Contains(got, `journal = {Nature}`) {
		t.Errorf("ToBibTeX() should contain journal, got:\n%s", got)
	}

	// Check year
	if !strings.Contains(got, `year = {2026}`) {
		t.Errorf("ToBibTeX() should contain year, got:\n%s", got)
	}

	// Check month
	if !strings.Contains(got, `month = {3}`) {
		t.Errorf("ToBibTeX() should contain month, got:\n%s", got)
	}

	// Check DOI
	if !strings.Contains(got, `doi = {10.1234/test}`) {
		t.Errorf("ToBibTeX() should contain DOI, got:\n%s", got)
	}

	// Check abstract
	if !strings.Contains(got, `abstract = {This is the abstract}`) {
		t.Errorf("ToBibTeX() should contain abstract, got:\n%s", got)
	}

	// Check closing brace
	if !strings.HasSuffix(strings.TrimSpace(got), "}") {
		t.Errorf("ToBibTeX() should end with }, got:\n%s", got)
	}
}

func TestToBibTeX_Inproceedings(t *testing.T) {
	ref := reference.Reference{
		ID:    "Conference2026",
		Title: "A Conference Paper",
		Authors: []reference.Author{
			{First: "Alice", Last: "Brown"},
		},
		Venue:     "Proceedings of ICML 2026",
		Published: reference.PublicationDate{Year: 2026},
	}

	got := ToBibTeX(ref)

	if !strings.HasPrefix(got, "@inproceedings{Conference2026,") {
		t.Errorf("ToBibTeX() conference paper should be @inproceedings, got:\n%s", got)
	}

	if !strings.Contains(got, `booktitle = {Proceedings of ICML 2026}`) {
		t.Errorf("ToBibTeX() conference paper should use booktitle, got:\n%s", got)
	}
}

func TestDetermineEntryType(t *testing.T) {
	tests := []struct {
		venue string
		want  string
	}{
		{"Nature", "article"},
		{"Science", "article"},
		{"bioRxiv", "article"},
		{"arXiv", "article"},
		{"medRxiv", "article"},
		{"Proceedings of NeurIPS", "inproceedings"},
		{"International Conference on Machine Learning", "inproceedings"},
		{"Workshop on AI Safety", "inproceedings"},
		{"Symposium on Theory of Computing", "inproceedings"},
		{"", "article"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.venue, func(t *testing.T) {
			ref := reference.Reference{Venue: tt.venue}
			got := determineEntryType(ref)
			if got != tt.want {
				t.Errorf("determineEntryType(%q) = %q, want %q", tt.venue, got, tt.want)
			}
		})
	}
}

func TestFormatAuthors(t *testing.T) {
	tests := []struct {
		name    string
		authors []reference.Author
		want    string
	}{
		{
			name: "single author",
			authors: []reference.Author{
				{First: "John", Last: "Smith"},
			},
			want: "Smith, John",
		},
		{
			name: "two authors",
			authors: []reference.Author{
				{First: "John", Last: "Smith"},
				{First: "Jane", Last: "Doe"},
			},
			want: "Smith, John and Doe, Jane",
		},
		{
			name: "three authors",
			authors: []reference.Author{
				{First: "Alice", Last: "Brown"},
				{First: "Bob", Last: "Jones"},
				{First: "Carol", Last: "White"},
			},
			want: "Brown, Alice and Jones, Bob and White, Carol",
		},
		{
			name: "author with only last name",
			authors: []reference.Author{
				{Last: "Corporation"},
			},
			want: "Corporation",
		},
		{
			name: "mixed authors",
			authors: []reference.Author{
				{First: "John", Last: "Smith"},
				{Last: "WHO"},
			},
			want: "Smith, John and WHO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAuthors(tt.authors)
			if got != tt.want {
				t.Errorf("formatAuthors() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEscapeLatex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plain text", "plain text"},
		{"100% effective", `100\% effective`},
		{"A & B", `A \& B`},
		{"$100 price", `\$100 price`},
		{"section #1", `section \#1`},
		{"under_score", `under\_score`},
		{"{braces}", `\{braces\}`},
		{"test~tilde", `test\textasciitilde{}tilde`},
		{"x^2", `x\textasciicircum{}2`},
		{"A & B: $100 for {item} #1", `A \& B: \$100 for \{item\} \#1`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeLatex(tt.input)
			if got != tt.want {
				t.Errorf("escapeLatex(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToBibTeX_OptionalFields(t *testing.T) {
	// Test with minimal fields
	ref := reference.Reference{
		ID:    "Minimal2026",
		Title: "Minimal Paper",
		Authors: []reference.Author{
			{First: "A", Last: "B"},
		},
		Published: reference.PublicationDate{Year: 2026},
	}

	got := ToBibTeX(ref)

	// Should NOT contain optional fields
	if strings.Contains(got, "doi = ") {
		t.Errorf("ToBibTeX() should not include empty DOI, got:\n%s", got)
	}
	if strings.Contains(got, "abstract = ") {
		t.Errorf("ToBibTeX() should not include empty abstract, got:\n%s", got)
	}
	if strings.Contains(got, "month = ") {
		t.Errorf("ToBibTeX() should not include zero month, got:\n%s", got)
	}
	if strings.Contains(got, "journal = ") && strings.Contains(got, "booktitle = ") {
		t.Errorf("ToBibTeX() should not include empty venue, got:\n%s", got)
	}
}

func TestToBibTeX_SpecialCharactersInTitle(t *testing.T) {
	ref := reference.Reference{
		ID:    "Special2026",
		Title: "A Study of α & β: 100% Complete",
		Authors: []reference.Author{
			{First: "Test", Last: "Author"},
		},
		Published: reference.PublicationDate{Year: 2026},
	}

	got := ToBibTeX(ref)

	// Title should have special chars escaped
	if !strings.Contains(got, `title = {A Study of α \& β: 100\% Complete}`) {
		t.Errorf("ToBibTeX() should escape special chars in title, got:\n%s", got)
	}
}

func TestToBibTeXList(t *testing.T) {
	refs := []reference.Reference{
		{
			ID:        "First2026",
			Title:     "First Paper",
			Authors:   []reference.Author{{First: "A", Last: "B"}},
			Published: reference.PublicationDate{Year: 2026},
		},
		{
			ID:        "Second2026",
			Title:     "Second Paper",
			Authors:   []reference.Author{{First: "C", Last: "D"}},
			Published: reference.PublicationDate{Year: 2025},
		},
	}

	got := ToBibTeXList(refs)

	// Should contain both entries
	if !strings.Contains(got, "@article{First2026,") {
		t.Errorf("ToBibTeXList() should contain first entry, got:\n%s", got)
	}
	if !strings.Contains(got, "@article{Second2026,") {
		t.Errorf("ToBibTeXList() should contain second entry, got:\n%s", got)
	}

	// Entries should be separated by newline
	parts := strings.Split(got, "@article{")
	if len(parts) != 3 { // Empty first part + 2 entries
		t.Errorf("ToBibTeXList() should have 2 entries separated properly, got %d parts", len(parts)-1)
	}
}

func TestToBibTeXList_Empty(t *testing.T) {
	got := ToBibTeXList([]reference.Reference{})
	if got != "" {
		t.Errorf("ToBibTeXList([]) should return empty string, got: %q", got)
	}
}

func TestToBibTeX_NoAuthors(t *testing.T) {
	ref := reference.Reference{
		ID:        "NoAuth2026",
		Title:     "Paper Without Authors",
		Authors:   []reference.Author{},
		Published: reference.PublicationDate{Year: 2026},
	}

	got := ToBibTeX(ref)

	// Should not include author field if empty
	if strings.Contains(got, "author = ") {
		t.Errorf("ToBibTeX() should not include empty authors, got:\n%s", got)
	}

	// But should still have title and year
	if !strings.Contains(got, "title = ") {
		t.Errorf("ToBibTeX() should still include title, got:\n%s", got)
	}
	if !strings.Contains(got, "year = ") {
		t.Errorf("ToBibTeX() should still include year, got:\n%s", got)
	}
}
