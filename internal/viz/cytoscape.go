package viz

import (
	"encoding/json"
	"fmt"
)

// CytoscapeElements represents the Cytoscape.js data format.
type CytoscapeElements struct {
	Nodes []CytoscapeNode `json:"nodes"`
	Edges []CytoscapeEdge `json:"edges"`
}

// CytoscapeNode represents a node in Cytoscape.js format.
type CytoscapeNode struct {
	Data Node `json:"data"`
}

// CytoscapeEdge represents an edge in Cytoscape.js format.
type CytoscapeEdge struct {
	Data CytoscapeEdgeData `json:"data"`
}

// CytoscapeEdgeData contains the edge data fields.
type CytoscapeEdgeData struct {
	ID               string `json:"id"`
	Source           string `json:"source"`
	Target           string `json:"target"`
	RelationshipType string `json:"relationshipType"`
	Summary          string `json:"summary"`
}

// ToCytoscapeJSON converts GraphData to Cytoscape.js JSON format.
func (g *GraphData) ToCytoscapeJSON() (string, error) {
	elements := CytoscapeElements{
		Nodes: make([]CytoscapeNode, 0, len(g.Nodes)),
		Edges: make([]CytoscapeEdge, 0, len(g.Edges)),
	}

	for _, n := range g.Nodes {
		elements.Nodes = append(elements.Nodes, CytoscapeNode{Data: n})
	}

	for i, e := range g.Edges {
		cyEdge := CytoscapeEdge{
			Data: CytoscapeEdgeData{
				ID:               edgeID(e.Source, e.Target, e.RelationshipType, i),
				Source:           e.Source,
				Target:           e.Target,
				RelationshipType: e.RelationshipType,
				Summary:          e.Summary,
			},
		}
		elements.Edges = append(elements.Edges, cyEdge)
	}

	jsonBytes, err := json.Marshal(elements)
	if err != nil {
		return "", fmt.Errorf("marshaling Cytoscape elements to JSON: %w", err)
	}
	return string(jsonBytes), nil
}

// edgeID generates a unique edge ID for the current visualization session.
// IDs are based on slice position and are not stable across different graph builds.
func edgeID(source, target, relType string, index int) string {
	return fmt.Sprintf("%s-%s-%s-%d", source, target, relType, index)
}
