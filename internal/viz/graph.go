package viz

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/concept"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// BuildGraphFromDatabase queries the database and constructs a complete GraphData
// structure for visualization, including filtered edges, connection counts, and
// type-specific nodes.
func BuildGraphFromDatabase(db *storage.DB) (*GraphData, error) {
	concepts, err := db.GetAllConcepts()
	if err != nil {
		return nil, err
	}

	allEdges, err := db.GetAllEdges()
	if err != nil {
		return nil, err
	}

	filteredEdges, paperIDs, connectionCounts := filterEdgesToConcepts(allEdges, concepts)

	paperNodes, err := buildPaperNodes(db, paperIDs)
	if err != nil {
		return nil, err
	}

	conceptNodes := buildConceptNodes(concepts, connectionCounts)

	return &GraphData{
		Nodes: append(paperNodes, conceptNodes...),
		Edges: filteredEdges,
	}, nil
}

// filterEdgesToConcepts filters edges to only paper->concept relationships and
// tracks which papers are involved and how many connections each concept has.
func filterEdgesToConcepts(allEdges []edge.Edge, concepts []concept.Concept) ([]Edge, map[string]bool, map[string]int) {
	conceptIDs := make(map[string]bool, len(concepts))
	for _, c := range concepts {
		conceptIDs[c.ID] = true
	}

	paperIDs := make(map[string]bool)
	connectionCounts := make(map[string]int)
	var filteredEdges []Edge

	for _, e := range allEdges {
		if conceptIDs[e.TargetID] {
			paperIDs[e.SourceID] = true
			connectionCounts[e.TargetID]++
			filteredEdges = append(filteredEdges, Edge{
				Source:           e.SourceID,
				Target:           e.TargetID,
				RelationshipType: e.RelationshipType,
				Summary:          e.Summary,
			})
		}
	}

	return filteredEdges, paperIDs, connectionCounts
}

// buildPaperNodes fetches paper details and constructs nodes for papers with concept edges.
func buildPaperNodes(db *storage.DB, paperIDs map[string]bool) ([]Node, error) {
	nodes := make([]Node, 0, len(paperIDs))

	for paperID := range paperIDs {
		ref, err := db.GetByID(paperID)
		if err != nil {
			return nil, fmt.Errorf("retrieving paper %s: %w", paperID, err)
		}
		if ref == nil {
			return nil, fmt.Errorf("data integrity error: edge references non-existent paper %s", paperID)
		}
		nodes = append(nodes, newPaperNode(ref))
	}

	return nodes, nil
}

// buildConceptNodes constructs nodes for all concepts with their connection counts.
func buildConceptNodes(concepts []concept.Concept, connectionCounts map[string]int) []Node {
	nodes := make([]Node, 0, len(concepts))

	for _, c := range concepts {
		nodes = append(nodes, newConceptNode(c, connectionCounts[c.ID]))
	}

	return nodes
}

// newPaperNode creates a visualization node from a paper reference.
func newPaperNode(ref *reference.Reference) Node {
	return Node{
		ID:      ref.ID,
		Type:    NodeTypePaper,
		Label:   ref.ID,
		Title:   ref.Title,
		Authors: authorsToString(ref.Authors),
		Year:    ref.Published.Year,
	}
}

// newConceptNode creates a visualization node from a concept with its connection count.
func newConceptNode(c concept.Concept, connectionCount int) Node {
	return Node{
		ID:              c.ID,
		Type:            NodeTypeConcept,
		Label:           c.Name,
		Name:            c.Name,
		Aliases:         c.Aliases,
		Description:     c.Description,
		ConnectionCount: connectionCount,
	}
}

// authorsToString converts a slice of Author to a comma-separated "First Last" format string.
func authorsToString(authors []reference.Author) string {
	if len(authors) == 0 {
		return ""
	}

	names := make([]string, 0, len(authors))
	for _, a := range authors {
		if a.First != "" {
			names = append(names, a.First+" "+a.Last)
		} else {
			names = append(names, a.Last)
		}
	}
	return strings.Join(names, ", ")
}
