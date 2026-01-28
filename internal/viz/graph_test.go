package viz

import (
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/repo"
)

func TestProcessEdges_PaperToConcept(t *testing.T) {
	conceptIDs := map[string]bool{
		"somatic-hypermutation": true,
		"affinity-maturation":   true,
	}
	projectIDs := map[string]bool{}

	tests := []struct {
		name                     string
		edges                    []edge.Edge
		wantEdgeCount            int
		wantPaperIDs             []string
		wantConceptConnCounts    map[string]int
		wantEdgeTargetUnprefixed bool
	}{
		{
			name: "edges with concept: prefix are matched",
			edges: []edge.Edge{
				{
					SourceID:         "Paper2023-ab",
					TargetID:         "concept:somatic-hypermutation",
					RelationshipType: "models",
					Summary:          "Models SHM targeting",
				},
			},
			wantEdgeCount:            1,
			wantPaperIDs:             []string{"Paper2023-ab"},
			wantConceptConnCounts:    map[string]int{"somatic-hypermutation": 1},
			wantEdgeTargetUnprefixed: true,
		},
		{
			name: "edges without prefix are matched",
			edges: []edge.Edge{
				{
					SourceID:         "Paper2023-cd",
					TargetID:         "affinity-maturation",
					RelationshipType: "applies",
					Summary:          "Applies AM",
				},
			},
			wantEdgeCount:            1,
			wantPaperIDs:             []string{"Paper2023-cd"},
			wantConceptConnCounts:    map[string]int{"affinity-maturation": 1},
			wantEdgeTargetUnprefixed: true,
		},
		{
			name: "edges to non-existent concepts are filtered out",
			edges: []edge.Edge{
				{
					SourceID:         "Paper2023-ef",
					TargetID:         "concept:unknown-concept",
					RelationshipType: "introduces",
					Summary:          "Introduces something",
				},
			},
			wantEdgeCount:         0,
			wantPaperIDs:          nil,
			wantConceptConnCounts: map[string]int{},
		},
		{
			name: "multiple edges from same paper to different concepts",
			edges: []edge.Edge{
				{
					SourceID:         "Paper2023-gh",
					TargetID:         "concept:somatic-hypermutation",
					RelationshipType: "models",
					Summary:          "Models SHM",
				},
				{
					SourceID:         "Paper2023-gh",
					TargetID:         "concept:affinity-maturation",
					RelationshipType: "applies",
					Summary:          "Studies AM",
				},
			},
			wantEdgeCount: 2,
			wantPaperIDs:  []string{"Paper2023-gh"},
			wantConceptConnCounts: map[string]int{
				"somatic-hypermutation": 1,
				"affinity-maturation":   1,
			},
			wantEdgeTargetUnprefixed: true,
		},
		{
			name: "multiple papers to same concept",
			edges: []edge.Edge{
				{
					SourceID:         "Paper2023-ij",
					TargetID:         "concept:somatic-hypermutation",
					RelationshipType: "models",
					Summary:          "First paper",
				},
				{
					SourceID:         "Paper2023-kl",
					TargetID:         "concept:somatic-hypermutation",
					RelationshipType: "applies",
					Summary:          "Second paper",
				},
			},
			wantEdgeCount:            2,
			wantPaperIDs:             []string{"Paper2023-ij", "Paper2023-kl"},
			wantConceptConnCounts:    map[string]int{"somatic-hypermutation": 2},
			wantEdgeTargetUnprefixed: true,
		},
		{
			name:                  "empty edges list",
			edges:                 []edge.Edge{},
			wantEdgeCount:         0,
			wantPaperIDs:          nil,
			wantConceptConnCounts: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEdges, gotPaperIDs, gotConceptConnCounts, _ := processEdges(tt.edges, conceptIDs, projectIDs)

			// Check edge count
			if len(gotEdges) != tt.wantEdgeCount {
				t.Errorf("got %d edges, want %d", len(gotEdges), tt.wantEdgeCount)
			}

			// Check paper IDs
			for _, wantPaperID := range tt.wantPaperIDs {
				if !gotPaperIDs[wantPaperID] {
					t.Errorf("missing paper ID %q in result", wantPaperID)
				}
			}
			if len(gotPaperIDs) != len(tt.wantPaperIDs) {
				t.Errorf("got %d paper IDs, want %d", len(gotPaperIDs), len(tt.wantPaperIDs))
			}

			// Check concept connection counts
			for conceptID, wantCount := range tt.wantConceptConnCounts {
				if gotConceptConnCounts[conceptID] != wantCount {
					t.Errorf("concept %q: got connection count %d, want %d",
						conceptID, gotConceptConnCounts[conceptID], wantCount)
				}
			}

			// Check that edge targets are unprefixed (matching concept node IDs)
			if tt.wantEdgeTargetUnprefixed {
				for _, e := range gotEdges {
					if strings.HasPrefix(e.Target, "concept:") {
						t.Errorf("edge target should be unprefixed, got %q", e.Target)
					}
				}
			}
		})
	}
}

func TestProcessEdges_ConceptToProject(t *testing.T) {
	conceptIDs := map[string]bool{
		"variational-inference": true,
		"somatic-hypermutation": true,
	}
	projectIDs := map[string]bool{
		"dasm2": true,
		"netam": true,
	}

	tests := []struct {
		name                  string
		edges                 []edge.Edge
		wantEdgeCount         int
		wantConceptConnCounts map[string]int
		wantProjectConnCounts map[string]int
	}{
		{
			name: "concept to project edge with prefixes",
			edges: []edge.Edge{
				{
					SourceID:         "concept:variational-inference",
					TargetID:         "project:dasm2",
					RelationshipType: "implemented-in",
					Summary:          "DASM2 uses VI",
				},
			},
			wantEdgeCount:         1,
			wantConceptConnCounts: map[string]int{"variational-inference": 1},
			wantProjectConnCounts: map[string]int{"dasm2": 1},
		},
		{
			name: "project to concept edge (reverse direction)",
			edges: []edge.Edge{
				{
					SourceID:         "project:netam",
					TargetID:         "concept:somatic-hypermutation",
					RelationshipType: "studied-by",
					Summary:          "Netam studies SHM",
				},
			},
			wantEdgeCount:         1,
			wantConceptConnCounts: map[string]int{"somatic-hypermutation": 1},
			wantProjectConnCounts: map[string]int{"netam": 1},
		},
		{
			name: "multiple concepts to same project",
			edges: []edge.Edge{
				{
					SourceID:         "concept:variational-inference",
					TargetID:         "project:dasm2",
					RelationshipType: "implemented-in",
					Summary:          "VI in DASM2",
				},
				{
					SourceID:         "concept:somatic-hypermutation",
					TargetID:         "project:dasm2",
					RelationshipType: "applied-in",
					Summary:          "SHM in DASM2",
				},
			},
			wantEdgeCount: 2,
			wantConceptConnCounts: map[string]int{
				"variational-inference": 1,
				"somatic-hypermutation": 1,
			},
			wantProjectConnCounts: map[string]int{"dasm2": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEdges, _, gotConceptConnCounts, gotProjectConnCounts := processEdges(tt.edges, conceptIDs, projectIDs)

			// Check edge count
			if len(gotEdges) != tt.wantEdgeCount {
				t.Errorf("got %d edges, want %d", len(gotEdges), tt.wantEdgeCount)
			}

			// Check concept connection counts
			for conceptID, wantCount := range tt.wantConceptConnCounts {
				if gotConceptConnCounts[conceptID] != wantCount {
					t.Errorf("concept %q: got connection count %d, want %d",
						conceptID, gotConceptConnCounts[conceptID], wantCount)
				}
			}

			// Check project connection counts
			for projectID, wantCount := range tt.wantProjectConnCounts {
				if gotProjectConnCounts[projectID] != wantCount {
					t.Errorf("project %q: got connection count %d, want %d",
						projectID, gotProjectConnCounts[projectID], wantCount)
				}
			}

			// Check that edge endpoints are unprefixed
			for _, e := range gotEdges {
				if strings.HasPrefix(e.Source, "concept:") || strings.HasPrefix(e.Source, "project:") {
					t.Errorf("edge source should be unprefixed, got %q", e.Source)
				}
				if strings.HasPrefix(e.Target, "concept:") || strings.HasPrefix(e.Target, "project:") {
					t.Errorf("edge target should be unprefixed, got %q", e.Target)
				}
			}
		})
	}
}

func TestBuildRepoProjectEdges(t *testing.T) {
	projectIDs := map[string]bool{
		"dasm":  true,
		"netam": true,
	}

	tests := []struct {
		name          string
		repos         []repo.Repo
		wantEdgeCount int
		wantEdges     []Edge
	}{
		{
			name: "repo with valid project creates edge",
			repos: []repo.Repo{
				{ID: "bipartite", Project: "dasm"},
			},
			wantEdgeCount: 1,
			wantEdges: []Edge{
				{Source: "bipartite", Target: "dasm", RelationshipType: RelationshipBelongsTo, Summary: ""},
			},
		},
		{
			name: "repo without project creates no edge",
			repos: []repo.Repo{
				{ID: "orphan-repo", Project: ""},
			},
			wantEdgeCount: 0,
			wantEdges:     nil,
		},
		{
			name: "repo with non-existent project creates no edge",
			repos: []repo.Repo{
				{ID: "missing-project-repo", Project: "unknown-project"},
			},
			wantEdgeCount: 0,
			wantEdges:     nil,
		},
		{
			name: "multiple repos with different projects",
			repos: []repo.Repo{
				{ID: "repo1", Project: "dasm"},
				{ID: "repo2", Project: "netam"},
				{ID: "repo3", Project: "dasm"},
			},
			wantEdgeCount: 3,
			wantEdges: []Edge{
				{Source: "repo1", Target: "dasm", RelationshipType: RelationshipBelongsTo, Summary: ""},
				{Source: "repo2", Target: "netam", RelationshipType: RelationshipBelongsTo, Summary: ""},
				{Source: "repo3", Target: "dasm", RelationshipType: RelationshipBelongsTo, Summary: ""},
			},
		},
		{
			name:          "empty repos list",
			repos:         []repo.Repo{},
			wantEdgeCount: 0,
			wantEdges:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEdges := buildRepoProjectEdges(tt.repos, projectIDs)

			if len(gotEdges) != tt.wantEdgeCount {
				t.Errorf("got %d edges, want %d", len(gotEdges), tt.wantEdgeCount)
			}

			for i, wantEdge := range tt.wantEdges {
				if i >= len(gotEdges) {
					break
				}
				gotEdge := gotEdges[i]
				if gotEdge.Source != wantEdge.Source {
					t.Errorf("edge %d: got source %q, want %q", i, gotEdge.Source, wantEdge.Source)
				}
				if gotEdge.Target != wantEdge.Target {
					t.Errorf("edge %d: got target %q, want %q", i, gotEdge.Target, wantEdge.Target)
				}
				if gotEdge.RelationshipType != wantEdge.RelationshipType {
					t.Errorf("edge %d: got relationshipType %q, want %q", i, gotEdge.RelationshipType, wantEdge.RelationshipType)
				}
				if gotEdge.Summary != wantEdge.Summary {
					t.Errorf("edge %d: got summary %q, want %q", i, gotEdge.Summary, wantEdge.Summary)
				}
			}
		})
	}
}

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"concept:somatic-hypermutation", "somatic-hypermutation"},
		{"project:dasm2", "dasm2"},
		{"Paper2023-ab", "Paper2023-ab"},
		{"no-prefix", "no-prefix"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeID(tt.input)
			if got != tt.want {
				t.Errorf("normalizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
