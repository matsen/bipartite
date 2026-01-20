package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/asta"
)

// astaOutputJSON outputs data as JSON to stdout.
func astaOutputJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// astaOutputError outputs an error in JSON or human format and returns the exit code.
func astaOutputError(err error, paperID string) int {
	exitCode := ExitASTAAPIError

	if asta.IsNotFound(err) {
		exitCode = ExitASTANotFound
	} else if asta.IsAuthError(err) {
		exitCode = ExitASTAAuthError
	} else if asta.IsRateLimited(err) {
		exitCode = ExitASTAAPIError
	}

	errCode := "api_error"
	if asta.IsNotFound(err) {
		errCode = "not_found"
	} else if asta.IsAuthError(err) {
		errCode = "auth_error"
	} else if asta.IsRateLimited(err) {
		errCode = "rate_limited"
	}

	if astaHuman {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		if paperID != "" {
			fmt.Fprintf(os.Stderr, "  Paper ID: %s\n", paperID)
		}
	} else {
		errResp := map[string]any{
			"error": map[string]any{
				"code":    errCode,
				"message": err.Error(),
			},
		}
		if paperID != "" {
			errResp["error"].(map[string]any)["paperId"] = paperID
		}
		_ = astaOutputJSON(errResp)
	}

	return exitCode
}

// formatASTAAuthors formats a list of ASTA authors for human display.
func formatASTAAuthors(authors []asta.ASTAAuthor) string {
	if len(authors) == 0 {
		return "Unknown"
	}
	names := make([]string, len(authors))
	for i, a := range authors {
		// Abbreviate to last name + first initial
		parts := strings.Fields(a.Name)
		if len(parts) >= 2 {
			names[i] = parts[len(parts)-1] + " " + string(parts[0][0])
		} else {
			names[i] = a.Name
		}
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
