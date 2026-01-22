// Package storage handles data persistence in JSONL and SQLite formats.
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/reference"
)

// MaxJSONLLineCapacity is the maximum buffer size for reading JSONL lines (1MB per line).
// This constant is shared across all JSONL file readers.
const MaxJSONLLineCapacity = 1024 * 1024

// RefWithAction pairs a reference with an import action.
type RefWithAction struct {
	Ref         reference.Reference
	Action      string // new, update
	ExistingIdx int    // Index in existing refs (for updates)
}

// ReadAll reads all references from a JSONL file.
func ReadAll(path string) ([]reference.Reference, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty file returns empty slice
		}
		return nil, fmt.Errorf("opening refs file: %w", err)
	}
	defer f.Close()

	var refs []reference.Reference
	scanner := bufio.NewScanner(f)

	// Increase buffer size for long lines
	buf := make([]byte, MaxJSONLLineCapacity)
	scanner.Buffer(buf, MaxJSONLLineCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var ref reference.Reference
		if err := json.Unmarshal(line, &ref); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		refs = append(refs, ref)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading refs file: %w", err)
	}

	return refs, nil
}

// Append adds a reference to the end of a JSONL file.
func Append(path string, ref reference.Reference) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening refs file for append: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(ref)
	if err != nil {
		return fmt.Errorf("encoding reference: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing reference: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WriteAll writes all references to a JSONL file, replacing existing content.
func WriteAll(path string, refs []reference.Reference) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating refs file: %w", err)
	}
	defer f.Close()

	for i, ref := range refs {
		data, err := json.Marshal(ref)
		if err != nil {
			return fmt.Errorf("encoding reference %d: %w", i, err)
		}

		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("writing reference %d: %w", i, err)
		}
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
	}

	return nil
}

// FindByDOI searches for a reference by DOI.
func FindByDOI(refs []reference.Reference, doi string) (int, bool) {
	if doi == "" {
		return -1, false
	}
	for i, ref := range refs {
		if ref.DOI == doi {
			return i, true
		}
	}
	return -1, false
}

// FindByID searches for a reference by ID.
func FindByID(refs []reference.Reference, id string) (int, bool) {
	for i, ref := range refs {
		if ref.ID == id {
			return i, true
		}
	}
	return -1, false
}

// FindBySourceID searches for a reference by import source type and ID.
func FindBySourceID(refs []reference.Reference, sourceType, sourceID string) (int, bool) {
	if sourceID == "" {
		return -1, false
	}
	for i, ref := range refs {
		if ref.Source.Type == sourceType && ref.Source.ID == sourceID {
			return i, true
		}
	}
	return -1, false
}

// GenerateUniqueID returns an ID that doesn't conflict with existing references.
// If the base ID exists, appends -2, -3, etc.
func GenerateUniqueID(refs []reference.Reference, baseID string) string {
	if _, found := FindByID(refs, baseID); !found {
		return baseID
	}

	// Start at 2: baseID is taken, so first duplicate becomes baseID-2
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", baseID, i)
		if _, found := FindByID(refs, candidate); !found {
			return candidate
		}
	}
}
