package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/matsen/bipartite/internal/reference"
)

// ASTA-specific API limit constants
const (
	AstaSearchLimit      = 10  // Default search limit for ASTA title searches
	PDFSearchLimit       = 5   // Limit for PDF-based searches
	GapsReferencesLimit  = 500 // Limit for references per paper in gaps analysis
	RateLimitBurstSize   = 1   // Rate limit burst size
	MinTitlePrefixLength = 30  // Minimum chars for prefix title matching
	MinAuthorMatchCount  = 2   // Require at least 2 authors to match (or 1 for single-author papers)
)

// formatAuthors converts reference authors to display strings.
func formatAuthors(authors []reference.Author) []string {
	result := make([]string, 0, len(authors))
	for _, a := range authors {
		if a.First != "" {
			result = append(result, a.First+" "+a.Last)
		} else {
			result = append(result, a.Last)
		}
	}
	return result
}

// capitalizeFirst capitalizes the first letter of a string using proper Unicode handling.
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// joinAuthorsDisplay formats a list of author names for human display.
func joinAuthorsDisplay(authors []string) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) == 1 {
		return authors[0]
	}
	if len(authors) == 2 {
		return authors[0] + " and " + authors[1]
	}
	result := ""
	for i, a := range authors {
		if i == len(authors)-1 {
			result += "and " + a
		} else {
			result += a + ", "
		}
	}
	return result
}

// GenericErrorResult is a generic error result that can be embedded in any result type.
type GenericErrorResult struct {
	Error *AstaErrorResult `json:"error,omitempty"`
}

// outputGenericError outputs an error in both human and JSON format and exits.
func outputGenericError(exitCode int, errCode, context string, err error) error {
	msg := context
	if err != nil {
		msg = fmt.Sprintf("%s: %v", context, err)
	}

	result := GenericErrorResult{
		Error: &AstaErrorResult{
			Code:    errCode,
			Message: msg,
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	} else {
		outputJSON(result)
	}
	os.Exit(exitCode)
	return nil
}

// outputGenericNotFound outputs a not-found error in both human and JSON format and exits.
func outputGenericNotFound(paperID, message string) error {
	result := GenericErrorResult{
		Error: &AstaErrorResult{
			Code:       "not_found",
			Message:    message,
			PaperID:    paperID,
			Suggestion: "Verify the paper ID is correct",
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
		fmt.Fprintf(os.Stderr, "  Paper ID: %s\n", paperID)
	} else {
		outputJSON(result)
	}
	os.Exit(ExitAstaNotFound)
	return nil
}

// outputGenericRateLimited outputs a rate limit error in both human and JSON format and exits.
func outputGenericRateLimited(err error) error {
	apiErr, _ := err.(*asta.APIError)
	retryAfter := 300
	if apiErr != nil && apiErr.RetryAfter > 0 {
		retryAfter = apiErr.RetryAfter
	}

	result := GenericErrorResult{
		Error: &AstaErrorResult{
			Code:       "rate_limited",
			Message:    "Semantic Scholar rate limit exceeded",
			Suggestion: fmt.Sprintf("Wait %d seconds or add S2_API_KEY to .env", retryAfter),
			RetryAfter: retryAfter,
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: Rate limit exceeded\n")
		fmt.Fprintf(os.Stderr, "  Wait %d seconds before retrying\n", retryAfter)
	} else {
		outputJSON(result)
	}
	os.Exit(ExitAstaAPIError)
	return nil
}

// warnAPIError logs a warning for API errors that are being skipped (not fatal).
func warnAPIError(context string, paperID string, err error) {
	if humanOutput {
		fmt.Fprintf(os.Stderr, "Warning: %s for %s: %v\n", context, paperID, err)
	}
}

// titlesMatchStrict compares two titles with stricter matching rules.
// Requires exact match after normalization, or substantial prefix overlap.
func titlesMatchStrict(t1, t2 string) bool {
	norm1 := normalizeTitleStrict(t1)
	norm2 := normalizeTitleStrict(t2)

	// Exact match after normalization
	if norm1 == norm2 {
		return true
	}

	// Only treat as match if the shorter title is substantial (>30 chars)
	// and is a prefix of the longer one
	if len(norm1) > MinTitlePrefixLength && strings.HasPrefix(norm2, norm1) {
		return true
	}
	if len(norm2) > MinTitlePrefixLength && strings.HasPrefix(norm1, norm2) {
		return true
	}

	return false
}

// normalizeTitleStrict normalizes a title for comparison.
func normalizeTitleStrict(title string) string {
	title = strings.ToLower(title)
	title = strings.TrimSpace(title)
	// Remove common punctuation
	title = strings.ReplaceAll(title, ":", " ")
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "  ", " ")
	return title
}

// authorsOverlapStrict checks if authors from a reference match S2 authors.
// Returns false if we can't verify (empty lists), to avoid false positives.
// Requires at least 2 authors to match (or 1 for single-author papers).
func authorsOverlapStrict(refAuthors []reference.Author, s2Authors []asta.S2Author) bool {
	// If either list is empty, we can't verify - be conservative
	if len(refAuthors) == 0 || len(s2Authors) == 0 {
		return false
	}

	matchCount := 0
	for _, refAuth := range refAuthors {
		for _, s2Auth := range s2Authors {
			if authorNamesMatch(refAuth, s2Auth.Name) {
				matchCount++
				break
			}
		}
		// Early exit conditions
		if matchCount >= MinAuthorMatchCount {
			return true
		}
		if matchCount >= 1 && len(refAuthors) == 1 {
			return true
		}
	}

	return false
}

// authorNamesMatch checks if a reference author matches an S2 author name.
// Uses word boundary matching to avoid substring false positives.
func authorNamesMatch(refAuth reference.Author, s2Name string) bool {
	lastNameLower := strings.ToLower(refAuth.Last)
	s2Lower := strings.ToLower(s2Name)

	// Require word boundary match to avoid substring false positives
	// e.g., "Lee" should not match "Charleston Lee" or "Lees"
	return strings.Contains(s2Lower, " "+lastNameLower) ||
		strings.HasPrefix(s2Lower, lastNameLower+" ") ||
		strings.HasSuffix(s2Lower, " "+lastNameLower) ||
		s2Lower == lastNameLower
}
