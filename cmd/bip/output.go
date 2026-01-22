package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/semantic"
	"github.com/matsen/bipartite/internal/storage"
)

// Constants for output formatting.
// Names indicate the context where each constant is used.
const (
	DefaultSearchLimit = 50 // Default limit for search/list commands

	// Title truncation lengths by context
	ImportTitleMaxLen = 60 // Used in import command output
	SearchTitleMaxLen = 70 // Used in search result summaries
	ListTitleMaxLen   = 50 // Used in list command output
	DetailTitleMaxLen = 70 // Used in get command detail view

	// Text wrapping widths
	TextWrapWidth       = 60 // Standard text wrap width
	DetailTextWrapWidth = 68 // Wider wrap for detail views
)

// outputJSON writes a value as formatted JSON to stdout.
func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// outputJSONCompact writes a value as compact JSON to stdout.
func outputJSONCompact(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(v)
}

// outputHuman writes a human-readable string to stdout.
func outputHuman(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// outputError writes an error message to stderr and returns the exit code.
func outputError(code int, format string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	return code
}

// exitWithError outputs an error in the appropriate format (human or JSON) and exits.
func exitWithError(code int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if humanOutput {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	} else {
		outputJSON(ErrorResponse{Error: msg})
	}
	os.Exit(code)
}

// StatusResponse is a generic response for commands that return status.
type StatusResponse struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

// ConfigResponse is the response for config get commands.
type ConfigResponse struct {
	PDFRoot    string `json:"pdf_root,omitempty"`
	PDFReader  string `json:"pdf_reader,omitempty"`
	PapersRepo string `json:"papers_repo,omitempty"`
}

// UpdateResponse is the response for config set commands.
type UpdateResponse struct {
	Status string `json:"status"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// ErrorResponse is a JSON error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// PaperSearchResult represents a paper in search results (semantic search and similar papers).
type PaperSearchResult struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Authors    []reference.Author `json:"authors"`
	Year       int                `json:"year"`
	Similarity float32            `json:"similarity"`
	Abstract   string             `json:"abstract,omitempty"`
}

// printSearchResultsHuman prints search results in human-readable format.
// This is used by both semantic search and similar papers commands.
func printSearchResultsHuman(results []PaperSearchResult) {
	for i, r := range results {
		fmt.Printf("%d. [%.2f] %s\n", i+1, r.Similarity, r.ID)
		fmt.Printf("   %s\n", truncateString(r.Title, SearchTitleMaxLen))
		fmt.Printf("   %s (%d)\n\n", formatAuthorsShort(r.Authors, 3), r.Year)
	}
}

// buildSearchResults converts semantic search results to PaperSearchResult slice.
// Set includeAbstract to true to populate the Abstract field.
//
// Papers that exist in the semantic index but are not found in the database
// (e.g., deleted after indexing) are silently skipped. This graceful degradation
// allows search to return partial results rather than failing entirely.
func buildSearchResults(results []semantic.SearchResult, db *storage.DB, includeAbstract bool) []PaperSearchResult {
	paperResults := make([]PaperSearchResult, 0, len(results))
	for _, r := range results {
		ref, err := db.GetByID(r.PaperID)
		if err != nil || ref == nil {
			continue // Skip papers deleted from DB after indexing
		}
		result := PaperSearchResult{
			ID:         ref.ID,
			Title:      ref.Title,
			Authors:    ref.Authors,
			Year:       ref.Published.Year,
			Similarity: r.Similarity,
		}
		if includeAbstract {
			result.Abstract = ref.Abstract
		}
		paperResults = append(paperResults, result)
	}
	return paperResults
}

// truncateString truncates a string to maxLen, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// wrapText wraps text to the specified width with indentation on subsequent lines.
func wrapText(text string, width int, indent string) string {
	if len(text) <= width {
		return text
	}

	var lines []string
	words := strings.Fields(text)
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n"+indent)
}

// formatIDList formats a list of IDs as a comma-separated string.
func formatIDList(ids []string) string {
	return strings.Join(ids, ", ")
}

// formatAuthorFull formats an author as "First Last".
func formatAuthorFull(a reference.Author) string {
	if a.First != "" {
		return a.First + " " + a.Last
	}
	return a.Last
}

// formatAuthorShort formats an author as "Last F" (abbreviated first name).
func formatAuthorShort(a reference.Author) string {
	if a.First != "" {
		return a.Last + " " + string(a.First[0])
	}
	return a.Last
}

// formatAuthorsFull formats all authors as "First Last, First Last, ...".
func formatAuthorsFull(authors []reference.Author) string {
	names := make([]string, len(authors))
	for i, a := range authors {
		names[i] = formatAuthorFull(a)
	}
	return strings.Join(names, ", ")
}

// formatAuthorsShort formats authors with abbreviation and "et al." for more than maxCount.
func formatAuthorsShort(authors []reference.Author, maxCount int) string {
	if len(authors) == 0 {
		return ""
	}

	var names []string
	for i, a := range authors {
		if i >= maxCount {
			names = append(names, "et al.")
			break
		}
		names = append(names, formatAuthorShort(a))
	}
	return strings.Join(names, ", ")
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// formatBytes formats bytes in a human-readable way.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatConceptHuman formats a concept for human-readable output.
// The prefix parameter is prepended to each line (e.g., "Created concept: ", "").
func formatConceptHuman(id, name string, aliases []string, description, prefix string) string {
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(id)
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  Name: %s\n", name))
	if len(aliases) > 0 {
		sb.WriteString(fmt.Sprintf("  Aliases: %s\n", strings.Join(aliases, ", ")))
	}
	if description != "" {
		sb.WriteString(fmt.Sprintf("  Description: %s\n", description))
	}
	return sb.String()
}

// formatEdgesGroupedByType formats paper-concept edges grouped by relationship type.
// Used by both concept papers and paper concepts commands.
// The idGetter function extracts the relevant ID from each edge (paper_id or concept_id).
func formatEdgesGroupedByType(edges []storage.PaperConceptEdge, idGetter func(storage.PaperConceptEdge) string) string {
	if len(edges) == 0 {
		return ""
	}

	// Group by relationship type
	byType := make(map[string][]storage.PaperConceptEdge)
	for _, e := range edges {
		byType[e.RelationshipType] = append(byType[e.RelationshipType], e)
	}

	var sb strings.Builder
	for relType, typeEdges := range byType {
		sb.WriteString(fmt.Sprintf("\n[%s]\n", relType))
		for _, e := range typeEdges {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", idGetter(e), e.Summary))
		}
	}
	return sb.String()
}
