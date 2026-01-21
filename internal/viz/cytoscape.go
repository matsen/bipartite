package viz

import "encoding/json"

// CytoscapeElements represents the Cytoscape.js data format.
type CytoscapeElements struct {
	Nodes []CytoscapeNode `json:"nodes"`
	Edges []CytoscapeEdge `json:"edges"`
}

// CytoscapeNode represents a node in Cytoscape.js format.
type CytoscapeNode struct {
	Data CytoscapeNodeData `json:"data"`
}

// CytoscapeNodeData contains the node data fields.
type CytoscapeNodeData struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`

	// Paper fields
	Title   string `json:"title,omitempty"`
	Authors string `json:"authors,omitempty"`
	Year    int    `json:"year,omitempty"`

	// Concept fields
	Name        string   `json:"name,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description,omitempty"`

	// Sizing
	ConnectionCount int `json:"connectionCount"`
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
		cyNode := CytoscapeNode{
			Data: CytoscapeNodeData{
				ID:              n.ID,
				Type:            n.Type,
				Label:           n.Label,
				Title:           n.Title,
				Authors:         n.Authors,
				Year:            n.Year,
				Name:            n.Name,
				Aliases:         n.Aliases,
				Description:     n.Description,
				ConnectionCount: n.ConnectionCount,
			},
		}
		elements.Nodes = append(elements.Nodes, cyNode)
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
		return "", err
	}
	return string(jsonBytes), nil
}

// edgeID generates a unique edge ID.
func edgeID(source, target, relType string, index int) string {
	return source + "-" + target + "-" + relType
}
