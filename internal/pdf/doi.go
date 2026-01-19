package pdf

import (
	"io"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// DOI pattern: 10.XXXX/... where XXXX is 4+ digits
// More specific: 10.\d{4,9}/[-._;()/:A-Z0-9]+
var doiPattern = regexp.MustCompile(`10\.\d{4,9}/[^\s<>"{}|\\^~\[\]` + "`" + `]+`)

// ExtractDOI extracts a DOI from a PDF file.
// It searches the first few pages for DOI patterns.
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

	for i := 1; i <= maxPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		if doi := findDOI(text); doi != "" {
			return doi, nil
		}
	}

	return "", nil // No DOI found (not an error)
}

// ExtractTitle attempts to extract the title from a PDF.
// This is a best-effort heuristic based on font size and position.
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

// ExtractText extracts all text from the first N pages of a PDF.
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

// ExtractTextReader extracts text from a PDF reader.
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
