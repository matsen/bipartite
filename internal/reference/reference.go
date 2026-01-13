// Package reference defines the core domain types for academic references.
package reference

// Reference represents an academic paper or article.
type Reference struct {
	// Identity
	ID  string `json:"id"`  // Internal stable identifier (from citekey)
	DOI string `json:"doi"` // Digital Object Identifier (primary deduplication key)

	// Metadata
	Title    string   `json:"title"`
	Authors  []Author `json:"authors"`
	Abstract string   `json:"abstract"`
	Venue    string   `json:"venue"` // Journal, conference, or preprint server

	// Publication Date
	Published PublicationDate `json:"published"`

	// File Paths (relative to configured PDF root)
	PDFPath         string   `json:"pdf_path"`
	SupplementPaths []string `json:"supplement_paths,omitempty"`

	// Import Tracking
	Source ImportSource `json:"source"`

	// Relationships
	Supersedes string `json:"supersedes,omitempty"` // DOI of paper this replaces
}

// PublicationDate represents a publication date with optional month and day.
type PublicationDate struct {
	Year  int `json:"year"`
	Month int `json:"month,omitempty"` // 1-12, 0 if unknown
	Day   int `json:"day,omitempty"`   // 1-31, 0 if unknown
}

// ImportSource tracks where a reference was imported from.
type ImportSource struct {
	Type string `json:"type"` // paperpile, zotero, mendeley, manual, asta
	ID   string `json:"id"`   // Original ID from source system
}
