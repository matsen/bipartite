package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestFlexibleString_String(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"string year", `"2026"`, "2026"},
		{"number year", `2026`, "2026"},
		{"null value", `null`, ""},
		{"float number", `2026.0`, "2026.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FlexibleString
			if err := json.Unmarshal([]byte(tt.input), &f); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got := f.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlexibleString_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"array", `[1,2,3]`},
		{"object", `{"key": "value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FlexibleString
			if err := json.Unmarshal([]byte(tt.input), &f); err == nil {
				t.Errorf("UnmarshalJSON() expected error for input %s", tt.input)
			}
		})
	}
}

func TestParsePaperpile_ValidEntry(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "Smith2026-ab",
		"doi": "10.1234/test",
		"title": "Test Paper",
		"abstract": "This is a test abstract",
		"journal": "Test Journal",
		"note": "SONIA (linear)",
		"published": {"year": "2026", "month": "3", "day": "15"},
		"author": [
			{"first": "John", "last": "Smith", "orcid": "0000-0001-2345-6789"},
			{"first": "Jane", "last": "Doe"}
		],
		"attachments": [
			{"_id": "att1", "article_pdf": 1, "filename": "Papers/main.pdf"},
			{"_id": "att2", "article_pdf": 0, "filename": "Papers/supplement.pdf"}
		]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("ParsePaperpile() returned %d refs, want 1", len(refs))
	}

	ref := refs[0]

	// Check identity fields
	if ref.ID != "Smith2026-ab" {
		t.Errorf("ID = %v, want Smith2026-ab", ref.ID)
	}
	if ref.DOI != "10.1234/test" {
		t.Errorf("DOI = %v, want 10.1234/test", ref.DOI)
	}

	// Check metadata
	if ref.Title != "Test Paper" {
		t.Errorf("Title = %v, want Test Paper", ref.Title)
	}
	if ref.Abstract != "This is a test abstract" {
		t.Errorf("Abstract = %v, want This is a test abstract", ref.Abstract)
	}
	if ref.Venue != "Test Journal" {
		t.Errorf("Venue = %v, want Test Journal", ref.Venue)
	}

	// Check authors
	if len(ref.Authors) != 2 {
		t.Fatalf("Authors count = %d, want 2", len(ref.Authors))
	}
	if ref.Authors[0].First != "John" || ref.Authors[0].Last != "Smith" {
		t.Errorf("Authors[0] = %+v, want John Smith", ref.Authors[0])
	}
	if ref.Authors[0].ORCID != "0000-0001-2345-6789" {
		t.Errorf("Authors[0].ORCID = %v, want 0000-0001-2345-6789", ref.Authors[0].ORCID)
	}

	// Check publication date
	if ref.Published.Year != 2026 {
		t.Errorf("Published.Year = %d, want 2026", ref.Published.Year)
	}
	if ref.Published.Month != 3 {
		t.Errorf("Published.Month = %d, want 3", ref.Published.Month)
	}
	if ref.Published.Day != 15 {
		t.Errorf("Published.Day = %d, want 15", ref.Published.Day)
	}

	// Check PDF paths
	if ref.PDFPath != "Papers/main.pdf" {
		t.Errorf("PDFPath = %v, want Papers/main.pdf", ref.PDFPath)
	}
	if len(ref.SupplementPaths) != 1 || ref.SupplementPaths[0] != "Papers/supplement.pdf" {
		t.Errorf("SupplementPaths = %v, want [Papers/supplement.pdf]", ref.SupplementPaths)
	}

	// Check notes
	if ref.Note != "SONIA (linear)" {
		t.Errorf("Notes = %v, want SONIA (linear)", ref.Note)
	}

	// Check import source
	if ref.Source.Type != "paperpile" {
		t.Errorf("Source.Type = %v, want paperpile", ref.Source.Type)
	}
	if ref.Source.ID != "abc123" {
		t.Errorf("Source.ID = %v, want abc123", ref.Source.ID)
	}
}

func TestParsePaperpile_Venue(t *testing.T) {
	// Conference papers carry the venue in booktitle (empty journal); articles
	// carry it in journal. Journal wins when both are present.
	tests := []struct {
		name      string
		data      string
		wantVenue string
	}{
		{
			name: "conference paper uses booktitle",
			data: `[{
				"_id": "abc", "citekey": "Prillo2024-bn", "title": "Conf paper",
				"published": {"year": "2024"}, "author": [{"last": "Prillo"}],
				"booktitle": "The Thirty-eighth Annual Conference on Neural Information Processing Systems"
			}]`,
			wantVenue: "The Thirty-eighth Annual Conference on Neural Information Processing Systems",
		},
		{
			name: "journal wins when both present",
			data: `[{
				"_id": "abc", "citekey": "Both2024", "title": "Paper",
				"published": {"year": "2024"}, "author": [{"last": "Smith"}],
				"journal": "Real Journal", "booktitle": "Some Proceedings"
			}]`,
			wantVenue: "Real Journal",
		},
		{
			name: "neither present yields empty venue",
			data: `[{
				"_id": "abc", "citekey": "None2024", "title": "Paper",
				"published": {"year": "2024"}, "author": [{"last": "Smith"}]
			}]`,
			wantVenue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, _, errs := ParsePaperpile([]byte(tt.data), true)
			if len(errs) > 0 {
				t.Fatalf("ParsePaperpile() returned errors: %v", errs)
			}
			if len(refs) != 1 {
				t.Fatalf("got %d refs, want 1", len(refs))
			}
			if refs[0].Venue != tt.wantVenue {
				t.Errorf("Venue = %q, want %q", refs[0].Venue, tt.wantVenue)
			}
		})
	}
}

func TestParsePaperpile_WithoutNotes(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "NoNotes2026",
		"title": "Paper without notes",
		"published": {"year": "2026"},
		"author": [{"first": "John", "last": "Smith"}]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if refs[0].Note != "" {
		t.Errorf("Notes = %v, want empty string", refs[0].Note)
	}
}

func TestParsePaperpile_NoCitekey(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"title": "Test Paper",
		"published": {"year": "2026"},
		"author": [{"first": "John", "last": "Smith"}]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("ParsePaperpile() returned %d refs, want 1", len(refs))
	}

	// When no citekey, should use Paperpile ID
	if refs[0].ID != "abc123" {
		t.Errorf("ID = %v, want abc123 (Paperpile ID)", refs[0].ID)
	}
}

func TestParsePaperpile_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "missing title",
			data: `[{"_id": "abc", "published": {"year": "2026"}, "author": [{"first": "John", "last": "Smith"}]}]`,
		},
		{
			name: "missing author",
			data: `[{"_id": "abc", "title": "Test", "published": {"year": "2026"}, "author": []}]`,
		},
		{
			name: "missing year",
			data: `[{"_id": "abc", "title": "Test", "author": [{"first": "John", "last": "Smith"}]}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, _, errs := ParsePaperpile([]byte(tt.data), true)
			if len(errs) == 0 {
				t.Errorf("ParsePaperpile() expected error for %s, got refs: %+v", tt.name, refs)
			}
		})
	}
}

func TestParsePaperpile_InvalidYear(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"title": "Test Paper",
		"published": {"year": "invalid"},
		"author": [{"first": "John", "last": "Smith"}]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) == 0 {
		t.Errorf("ParsePaperpile() expected error for invalid year, got refs: %+v", refs)
	}
}

func TestParsePaperpile_NumericYearMonth(t *testing.T) {
	// Test that numeric year/month values work (Paperpile exports both formats)
	data := []byte(`[{
		"_id": "abc123",
		"title": "Test Paper",
		"published": {"year": 2026, "month": 6},
		"author": [{"first": "John", "last": "Smith"}]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if refs[0].Published.Year != 2026 {
		t.Errorf("Published.Year = %d, want 2026", refs[0].Published.Year)
	}
	if refs[0].Published.Month != 6 {
		t.Errorf("Published.Month = %d, want 6", refs[0].Published.Month)
	}
}

func TestParsePaperpile_InvalidJSON(t *testing.T) {
	data := []byte(`not valid json`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) == 0 {
		t.Errorf("ParsePaperpile() expected error for invalid JSON, got refs: %+v", refs)
	}
}

func TestParsePaperpile_EmptyArray(t *testing.T) {
	data := []byte(`[]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs) != 0 {
		t.Errorf("ParsePaperpile() returned %d refs, want 0", len(refs))
	}
}

func TestParsePaperpile_MultipleEntries(t *testing.T) {
	data := []byte(`[
		{"_id": "1", "citekey": "A2026", "title": "Paper A", "published": {"year": "2026"}, "author": [{"last": "A"}]},
		{"_id": "2", "citekey": "B2026", "title": "Paper B", "published": {"year": "2025"}, "author": [{"last": "B"}]}
	]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs) != 2 {
		t.Fatalf("ParsePaperpile() returned %d refs, want 2", len(refs))
	}
	if refs[0].ID != "A2026" || refs[1].ID != "B2026" {
		t.Errorf("IDs = [%s, %s], want [A2026, B2026]", refs[0].ID, refs[1].ID)
	}
}

func TestParsePaperpile_PartialErrors(t *testing.T) {
	// Mix of valid and invalid entries - should return valid ones and errors for invalid
	data := []byte(`[
		{"_id": "1", "citekey": "Valid2026", "title": "Valid", "published": {"year": "2026"}, "author": [{"last": "Valid"}]},
		{"_id": "2", "citekey": "Invalid", "title": "", "published": {"year": "2026"}, "author": [{"last": "Invalid"}]},
		{"_id": "3", "citekey": "AlsoValid2026", "title": "Also Valid", "published": {"year": "2025"}, "author": [{"last": "Also"}]}
	]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(refs) != 2 {
		t.Errorf("ParsePaperpile() returned %d valid refs, want 2", len(refs))
	}
	if len(errs) != 1 {
		t.Errorf("ParsePaperpile() returned %d errors, want 1", len(errs))
	}
}

func TestParsePaperpile_RealTestData(t *testing.T) {
	// Test against the actual sample data file
	testFile := filepath.Join("..", "..", "testdata", "paperpile_sample.json")
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Skipf("Test data file not found: %v", err)
	}

	strictRefs, _, strictErrs := ParsePaperpile(data, true)
	if len(strictErrs) > 0 {
		t.Errorf("strict mode returned %d errors parsing real test data: %v", len(strictErrs), strictErrs)
	}
	if len(strictRefs) == 0 {
		t.Error("strict mode returned 0 refs from test data")
	}

	// Check that the first reference has expected structure. The fixture
	// contains complete entries, so strict-mode results should not contain
	// sentinel values.
	if len(strictRefs) > 0 {
		ref := strictRefs[0]
		if ref.ID == "" {
			t.Error("First ref has empty ID")
		}
		if ref.Title == "" || ref.Title == UnknownTitle {
			t.Errorf("First ref Title = %q (sentinel or empty)", ref.Title)
		}
		if len(ref.Authors) == 0 {
			t.Error("First ref has no Authors")
		}
		if ref.Published.Year == UnknownYear {
			t.Errorf("First ref Year = %d (sentinel)", ref.Published.Year)
		}
		if ref.Source.Type != "paperpile" {
			t.Errorf("First ref Source.Type = %s, want paperpile", ref.Source.Type)
		}
	}

	// Lenient mode should never drop more entries than strict and should
	// produce structurally valid warnings for any entries that hit fallbacks.
	lenientRefs, lenientWarnings, lenientErrs := ParsePaperpile(data, false)
	if len(lenientErrs) > len(strictErrs) {
		t.Errorf("lenient mode returned more errors (%d) than strict (%d)", len(lenientErrs), len(strictErrs))
	}
	if len(lenientRefs) < len(strictRefs) {
		t.Errorf("lenient mode dropped more entries (%d refs) than strict (%d refs)", len(lenientRefs), len(strictRefs))
	}
	for i, w := range lenientWarnings {
		if w.ID == "" {
			t.Errorf("lenient warning %d has empty ID", i)
		}
		if len(w.Fields) == 0 {
			t.Errorf("lenient warning %d (%s) has no Fields", i, w.ID)
		}
	}
}

func TestPaperpileEntry_AuthorWithOnlyLast(t *testing.T) {
	// Some papers have authors with only last names (e.g., corporate authors)
	data := []byte(`[{
		"_id": "abc123",
		"title": "Test Paper",
		"published": {"year": "2026"},
		"author": [{"last": "Corporation"}]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}

	if refs[0].Authors[0].Last != "Corporation" {
		t.Errorf("Authors[0].Last = %v, want Corporation", refs[0].Authors[0].Last)
	}
	if refs[0].Authors[0].First != "" {
		t.Errorf("Authors[0].First = %v, want empty string", refs[0].Authors[0].First)
	}
}

func TestParsePaperpile_WithLabelsAndFolders(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "Tagged2026",
		"title": "Tagged Paper",
		"published": {"year": "2026"},
		"author": [{"first": "John", "last": "Smith"}],
		"labelsNamed": ["antibody", "vaccine"],
		"foldersNamed": ["my papers"]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs[0].Tags) != 3 {
		t.Fatalf("Tags count = %d, want 3, got %v", len(refs[0].Tags), refs[0].Tags)
	}
	if refs[0].Tags[0] != "antibody" || refs[0].Tags[1] != "vaccine" || refs[0].Tags[2] != "my papers" {
		t.Errorf("Tags = %v, want [antibody vaccine my papers]", refs[0].Tags)
	}
}

func TestParsePaperpile_DuplicateLabelAndFolder(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "Dedup2026",
		"title": "Dedup Paper",
		"published": {"year": "2026"},
		"author": [{"first": "John", "last": "Smith"}],
		"labelsNamed": ["antibody"],
		"foldersNamed": ["antibody"]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs[0].Tags) != 1 {
		t.Errorf("Tags count = %d, want 1 (deduplicated), got %v", len(refs[0].Tags), refs[0].Tags)
	}
}

func TestParsePaperpile_NoLabelsOrFolders(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "NoTags2026",
		"title": "No Tags Paper",
		"published": {"year": "2026"},
		"author": [{"first": "John", "last": "Smith"}]
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile() returned errors: %v", errs)
	}
	if len(refs[0].Tags) != 0 {
		t.Errorf("Tags = %v, want empty", refs[0].Tags)
	}
}

func TestParsePaperpile_LenientMissingYear(t *testing.T) {
	// Real-world case from issue #138: eLife reviewed preprint with no year.
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "Sung2025-thrifty",
		"title": "Thrifty wide-context models of B cell receptor somatic hypermutation",
		"author": [{"first": "Kevin", "last": "Sung"}],
		"published": {}
	}]`)

	refs, warnings, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("ParsePaperpile(lenient) returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if refs[0].Published.Year != UnknownYear {
		t.Errorf("Year = %d, want %d (sentinel)", refs[0].Published.Year, UnknownYear)
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if len(warnings[0].Fields) != 1 || warnings[0].Fields[0] != "published.year" {
		t.Errorf("warning fields = %v, want [published.year]", warnings[0].Fields)
	}
	if warnings[0].ID != "Sung2025-thrifty" {
		t.Errorf("warning ID = %s, want Sung2025-thrifty", warnings[0].ID)
	}
}

func TestParsePaperpile_LenientMissingTitle(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "NoTitle2026",
		"author": [{"last": "Smith"}],
		"published": {"year": "2026"}
	}]`)

	refs, warnings, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("returned errors: %v", errs)
	}
	if len(refs) != 1 || refs[0].Title != UnknownTitle {
		t.Errorf("Title = %q, want %q", refs[0].Title, UnknownTitle)
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if len(warnings[0].Fields) != 1 || warnings[0].Fields[0] != "title" {
		t.Errorf("warning fields = %v, want [title]", warnings[0].Fields)
	}
	if warnings[0].ID != "NoTitle2026" {
		t.Errorf("warning ID = %s, want NoTitle2026", warnings[0].ID)
	}
}

func TestParsePaperpile_LenientMissingAuthor(t *testing.T) {
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "NoAuthor2026",
		"title": "Some paper",
		"published": {"year": "2026"}
	}]`)

	refs, warnings, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if len(refs[0].Authors) != 1 || refs[0].Authors[0].Last != UnknownAuthor {
		t.Errorf("Authors = %+v, want one author with Last=%q", refs[0].Authors, UnknownAuthor)
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if len(warnings[0].Fields) != 1 || warnings[0].Fields[0] != "author" {
		t.Errorf("warning fields = %v, want [author]", warnings[0].Fields)
	}
	if warnings[0].ID != "NoAuthor2026" {
		t.Errorf("warning ID = %s, want NoAuthor2026", warnings[0].ID)
	}
}

func TestParsePaperpile_LenientMultipleMissingFields(t *testing.T) {
	// Has DOI, so not pure junk — should still import with all three sentinels.
	data := []byte(`[{
		"_id": "abc123",
		"citekey": "OnlyDOI",
		"doi": "10.1234/whatever",
		"published": {}
	}]`)

	refs, warnings, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if len(warnings[0].Fields) != 3 {
		t.Errorf("warning fields = %v, want all three (title, author, published.year)", warnings[0].Fields)
	}
}

func TestParsePaperpile_LenientJunkEntryDropped(t *testing.T) {
	// Pure junk: no title, no author, no year, no DOI. Paperpile auto-stub for
	// an unparsed web page. Even in lenient mode this should be dropped to
	// avoid flooding the nexus with placeholders.
	data := []byte(`[{
		"_id": "abc",
		"citekey": "noauthor_undated-aa",
		"published": {}
	}]`)

	refs, warnings, errs := ParsePaperpile(data, false)
	if len(refs) != 0 {
		t.Errorf("got %d refs, want 0 (junk should be dropped)", len(refs))
	}
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0", len(warnings))
	}
	if len(errs) != 1 {
		t.Errorf("got %d errors, want 1 (junk dropped with explanation)", len(errs))
	}
}

func TestParsePaperpile_StrictDropsAnyMissingField(t *testing.T) {
	// Verify strict mode preserves the original behavior: any of the three
	// required fields (title, author, year) being missing drops the entry,
	// with no warnings produced.
	cases := []struct {
		name string
		data string
	}{
		{
			name: "missing year",
			data: `[{"_id": "a", "citekey": "MissingYear", "title": "T", "author": [{"last": "S"}], "published": {}}]`,
		},
		{
			name: "missing title",
			data: `[{"_id": "a", "citekey": "MissingTitle", "author": [{"last": "S"}], "published": {"year": "2026"}}]`,
		},
		{
			name: "missing author",
			data: `[{"_id": "a", "citekey": "MissingAuthor", "title": "T", "published": {"year": "2026"}}]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refs, warnings, errs := ParsePaperpile([]byte(tc.data), true)
			if len(refs) != 0 {
				t.Errorf("strict mode: got %d refs, want 0", len(refs))
			}
			if len(warnings) != 0 {
				t.Errorf("strict mode should not produce warnings, got %d", len(warnings))
			}
			if len(errs) != 1 {
				t.Errorf("strict mode: got %d errors, want 1", len(errs))
			}
		})
	}
}

func TestParsePaperpile_LenientMixedArray(t *testing.T) {
	// Realistic Paperpile shape: one valid entry, one recoverable (missing
	// year only), one pure junk. Verify that the warning for the recoverable
	// entry is emitted alongside an error for the junk entry without
	// corrupting either, and that the valid entry imports cleanly.
	data := []byte(`[
		{"_id": "1", "citekey": "Good2026", "title": "Good", "author": [{"last": "G"}], "published": {"year": "2026"}},
		{"_id": "2", "citekey": "Recoverable", "title": "Recoverable paper", "author": [{"last": "R"}], "published": {}},
		{"_id": "3", "citekey": "noauthor_undated-zz", "published": {}}
	]`)

	refs, warnings, errs := ParsePaperpile(data, false)
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2 (good + recoverable)", len(refs))
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1 (recoverable only)", len(warnings))
	}
	if warnings[0].ID != "Recoverable" {
		t.Errorf("warning attached to %s, want Recoverable", warnings[0].ID)
	}
	if len(warnings[0].Fields) != 1 || warnings[0].Fields[0] != "published.year" {
		t.Errorf("warning fields = %v, want [published.year]", warnings[0].Fields)
	}
	if len(errs) != 1 {
		t.Errorf("got %d errors, want 1 (junk drop)", len(errs))
	}
}

func tagsContain(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

func countTag(tags []string, want string) int {
	n := 0
	for _, t := range tags {
		if t == want {
			n++
		}
	}
	return n
}

func TestParsePaperpile_LenientFallbackAddsIncompleteTag(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{
			name: "missing year",
			data: `[{"_id": "a", "citekey": "MissingYear", "title": "T", "author": [{"last": "S"}], "published": {}}]`,
		},
		{
			name: "missing title",
			data: `[{"_id": "a", "citekey": "MissingTitle", "author": [{"last": "S"}], "published": {"year": "2026"}}]`,
		},
		{
			name: "missing author",
			data: `[{"_id": "a", "citekey": "MissingAuthor", "title": "T", "published": {"year": "2026"}}]`,
		},
		{
			name: "all three missing (DOI present)",
			data: `[{"_id": "a", "citekey": "OnlyDOI", "doi": "10.1234/x", "published": {}}]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refs, _, errs := ParsePaperpile([]byte(tc.data), false)
			if len(errs) > 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if len(refs) != 1 {
				t.Fatalf("got %d refs, want 1", len(refs))
			}
			if !tagsContain(refs[0].Tags, IncompleteTag) {
				t.Errorf("Tags = %v, want to contain %q", refs[0].Tags, IncompleteTag)
			}
		})
	}
}

func TestParsePaperpile_StrictCompleteEntryDoesNotGetIncompleteTag(t *testing.T) {
	// In strict mode no fallbacks fire (entries with missing fields are
	// dropped, complete entries import without sentinels), so the tag
	// should never be set.
	data := []byte(`[{
		"_id": "abc",
		"citekey": "Complete2026",
		"title": "Complete paper",
		"author": [{"last": "Smith"}],
		"published": {"year": "2026"}
	}]`)

	refs, _, errs := ParsePaperpile(data, true)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if tagsContain(refs[0].Tags, IncompleteTag) {
		t.Errorf("Tags = %v, should not contain %q in strict mode", refs[0].Tags, IncompleteTag)
	}
}

func TestParsePaperpile_ValidEntryDoesNotGetIncompleteTag(t *testing.T) {
	data := []byte(`[{
		"_id": "abc",
		"citekey": "Complete2026",
		"title": "Complete paper",
		"author": [{"last": "Smith"}],
		"published": {"year": "2026"}
	}]`)

	refs, _, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if tagsContain(refs[0].Tags, IncompleteTag) {
		t.Errorf("Tags = %v, should not contain %q for a complete entry", refs[0].Tags, IncompleteTag)
	}
}

func TestParsePaperpile_LenientMixedArrayTagsRecoverableOnly(t *testing.T) {
	data := []byte(`[
		{"_id": "1", "citekey": "Good2026", "title": "Good", "author": [{"last": "G"}], "published": {"year": "2026"}},
		{"_id": "2", "citekey": "Recoverable", "title": "Recoverable paper", "author": [{"last": "R"}], "published": {}}
	]`)

	refs, _, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}

	var good, recoverable reference.Reference
	for _, r := range refs {
		switch r.ID {
		case "Good2026":
			good = r
		case "Recoverable":
			recoverable = r
		}
	}
	if tagsContain(good.Tags, IncompleteTag) {
		t.Errorf("complete entry Tags = %v, should not contain %q", good.Tags, IncompleteTag)
	}
	if !tagsContain(recoverable.Tags, IncompleteTag) {
		t.Errorf("recoverable entry Tags = %v, want to contain %q", recoverable.Tags, IncompleteTag)
	}
}

func TestParsePaperpile_LenientUserLabelDedupedWithIncompleteTag(t *testing.T) {
	// If a Paperpile user has manually labeled an entry "paperpile:incomplete"
	// AND the entry hits a fallback, the existing dedup `seen` map should
	// collapse the two into a single occurrence in Tags.
	data := []byte(`[{
		"_id": "abc",
		"citekey": "DoubleTagged",
		"title": "T",
		"author": [{"last": "S"}],
		"published": {},
		"labelsNamed": ["paperpile:incomplete"]
	}]`)

	refs, _, errs := ParsePaperpile(data, false)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	if got := countTag(refs[0].Tags, IncompleteTag); got != 1 {
		t.Errorf("Tags = %v: %q occurs %d times, want 1", refs[0].Tags, IncompleteTag, got)
	}
}

// Helper function for comparing references
func refsEqual(a, b reference.Reference) bool {
	if a.ID != b.ID || a.DOI != b.DOI || a.Title != b.Title {
		return false
	}
	if len(a.Authors) != len(b.Authors) {
		return false
	}
	for i := range a.Authors {
		if a.Authors[i] != b.Authors[i] {
			return false
		}
	}
	return true
}
