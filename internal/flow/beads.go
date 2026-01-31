package flow

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// GitHub reference pattern: "GitHub: org/repo#N"
var gitHubRefPattern = regexp.MustCompile(`GitHub:\s*([^#\s]+)#(\d+)`)

// LoadBeads loads all beads from .beads/issues.jsonl.
func LoadBeads() ([]Bead, error) {
	beadsPath := filepath.Join(BeadsDir, BeadsFile)
	file, err := os.Open(beadsPath)
	if os.IsNotExist(err) {
		return nil, nil // No beads file is not an error
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var beads []Bead
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var bead Bead
		if err := json.Unmarshal([]byte(line), &bead); err != nil {
			continue // Skip malformed lines
		}
		beads = append(beads, bead)
	}

	return beads, scanner.Err()
}

// LoadP0Beads returns all P0 (priority=0) beads.
func LoadP0Beads() ([]Bead, error) {
	beads, err := LoadBeads()
	if err != nil {
		return nil, err
	}

	var p0 []Bead
	for _, b := range beads {
		if b.Priority == 0 {
			p0 = append(p0, b)
		}
	}
	return p0, nil
}

// ExtractGitHubRefsFromDescription extracts all GitHub references from a description.
func ExtractGitHubRefsFromDescription(desc string) []string {
	matches := gitHubRefPattern.FindAllStringSubmatch(desc, -1)
	var refs []string
	for _, m := range matches {
		refs = append(refs, m[1]+"#"+m[2])
	}
	return refs
}

// CollectAllGitHubRefs builds a set of all GitHub references across all beads.
func CollectAllGitHubRefs() (map[string]bool, error) {
	beads, err := LoadBeads()
	if err != nil {
		return nil, err
	}

	refs := make(map[string]bool)
	for _, b := range beads {
		for _, ref := range ExtractGitHubRefsFromDescription(b.Description) {
			refs[ref] = true
		}
	}
	return refs, nil
}

// errInvalidInteger is returned when parsing an invalid integer string.
var errInvalidInteger = errors.New("invalid integer format")

// mustParseInt parses a string as a positive integer, panicking on error.
// Use only when input is known to be valid (e.g., from regex capture groups).
func mustParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic("mustParseInt: " + err.Error())
	}
	return n
}

// parsePositiveInt parses a string as a positive integer.
func parsePositiveInt(s string) (int, error) {
	if s == "" {
		return 0, errInvalidInteger
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errInvalidInteger
	}
	if n <= 0 {
		return 0, errInvalidInteger
	}
	return n, nil
}
