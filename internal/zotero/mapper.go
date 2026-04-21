package zotero

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/matsen/bipartite/internal/reference"
)

// MapZoteroToReference converts a Zotero API item to a Reference.
func MapZoteroToReference(item ZoteroItem) (reference.Reference, error) {
	data := item.Data

	if data.Title == "" {
		return reference.Reference{}, fmt.Errorf("missing title")
	}
	if len(data.Creators) == 0 {
		return reference.Reference{}, fmt.Errorf("missing creators")
	}

	// Convert creators to authors
	authors := make([]reference.Author, 0, len(data.Creators))
	for _, c := range data.Creators {
		if c.CreatorType != "author" {
			continue
		}
		if c.Name != "" {
			// Institutional/single-field name
			authors = append(authors, reference.Author{Last: c.Name})
		} else {
			authors = append(authors, reference.Author{
				First: c.FirstName,
				Last:  c.LastName,
			})
		}
	}
	if len(authors) == 0 {
		// Fall back to all creators if no "author" type
		for _, c := range data.Creators {
			if c.Name != "" {
				authors = append(authors, reference.Author{Last: c.Name})
			} else {
				authors = append(authors, reference.Author{
					First: c.FirstName,
					Last:  c.LastName,
				})
			}
		}
	}

	// Parse date
	pubDate := parseZoteroDate(data.Date)
	if pubDate.Year == 0 {
		return reference.Reference{}, fmt.Errorf("missing or invalid date")
	}

	// Extract PMID/PMCID/arXiv from Extra field
	pmid, pmcid, arxiv := parseExtra(data.Extra)

	// Generate citation key
	id := generateCiteKey(authors, pubDate.Year, data.Title)

	ref := reference.Reference{
		ID:        id,
		DOI:       data.DOI,
		Title:     data.Title,
		Authors:   authors,
		Abstract:  data.AbstractNote,
		Venue:     data.PublicationTitle,
		Published: pubDate,
		PMID:      pmid,
		PMCID:     pmcid,
		ArXivID:   arxiv,
		Source: reference.ImportSource{
			Type: "zotero",
			ID:   item.Key,
		},
	}

	return ref, nil
}

// MapReferenceToZotero converts a Reference to a ZoteroItemData for creation.
func MapReferenceToZotero(ref reference.Reference) ZoteroItemData {
	creators := make([]ZoteroCreator, len(ref.Authors))
	for i, a := range ref.Authors {
		if a.First == "" && a.Last != "" {
			// Institutional author
			creators[i] = ZoteroCreator{
				CreatorType: "author",
				Name:        a.Last,
			}
		} else {
			creators[i] = ZoteroCreator{
				CreatorType: "author",
				FirstName:   a.First,
				LastName:    a.Last,
			}
		}
	}

	// Build date string
	dateStr := strconv.Itoa(ref.Published.Year)
	if ref.Published.Month > 0 {
		dateStr += fmt.Sprintf("-%02d", ref.Published.Month)
		if ref.Published.Day > 0 {
			dateStr += fmt.Sprintf("-%02d", ref.Published.Day)
		}
	}

	// Build Extra field for PMID/PMCID/arXiv
	var extraParts []string
	if ref.PMID != "" {
		extraParts = append(extraParts, "PMID: "+ref.PMID)
	}
	if ref.PMCID != "" {
		extraParts = append(extraParts, "PMCID: "+ref.PMCID)
	}
	if ref.ArXivID != "" {
		extraParts = append(extraParts, "arXiv: "+ref.ArXivID)
	}

	item := ZoteroItemData{
		ItemType:         "journalArticle",
		Title:            ref.Title,
		Creators:         creators,
		AbstractNote:     ref.Abstract,
		PublicationTitle: ref.Venue,
		Date:             dateStr,
		DOI:              ref.DOI,
		Extra:            strings.Join(extraParts, "\n"),
	}

	return item
}

// parseZoteroDate parses Zotero's free-form date string.
// Handles: "2023-08-22", "2023-08", "2023", "August 22, 2023", etc.
func parseZoteroDate(date string) reference.PublicationDate {
	date = strings.TrimSpace(date)
	if date == "" {
		return reference.PublicationDate{}
	}

	// Try YYYY-MM-DD format first
	parts := strings.Split(date, "-")
	if len(parts) >= 1 {
		if y, err := strconv.Atoi(parts[0]); err == nil && y > 0 {
			pub := reference.PublicationDate{Year: y}
			if len(parts) >= 2 {
				if m, err := strconv.Atoi(parts[1]); err == nil && m >= 1 && m <= 12 {
					pub.Month = m
				}
			}
			if len(parts) >= 3 {
				if d, err := strconv.Atoi(parts[2]); err == nil && d >= 1 && d <= 31 {
					pub.Day = d
				}
			}
			return pub
		}
	}

	// Try to extract a 4-digit year from anywhere in the string
	for _, word := range strings.Fields(date) {
		word = strings.Trim(word, ",.")
		if len(word) == 4 {
			if y, err := strconv.Atoi(word); err == nil && y > 1000 && y < 3000 {
				return reference.PublicationDate{Year: y}
			}
		}
	}

	return reference.PublicationDate{}
}

// parseExtra extracts PMID, PMCID, and arXiv ID from the Extra field.
// Zotero stores these as "PMID: 12345\nPMCID: PMC12345\narXiv: 2106.15928"
func parseExtra(extra string) (pmid, pmcid, arxiv string) {
	for _, line := range strings.Split(extra, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PMID:") {
			pmid = strings.TrimSpace(strings.TrimPrefix(line, "PMID:"))
		} else if strings.HasPrefix(line, "PMCID:") {
			pmcid = strings.TrimSpace(strings.TrimPrefix(line, "PMCID:"))
		} else if strings.HasPrefix(line, "arXiv:") {
			arxiv = strings.TrimSpace(strings.TrimPrefix(line, "arXiv:"))
		}
	}
	return
}

// generateCiteKey generates a citation key from metadata.
// Format: LastName + Year + suffix (e.g., "Zhang2018-vi")
func generateCiteKey(authors []reference.Author, year int, title string) string {
	lastName := "Unknown"
	if len(authors) > 0 && authors[0].Last != "" {
		lastName = sanitizeForCiteKey(authors[0].Last)
	}

	if year == 0 {
		year = 9999
	}

	suffix := generateTitleSuffix(title)
	return fmt.Sprintf("%s%d-%s", lastName, year, suffix)
}

func sanitizeForCiteKey(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func generateTitleSuffix(title string) string {
	words := strings.Fields(strings.ToLower(title))
	stopWords := map[string]bool{"a": true, "an": true, "the": true, "of": true, "and": true, "in": true, "on": true, "for": true, "to": true, "with": true}

	var suffix strings.Builder
	for _, word := range words {
		if !stopWords[word] && len(word) > 0 {
			suffix.WriteByte(word[0])
			if suffix.Len() >= 2 {
				break
			}
		}
	}
	for suffix.Len() < 2 {
		suffix.WriteByte('x')
	}
	return suffix.String()
}
