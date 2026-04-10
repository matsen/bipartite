package zotero

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestMapZoteroToReference_ValidItem(t *testing.T) {
	item := ZoteroItem{
		Key:     "ABCD1234",
		Version: 42,
		Data: ZoteroItemData{
			ItemType:         "journalArticle",
			Title:            "Test Paper Title",
			AbstractNote:     "An abstract.",
			PublicationTitle: "Nature Methods",
			DOI:              "10.1038/test",
			Date:             "2023-08-22",
			Extra:            "PMID: 12345\nPMCID: PMC67890\narXiv: 2106.15928",
			Creators: []ZoteroCreator{
				{CreatorType: "author", FirstName: "Jane", LastName: "Doe"},
				{CreatorType: "author", FirstName: "John", LastName: "Smith"},
			},
		},
	}

	ref, err := MapZoteroToReference(item)
	if err != nil {
		t.Fatalf("MapZoteroToReference() error: %v", err)
	}

	if ref.Title != "Test Paper Title" {
		t.Errorf("Title = %q, want 'Test Paper Title'", ref.Title)
	}
	if ref.DOI != "10.1038/test" {
		t.Errorf("DOI = %q, want '10.1038/test'", ref.DOI)
	}
	if ref.Venue != "Nature Methods" {
		t.Errorf("Venue = %q, want 'Nature Methods'", ref.Venue)
	}
	if ref.Abstract != "An abstract." {
		t.Errorf("Abstract = %q, want 'An abstract.'", ref.Abstract)
	}
	if ref.Published.Year != 2023 || ref.Published.Month != 8 || ref.Published.Day != 22 {
		t.Errorf("Published = %+v, want 2023-08-22", ref.Published)
	}
	if len(ref.Authors) != 2 {
		t.Fatalf("Authors count = %d, want 2", len(ref.Authors))
	}
	if ref.Authors[0].First != "Jane" || ref.Authors[0].Last != "Doe" {
		t.Errorf("Author[0] = %+v, want Jane Doe", ref.Authors[0])
	}
	if ref.PMID != "12345" {
		t.Errorf("PMID = %q, want '12345'", ref.PMID)
	}
	if ref.PMCID != "PMC67890" {
		t.Errorf("PMCID = %q, want 'PMC67890'", ref.PMCID)
	}
	if ref.ArXivID != "2106.15928" {
		t.Errorf("ArXivID = %q, want '2106.15928'", ref.ArXivID)
	}
	if ref.Source.Type != "zotero" {
		t.Errorf("Source.Type = %q, want 'zotero'", ref.Source.Type)
	}
	if ref.Source.ID != "ABCD1234" {
		t.Errorf("Source.ID = %q, want 'ABCD1234'", ref.Source.ID)
	}
}

func TestMapZoteroToReference_InstitutionalAuthor(t *testing.T) {
	item := ZoteroItem{
		Key: "KEY1",
		Data: ZoteroItemData{
			Title: "Report",
			Date:  "2020",
			Creators: []ZoteroCreator{
				{CreatorType: "author", Name: "World Health Organization"},
			},
		},
	}

	ref, err := MapZoteroToReference(item)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if ref.Authors[0].Last != "World Health Organization" || ref.Authors[0].First != "" {
		t.Errorf("Author = %+v, want institutional name in Last only", ref.Authors[0])
	}
}

func TestMapZoteroToReference_EditorFallback(t *testing.T) {
	item := ZoteroItem{
		Key: "KEY2",
		Data: ZoteroItemData{
			Title: "Edited Volume",
			Date:  "2021",
			Creators: []ZoteroCreator{
				{CreatorType: "editor", FirstName: "Ed", LastName: "Smith"},
			},
		},
	}

	ref, err := MapZoteroToReference(item)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should fall back to editors when no authors
	if len(ref.Authors) != 1 || ref.Authors[0].Last != "Smith" {
		t.Errorf("Expected editor fallback, got %+v", ref.Authors)
	}
}

func TestMapZoteroToReference_MissingTitle(t *testing.T) {
	item := ZoteroItem{
		Key: "KEY3",
		Data: ZoteroItemData{
			Date:     "2020",
			Creators: []ZoteroCreator{{CreatorType: "author", LastName: "X"}},
		},
	}

	_, err := MapZoteroToReference(item)
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func TestMapZoteroToReference_MissingCreators(t *testing.T) {
	item := ZoteroItem{
		Key: "KEY4",
		Data: ZoteroItemData{
			Title: "No Authors",
			Date:  "2020",
		},
	}

	_, err := MapZoteroToReference(item)
	if err == nil {
		t.Error("expected error for missing creators")
	}
}

func TestMapZoteroToReference_MissingDate(t *testing.T) {
	item := ZoteroItem{
		Key: "KEY5",
		Data: ZoteroItemData{
			Title:    "No Date",
			Creators: []ZoteroCreator{{CreatorType: "author", LastName: "X"}},
		},
	}

	_, err := MapZoteroToReference(item)
	if err == nil {
		t.Error("expected error for missing date")
	}
}

func TestMapReferenceToZotero(t *testing.T) {
	ref := reference.Reference{
		Title:    "Test Paper",
		DOI:      "10.1234/test",
		Abstract: "Abstract text",
		Venue:    "Science",
		Authors: []reference.Author{
			{First: "Jane", Last: "Doe"},
			{Last: "WHO"}, // institutional
		},
		Published: reference.PublicationDate{Year: 2023, Month: 5, Day: 15},
		PMID:      "99999",
		ArXivID:   "2301.00001",
	}

	item := MapReferenceToZotero(ref)

	if item.ItemType != "journalArticle" {
		t.Errorf("ItemType = %q, want 'journalArticle'", item.ItemType)
	}
	if item.Title != "Test Paper" {
		t.Errorf("Title = %q", item.Title)
	}
	if item.DOI != "10.1234/test" {
		t.Errorf("DOI = %q", item.DOI)
	}
	if item.Date != "2023-05-15" {
		t.Errorf("Date = %q, want '2023-05-15'", item.Date)
	}
	if len(item.Creators) != 2 {
		t.Fatalf("Creators count = %d, want 2", len(item.Creators))
	}
	if item.Creators[0].FirstName != "Jane" || item.Creators[0].LastName != "Doe" {
		t.Errorf("Creator[0] = %+v", item.Creators[0])
	}
	// Institutional author should use Name field
	if item.Creators[1].Name != "WHO" {
		t.Errorf("Creator[1] = %+v, want Name='WHO'", item.Creators[1])
	}
	if item.Extra != "PMID: 99999\narXiv: 2301.00001" {
		t.Errorf("Extra = %q", item.Extra)
	}
}

func TestParseZoteroDate(t *testing.T) {
	tests := []struct {
		input string
		want  reference.PublicationDate
	}{
		{"2023-08-22", reference.PublicationDate{Year: 2023, Month: 8, Day: 22}},
		{"2023-08", reference.PublicationDate{Year: 2023, Month: 8}},
		{"2023", reference.PublicationDate{Year: 2023}},
		{"August 22, 2023", reference.PublicationDate{Year: 2023}},
		{"", reference.PublicationDate{}},
		{"no date here", reference.PublicationDate{}},
	}

	for _, tt := range tests {
		got := parseZoteroDate(tt.input)
		if got != tt.want {
			t.Errorf("parseZoteroDate(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestParseExtra(t *testing.T) {
	tests := []struct {
		input                  string
		wantPMID, wantPMCID, wantArXiv string
	}{
		{"PMID: 12345\nPMCID: PMC67890\narXiv: 2106.15928", "12345", "PMC67890", "2106.15928"},
		{"PMID: 12345", "12345", "", ""},
		{"some other stuff", "", "", ""},
		{"", "", "", ""},
		{"PMID: 111\nSome noise\narXiv: 222", "111", "", "222"},
	}

	for _, tt := range tests {
		pmid, pmcid, arxiv := parseExtra(tt.input)
		if pmid != tt.wantPMID || pmcid != tt.wantPMCID || arxiv != tt.wantArXiv {
			t.Errorf("parseExtra(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.input, pmid, pmcid, arxiv, tt.wantPMID, tt.wantPMCID, tt.wantArXiv)
		}
	}
}

func TestGenerateCiteKey(t *testing.T) {
	authors := []reference.Author{{First: "Jane", Last: "Doe"}}
	key := generateCiteKey(authors, 2023, "Attention Is All You Need")
	if key != "Doe2023-ai" {
		t.Errorf("generateCiteKey() = %q, want 'Doe2023-ai'", key)
	}
}
