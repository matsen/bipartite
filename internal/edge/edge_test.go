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
