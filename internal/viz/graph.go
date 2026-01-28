package viz

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/concept"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/project"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/repo"
	"github.com/matsen/bipartite/internal/storage"
)

// ID prefixes used in edge source/target IDs.
const (
	conceptPrefix = "concept:"
	projectPrefix = "project:"
)

// BuildGraphFromDatabase queries the database and constructs a complete GraphData
// structure for visualization, including papers, concepts, projects, repos, and edges.
func BuildGraphFromDatabase(db *storage.DB) (*GraphData, error) {
	// Load all entity types
	concepts, err := db.GetAllConcepts()
	if err != nil {
		return nil, err
	}

	projects, err := db.GetAllProjects()
	if err != nil {
		return nil, err
	}

	repos, err := db.GetAllRepos()
	if err != nil {
		return nil, err
	}

	allEdges, err := db.GetAllEdges()
	if err != nil {
		return nil, err
	}

	// Build ID lookup maps
	conceptIDs := make(map[string]bool, len(concepts))
	for _, c := range concepts {
		conceptIDs[c.ID] = true
	}

	projectIDs := make(map[string]bool, len(projects))
	for _, p := range projects {
		projectIDs[p.ID] = true
	}

	// Process edges and collect connected node IDs
	vizEdges, paperIDs, conceptConnectionCounts, projectConnectionCounts := processEdges(allEdges, conceptIDs, projectIDs)

	// Build paper nodes (only those connected via edges)
	paperNodes, err := buildPaperNodes(db, paperIDs)
	if err != nil {
		return nil, err
	}

	// Build concept nodes with connection counts
	conceptNodes := buildConceptNodes(concepts, conceptConnectionCounts)

	// Build project nodes with connection counts
	projectNodes := buildProjectNodes(projects, projectConnectionCounts)

	// Build repo nodes (grouped under their projects)
	repoNodes := buildRepoNodes(repos)

	// Build repo→project edges (visual-only, derived from repo.Project field)
	repoProjectEdges := buildRepoProjectEdges(repos, projectIDs)
	vizEdges = append(vizEdges, repoProjectEdges...)

	// Combine all nodes
	var allNodes []Node
	allNodes = append(allNodes, paperNodes...)
	allNodes = append(allNodes, conceptNodes...)
	allNodes = append(allNodes, projectNodes...)
	allNodes = append(allNodes, repoNodes...)

	return &GraphData{
		Nodes: allNodes,
		Edges: vizEdges,
	}, nil
}

// processEdges processes all edges, normalizing prefixed IDs and tracking connections.
// Returns visualization edges, paper IDs to fetch, and connection counts for concepts and projects.
func processEdges(allEdges []edge.Edge, conceptIDs, projectIDs map[string]bool) ([]Edge, map[string]bool, map[string]int, map[string]int) {
	paperIDs := make(map[string]bool)
	conceptConnectionCounts := make(map[string]int)
	projectConnectionCounts := make(map[string]int)
	var vizEdges []Edge

	for _, e := range allEdges {
		sourceID := normalizeID(e.SourceID)
		targetID := normalizeID(e.TargetID)

		// Determine edge type and validate endpoints
		sourceIsConcept := conceptIDs[sourceID]
		targetIsConcept := conceptIDs[targetID]
		sourceIsProject := projectIDs[sourceID]
		targetIsProject := projectIDs[targetID]

		// Paper → Concept edges
		if targetIsConcept && !sourceIsConcept && !sourceIsProject {
			paperIDs[sourceID] = true
			conceptConnectionCounts[targetID]++
			vizEdges = append(vizEdges, Edge{
				Source:           sourceID,
				Target:           targetID,
				RelationshipType: e.RelationshipType,
				Summary:          e.Summary,
			})
			continue
		}

		// Concept → Project edges
		if sourceIsConcept && targetIsProject {
			conceptConnectionCounts[sourceID]++
			projectConnectionCounts[targetID]++
			vizEdges = append(vizEdges, Edge{
				Source:           sourceID,
				Target:           targetID,
				RelationshipType: e.RelationshipType,
				Summary:          e.Summary,
			})
			continue
		}

		// Project → Concept edges (reverse direction)
		if sourceIsProject && targetIsConcept {
			projectConnectionCounts[sourceID]++
			conceptConnectionCounts[targetID]++
			vizEdges = append(vizEdges, Edge{
				Source:           sourceID,
				Target:           targetID,
				RelationshipType: e.RelationshipType,
				Summary:          e.Summary,
			})
			continue
		}

		// Skip other edge types (e.g., paper-paper, invalid edges)
	}

	return vizEdges, paperIDs, conceptConnectionCounts, projectConnectionCounts
}

// normalizeID strips type prefixes from IDs for matching against node IDs.
func normalizeID(id string) string {
	if strings.HasPrefix(id, conceptPrefix) {
		return strings.TrimPrefix(id, conceptPrefix)
	}
	if strings.HasPrefix(id, projectPrefix) {
		return strings.TrimPrefix(id, projectPrefix)
	}
	return id
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

// buildProjectNodes constructs nodes for all projects with their connection counts.
func buildProjectNodes(projects []project.Project, connectionCounts map[string]int) []Node {
	nodes := make([]Node, 0, len(projects))

	for _, p := range projects {
		nodes = append(nodes, newProjectNode(p, connectionCounts[p.ID]))
	}

	return nodes
}

// buildRepoNodes constructs nodes for all repos.
func buildRepoNodes(repos []repo.Repo) []Node {
	nodes := make([]Node, 0, len(repos))

	for _, r := range repos {
		nodes = append(nodes, newRepoNode(r))
	}

	return nodes
}

// buildRepoProjectEdges creates visual edges from repos to their parent projects.
// These are not stored in the edge data - they're derived from the repo's project field.
func buildRepoProjectEdges(repos []repo.Repo, projectIDs map[string]bool) []Edge {
	var edges []Edge

	for _, r := range repos {
		if r.Project == "" {
			continue
		}
		// Only create edge if the parent project exists
		if !projectIDs[r.Project] {
			continue
		}
		edges = append(edges, Edge{
			Source:           r.ID,
			Target:           r.Project,
			RelationshipType: "belongs-to",
			Summary:          "",
		})
	}

	return edges
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

// newProjectNode creates a visualization node from a project with its connection count.
func newProjectNode(p project.Project, connectionCount int) Node {
	return Node{
		ID:              p.ID,
		Type:            NodeTypeProject,
		Label:           p.Name,
		Name:            p.Name,
		Description:     p.Description,
		ConnectionCount: connectionCount,
	}
}

// newRepoNode creates a visualization node from a repo.
func newRepoNode(r repo.Repo) Node {
	return Node{
		ID:          r.ID,
		Type:        NodeTypeRepo,
		Label:       r.Name,
		Name:        r.Name,
		Description: r.Description,
		ProjectID:   r.Project,
		RepoType:    r.Type,
		GitHubURL:   r.GitHubURL,
		Language:    r.Language,
		Topics:      r.Topics,
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
