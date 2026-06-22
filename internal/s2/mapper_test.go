package s2

import "testing"

func TestSplitAuthorName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFirst string
		wantLast  string
	}{
		{"empty", "", "", ""},
		{"whitespace only", "   ", "", ""},
		{"single name", "Madonna", "", "Madonna"},
		{"first last", "John Smith", "John", "Smith"},
		{"first middle last", "John Quincy Adams", "John Quincy", "Adams"},
		{"suffix Jr", "John Smith Jr", "John", "Smith Jr"},
		{"suffix III", "John Smith III", "John", "Smith III"},
		{"suffix with middle", "John Quincy Smith Jr", "John Quincy", "Smith Jr"},
		{"two-part suffix-like is just last", "Smith Jr", "Smith", "Jr"},
		{"surrounding whitespace", "  John Smith  ", "John", "Smith"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := splitAuthorName(tt.input)
			if first != tt.wantFirst {
				t.Errorf("first = %q, want %q", first, tt.wantFirst)
			}
			if last != tt.wantLast {
				t.Errorf("last = %q, want %q", last, tt.wantLast)
			}
		})
	}
}

func TestParsePublicationDate(t *testing.T) {
	tests := []struct {
		name             string
		year             int
		dateStr          string
		wantY, wantM, wD int
	}{
		{"year only, no date string", 2018, "", 2018, 0, 0},
		{"full date overrides year", 2018, "2019-03-15", 2019, 3, 15},
		{"year-only date string", 0, "2020", 2020, 0, 0},
		{"year-month", 0, "2020-07", 2020, 7, 0},
		{"invalid month ignored", 2021, "2021-13-01", 2021, 0, 1},
		{"invalid day ignored", 2021, "2021-06-40", 2021, 6, 0},
		{"non-numeric year keeps fallback", 2022, "abc-01-01", 2022, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub := parsePublicationDate(tt.year, tt.dateStr)
			if pub.Year != tt.wantY {
				t.Errorf("Year = %d, want %d", pub.Year, tt.wantY)
			}
			if pub.Month != tt.wantM {
				t.Errorf("Month = %d, want %d", pub.Month, tt.wantM)
			}
			if pub.Day != tt.wD {
				t.Errorf("Day = %d, want %d", pub.Day, tt.wD)
			}
		})
	}
}

func TestSanitizeForCiteKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Smith", "Smith"},
		{"O'Brien", "OBrien"},
		{"van der Waals", "vanderWaals"},
		{"Smith-Jones", "SmithJones"},
		{"José", "José"},
		{"Author 3rd", "Author3rd"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeForCiteKey(tt.input); got != tt.want {
				t.Errorf("sanitizeForCiteKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateTitleSuffix(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"two significant words", "Variational Inference", "vi"},
		{"stop words skipped", "The Origin of Species", "os"},
		{"single significant word padded", "Phylogenetics", "px"},
		{"empty padded", "", "xx"},
		{"all stop words padded", "the of and", "xx"},
		{"leading stop word", "A Neural Network", "nn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateTitleSuffix(tt.title); got != tt.want {
				t.Errorf("generateTitleSuffix(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestGenerateCiteKey(t *testing.T) {
	tests := []struct {
		name  string
		paper S2Paper
		want  string
	}{
		{
			name: "standard",
			paper: S2Paper{
				Authors: []S2Author{{Name: "Cheng Zhang"}},
				Year:    2018,
				Title:   "Variational Inference",
			},
			want: "Zhang2018-vi",
		},
		{
			name: "no authors -> Unknown",
			paper: S2Paper{
				Year:  2020,
				Title: "Mystery Paper",
			},
			want: "Unknown2020-mp",
		},
		{
			name: "no year -> 9999",
			paper: S2Paper{
				Authors: []S2Author{{Name: "Jane Doe"}},
				Title:   "Neural Networks",
			},
			want: "Doe9999-nn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateCiteKey(tt.paper); got != tt.want {
				t.Errorf("generateCiteKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapS2ToReference(t *testing.T) {
	paper := S2Paper{
		PaperID: "abc123",
		ExternalIDs: ExternalIDs{
			DOI:           "10.1038/nature12373",
			ArXiv:         "2106.15928",
			PubMed:        "19872477",
			PubMedCentral: "PMC2323736",
		},
		Title:    "A Great Paper",
		Abstract: "An abstract.",
		Venue:    "Nature",
		Authors:  []S2Author{{Name: "Cheng Zhang"}, {Name: "Jane Doe"}},
		Year:     2018,
		PubDate:  "2018-05-10",
	}

	ref := MapS2ToReference(paper)

	if ref.ID != "Zhang2018-gp" {
		t.Errorf("ID = %q, want %q", ref.ID, "Zhang2018-gp")
	}
	if ref.DOI != "10.1038/nature12373" {
		t.Errorf("DOI = %q, want %q", ref.DOI, "10.1038/nature12373")
	}
	if ref.Title != "A Great Paper" {
		t.Errorf("Title = %q", ref.Title)
	}
	if ref.Abstract != "An abstract." {
		t.Errorf("Abstract = %q", ref.Abstract)
	}
	if ref.Venue != "Nature" {
		t.Errorf("Venue = %q", ref.Venue)
	}
	if ref.PMID != "19872477" {
		t.Errorf("PMID = %q", ref.PMID)
	}
	if ref.PMCID != "PMC2323736" {
		t.Errorf("PMCID = %q", ref.PMCID)
	}
	if ref.ArXivID != "2106.15928" {
		t.Errorf("ArXivID = %q", ref.ArXivID)
	}
	if ref.S2ID != "abc123" {
		t.Errorf("S2ID = %q", ref.S2ID)
	}
	if ref.Source.Type != "s2" || ref.Source.ID != "abc123" {
		t.Errorf("Source = %+v, want {s2 abc123}", ref.Source)
	}
	if ref.Published.Year != 2018 || ref.Published.Month != 5 || ref.Published.Day != 10 {
		t.Errorf("Published = %+v, want 2018-05-10", ref.Published)
	}
	if len(ref.Authors) != 2 {
		t.Fatalf("len(Authors) = %d, want 2", len(ref.Authors))
	}
	if ref.Authors[0].First != "Cheng" || ref.Authors[0].Last != "Zhang" {
		t.Errorf("Authors[0] = %+v, want {Cheng Zhang}", ref.Authors[0])
	}
}
