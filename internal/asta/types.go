// Package asta provides a client for the Semantic Scholar Academic Graph API.
package asta

// S2Paper represents a paper from the Semantic Scholar API.
type S2Paper struct {
	PaperID      string      `json:"paperId"`
	ExternalIDs  ExternalIDs `json:"externalIds,omitempty"`
	Title        string      `json:"title"`
	Abstract     string      `json:"abstract,omitempty"`
	Authors      []S2Author  `json:"authors,omitempty"`
	Year         int         `json:"year,omitempty"`
	Venue        string      `json:"venue,omitempty"`
	PubDate      string      `json:"publicationDate,omitempty"` // YYYY-MM-DD format
	Citations    int         `json:"citationCount,omitempty"`
	References   int         `json:"referenceCount,omitempty"`
	IsOpenAccess bool        `json:"isOpenAccess,omitempty"`
	Fields       []string    `json:"fieldsOfStudy,omitempty"`
}

// ExternalIDs contains various external identifiers for a paper.
type ExternalIDs struct {
	DOI           string `json:"DOI,omitempty"`
	ArXiv         string `json:"ArXiv,omitempty"`
	PubMed        string `json:"PubMed,omitempty"`
	PubMedCentral string `json:"PubMedCentral,omitempty"`
	CorpusID      int    `json:"CorpusId,omitempty"`
}

// S2Author represents an author from the Semantic Scholar API.
type S2Author struct {
	AuthorID string `json:"authorId,omitempty"`
	Name     string `json:"name"`
}

// PaperIdentifier represents a parsed paper identifier.
type PaperIdentifier struct {
	Type  string // DOI, ARXIV, PMID, PMCID, CorpusId, S2, URL
	Value string // The identifier value
}

// String returns the S2 API format for the identifier.
func (p PaperIdentifier) String() string {
	switch p.Type {
	case "S2":
		return p.Value // Raw S2 ID doesn't need prefix
	default:
		return p.Type + ":" + p.Value
	}
}

// CitationResult represents a citation or reference in API responses.
type CitationResult struct {
	CitingPaper *S2Paper `json:"citingPaper,omitempty"` // For citations endpoint
	CitedPaper  *S2Paper `json:"citedPaper,omitempty"`  // For references endpoint
}

// CitationsResponse is the response from the citations endpoint.
type CitationsResponse struct {
	Offset int              `json:"offset"`
	Next   int              `json:"next,omitempty"`
	Data   []CitationResult `json:"data"`
}

// ReferencesResponse is the response from the references endpoint.
type ReferencesResponse struct {
	Offset int              `json:"offset"`
	Next   int              `json:"next,omitempty"`
	Data   []CitationResult `json:"data"`
}

// SearchResponse is the response from the paper search endpoint.
type SearchResponse struct {
	Total  int       `json:"total"`
	Offset int       `json:"offset"`
	Next   int       `json:"next,omitempty"`
	Data   []S2Paper `json:"data"`
}

// PaperBatchRequest is the request body for the batch paper lookup.
type PaperBatchRequest struct {
	IDs []string `json:"ids"`
}
