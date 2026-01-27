package s2

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/matsen/bipartite/internal/reference"
)

// Common name suffixes to keep with the last name.
var nameSuffixes = map[string]bool{
	"jr":   true,
	"jr.":  true,
	"sr":   true,
	"sr.":  true,
	"ii":   true,
	"iii":  true,
	"iv":   true,
	"v":    true,
	"phd":  true,
	"ph.d": true,
	"md":   true,
	"m.d":  true,
}

// MapS2ToReference converts an S2Paper to a Reference.
func MapS2ToReference(paper S2Paper) reference.Reference {
	ref := reference.Reference{
		ID:       generateCiteKey(paper),
		DOI:      paper.ExternalIDs.DOI,
		Title:    paper.Title,
		Abstract: paper.Abstract,
		Venue:    paper.Venue,
		Authors:  mapAuthors(paper.Authors),
		Source: reference.ImportSource{
			Type: "s2",
			ID:   paper.PaperID,
		},
		// External identifiers
		PMID:    paper.ExternalIDs.PubMed,
		PMCID:   paper.ExternalIDs.PubMedCentral,
		ArXivID: paper.ExternalIDs.ArXiv,
		S2ID:    paper.PaperID,
	}

	// Parse publication date
	ref.Published = parsePublicationDate(paper.Year, paper.PubDate)

	return ref
}

// mapAuthors converts S2 authors to Reference authors.
func mapAuthors(s2Authors []S2Author) []reference.Author {
	authors := make([]reference.Author, 0, len(s2Authors))
	for _, a := range s2Authors {
		first, last := splitAuthorName(a.Name)
		authors = append(authors, reference.Author{
			First: first,
			Last:  last,
		})
	}
	return authors
}

// splitAuthorName splits a full name into first and last name.
// Handles common suffixes (Jr, Sr, II, III, IV, PhD, MD).
//
// Known limitations:
// - Multi-part surnames (von Neumann, van der Waals) split incorrectly
// - Non-Western name formats may not be handled correctly
// - Middle names are included in the first name
func splitAuthorName(name string) (first, last string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}

	parts := strings.Fields(name)
	if len(parts) == 1 {
		// Single name (e.g., "Madonna")
		return "", parts[0]
	}

	// Check if the last part is a suffix
	lastPart := strings.ToLower(parts[len(parts)-1])
	if nameSuffixes[lastPart] && len(parts) > 2 {
		// Keep suffix with last name
		last = parts[len(parts)-2] + " " + parts[len(parts)-1]
		first = strings.Join(parts[:len(parts)-2], " ")
	} else {
		// Standard split: last part is last name
		last = parts[len(parts)-1]
		first = strings.Join(parts[:len(parts)-1], " ")
	}

	return first, last
}

// parsePublicationDate parses year and optional date string.
func parsePublicationDate(year int, dateStr string) reference.PublicationDate {
	pub := reference.PublicationDate{Year: year}

	if dateStr == "" {
		return pub
	}

	// Parse YYYY-MM-DD format
	parts := strings.Split(dateStr, "-")
	if len(parts) >= 1 {
		if y, err := strconv.Atoi(parts[0]); err == nil {
			pub.Year = y
		}
	}
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

// generateCiteKey generates a citation key from paper metadata.
// Format: LastName + Year + suffix (e.g., "Zhang2018-vi")
// Note: Not guaranteed globally unique - caller should use storage.GenerateUniqueID()
// to handle collisions before persisting.
func generateCiteKey(paper S2Paper) string {
	lastName := "Unknown"
	if len(paper.Authors) > 0 {
		_, last := splitAuthorName(paper.Authors[0].Name)
		// Remove spaces and special chars from last name
		lastName = sanitizeForCiteKey(last)
	}

	year := paper.Year
	if year == 0 {
		year = 9999
	}

	// Generate a short suffix from the title
	suffix := generateTitleSuffix(paper.Title)

	return fmt.Sprintf("%s%d-%s", lastName, year, suffix)
}

// sanitizeForCiteKey removes non-alphanumeric characters.
func sanitizeForCiteKey(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// generateTitleSuffix creates a 2-letter suffix from the title.
func generateTitleSuffix(title string) string {
	// Get first letters of first few significant words
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

	// Pad if needed
	for suffix.Len() < 2 {
		suffix.WriteByte('x')
	}

	return suffix.String()
}
