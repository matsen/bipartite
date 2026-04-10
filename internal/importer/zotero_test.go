package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseZotero_ValidEntry(t *testing.T) {
	data := []byte(`[{
		"id": "http://zotero.org/users/12345/items/ABCD1234",
		"type": "article-journal",
		"citation-key": "Bloom2023-ne",
		"title": "Fitness effects of mutations",
		"author": [
			{"family": "Bloom", "given": "Jesse D"},
			{"family": "Neher", "given": "Richard A"}
		],
		"container-title": "Virus Evolution",
		"DOI": "10.1093/ve/vead055",
		"PMID": "37727785",
		"PMCID": "PMC10506396",
		"abstract": "Knowledge of fitness effects.",
		"issued": {"date-parts": [[2023, 8, 22]]},
		"note": "Key paper"
	}]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("ParseZotero() returned %d refs, want 1", len(refs))
	}

	ref := refs[0]
	if ref.ID != "Bloom2023-ne" {
		t.Errorf("ID = %v, want Bloom2023-ne", ref.ID)
	}
	if ref.DOI != "10.1093/ve/vead055" {
		t.Errorf("DOI = %v, want 10.1093/ve/vead055", ref.DOI)
	}
	if ref.Title != "Fitness effects of mutations" {
		t.Errorf("Title = %v, want 'Fitness effects of mutations'", ref.Title)
	}
	if len(ref.Authors) != 2 {
		t.Fatalf("Authors count = %d, want 2", len(ref.Authors))
	}
	if ref.Authors[0].Last != "Bloom" || ref.Authors[0].First != "Jesse D" {
		t.Errorf("Author[0] = %+v, want Bloom, Jesse D", ref.Authors[0])
	}
	if ref.Authors[1].Last != "Neher" || ref.Authors[1].First != "Richard A" {
		t.Errorf("Author[1] = %+v, want Neher, Richard A", ref.Authors[1])
	}
	if ref.Venue != "Virus Evolution" {
		t.Errorf("Venue = %v, want 'Virus Evolution'", ref.Venue)
	}
	if ref.Abstract != "Knowledge of fitness effects." {
		t.Errorf("Abstract = %v, want 'Knowledge of fitness effects.'", ref.Abstract)
	}
	if ref.Published.Year != 2023 {
		t.Errorf("Published.Year = %d, want 2023", ref.Published.Year)
	}
	if ref.Published.Month != 8 {
		t.Errorf("Published.Month = %d, want 8", ref.Published.Month)
	}
	if ref.Published.Day != 22 {
		t.Errorf("Published.Day = %d, want 22", ref.Published.Day)
	}
	if ref.PMID != "37727785" {
		t.Errorf("PMID = %v, want 37727785", ref.PMID)
	}
	if ref.PMCID != "PMC10506396" {
		t.Errorf("PMCID = %v, want PMC10506396", ref.PMCID)
	}
	if ref.Note != "Key paper" {
		t.Errorf("Note = %v, want 'Key paper'", ref.Note)
	}
	if ref.Source.Type != "zotero" {
		t.Errorf("Source.Type = %v, want zotero", ref.Source.Type)
	}
	if ref.Source.ID != "ABCD1234" {
		t.Errorf("Source.ID = %v, want ABCD1234", ref.Source.ID)
	}
}

func TestParseZotero_NoCitationKey(t *testing.T) {
	data := []byte(`[{
		"id": "http://zotero.org/users/12345/items/WXYZ9999",
		"type": "article-journal",
		"title": "A paper without citation-key",
		"author": [{"family": "Smith", "given": "John"}],
		"issued": {"date-parts": [[2024]]}
	}]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if len(refs) != 1 {
		t.Fatalf("ParseZotero() returned %d refs, want 1", len(refs))
	}

	// Without citation-key, ID should be the Zotero item key
	if refs[0].ID != "WXYZ9999" {
		t.Errorf("ID = %v, want WXYZ9999 (Zotero item key)", refs[0].ID)
	}
}

func TestParseZotero_PlainIDNotURL(t *testing.T) {
	data := []byte(`[{
		"id": "some-plain-id",
		"type": "article-journal",
		"title": "A paper with plain ID",
		"author": [{"family": "Doe", "given": "Jane"}],
		"issued": {"date-parts": [[2020]]}
	}]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if refs[0].ID != "some-plain-id" {
		t.Errorf("ID = %v, want some-plain-id", refs[0].ID)
	}
	if refs[0].Source.ID != "some-plain-id" {
		t.Errorf("Source.ID = %v, want some-plain-id", refs[0].Source.ID)
	}
}

func TestParseZotero_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "missing title",
			data: `[{"id": "x", "author": [{"family": "A", "given": "B"}], "issued": {"date-parts": [[2020]]}}]`,
		},
		{
			name: "missing author",
			data: `[{"id": "x", "title": "Test", "issued": {"date-parts": [[2020]]}}]`,
		},
		{
			name: "missing issued",
			data: `[{"id": "x", "title": "Test", "author": [{"family": "A", "given": "B"}]}]`,
		},
		{
			name: "empty date-parts",
			data: `[{"id": "x", "title": "Test", "author": [{"family": "A", "given": "B"}], "issued": {"date-parts": []}}]`,
		},
		{
			name: "zero year",
			data: `[{"id": "x", "title": "Test", "author": [{"family": "A", "given": "B"}], "issued": {"date-parts": [[0]]}}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, errs := ParseZotero([]byte(tt.data))
			if len(refs) > 0 {
				t.Errorf("ParseZotero() expected error for %s, got refs: %+v", tt.name, refs)
			}
			if len(errs) == 0 {
				t.Errorf("ParseZotero() expected error for %s, got none", tt.name)
			}
		})
	}
}

func TestParseZotero_DatePartsYearOnly(t *testing.T) {
	data := []byte(`[{
		"id": "x",
		"title": "Year only",
		"author": [{"family": "Test", "given": "A"}],
		"issued": {"date-parts": [[2017]]}
	}]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if refs[0].Published.Year != 2017 {
		t.Errorf("Year = %d, want 2017", refs[0].Published.Year)
	}
	if refs[0].Published.Month != 0 {
		t.Errorf("Month = %d, want 0", refs[0].Published.Month)
	}
	if refs[0].Published.Day != 0 {
		t.Errorf("Day = %d, want 0", refs[0].Published.Day)
	}
}

func TestParseZotero_DatePartsYearMonth(t *testing.T) {
	data := []byte(`[{
		"id": "x",
		"title": "Year and month",
		"author": [{"family": "Test", "given": "A"}],
		"issued": {"date-parts": [[2014, 5]]}
	}]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if refs[0].Published.Year != 2014 {
		t.Errorf("Year = %d, want 2014", refs[0].Published.Year)
	}
	if refs[0].Published.Month != 5 {
		t.Errorf("Month = %d, want 5", refs[0].Published.Month)
	}
	if refs[0].Published.Day != 0 {
		t.Errorf("Day = %d, want 0", refs[0].Published.Day)
	}
}

func TestParseZotero_LiteralAuthor(t *testing.T) {
	data := []byte(`[{
		"id": "x",
		"title": "Institutional author test",
		"author": [{"literal": "World Health Organization"}],
		"issued": {"date-parts": [[2020]]}
	}]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if refs[0].Authors[0].Last != "World Health Organization" {
		t.Errorf("Author Last = %v, want 'World Health Organization'", refs[0].Authors[0].Last)
	}
	if refs[0].Authors[0].First != "" {
		t.Errorf("Author First = %v, want empty", refs[0].Authors[0].First)
	}
}

func TestParseZotero_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	refs, errs := ParseZotero(data)
	if len(refs) > 0 {
		t.Errorf("ParseZotero() expected error for invalid JSON, got refs: %+v", refs)
	}
	if len(errs) == 0 {
		t.Error("ParseZotero() expected error for invalid JSON, got none")
	}
}

func TestParseZotero_EmptyArray(t *testing.T) {
	data := []byte(`[]`)
	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if len(refs) != 0 {
		t.Errorf("ParseZotero() returned %d refs, want 0", len(refs))
	}
}

func TestParseZotero_MultipleEntries(t *testing.T) {
	data := []byte(`[
		{"id": "a", "title": "First", "author": [{"family": "A", "given": "B"}], "issued": {"date-parts": [[2020]]}},
		{"id": "b", "title": "Second", "author": [{"family": "C", "given": "D"}], "issued": {"date-parts": [[2021]]}}
	]`)

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Fatalf("ParseZotero() returned errors: %v", errs)
	}
	if len(refs) != 2 {
		t.Fatalf("ParseZotero() returned %d refs, want 2", len(refs))
	}
}

func TestParseZotero_PartialErrors(t *testing.T) {
	data := []byte(`[
		{"id": "a", "title": "Valid", "author": [{"family": "A", "given": "B"}], "issued": {"date-parts": [[2020]]}},
		{"id": "b", "title": "", "author": [{"family": "C", "given": "D"}], "issued": {"date-parts": [[2021]]}},
		{"id": "c", "title": "Also valid", "author": [{"family": "E", "given": "F"}], "issued": {"date-parts": [[2022]]}}
	]`)

	refs, errs := ParseZotero(data)
	if len(refs) != 2 {
		t.Errorf("ParseZotero() returned %d valid refs, want 2", len(refs))
	}
	if len(errs) != 1 {
		t.Errorf("ParseZotero() returned %d errors, want 1", len(errs))
	}
}

func TestParseZotero_RealTestData(t *testing.T) {
	testFile := filepath.Join("..", "..", "testdata", "zotero_csl_sample.json")
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	refs, errs := ParseZotero(data)
	if len(errs) > 0 {
		t.Errorf("ParseZotero() returned %d errors parsing real test data: %v", len(errs), errs)
	}
	if len(refs) == 0 {
		t.Error("ParseZotero() returned 0 refs from test data")
	}
	if len(refs) != 5 {
		t.Errorf("ParseZotero() returned %d refs, want 5", len(refs))
	}

	// Verify first entry
	ref := refs[0]
	if ref.ID != "Bloom2023-ne" {
		t.Errorf("First ref ID = %s, want Bloom2023-ne", ref.ID)
	}
	if ref.Source.Type != "zotero" {
		t.Errorf("First ref Source.Type = %s, want zotero", ref.Source.Type)
	}
	if ref.Source.ID != "ABCD1234" {
		t.Errorf("First ref Source.ID = %s, want ABCD1234", ref.Source.ID)
	}
	if ref.PMID != "37727785" {
		t.Errorf("First ref PMID = %s, want 37727785", ref.PMID)
	}

	// Verify institutional author entry (4th)
	who := refs[3]
	if who.Authors[0].Last != "World Health Organization" {
		t.Errorf("WHO entry author = %+v, want literal 'World Health Organization'", who.Authors[0])
	}
	if who.Authors[0].First != "" {
		t.Errorf("WHO entry author First = %v, want empty", who.Authors[0].First)
	}

	// Entry without citation-key should use Zotero item key
	dudas := refs[2]
	if dudas.ID != "IJKL9012" {
		t.Errorf("Entry without citation-key ID = %s, want IJKL9012", dudas.ID)
	}
}

func TestExtractZoteroItemKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://zotero.org/users/12345/items/ABCD1234", "ABCD1234"},
		{"https://zotero.org/users/12345/items/XYZ", "XYZ"},
		{"http://zotero.org/groups/99/items/KEY123", "KEY123"},
		{"plain-id", "plain-id"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractZoteroItemKey(tt.input)
		if got != tt.want {
			t.Errorf("extractZoteroItemKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
