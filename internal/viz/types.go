// Package viz provides knowledge graph visualization functionality.
package viz

// GraphData contains all data needed to render the visualization.
type GraphData struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node represents a paper or concept in the graph.
type Node struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "paper" or "concept"

	// Display
	Label string `json:"label"`

	// Paper-specific fields (for tooltips)
	Title   string `json:"title,omitempty"`
	Authors string `json:"authors,omitempty"` // Formatted string "First Last, First Last"
	Year    int    `json:"year,omitempty"`

	// Concept-specific fields (for tooltips)
	Name        string   `json:"name,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description,omitempty"`

	// Sizing (for concept nodes)
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
