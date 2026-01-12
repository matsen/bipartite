// Package importer provides functions to import references from external formats.
package importer

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/matsen/bipartite/internal/reference"
)

// FlexibleString can unmarshal from either string or number JSON values.
type FlexibleString string

func (f *FlexibleString) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		*f = ""
		return nil
	}

	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleString(s)
		return nil
	}

	// Try number
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleString(n.String())
		return nil
	}

	// Try int directly
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*f = FlexibleString(strconv.Itoa(i))
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

func (f FlexibleString) String() string {
	return string(f)
}

// PaperpileEntry represents a single entry from a Paperpile JSON export.
type PaperpileEntry struct {
	ID        string `json:"_id"`
	Citekey   string `json:"citekey"`
	DOI       string `json:"doi"`
	Title     string `json:"title"`
	Abstract  string `json:"abstract"`
	Journal   string `json:"journal"`
	Published struct {
		Year  FlexibleString `json:"year"`
		Month FlexibleString `json:"month"`
		Day   FlexibleString `json:"day"`
	} `json:"published"`
	Author []struct {
		First string `json:"first"`
		Last  string `json:"last"`
		ORCID string `json:"orcid"`
	} `json:"author"`
	Attachments []struct {
		ID         string `json:"_id"`
		ArticlePDF int    `json:"article_pdf"` // 1 = main PDF, 0 = supplement
		Filename   string `json:"filename"`
	} `json:"attachments"`
}

// ParsePaperpile parses a Paperpile JSON export and returns references.
func ParsePaperpile(data []byte) ([]reference.Reference, []error) {
	var entries []PaperpileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, []error{fmt.Errorf("parsing Paperpile JSON: %w", err)}
	}

	var refs []reference.Reference
	var errs []error

	for i, entry := range entries {
		ref, err := paperpileEntryToReference(entry)
		if err != nil {
			errs = append(errs, fmt.Errorf("entry %d (%s): %w", i+1, entry.Citekey, err))
			continue
		}
		refs = append(refs, ref)
	}

	return refs, errs
}

// paperpileEntryToReference converts a Paperpile entry to our Reference type.
func paperpileEntryToReference(entry PaperpileEntry) (reference.Reference, error) {
	// Validate required fields
	if entry.Title == "" {
		return reference.Reference{}, fmt.Errorf("missing required field 'title'")
	}
	if len(entry.Author) == 0 {
		return reference.Reference{}, fmt.Errorf("missing required field 'author'")
	}
	if entry.Published.Year.String() == "" {
		return reference.Reference{}, fmt.Errorf("missing required field 'published.year'")
	}

	// Convert authors
	authors := make([]reference.Author, len(entry.Author))
	for i, a := range entry.Author {
		authors[i] = reference.Author{
			First: a.First,
			Last:  a.Last,
			ORCID: a.ORCID,
		}
	}

	// Convert publication date
	year, err := strconv.Atoi(entry.Published.Year.String())
	if err != nil {
		return reference.Reference{}, fmt.Errorf("invalid year: %s", entry.Published.Year.String())
	}

	pubDate := reference.PublicationDate{Year: year}
	if entry.Published.Month.String() != "" {
		month, err := strconv.Atoi(entry.Published.Month.String())
		if err == nil && month >= 1 && month <= 12 {
			pubDate.Month = month
		}
	}
	if entry.Published.Day.String() != "" {
		day, err := strconv.Atoi(entry.Published.Day.String())
		if err == nil && day >= 1 && day <= 31 {
			pubDate.Day = day
		}
	}

	// Extract PDFs from attachments
	var pdfPath string
	var supplementPaths []string

	for _, att := range entry.Attachments {
		if att.ArticlePDF == 1 {
			pdfPath = att.Filename
		} else {
			supplementPaths = append(supplementPaths, att.Filename)
		}
	}

	// Use citekey as ID, falling back to Paperpile ID if no citekey
	id := entry.Citekey
	if id == "" {
		id = entry.ID
	}

	ref := reference.Reference{
		ID:              id,
		DOI:             entry.DOI,
		Title:           entry.Title,
		Authors:         authors,
		Abstract:        entry.Abstract,
		Venue:           entry.Journal,
		Published:       pubDate,
		PDFPath:         pdfPath,
		SupplementPaths: supplementPaths,
		Source: reference.ImportSource{
			Type: "paperpile",
			ID:   entry.ID,
		},
	}

	return ref, nil
}
