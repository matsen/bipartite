package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/edge"
)

func TestReadAllEdges(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantEdges   int
		wantErr     bool
		wantErrLine int // 0 means no specific line expected
	}{
		{
			name:      "empty file",
			content:   "",
			wantEdges: 0,
		},
		{
			name:      "single edge",
			content:   `{"source_id":"A","target_id":"B","relationship_type":"cites","summary":"A cites B"}`,
			wantEdges: 1,
		},
		{
			name: "multiple edges",
			content: `{"source_id":"A","target_id":"B","relationship_type":"cites","summary":"A cites B"}
{"source_id":"B","target_id":"C","relationship_type":"extends","summary":"B extends C"}
{"source_id":"A","target_id":"C","relationship_type":"contradicts","summary":"A contradicts C"}`,
			wantEdges: 3,
		},
		{
			name: "with empty lines",
			content: `{"source_id":"A","target_id":"B","relationship_type":"cites","summary":"A cites B"}

{"source_id":"B","target_id":"C","relationship_type":"extends","summary":"B extends C"}`,
			wantEdges: 2,
		},
		{
			name:        "invalid JSON",
			content:     `{"source_id":"A","target_id":"B"`,
			wantErr:     true,
			wantErrLine: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "edges.jsonl")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			edges, err := ReadAllEdges(path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(edges) != tt.wantEdges {
				t.Errorf("got %d edges, want %d", len(edges), tt.wantEdges)
			}
		})
	}
}

func TestReadAllEdges_FileNotExists(t *testing.T) {
	edges, err := ReadAllEdges("/nonexistent/path/edges.jsonl")
	if err != nil {
		t.Errorf("expected nil error for missing file, got: %v", err)
	}
	if edges != nil {
		t.Errorf("expected nil slice for missing file, got: %v", edges)
	}
}

func TestAppendEdge(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "edges.jsonl")

	e := edge.Edge{
		SourceID:         "Smith2024",
		TargetID:         "Jones2023",
		RelationshipType: "extends",
		Summary:          "Smith extends Jones's framework",
	}

	if err := AppendEdge(path, e); err != nil {
		t.Fatalf("AppendEdge failed: %v", err)
	}

	// Read back
	edges, err := ReadAllEdges(path)
	if err != nil {
		t.Fatalf("ReadAllEdges failed: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].SourceID != "Smith2024" {
		t.Errorf("got source_id %q, want %q", edges[0].SourceID, "Smith2024")
	}
}

func TestWriteAllEdges(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "edges.jsonl")

	edges := []edge.Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "A cites B"},
		{SourceID: "B", TargetID: "C", RelationshipType: "extends", Summary: "B extends C"},
	}

	if err := WriteAllEdges(path, edges); err != nil {
		t.Fatalf("WriteAllEdges failed: %v", err)
	}

	// Read back
	readEdges, err := ReadAllEdges(path)
	if err != nil {
		t.Fatalf("ReadAllEdges failed: %v", err)
	}
	if len(readEdges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(readEdges))
	}
}

func TestFindEdgeByKey(t *testing.T) {
	edges := []edge.Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "A cites B"},
		{SourceID: "B", TargetID: "C", RelationshipType: "extends", Summary: "B extends C"},
		{SourceID: "A", TargetID: "C", RelationshipType: "cites", Summary: "A cites C"},
	}

	tests := []struct {
		name      string
		key       edge.EdgeKey
		wantIdx   int
		wantFound bool
	}{
		{
			name:      "found first",
			key:       edge.EdgeKey{SourceID: "A", TargetID: "B", RelationshipType: "cites"},
			wantIdx:   0,
			wantFound: true,
		},
		{
			name:      "found middle",
			key:       edge.EdgeKey{SourceID: "B", TargetID: "C", RelationshipType: "extends"},
			wantIdx:   1,
			wantFound: true,
		},
		{
			name:      "not found - different type",
			key:       edge.EdgeKey{SourceID: "A", TargetID: "B", RelationshipType: "extends"},
			wantIdx:   -1,
			wantFound: false,
		},
		{
			name:      "not found - no such edge",
			key:       edge.EdgeKey{SourceID: "X", TargetID: "Y", RelationshipType: "cites"},
			wantIdx:   -1,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, found := FindEdgeByKey(edges, tt.key)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if idx != tt.wantIdx {
				t.Errorf("idx = %d, want %d", idx, tt.wantIdx)
			}
		})
	}
}

func TestUpsertEdge(t *testing.T) {
	t.Run("add new edge", func(t *testing.T) {
		edges := []edge.Edge{
			{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "A cites B"},
		}
		newEdge := edge.Edge{
			SourceID:         "B",
			TargetID:         "C",
			RelationshipType: "extends",
			Summary:          "B extends C",
		}

		result, updated := UpsertEdge(edges, newEdge)
		if updated {
			t.Error("expected add, got update")
		}
		if len(result) != 2 {
			t.Errorf("expected 2 edges, got %d", len(result))
		}
		if result[1].CreatedAt == "" {
			t.Error("expected CreatedAt to be set for new edge")
		}
	})

	t.Run("update existing edge", func(t *testing.T) {
		edges := []edge.Edge{
			{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "Original summary", CreatedAt: "2024-01-01T00:00:00Z"},
		}
		newEdge := edge.Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "Updated summary",
		}

		result, updated := UpsertEdge(edges, newEdge)
		if !updated {
			t.Error("expected update, got add")
		}
		if len(result) != 1 {
			t.Errorf("expected 1 edge, got %d", len(result))
		}
		if result[0].Summary != "Updated summary" {
			t.Errorf("expected updated summary, got %q", result[0].Summary)
		}
		if result[0].CreatedAt != "2024-01-01T00:00:00Z" {
			t.Errorf("expected CreatedAt to be preserved, got %q", result[0].CreatedAt)
		}
	})
}
