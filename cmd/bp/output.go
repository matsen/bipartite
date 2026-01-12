package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Constants for output formatting
const (
	DefaultSearchLimit   = 50
	TitleTruncateLen     = 60
	SummaryTitleLen      = 70
	ListTitleTruncateLen = 50
	TextWrapWidth        = 60
	DetailTextWrapWidth  = 68
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
	PDFRoot   string `json:"pdf_root,omitempty"`
	PDFReader string `json:"pdf_reader,omitempty"`
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

// outputErrorJSON writes an error as JSON and returns the exit code.
func outputErrorJSON(code int, message string) int {
	outputJSON(ErrorResponse{Error: message})
	return code
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
	currentLine := ""

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n"+indent)
}

// formatIDList formats a list of IDs as a comma-separated string.
func formatIDList(ids []string) string {
	return strings.Join(ids, ", ")
}
