package importer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
)

// CSLItem represents a single entry from a CSL-JSON export (Zotero's native export format).
type CSLItem struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	CitationKey    string    `json:"citation-key"`
	Title          string    `json:"title"`
	DOI            string    `json:"DOI"`
	Abstract       string    `json:"abstract"`
	ContainerTitle string    `json:"container-title"`
	Note           string    `json:"note"`
	PMID           string    `json:"PMID"`
	PMCID          string    `json:"PMCID"`
	Author         []CSLName `json:"author"`
	Issued         CSLDate   `json:"issued"`
	Volume         string    `json:"volume"`
	Issue          string    `json:"issue"`
}

// CSLName represents an author in CSL-JSON format.
type CSLName struct {
	Family  string `json:"family"`
	Given   string `json:"given"`
	Literal string `json:"literal"` // For institutional authors
}

// CSLDate represents a date in CSL-JSON format.
type CSLDate struct {
	DateParts [][]json.Number `json:"date-parts"`
}

// ParseZotero parses a CSL-JSON export (as produced by Zotero) and returns references.
func ParseZotero(data []byte) ([]reference.Reference, []error) {
	var items []CSLItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, []error{fmt.Errorf("parsing CSL-JSON: %w", err)}
	}

	var refs []reference.Reference
	var errs []error

	for i, item := range items {
		ref, err := cslItemToReference(item)
		if err != nil {
			label := item.CitationKey
			if label == "" {
				label = item.ID
			}
			errs = append(errs, fmt.Errorf("entry %d (%s): %w", i+1, label, err))
			continue
		}
		refs = append(refs, ref)
	}

	return refs, errs
}

// cslItemToReference converts a CSL-JSON item to a Reference.
func cslItemToReference(item CSLItem) (reference.Reference, error) {
	// Validate required fields
	if item.Title == "" {
		return reference.Reference{}, fmt.Errorf("missing required field 'title'")
	}
	if len(item.Author) == 0 {
		return reference.Reference{}, fmt.Errorf("missing required field 'author'")
	}

	// Parse publication date
	pubDate, err := parseCSLDate(item.Issued)
	if err != nil {
		return reference.Reference{}, err
	}

	// Convert authors
	authors := make([]reference.Author, len(item.Author))
	for i, a := range item.Author {
		if a.Literal != "" {
			// Institutional author: store in Last, leave First empty
			authors[i] = reference.Author{Last: a.Literal}
		} else {
			authors[i] = reference.Author{
				First: a.Given,
				Last:  a.Family,
			}
		}
	}

	// Determine ID: prefer citation-key, fall back to Zotero item key from URL
	id := item.CitationKey
	if id == "" {
		id = extractZoteroItemKey(item.ID)
	}

	// Extract source ID (Zotero item key)
	sourceID := extractZoteroItemKey(item.ID)

	ref := reference.Reference{
		ID:        id,
		DOI:       item.DOI,
		Title:     item.Title,
		Authors:   authors,
		Abstract:  item.Abstract,
		Venue:     item.ContainerTitle,
		Note:      item.Note,
		Published: pubDate,
		PMID:      item.PMID,
		PMCID:     item.PMCID,
		Source: reference.ImportSource{
			Type: "zotero",
			ID:   sourceID,
		},
	}

	return ref, nil
}

// parseCSLDate parses a CSL-JSON date object into a PublicationDate.
func parseCSLDate(d CSLDate) (reference.PublicationDate, error) {
	if len(d.DateParts) == 0 || len(d.DateParts[0]) == 0 {
		return reference.PublicationDate{}, fmt.Errorf("missing required field 'issued' (date)")
	}

	parts := d.DateParts[0]

	year, err := strconv.Atoi(parts[0].String())
	if err != nil || year == 0 {
		return reference.PublicationDate{}, fmt.Errorf("invalid or missing year in 'issued'")
	}

	pub := reference.PublicationDate{Year: year}

	if len(parts) >= 2 {
		month, err := strconv.Atoi(parts[1].String())
		if err == nil && month >= 1 && month <= 12 {
			pub.Month = month
		}
	}

	if len(parts) >= 3 {
		day, err := strconv.Atoi(parts[2].String())
		if err == nil && day >= 1 && day <= 31 {
			pub.Day = day
		}
	}

	return pub, nil
}

// extractZoteroItemKey extracts the 8-char item key from a Zotero ID URL.
// e.g., "http://zotero.org/users/12345/items/ABCD1234" → "ABCD1234"
// If the ID is not a URL, returns it as-is.
func extractZoteroItemKey(id string) string {
	const prefix = "/items/"
	if idx := strings.Index(id, prefix); idx != -1 {
		return id[idx+len(prefix):]
	}
	return id
}
