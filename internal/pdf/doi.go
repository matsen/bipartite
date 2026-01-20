// Package pdf provides utilities for extracting information from PDF files,
// including DOI extraction and title detection for academic papers.
package pdf

import (
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Common errors returned by PDF extraction functions.
var (
	// ErrNoDOIFound indicates no DOI pattern was found in the PDF.
	ErrNoDOIFound = errors.New("no DOI found in PDF")

	// ErrNoTextExtracted indicates text extraction failed for all pages.
	ErrNoTextExtracted = errors.New("could not extract text from PDF")
)

// DOI pattern: 10.XXXX/... where XXXX is 4+ digits
// More specific: 10.\d{4,9}/[-._;()/:A-Z0-9]+
var doiPattern = regexp.MustCompile(`10\.\d{4,9}/[^\s<>"{}|\\^~\[\]` + "`" + `]+`)

// ExtractDOI extracts a DOI from a PDF file by searching the first few pages.
// Returns the first valid DOI found, or ErrNoDOIFound if no DOI is present.
// Returns other errors for PDF parsing failures.
func ExtractDOI(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Search first 3 pages (DOI is usually on first page)
	maxPages := 3
	if r.NumPage() < maxPages {
		maxPages = r.NumPage()
	}

	extractedText := false
	for i := 1; i <= maxPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		extractedText = true

		if doi := findDOI(text); doi != "" {
			return doi, nil
		}
	}

	if !extractedText {
		return "", ErrNoTextExtracted
	}
	return "", ErrNoDOIFound
}

// ExtractTitle attempts to extract the title from a PDF using heuristics.
// It returns the first substantial line (>20 chars) from the first page that
// doesn't appear to be a header/footer. This is best-effort and may return
// an empty string if no suitable title candidate is found.
func ExtractTitle(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Just get text from first page and use first non-empty line as title
	if r.NumPage() < 1 {
		return "", nil
	}

	page := r.Page(1)
	if page.V.IsNull() {
		return "", nil
	}

	text, err := page.GetPlainText(nil)
	if err != nil {
		return "", nil
	}

	// Find first substantial line (likely title)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip short lines, headers, etc.
		if len(line) > 20 && !isHeaderLine(line) {
			return line, nil
		}
	}

	return "", nil
}

// ExtractText extracts all text from the first N pages of a PDF file.
// If maxPages is 0 or greater than the document length, extracts all pages.
func ExtractText(filePath string, maxPages int) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if maxPages <= 0 || maxPages > r.NumPage() {
		maxPages = r.NumPage()
	}

	var builder strings.Builder
	for i := 1; i <= maxPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		builder.WriteString(text)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// ExtractTextReader extracts text from a PDF via io.ReaderAt interface.
// This is useful when the PDF is already in memory or from a non-file source.
func ExtractTextReader(r io.ReaderAt, size int64, maxPages int) (string, error) {
	pdfReader, err := pdf.NewReader(r, size)
	if err != nil {
		return "", err
	}

	if maxPages <= 0 || maxPages > pdfReader.NumPage() {
		maxPages = pdfReader.NumPage()
	}

	var builder strings.Builder
	for i := 1; i <= maxPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		builder.WriteString(text)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// findDOI finds a DOI in text.
func findDOI(text string) string {
	matches := doiPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}

	// Clean up matches and return the first valid one
	for _, match := range matches {
		// Remove trailing punctuation
		match = strings.TrimRight(match, ".,;:)")
		// Validate it looks like a real DOI
		if isValidDOI(match) {
			return match
		}
	}

	return ""
}

// isValidDOI performs basic validation on a DOI.
func isValidDOI(doi string) bool {
	if len(doi) < 10 {
		return false
	}
	// Must start with 10. and have something after the /
	if !strings.HasPrefix(doi, "10.") {
		return false
	}
	slashIdx := strings.Index(doi, "/")
	if slashIdx == -1 || slashIdx >= len(doi)-1 {
		return false
	}
	return true
}

// isHeaderLine checks if a line is likely a header/footer.
func isHeaderLine(line string) bool {
	lower := strings.ToLower(line)
	// Common header patterns
	if strings.Contains(lower, "journal") {
		return true
	}
	if strings.Contains(lower, "volume") && strings.Contains(lower, "issue") {
		return true
	}
	if strings.Contains(lower, "copyright") {
		return true
	}
	if strings.Contains(lower, "article") && strings.Contains(lower, "published") {
		return true
	}
	return false
}
