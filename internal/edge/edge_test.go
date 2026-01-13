package edge

import "testing"

func TestEdge_ValidateForCreate(t *testing.T) {
	tests := []struct {
		name    string
		edge    Edge
		wantErr error
	}{
		{
			name: "valid edge",
			edge: Edge{
				SourceID:         "Smith2024",
				TargetID:         "Jones2023",
				RelationshipType: "extends",
				Summary:          "Smith extends Jones's framework",
			},
			wantErr: nil,
		},
		{
			name: "empty source_id",
			edge: Edge{
				SourceID:         "",
				TargetID:         "Jones2023",
				RelationshipType: "extends",
				Summary:          "summary",
			},
			wantErr: ErrEmptySourceID,
		},
		{
			name: "empty target_id",
			edge: Edge{
				SourceID:         "Smith2024",
				TargetID:         "",
				RelationshipType: "extends",
				Summary:          "summary",
			},
			wantErr: ErrEmptyTargetID,
		},
		{
			name: "empty relationship_type",
			edge: Edge{
				SourceID:         "Smith2024",
				TargetID:         "Jones2023",
				RelationshipType: "",
				Summary:          "summary",
			},
			wantErr: ErrEmptyRelationshipType,
		},
		{
			name: "empty summary",
			edge: Edge{
				SourceID:         "Smith2024",
				TargetID:         "Jones2023",
				RelationshipType: "extends",
				Summary:          "",
			},
			wantErr: ErrEmptySummary,
		},
		{
			name: "self edge",
			edge: Edge{
				SourceID:         "Smith2024",
				TargetID:         "Smith2024",
				RelationshipType: "cites",
				Summary:          "Self-citation",
			},
			wantErr: ErrSelfEdge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.ValidateForCreate()
			if err != tt.wantErr {
				t.Errorf("ValidateForCreate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEdge_SetCreatedAt(t *testing.T) {
	t.Run("sets timestamp when empty", func(t *testing.T) {
		e := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "summary",
		}
		e.SetCreatedAt()
		if e.CreatedAt == "" {
			t.Error("expected CreatedAt to be set")
		}
	})

	t.Run("preserves existing timestamp", func(t *testing.T) {
		e := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "summary",
			CreatedAt:        "2024-01-01T00:00:00Z",
		}
		e.SetCreatedAt()
		if e.CreatedAt != "2024-01-01T00:00:00Z" {
			t.Errorf("expected CreatedAt to be preserved, got %q", e.CreatedAt)
		}
	})
}

func TestEdge_MergeCreatedAt(t *testing.T) {
	t.Run("copies timestamp from existing when new is empty", func(t *testing.T) {
		newEdge := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "new summary",
		}
		existing := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "old summary",
			CreatedAt:        "2024-01-01T00:00:00Z",
		}
		newEdge.MergeCreatedAt(existing)
		if newEdge.CreatedAt != "2024-01-01T00:00:00Z" {
			t.Errorf("expected CreatedAt to be copied, got %q", newEdge.CreatedAt)
		}
	})

	t.Run("preserves new timestamp when set", func(t *testing.T) {
		newEdge := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "new summary",
			CreatedAt:        "2025-01-01T00:00:00Z",
		}
		existing := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "old summary",
			CreatedAt:        "2024-01-01T00:00:00Z",
		}
		newEdge.MergeCreatedAt(existing)
		if newEdge.CreatedAt != "2025-01-01T00:00:00Z" {
			t.Errorf("expected new CreatedAt to be preserved, got %q", newEdge.CreatedAt)
		}
	})

	t.Run("handles empty existing timestamp", func(t *testing.T) {
		newEdge := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "new summary",
		}
		existing := Edge{
			SourceID:         "A",
			TargetID:         "B",
			RelationshipType: "cites",
			Summary:          "old summary",
		}
		newEdge.MergeCreatedAt(existing)
		if newEdge.CreatedAt != "" {
			t.Errorf("expected CreatedAt to remain empty, got %q", newEdge.CreatedAt)
		}
	})
}

func TestEdge_Key(t *testing.T) {
	e := Edge{
		SourceID:         "Smith2024",
		TargetID:         "Jones2023",
		RelationshipType: "extends",
		Summary:          "summary",
	}
	key := e.Key()
	if key.SourceID != "Smith2024" {
		t.Errorf("SourceID = %q, want %q", key.SourceID, "Smith2024")
	}
	if key.TargetID != "Jones2023" {
		t.Errorf("TargetID = %q, want %q", key.TargetID, "Jones2023")
	}
	if key.RelationshipType != "extends" {
		t.Errorf("RelationshipType = %q, want %q", key.RelationshipType, "extends")
	}
}

func TestDetectOrphanedEdges(t *testing.T) {
	edges := []Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "A", TargetID: "X", RelationshipType: "extends", Summary: "s2"}, // X missing
		{SourceID: "Y", TargetID: "B", RelationshipType: "cites", Summary: "s3"},   // Y missing
		{SourceID: "Y", TargetID: "X", RelationshipType: "cites", Summary: "s4"},   // both missing
	}
	validIDs := map[string]bool{"A": true, "B": true}

	orphaned, valid := DetectOrphanedEdges(edges, validIDs)

	if len(valid) != 1 {
		t.Errorf("expected 1 valid edge, got %d", len(valid))
	}
	if len(orphaned) != 3 {
		t.Errorf("expected 3 orphaned edges, got %d", len(orphaned))
	}

	// Check reasons
	reasonCounts := map[string]int{}
	for _, o := range orphaned {
		reasonCounts[o.Reason]++
	}
	if reasonCounts["missing_target"] != 1 {
		t.Errorf("expected 1 missing_target, got %d", reasonCounts["missing_target"])
	}
	if reasonCounts["missing_source"] != 1 {
		t.Errorf("expected 1 missing_source, got %d", reasonCounts["missing_source"])
	}
	if reasonCounts["missing_both"] != 1 {
		t.Errorf("expected 1 missing_both, got %d", reasonCounts["missing_both"])
	}
}

func TestDetectOrphanedEdges_NoOrphans(t *testing.T) {
	edges := []Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "B", TargetID: "A", RelationshipType: "extends", Summary: "s2"},
	}
	validIDs := map[string]bool{"A": true, "B": true}

	orphaned, valid := DetectOrphanedEdges(edges, validIDs)

	if len(valid) != 2 {
		t.Errorf("expected 2 valid edges, got %d", len(valid))
	}
	if len(orphaned) != 0 {
		t.Errorf("expected 0 orphaned edges, got %d", len(orphaned))
	}
}

func TestFindDuplicateEdges(t *testing.T) {
	edges := []Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s2"}, // duplicate
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s3"}, // duplicate
		{SourceID: "A", TargetID: "C", RelationshipType: "extends", Summary: "s4"},
		{SourceID: "B", TargetID: "C", RelationshipType: "cites", Summary: "s5"},
	}

	duplicates := FindDuplicateEdges(edges)

	if len(duplicates) != 1 {
		t.Errorf("expected 1 duplicate key, got %d", len(duplicates))
	}

	key := EdgeKey{SourceID: "A", TargetID: "B", RelationshipType: "cites"}
	if count, ok := duplicates[key]; !ok {
		t.Error("expected duplicate for A->B cites")
	} else if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestFindDuplicateEdges_NoDuplicates(t *testing.T) {
	edges := []Edge{
		{SourceID: "A", TargetID: "B", RelationshipType: "cites", Summary: "s1"},
		{SourceID: "A", TargetID: "C", RelationshipType: "extends", Summary: "s2"},
		{SourceID: "B", TargetID: "C", RelationshipType: "cites", Summary: "s3"},
	}

	duplicates := FindDuplicateEdges(edges)

	if len(duplicates) != 0 {
		t.Errorf("expected 0 duplicates, got %d", len(duplicates))
	}
}
