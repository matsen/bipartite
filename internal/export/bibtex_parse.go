package export

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// BibTeXIndex indexes existing BibTeX entries for deduplication.
type BibTeXIndex struct {
	// Keys maps citation keys to true for existence check
	Keys map[string]bool
	// DOIs maps DOI values to citation keys
	DOIs map[string]string
}

// NewBibTeXIndex creates an empty BibTeX index.
func NewBibTeXIndex() *BibTeXIndex {
	return &BibTeXIndex{
		Keys: make(map[string]bool),
		DOIs: make(map[string]string),
	}
}

// HasEntry returns true if the entry already exists (by DOI or key).
// DOI is the primary match; citation key is the fallback if no DOI.
func (idx *BibTeXIndex) HasEntry(key, doi string) bool {
	// Primary: match by DOI if available
	if doi != "" {
		if _, exists := idx.DOIs[normalizeDOI(doi)]; exists {
			return true
		}
	}

	// Fallback: match by citation key
	return idx.Keys[key]
}

// ParseBibTeXFile builds an index from an existing .bib file.
// Returns an empty index if the file doesn't exist or is empty.
func ParseBibTeXFile(path string) (*BibTeXIndex, error) {
	idx := NewBibTeXIndex()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return nil, err
	}
	defer file.Close()

	// Regex patterns
	// Match entry start: @type{key,
	entryStartRegex := regexp.MustCompile(`@\w+\{([^,]+),`)
	// Match DOI field: doi = {value} or doi = "value"
	doiFieldRegex := regexp.MustCompile(`(?i)^\s*doi\s*=\s*[\{"]([^\}"]+)[\}"]`)

	scanner := bufio.NewScanner(file)
	var currentKey string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for entry start
		if matches := entryStartRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentKey = strings.TrimSpace(matches[1])
			idx.Keys[currentKey] = true
		}

		// Check for DOI field
		if matches := doiFieldRegex.FindStringSubmatch(line); len(matches) > 1 {
			doi := normalizeDOI(matches[1])
			if doi != "" && currentKey != "" {
				idx.DOIs[doi] = currentKey
			}
		}
	}

	return idx, scanner.Err()
}

// normalizeDOI normalizes a DOI for comparison.
// Removes common prefixes like "https://doi.org/" and lowercases.
func normalizeDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	doi = strings.TrimPrefix(doi, "https://doi.org/")
	doi = strings.TrimPrefix(doi, "http://doi.org/")
	doi = strings.TrimPrefix(doi, "doi.org/")
	doi = strings.TrimPrefix(doi, "DOI:")
	doi = strings.TrimPrefix(doi, "doi:")
	return strings.ToLower(doi)
}

// AppendToBibFile appends BibTeX content to a file.
func AppendToBibFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Ensure we start on a new line
	_, err = file.WriteString("\n" + content)
	return err
}
