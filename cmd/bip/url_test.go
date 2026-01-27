package main

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestGenerateURL(t *testing.T) {
	ref := &reference.Reference{
		ID:      "Smith2024-ab",
		DOI:     "10.1234/example",
		PMID:    "12345678",
		PMCID:   "PMC1234567",
		ArXivID: "2106.15928",
		S2ID:    "649def34f8be52c8b66281af98ae884c09aef38b",
	}

	tests := []struct {
		name       string
		format     string
		wantURL    string
		wantErrMsg string
	}{
		{
			name:    "DOI format",
			format:  "doi",
			wantURL: "https://doi.org/10.1234/example",
		},
		{
			name:    "PubMed format",
			format:  "pubmed",
			wantURL: "https://pubmed.ncbi.nlm.nih.gov/12345678/",
		},
		{
			name:    "PMC format",
			format:  "pmc",
			wantURL: "https://www.ncbi.nlm.nih.gov/pmc/articles/PMC1234567/",
		},
		{
			name:    "arXiv format",
			format:  "arxiv",
			wantURL: "https://arxiv.org/abs/2106.15928",
		},
		{
			name:    "S2 format",
			format:  "s2",
			wantURL: "https://www.semanticscholar.org/paper/649def34f8be52c8b66281af98ae884c09aef38b",
		},
		{
			name:       "unknown format",
			format:     "unknown",
			wantErrMsg: "unknown URL format: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := generateURL(ref, tt.format)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErrMsg)
				} else if err.Error() != tt.wantErrMsg {
					t.Errorf("expected error %q, got %q", tt.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tt.wantURL {
				t.Errorf("expected URL %q, got %q", tt.wantURL, url)
			}
		})
	}
}

func TestGenerateURL_MissingIDs(t *testing.T) {
	// Reference with no external IDs
	ref := &reference.Reference{
		ID: "Smith2024-ab",
	}

	tests := []struct {
		format     string
		wantErrMsg string
	}{
		{"doi", "no DOI available for Smith2024-ab"},
		{"pubmed", "no PubMed ID available for Smith2024-ab"},
		{"pmc", "no PMC ID available for Smith2024-ab"},
		{"arxiv", "no arXiv ID available for Smith2024-ab"},
		{"s2", "no Semantic Scholar ID available for Smith2024-ab"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			_, err := generateURL(ref, tt.format)
			if err == nil {
				t.Errorf("expected error for missing %s ID", tt.format)
				return
			}
			if err.Error() != tt.wantErrMsg {
				t.Errorf("expected error %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}

func TestGetSelectedFormat(t *testing.T) {
	// Save original flag values
	origPubmed := urlPubmedFlag
	origPmc := urlPmcFlag
	origArxiv := urlArxivFlag
	origS2 := urlS2Flag

	// Reset flags after test
	defer func() {
		urlPubmedFlag = origPubmed
		urlPmcFlag = origPmc
		urlArxivFlag = origArxiv
		urlS2Flag = origS2
	}()

	tests := []struct {
		name       string
		pubmed     bool
		pmc        bool
		arxiv      bool
		s2         bool
		wantFormat string
		wantErr    bool
	}{
		{
			name:       "no flags = doi default",
			wantFormat: "doi",
		},
		{
			name:       "pubmed flag",
			pubmed:     true,
			wantFormat: "pubmed",
		},
		{
			name:       "pmc flag",
			pmc:        true,
			wantFormat: "pmc",
		},
		{
			name:       "arxiv flag",
			arxiv:      true,
			wantFormat: "arxiv",
		},
		{
			name:       "s2 flag",
			s2:         true,
			wantFormat: "s2",
		},
		{
			name:    "multiple flags = error",
			pubmed:  true,
			pmc:     true,
			wantErr: true,
		},
		{
			name:    "all flags = error",
			pubmed:  true,
			pmc:     true,
			arxiv:   true,
			s2:      true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flags
			urlPubmedFlag = tt.pubmed
			urlPmcFlag = tt.pmc
			urlArxivFlag = tt.arxiv
			urlS2Flag = tt.s2

			format, err := getSelectedFormat()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error for multiple flags")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if format != tt.wantFormat {
				t.Errorf("expected format %q, got %q", tt.wantFormat, format)
			}
		})
	}
}

func TestURLFormats_AllDefined(t *testing.T) {
	// Verify all expected formats are defined
	expectedFormats := []string{"doi", "pubmed", "pmc", "arxiv", "s2"}

	for _, name := range expectedFormats {
		format, ok := urlFormats[name]
		if !ok {
			t.Errorf("urlFormats missing format %q", name)
			continue
		}
		if format.name != name {
			t.Errorf("format %q has mismatched name field: %q", name, format.name)
		}
		if format.template == "" {
			t.Errorf("format %q has empty template", name)
		}
		if format.getID == nil {
			t.Errorf("format %q has nil getID function", name)
		}
		if format.idName == "" {
			t.Errorf("format %q has empty idName", name)
		}
	}
}
