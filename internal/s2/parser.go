package s2

import (
	"regexp"
	"strings"
)

// Common identifier prefixes supported by Semantic Scholar.
var identifierPrefixes = []string{
	"DOI:",
	"ARXIV:",
	"PMID:",
	"PMCID:",
	"CorpusId:",
	"URL:",
	"MAG:",
	"ACL:",
}

// s2IDPattern matches a 40-character hex string (raw S2 paper ID).
var s2IDPattern = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// ParsePaperID parses a paper identifier string into a PaperIdentifier.
// Supports formats:
//   - DOI:10.1038/nature12373
//   - ARXIV:2106.15928
//   - PMID:19872477
//   - PMCID:2323736
//   - CorpusId:215416146
//   - URL:https://arxiv.org/abs/2106.15928
//   - Raw 40-character S2 paper ID
func ParsePaperID(id string) PaperIdentifier {
	id = strings.TrimSpace(id)

	// Check for known prefixes
	for _, prefix := range identifierPrefixes {
		if strings.HasPrefix(strings.ToUpper(id), strings.ToUpper(prefix)) {
			return PaperIdentifier{
				Type:  strings.TrimSuffix(prefix, ":"),
				Value: id[len(prefix):],
			}
		}
	}

	// Check for raw S2 ID (40 hex characters)
	if s2IDPattern.MatchString(id) {
		return PaperIdentifier{
			Type:  "S2",
			Value: id,
		}
	}

	// Assume it might be a local ID that needs resolution
	return PaperIdentifier{
		Type:  "LOCAL",
		Value: id,
	}
}

// IsExternalID returns true if the identifier represents an external
// paper ID (DOI, ArXiv, PMID, etc.) rather than a local collection ID.
func (p PaperIdentifier) IsExternalID() bool {
	return p.Type != "LOCAL"
}

// NormalizeDOI normalizes a DOI to a consistent format for comparison.
// It removes common URL prefixes (https://doi.org/, DOI:) and converts to lowercase.
func NormalizeDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	doi = strings.TrimPrefix(doi, "https://doi.org/")
	doi = strings.TrimPrefix(doi, "http://doi.org/")
	doi = strings.TrimPrefix(doi, "doi.org/")
	doi = strings.TrimPrefix(doi, "DOI:")
	return strings.ToLower(doi)
}
