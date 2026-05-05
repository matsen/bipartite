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
	Note         string   `json:"note"`
	LabelsNamed  []string `json:"labelsNamed"`
	FoldersNamed []string `json:"foldersNamed"`
}

// ImportWarning describes a non-fatal issue with an imported entry.
// In lenient mode, missing required fields are filled with sentinels and a
// warning is recorded so the user can see what was defaulted.
type ImportWarning struct {
	ID      string   `json:"id"`      // bip ID assigned to the imported reference
	Citekey string   `json:"citekey"` // Paperpile citekey (may equal ID)
	Title   string   `json:"title"`   // Title (post-fallback) for human identification
	Fields  []string `json:"fields"`  // Names of required fields that hit fallbacks
}

// String renders an ImportWarning for human-readable output.
func (w ImportWarning) String() string {
	title := w.Title
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	return fmt.Sprintf("entry %s (%q): defaulted %v", w.ID, title, w.Fields)
}

// Sentinel values used when a required field is missing in lenient mode.
const (
	UnknownYear   = 0            // PublicationDate.Year sentinel for "year unknown"
	UnknownAuthor = "Unknown"    // Author.Last sentinel for "author unknown"
	UnknownTitle  = "[no title]" // Title sentinel for "title unknown"
)

// ParsePaperpile parses a Paperpile JSON export and returns references.
//
// When strict is true, entries missing any of {title, author, published.year}
// are dropped and reported in errs. When strict is false, those entries are
// imported with sentinel values and reported in warnings instead — except for
// "junk" entries with no useful metadata at all (no title, author, year, or
// DOI), which are still dropped to avoid flooding the nexus with placeholders.
func ParsePaperpile(data []byte, strict bool) (refs []reference.Reference, warnings []ImportWarning, errs []error) {
	var entries []PaperpileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, nil, []error{fmt.Errorf("parsing Paperpile JSON: %w", err)}
	}

	for i, entry := range entries {
		ref, w, err := paperpileEntryToReference(entry, strict)
		if err != nil {
			errs = append(errs, fmt.Errorf("entry %d (%s): %w", i+1, entry.Citekey, err))
			continue
		}
		refs = append(refs, ref)
		if w != nil {
			warnings = append(warnings, *w)
		}
	}

	return refs, warnings, errs
}

// paperpileEntryToReference converts a Paperpile entry to our Reference type.
//
// Returns an error if the entry should be dropped. Returns a non-nil
// ImportWarning when missing fields were filled with sentinels.
func paperpileEntryToReference(entry PaperpileEntry, strict bool) (reference.Reference, *ImportWarning, error) {
	missingTitle := entry.Title == ""
	missingAuthor := len(entry.Author) == 0
	missingYear := entry.Published.Year.String() == ""

	if strict {
		if missingTitle {
			return reference.Reference{}, nil, fmt.Errorf("missing required field 'title'")
		}
		if missingAuthor {
			return reference.Reference{}, nil, fmt.Errorf("missing required field 'author'")
		}
		if missingYear {
			return reference.Reference{}, nil, fmt.Errorf("missing required field 'published.year'")
		}
	} else if missingTitle && missingAuthor && missingYear && entry.DOI == "" {
		// Junk heuristic: no useful identifying metadata at all (e.g. Paperpile's
		// auto-generated stubs for unparsed web pages). Skip rather than flood
		// the nexus with "[no title] / Unknown / 0" placeholders.
		return reference.Reference{}, nil, fmt.Errorf("entry has no usable metadata (no title, author, year, or DOI)")
	}

	var fallbackFields []string

	// Title fallback
	title := entry.Title
	if missingTitle {
		title = UnknownTitle
		fallbackFields = append(fallbackFields, "title")
	}

	// Author fallback
	var authors []reference.Author
	if missingAuthor {
		authors = []reference.Author{{Last: UnknownAuthor}}
		fallbackFields = append(fallbackFields, "author")
	} else {
		authors = make([]reference.Author, len(entry.Author))
		for i, a := range entry.Author {
			authors[i] = reference.Author{
				First: a.First,
				Last:  a.Last,
				ORCID: a.ORCID,
			}
		}
	}

	// Year fallback
	var year int
	if missingYear {
		year = UnknownYear
		fallbackFields = append(fallbackFields, "published.year")
	} else {
		var err error
		year, err = strconv.Atoi(entry.Published.Year.String())
		if err != nil {
			return reference.Reference{}, nil, fmt.Errorf("invalid year: %s", entry.Published.Year.String())
		}
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

	// Collect tags from labels and folders (deduplicated)
	var tags []string
	seen := make(map[string]bool)
	for _, label := range entry.LabelsNamed {
		if label != "" && !seen[label] {
			tags = append(tags, label)
			seen[label] = true
		}
	}
	for _, folder := range entry.FoldersNamed {
		if folder != "" && !seen[folder] {
			tags = append(tags, folder)
			seen[folder] = true
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
		Title:           title,
		Authors:         authors,
		Abstract:        entry.Abstract,
		Venue:           entry.Journal,
		Note:            entry.Note,
		Tags:            tags,
		Published:       pubDate,
		PDFPath:         pdfPath,
		SupplementPaths: supplementPaths,
		Source: reference.ImportSource{
			Type: "paperpile",
			ID:   entry.ID,
		},
	}

	var warning *ImportWarning
	if len(fallbackFields) > 0 {
		warning = &ImportWarning{
			ID:      id,
			Citekey: entry.Citekey,
			Title:   title,
			Fields:  fallbackFields,
		}
	}

	return ref, warning, nil
}
