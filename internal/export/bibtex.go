// Package export provides functions to export references to various formats.
package export

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
)

// ToBibTeX converts a reference to BibTeX format.
func ToBibTeX(ref reference.Reference) string {
	entryType := determineEntryType(ref)
	var b strings.Builder

	b.WriteString(fmt.Sprintf("@%s{%s,\n", entryType, ref.ID))

	// Authors
	if len(ref.Authors) > 0 {
		b.WriteString(fmt.Sprintf("  author = {%s},\n", formatAuthors(ref.Authors)))
	}

	// Title
	b.WriteString(fmt.Sprintf("  title = {%s},\n", escapeLatex(ref.Title)))

	// Venue
	if ref.Venue != "" {
		fieldName := "journal"
		if entryType == "inproceedings" {
			fieldName = "booktitle"
		}
		b.WriteString(fmt.Sprintf("  %s = {%s},\n", fieldName, escapeLatex(ref.Venue)))
	}

	// Year
	b.WriteString(fmt.Sprintf("  year = {%d},\n", ref.Published.Year))

	// Month (optional)
	if ref.Published.Month > 0 {
		b.WriteString(fmt.Sprintf("  month = {%d},\n", ref.Published.Month))
	}

	// DOI (optional)
	if ref.DOI != "" {
		b.WriteString(fmt.Sprintf("  doi = {%s},\n", ref.DOI))
	}

	// Abstract (optional, if present)
	if ref.Abstract != "" {
		b.WriteString(fmt.Sprintf("  abstract = {%s},\n", escapeLatex(ref.Abstract)))
	}

	b.WriteString("}\n")

	return b.String()
}

// ToBibTeXList converts multiple references to BibTeX format.
func ToBibTeXList(refs []reference.Reference) string {
	var entries []string
	for _, ref := range refs {
		entries = append(entries, ToBibTeX(ref))
	}
	return strings.Join(entries, "\n")
}

// determineEntryType returns the BibTeX entry type for a reference.
func determineEntryType(ref reference.Reference) string {
	venue := strings.ToLower(ref.Venue)

	// Preprints
	if strings.Contains(venue, "arxiv") ||
		strings.Contains(venue, "biorxiv") ||
		strings.Contains(venue, "medrxiv") {
		return "article"
	}

	// Conference proceedings
	if strings.Contains(venue, "proceedings") ||
		strings.Contains(venue, "conference") ||
		strings.Contains(venue, "workshop") ||
		strings.Contains(venue, "symposium") {
		return "inproceedings"
	}

	// Default to article
	return "article"
}

// formatAuthors formats authors in BibTeX style: "Last, First and Last, First"
func formatAuthors(authors []reference.Author) string {
	var formatted []string
	for _, a := range authors {
		if a.First != "" {
			formatted = append(formatted, fmt.Sprintf("%s, %s", a.Last, a.First))
		} else {
			formatted = append(formatted, a.Last)
		}
	}
	return strings.Join(formatted, " and ")
}

// escapeLatex escapes special LaTeX characters.
func escapeLatex(s string) string {
	// Order matters: & must be first (before other escapes that might produce &)
	replacer := strings.NewReplacer(
		"&", `\&`,
		"%", `\%`,
		"$", `\$`,
		"#", `\#`,
		"_", `\_`,
		"{", `\{`,
		"}", `\}`,
		"~", `\textasciitilde{}`,
		"^", `\textasciicircum{}`,
	)
	return replacer.Replace(s)
}
