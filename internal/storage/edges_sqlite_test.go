package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/edge"
)

func TestDB_RebuildEdgesFromJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "edges.jsonl")

	// Create test JSONL file
	content := `{"source_id":"A","target_id":"B","relationship_type":"cites","summary":"A cites B","created_at":"2024-01-01T00:00:00Z"}
{"source_id":"B","target_id":"C","relationship_type":"extends","summary":"B extends C","created_at":"2024-01-02T00:00:00Z"}`
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	count, err := db.RebuildEdgesFromJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("RebuildEdgesFromJSONL failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 edges, got %d", count)
	}

	// Verify edges are queryable
	edges, err := db.GetAllEdges()
	if err != nil {
		t.Fatalf("GetAllEdges failed: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 edges from GetAllEdges, got %d", len(edges))
	}
}

func TestDB_RebuildEdgesFromJSONL_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "edges.jsonl")

	// Create empty JSONL file
	if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	count, err := db.RebuildEdgesFromJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("RebuildEdgesFromJSONL failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 edges, got %d", count)
	}
}

func TestDB_InsertEdge(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	e := edge.Edge{
		SourceID:         "Smith2024",
		TargetID:         "Jones2023",
		RelationshipType: "extends",
		Summary:          "Smith extends Jones's framework",
		CreatedAt:        "2024-01-01T00:00:00Z",
	}

	if err := db.InsertEdge(e); err != nil {
		t.Fatalf("InsertEdge failed: %v", err)
	}

	// Verify edge is queryable
	edges, err := db.GetEdgesBySource("Smith2024")
	if err != nil {
		t.Fatalf("GetEdgesBySource failed: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Summary != "Smith extends Jones's framework" {
		t.Errorf("unexpected summary: %q", edges[0].Summary)
	}
}

func TestDB_InsertEdge_Upsert(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert initial edge
	e1 := edge.Edge{
		SourceID:         "A",
		TargetID:         "B",
		RelationshipType: "cites",
		Summary:          "Original summary",
	}
	if err := db.InsertEdge(e1); err != nil {
		t.Fatalf("InsertEdge failed: %v", err)
	}

	// Upsert with same key but different summary
	e2 := edge.Edge{
		SourceID:         "A",
		TargetID:         "B",
		RelationshipType: "cites",
		Summary:          "Updated summary",
	}
	if err := db.InsertEdge(e2); err != nil {
		t.Fatalf("InsertEdge (upsert) failed: %v", err)
	}

	// Verify only one edge exists with updated summary
	edges, err := db.GetAllEdges()
	if err != nil {
		t.Fatalf("GetAllEdges failed: %v", err)
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 edge after upsert, got %d", len(edges))
	}
	if edges[0].Summary != "Updated summary" {
		t.Errorf("expected updated summary, got %q", edges[0].Summary)
	}
}

func TestDB_GetEdgesBySource(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test edges
	testEdges := []edge.Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "A", TargetID: "C", RelationshipType: "extends", Summary: "s2"},
		{SourceID: "B", TargetID: "C", RelationshipType: "cites", Summary: "s3"},
	}
	for _, e := range testEdges {
		if err := db.InsertEdge(e); err != nil {
			t.Fatal(err)
		}
	}

	edges, err := db.GetEdgesBySource("A")
	if err != nil {
		t.Fatalf("GetEdgesBySource failed: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 edges from A, got %d", len(edges))
	}
}

func TestDB_GetEdgesByTarget(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test edges
	testEdges := []edge.Edge{
		{SourceID: "A", TargetID: "C", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "B", TargetID: "C", RelationshipType: "extends", Summary: "s2"},
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s3"},
	}
	for _, e := range testEdges {
		if err := db.InsertEdge(e); err != nil {
			t.Fatal(err)
		}
	}

	edges, err := db.GetEdgesByTarget("C")
	if err != nil {
		t.Fatalf("GetEdgesByTarget failed: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 edges to C, got %d", len(edges))
	}
}

func TestDB_GetEdgesByType(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test edges
	testEdges := []edge.Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "B", TargetID: "C", RelationshipType: "extends", Summary: "s2"},
		{SourceID: "C", TargetID: "D", RelationshipType: "cites", Summary: "s3"},
	}
	for _, e := range testEdges {
		if err := db.InsertEdge(e); err != nil {
			t.Fatal(err)
		}
	}

	edges, err := db.GetEdgesByType("cites")
	if err != nil {
		t.Fatalf("GetEdgesByType failed: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 'cites' edges, got %d", len(edges))
	}
}

func TestDB_GetEdgesByPaper(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert test edges
	testEdges := []edge.Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "B", TargetID: "C", RelationshipType: "extends", Summary: "s2"},
		{SourceID: "C", TargetID: "D", RelationshipType: "cites", Summary: "s3"},
	}
	for _, e := range testEdges {
		if err := db.InsertEdge(e); err != nil {
			t.Fatal(err)
		}
	}

	// B is involved in 2 edges (as target of A->B and source of B->C)
	edges, err := db.GetEdgesByPaper("B")
	if err != nil {
		t.Fatalf("GetEdgesByPaper failed: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 edges involving B, got %d", len(edges))
	}
}

func TestDB_CountEdges(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Initially, table doesn't exist (or is empty)
	count, err := db.CountEdges()
	if err != nil {
		t.Fatalf("CountEdges failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 edges initially, got %d", count)
	}

	// Insert some edges
	testEdges := []edge.Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "B", TargetID: "C", RelationshipType: "extends", Summary: "s2"},
	}
	for _, e := range testEdges {
		if err := db.InsertEdge(e); err != nil {
			t.Fatal(err)
		}
	}

	count, err = db.CountEdges()
	if err != nil {
		t.Fatalf("CountEdges failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 edges, got %d", count)
	}
}

func TestDB_GetAllEdges_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create the schema first
	if err := db.createEdgesSchema(); err != nil {
		t.Fatal(err)
	}

	edges, err := db.GetAllEdges()
	if err != nil {
		t.Fatalf("GetAllEdges failed: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}
