package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/asta"
)

// errorMapping holds the exit code and error code for a specific error type.
type errorMapping struct {
	exitCode int
	errCode  string
}

// classifyError determines the exit code and error code for an error.
func classifyError(err error) errorMapping {
	switch {
	case asta.IsNotFound(err):
		return errorMapping{ExitASTANotFound, "not_found"}
	case asta.IsAuthError(err):
		return errorMapping{ExitASTAAuthError, "auth_error"}
	case asta.IsRateLimited(err):
		return errorMapping{ExitASTAAPIError, "rate_limited"}
	default:
		return errorMapping{ExitASTAAPIError, "api_error"}
	}
}

// astaOutputJSON outputs data as JSON to stdout.
func astaOutputJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// astaOutputError outputs an error in JSON or human format and returns the exit code.
func astaOutputError(err error, paperID string) int {
	mapping := classifyError(err)

	if astaHuman {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		if paperID != "" {
			fmt.Fprintf(os.Stderr, "  Paper ID: %s\n", paperID)
		}
	} else {
		errResp := map[string]any{
			"error": map[string]any{
				"code":    mapping.errCode,
				"message": err.Error(),
			},
		}
		if paperID != "" {
			errResp["error"].(map[string]any)["paperId"] = paperID
		}
		_ = astaOutputJSON(errResp)
	}

	return mapping.exitCode
}

// astaExecute is a generic command executor that handles the common pattern of
// calling an ASTA API method and formatting the output.
func astaExecute(
	apiCall func(context.Context, *asta.Client) (any, error),
	humanFormatter func(result any),
	paperID string,
) {
	client := asta.NewClient()
	result, err := apiCall(context.Background(), client)
	if err != nil {
		os.Exit(astaOutputError(err, paperID))
	}

	if astaHuman {
		humanFormatter(result)
	} else {
		if err := astaOutputJSON(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(ExitError)
		}
	}
}

// Input Validation

// ErrEmptyInput is returned when a required input is empty.
var ErrEmptyInput = errors.New("input cannot be empty")

// validateRequiredString validates that a string is not empty after trimming.
func validateRequiredString(value, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s: %w", fieldName, ErrEmptyInput)
	}
	return trimmed, nil
}

// validatePaperID validates and returns a trimmed paper ID.
func validatePaperID(paperID string) (string, error) {
	return validateRequiredString(paperID, "paper ID")
}

// validateAuthorID validates and returns a trimmed author ID.
func validateAuthorID(authorID string) (string, error) {
	return validateRequiredString(authorID, "author ID")
}

// validateQuery validates and returns a trimmed search query.
func validateQuery(query string) (string, error) {
	return validateRequiredString(query, "query")
}

// abbreviateAuthorName converts "First Last" to "Last F" with proper Unicode handling.
func abbreviateAuthorName(name string) string {
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return name
	}

	lastName := parts[len(parts)-1]
	firstName := parts[0]

	// Use runes for proper Unicode handling
	runes := []rune(firstName)
	if len(runes) == 0 {
		return lastName
	}

	return lastName + " " + string(runes[0])
}

// formatASTAAuthors formats a list of ASTA authors for human display.
func formatASTAAuthors(authors []asta.ASTAAuthor) string {
	if len(authors) == 0 {
		return "Unknown"
	}
	names := make([]string, len(authors))
	for i, a := range authors {
		names[i] = abbreviateAuthorName(a.Name)
	}
	if len(names) > 3 {
		return strings.Join(names[:3], ", ") + " et al."
	}
	return strings.Join(names, ", ")
}

// formatPaperHuman formats a paper for human-readable output.
func formatPaperHuman(p asta.ASTAPaper, index int) string {
	var sb strings.Builder
	if index > 0 {
		sb.WriteString(fmt.Sprintf("%d. %s\n", index, p.Title))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n", p.Title))
	}
	sb.WriteString(fmt.Sprintf("   %s (%d)", formatASTAAuthors(p.Authors), p.Year))
	if p.Venue != "" {
		sb.WriteString(fmt.Sprintf(" - %s", p.Venue))
	}
	sb.WriteString("\n")
	if p.CitationCount > 0 || p.IsOpenAccess {
		sb.WriteString("   ")
		if p.CitationCount > 0 {
			sb.WriteString(fmt.Sprintf("Citations: %d", p.CitationCount))
		}
		if p.IsOpenAccess {
			if p.CitationCount > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString("Open Access")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatSnippetHuman formats a snippet for human-readable output.
func formatSnippetHuman(s asta.ASTASnippet, index int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d. [%.2f] %s (%s %d)\n", index, s.Score, s.Paper.Title, formatASTAAuthors(s.Paper.Authors), s.Paper.Year))
	// Indent and wrap the snippet text
	snippet := strings.TrimSpace(s.Snippet)
	if len(snippet) > 200 {
		snippet = snippet[:197] + "..."
	}
	sb.WriteString(fmt.Sprintf("   \"%s\"\n", snippet))
	return sb.String()
}

// formatAuthorHuman formats an author for human-readable output.
func formatAuthorHuman(a asta.ASTAAuthor, index int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d. %s", index, a.Name))
	if a.AuthorID != "" {
		sb.WriteString(fmt.Sprintf(" (ID: %s)", a.AuthorID))
	}
	sb.WriteString("\n")
	if len(a.Affiliations) > 0 {
		sb.WriteString(fmt.Sprintf("   %s\n", strings.Join(a.Affiliations, ", ")))
	}
	sb.WriteString(fmt.Sprintf("   Papers: %d | Citations: %d | h-index: %d\n", a.PaperCount, a.CitationCount, a.HIndex))
	return sb.String()
}

// Human output formatters for use with astaExecute

// formatSearchResultsHuman formats search results for human output.
func formatSearchResultsHuman(result any) {
	r := result.(*asta.SearchResponse)
	if r.Total == 0 {
		fmt.Println("No papers found")
		return
	}
	fmt.Printf("Found %d papers\n\n", r.Total)
	for i, p := range r.Papers {
		fmt.Print(formatPaperHuman(p, i+1))
		fmt.Println()
	}
}

// formatSnippetResultsHuman formats snippet results for human output.
func formatSnippetResultsHuman(result any) {
	r := result.(*asta.SnippetResponse)
	if len(r.Snippets) == 0 {
		fmt.Println("No snippets found")
		return
	}
	fmt.Printf("Found %d snippets\n\n", len(r.Snippets))
	for i, s := range r.Snippets {
		fmt.Print(formatSnippetHuman(s, i+1))
		fmt.Println()
	}
}

// formatPaperDetailsHuman formats paper details for human output.
func formatPaperDetailsHuman(result any) {
	paper := result.(*asta.ASTAPaper)
	fmt.Println(paper.Title)
	fmt.Printf("Authors: %s\n", formatASTAAuthors(paper.Authors))
	if paper.Year > 0 {
		fmt.Printf("Year: %d\n", paper.Year)
	}
	if paper.Venue != "" {
		fmt.Printf("Venue: %s\n", paper.Venue)
	}
	if paper.PublicationDate != "" {
		fmt.Printf("Published: %s\n", paper.PublicationDate)
	}
	if paper.Abstract != "" {
		fmt.Printf("\nAbstract:\n%s\n", paper.Abstract)
	}
	fmt.Println()
	fmt.Printf("Citations: %d | References: %d", paper.CitationCount, paper.ReferenceCount)
	if paper.IsOpenAccess {
		fmt.Print(" | Open Access")
	}
	fmt.Println()
	if len(paper.FieldsOfStudy) > 0 {
		fmt.Printf("Fields: %s\n", strings.Join(paper.FieldsOfStudy, ", "))
	}
	if paper.URL != "" {
		fmt.Printf("URL: %s\n", paper.URL)
	}
}

// formatCitationsHuman formats citations for human output.
func formatCitationsHuman(paperID string) func(result any) {
	return func(result any) {
		r := result.(*asta.CitationsResponse)
		if len(r.Citations) == 0 {
			fmt.Printf("No citations found for %s\n", paperID)
			return
		}
		fmt.Printf("Found %d citations for %s\n\n", r.CitationCount, paperID)
		for i, p := range r.Citations {
			fmt.Print(formatPaperHuman(p, i+1))
			fmt.Println()
		}
	}
}

// formatReferencesHuman formats references for human output.
func formatReferencesHuman(paperID string) func(result any) {
	return func(result any) {
		r := result.(*asta.ReferencesResponse)
		if len(r.References) == 0 {
			fmt.Printf("No references found for %s\n", paperID)
			return
		}
		fmt.Printf("Found %d references for %s\n\n", r.ReferenceCount, paperID)
		for i, p := range r.References {
			fmt.Print(formatPaperHuman(p, i+1))
			fmt.Println()
		}
	}
}

// formatAuthorsHuman formats author search results for human output.
func formatAuthorsHuman(name string) func(result any) {
	return func(result any) {
		r := result.(*asta.AuthorsResponse)
		if len(r.Authors) == 0 {
			fmt.Printf("No authors found for \"%s\"\n", name)
			return
		}
		fmt.Printf("Found %d authors matching \"%s\"\n\n", len(r.Authors), name)
		for i, a := range r.Authors {
			fmt.Print(formatAuthorHuman(a, i+1))
			fmt.Println()
		}
	}
}

// formatAuthorPapersHuman formats author papers for human output.
func formatAuthorPapersHuman(authorID string) func(result any) {
	return func(result any) {
		r := result.(*asta.AuthorPapersResponse)
		if len(r.Papers) == 0 {
			fmt.Printf("No papers found for author %s\n", authorID)
			return
		}
		fmt.Printf("Found %d papers by author %s\n\n", len(r.Papers), authorID)
		for i, p := range r.Papers {
			fmt.Print(formatPaperHuman(p, i+1))
			fmt.Println()
		}
	}
}
