package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matsen/bipartite/internal/concept"
)

// ReadAllConcepts reads all concepts from a JSONL file.
// Returns an error if any concept fails structural validation (fail-fast).
func ReadAllConcepts(path string) ([]concept.Concept, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty file returns empty slice
		}
		return nil, fmt.Errorf("opening concepts file: %w", err)
	}
	defer f.Close()

	var concepts []concept.Concept
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

		var c concept.Concept
		if err := json.Unmarshal(line, &c); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}

		// Fail fast: validate concept structure before adding to collection
		if err := c.ValidateForCreate(); err != nil {
			return nil, fmt.Errorf("invalid concept at line %d: %w", lineNum, err)
		}

		concepts = append(concepts, c)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading concepts file: %w", err)
	}

	return concepts, nil
}

// writeConceptJSONL marshals a concept to JSON and writes it as a JSONL line.
func writeConceptJSONL(w io.Writer, c concept.Concept) error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encoding concept: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing concept: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	return nil
}

// AppendConcept adds a concept to the end of a JSONL file.
func AppendConcept(path string, c concept.Concept) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening concepts file for append: %w", err)
	}
	defer f.Close()

	return writeConceptJSONL(f, c)
}

// WriteAllConcepts writes all concepts to a JSONL file, replacing existing content.
func WriteAllConcepts(path string, concepts []concept.Concept) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating concepts file: %w", err)
	}
	defer f.Close()

	for _, c := range concepts {
		if err := writeConceptJSONL(f, c); err != nil {
			return err
		}
	}

	return nil
}

// FindConceptByID searches for a concept by its ID in an in-memory slice.
// Returns the index and true if found, -1 and false otherwise.
func FindConceptByID(concepts []concept.Concept, id string) (int, bool) {
	for i, c := range concepts {
		if c.ID == id {
			return i, true
		}
	}
	return -1, false
}

// UpsertConceptInSlice adds or updates a concept in an in-memory slice.
// Returns the updated slice and true if the concept was updated, false if added.
func UpsertConceptInSlice(concepts []concept.Concept, newConcept concept.Concept) ([]concept.Concept, bool) {
	idx, found := FindConceptByID(concepts, newConcept.ID)
	if found {
		concepts[idx] = newConcept
		return concepts, true
	}
	return append(concepts, newConcept), false
}

// DeleteConceptFromSlice removes a concept from an in-memory slice.
// Returns the updated slice and true if the concept was found and removed, false otherwise.
// Note: This operation does not preserve slice order; it swaps the deleted element with
// the last element for O(1) performance. If ordering matters, callers should sort after deletion.
func DeleteConceptFromSlice(concepts []concept.Concept, id string) ([]concept.Concept, bool) {
	idx, found := FindConceptByID(concepts, id)
	if !found {
		return concepts, false
	}
	// Remove by replacing with last element and truncating (O(1) but changes order)
	concepts[idx] = concepts[len(concepts)-1]
	return concepts[:len(concepts)-1], true
}

// LoadConceptIDSet loads all concept IDs and returns them as a set for O(1) lookup.
func LoadConceptIDSet(path string) (map[string]bool, error) {
	concepts, err := ReadAllConcepts(path)
	if err != nil {
		return nil, err
	}

	idSet := make(map[string]bool, len(concepts))
	for _, c := range concepts {
		idSet[c.ID] = true
	}
	return idSet, nil
}
