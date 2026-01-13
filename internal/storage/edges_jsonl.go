package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/edge"
)

// EdgesFile is the name of the edges JSONL file.
const EdgesFile = "edges.jsonl"

// ReadAllEdges reads all edges from a JSONL file.
func ReadAllEdges(path string) ([]edge.Edge, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty file returns empty slice
		}
		return nil, fmt.Errorf("opening edges file: %w", err)
	}
	defer f.Close()

	var edges []edge.Edge
	scanner := bufio.NewScanner(f)

	// Increase buffer size for long lines
	const maxCapacity = 1024 * 1024 // 1MB per line max
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var e edge.Edge
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		edges = append(edges, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading edges file: %w", err)
	}

	return edges, nil
}

// AppendEdge adds an edge to the end of a JSONL file.
func AppendEdge(path string, e edge.Edge) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening edges file for append: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encoding edge: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing edge: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WriteAllEdges writes all edges to a JSONL file, replacing existing content.
func WriteAllEdges(path string, edges []edge.Edge) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating edges file: %w", err)
	}
	defer f.Close()

	for i, e := range edges {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("encoding edge %d: %w", i, err)
		}

		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("writing edge %d: %w", i, err)
		}
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
	}

	return nil
}

// FindEdgeByKey searches for an edge by its key (source_id, target_id, relationship_type).
func FindEdgeByKey(edges []edge.Edge, key edge.EdgeKey) (int, bool) {
	for i, e := range edges {
		if e.SourceID == key.SourceID && e.TargetID == key.TargetID && e.RelationshipType == key.RelationshipType {
			return i, true
		}
	}
	return -1, false
}

// UpsertEdge adds or updates an edge in the list.
// Returns true if the edge was updated, false if it was added.
func UpsertEdge(edges []edge.Edge, newEdge edge.Edge) ([]edge.Edge, bool) {
	key := newEdge.Key()
	idx, found := FindEdgeByKey(edges, key)
	if found {
		// Update existing edge (preserve created_at if not provided)
		if newEdge.CreatedAt == "" && edges[idx].CreatedAt != "" {
			newEdge.CreatedAt = edges[idx].CreatedAt
		}
		edges[idx] = newEdge
		return edges, true
	}
	// Add new edge
	newEdge.SetCreatedAt()
	return append(edges, newEdge), false
}

// PaperExists checks if a paper ID exists in the refs slice.
func PaperExists(paperID string, refIDs []string) bool {
	for _, id := range refIDs {
		if id == paperID {
			return true
		}
	}
	return false
}
