// Package viz provides knowledge graph visualization functionality.
package viz

// Node type constants for type-safe node classification.
const (
	NodeTypePaper   = "paper"
	NodeTypeConcept = "concept"
	NodeTypeProject = "project"
	NodeTypeRepo    = "repo"
)

// GraphData contains all data needed to render the visualization.
type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node represents a paper, concept, project, or repo in the graph.
type Node struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // NodeTypePaper, NodeTypeConcept, NodeTypeProject, or NodeTypeRepo
	Label string `json:"label"`

	// Paper-specific fields (for tooltips)
	Title   string `json:"title,omitempty"`
	Authors string `json:"authors,omitempty"` // Formatted string "First Last, First Last"
	Year    int    `json:"year,omitempty"`

	// Concept-specific fields (for tooltips)
	Name        string   `json:"name,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description,omitempty"`

	// Project-specific fields (for tooltips)
	// Uses Name and Description (shared with concept)

	// Repo-specific fields (for tooltips)
	ProjectID string   `json:"projectId,omitempty"` // Parent project ID
	RepoType  string   `json:"repoType,omitempty"`  // "github" or "manual"
	GitHubURL string   `json:"githubUrl,omitempty"`
	Language  string   `json:"language,omitempty"`
	Topics    []string `json:"topics,omitempty"`

	// Sizing (for concept and project nodes)
	ConnectionCount int `json:"connectionCount"`
}

// Edge represents a paper-concept relationship.
type Edge struct {
	Source           string `json:"source"`
	Target           string `json:"target"`
	RelationshipType string `json:"relationshipType"`
	Summary          string `json:"summary"`
}

// IsEmpty returns true if the graph has no nodes.
func (g *GraphData) IsEmpty() bool {
	return len(g.Nodes) == 0
}
