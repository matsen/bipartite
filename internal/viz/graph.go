package viz

import (
	"strings"

	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// ExtractGraphData queries the database and builds GraphData for visualization.
// It includes:
// - Paper nodes: only papers that have edges to concepts
// - Concept nodes: all concepts, with connection counts
// - Edges: all paper-to-concept edges
func ExtractGraphData(db *storage.DB) (*GraphData, error) {
	graph := &GraphData{
		Nodes: []Node{},
		Edges: []Edge{},
	}

	// Get all concepts
	concepts, err := db.GetAllConcepts()
	if err != nil {
		return nil, err
	}

	// Build a set of concept IDs for edge filtering
	conceptIDs := make(map[string]bool)
	for _, c := range concepts {
		conceptIDs[c.ID] = true
	}

	// Get all edges
	allEdges, err := db.GetAllEdges()
	if err != nil {
		return nil, err
	}

	// Filter to paper->concept edges and track which papers are involved
	paperIDs := make(map[string]bool)
	connectionCounts := make(map[string]int)
	var vizEdges []Edge

	for _, e := range allEdges {
		// Only include edges where target is a concept
		if conceptIDs[e.TargetID] {
			paperIDs[e.SourceID] = true
			connectionCounts[e.TargetID]++
			vizEdges = append(vizEdges, Edge{
				Source:           e.SourceID,
				Target:           e.TargetID,
				RelationshipType: e.RelationshipType,
				Summary:          e.Summary,
			})
		}
	}
	graph.Edges = vizEdges

	// Get paper details for papers with concept edges
	for paperID := range paperIDs {
		ref, err := db.GetByID(paperID)
		if err != nil {
			return nil, err
		}
		if ref == nil {
			// Paper referenced in edge but not found in database - skip
			continue
		}

		node := Node{
			ID:      ref.ID,
			Type:    "paper",
			Label:   ref.ID,
			Title:   ref.Title,
			Authors: formatAuthors(ref.Authors),
			Year:    ref.Published.Year,
		}
		graph.Nodes = append(graph.Nodes, node)
	}

	// Add concept nodes with connection counts
	for _, c := range concepts {
		node := Node{
			ID:              c.ID,
			Type:            "concept",
			Label:           c.Name,
			Name:            c.Name,
			Aliases:         c.Aliases,
			Description:     c.Description,
			ConnectionCount: connectionCounts[c.ID],
		}
		graph.Nodes = append(graph.Nodes, node)
	}

	return graph, nil
}

// formatAuthors converts a slice of Author to "First Last, First Last" format.
func formatAuthors(authors []reference.Author) string {
	if len(authors) == 0 {
		return ""
	}

	var names []string
	for _, a := range authors {
		if a.First != "" {
			names = append(names, a.First+" "+a.Last)
		} else {
			names = append(names, a.Last)
		}
	}
	return strings.Join(names, ", ")
}
