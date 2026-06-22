package s2

import "testing"

func TestParsePaperID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantVal  string
	}{
		{"doi", "DOI:10.1038/nature12373", "DOI", "10.1038/nature12373"},
		{"arxiv", "ARXIV:2106.15928", "ARXIV", "2106.15928"},
		{"pmid", "PMID:19872477", "PMID", "19872477"},
		{"pmcid", "PMCID:2323736", "PMCID", "2323736"},
		{"corpusid", "CorpusId:215416146", "CorpusId", "215416146"},
		{"url", "URL:https://arxiv.org/abs/2106.15928", "URL", "https://arxiv.org/abs/2106.15928"},
		{"lowercase prefix", "doi:10.1038/nature12373", "DOI", "10.1038/nature12373"},
		{"raw s2 id", "649def34f8be52c8b66281af98ae884c09aef38b", "S2", "649def34f8be52c8b66281af98ae884c09aef38b"},
		{"leading whitespace", "  DOI:10.1/x  ", "DOI", "10.1/x"},
		{"bare local id", "Zhang2018-vi", "LOCAL", "Zhang2018-vi"},
		{"short hex not s2", "abc123", "LOCAL", "abc123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePaperID(tt.input)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Value != tt.wantVal {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantVal)
			}
		})
	}
}

func TestPaperIdentifierIsExternalID(t *testing.T) {
	tests := []struct {
		typ  string
		want bool
	}{
		{"DOI", true},
		{"ARXIV", true},
		{"S2", true},
		{"PMID", true},
		{"CorpusId", true},
		{"MAG", true},
		{"ACL", true},
		{"LOCAL", false},
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			p := PaperIdentifier{Type: tt.typ, Value: "x"}
			if got := p.IsExternalID(); got != tt.want {
				t.Errorf("IsExternalID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeDOI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "10.1038/Nature12373", "10.1038/nature12373"},
		{"https doi.org", "https://doi.org/10.1038/Nature12373", "10.1038/nature12373"},
		{"http doi.org", "http://doi.org/10.1038/NATURE12373", "10.1038/nature12373"},
		{"bare doi.org", "doi.org/10.1038/nature12373", "10.1038/nature12373"},
		{"DOI prefix", "DOI:10.1038/Nature12373", "10.1038/nature12373"},
		{"surrounding whitespace", "  10.1038/Nature12373  ", "10.1038/nature12373"},
		{"already normalized", "10.1234/foo", "10.1234/foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeDOI(tt.input); got != tt.want {
				t.Errorf("NormalizeDOI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
