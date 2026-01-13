package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matsen/bipartite/internal/edge"
)

// EdgesFile is the name of the edges JSONL file.
const EdgesFile = "edges.jsonl"

// ReadAllEdges reads all edges from a JSONL file.
// Returns an error if any edge fails structural validation (fail-fast).
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

		// Fail fast: validate edge structure before adding to collection
		if err := e.ValidateForCreate(); err != nil {
			return nil, fmt.Errorf("invalid edge at line %d: %w", lineNum, err)
		}

		edges = append(edges, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading edges file: %w", err)
	}

	return edges, nil
}

// writeEdgeJSONL marshals an edge to JSON and writes it as a JSONL line.
func writeEdgeJSONL(w io.Writer, e edge.Edge) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encoding edge: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing edge: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	return nil
}

// AppendEdge adds an edge to the end of a JSONL file.
func AppendEdge(path string, e edge.Edge) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening edges file for append: %w", err)
	}
	defer f.Close()

	return writeEdgeJSONL(f, e)
}

// WriteAllEdges writes all edges to a JSONL file, replacing existing content.
func WriteAllEdges(path string, edges []edge.Edge) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating edges file: %w", err)
	}
	defer f.Close()

	for _, e := range edges {
		if err := writeEdgeJSONL(f, e); err != nil {
			return err
		}
	}

	return nil
}

// FindEdgeInSlice searches for an edge by its key in an in-memory slice.
// Returns the index and true if found, -1 and false otherwise.
func FindEdgeInSlice(edges []edge.Edge, key edge.EdgeKey) (int, bool) {
	for i, e := range edges {
		if e.SourceID == key.SourceID && e.TargetID == key.TargetID && e.RelationshipType == key.RelationshipType {
			return i, true
		}
	}
	return -1, false
}

// UpsertEdgeInSlice adds or updates an edge in an in-memory slice.
// Returns the updated slice and true if the edge was updated, false if added.
func UpsertEdgeInSlice(edges []edge.Edge, newEdge edge.Edge) ([]edge.Edge, bool) {
	key := newEdge.Key()
	idx, found := FindEdgeInSlice(edges, key)
	if found {
		newEdge.MergeCreatedAt(edges[idx])
		edges[idx] = newEdge
		return edges, true
	}
	newEdge.SetCreatedAt()
	return append(edges, newEdge), false
}

// FindEdgeByKey is an alias for FindEdgeInSlice for backward compatibility.
// Deprecated: Use FindEdgeInSlice instead.
func FindEdgeByKey(edges []edge.Edge, key edge.EdgeKey) (int, bool) {
	return FindEdgeInSlice(edges, key)
}

// UpsertEdge is an alias for UpsertEdgeInSlice for backward compatibility.
// Deprecated: Use UpsertEdgeInSlice instead.
func UpsertEdge(edges []edge.Edge, newEdge edge.Edge) ([]edge.Edge, bool) {
	return UpsertEdgeInSlice(edges, newEdge)
}
