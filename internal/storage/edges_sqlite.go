package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/edge"
)

// ensureEdgesSchema ensures the edges schema exists (idempotent via CREATE IF NOT EXISTS).
func (d *DB) ensureEdgesSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS edges (
			source_id TEXT NOT NULL,
			target_id TEXT NOT NULL,
			relationship_type TEXT NOT NULL,
			summary TEXT NOT NULL,
			created_at TEXT,
			PRIMARY KEY (source_id, target_id, relationship_type)
		);

		CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
		CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
		CREATE INDEX IF NOT EXISTS idx_edges_type ON edges(relationship_type);
	`
	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("creating edges schema: %w", err)
	}
	return nil
}

// queryEdges executes a query and scans the results into edges.
// Ensures schema exists before querying.
func (d *DB) queryEdges(query string, errorContext string, args ...interface{}) ([]edge.Edge, error) {
	if err := d.ensureEdgesSchema(); err != nil {
		return nil, err
	}
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errorContext, err)
	}
	defer rows.Close()
	return scanEdges(rows)
}

// RebuildEdgesFromJSONL clears the edges table and rebuilds it from a JSONL file.
func (d *DB) RebuildEdgesFromJSONL(jsonlPath string) (int, error) {
	if err := d.ensureEdgesSchema(); err != nil {
		return 0, err
	}

	// Read all edges from JSONL
	edges, err := ReadAllEdges(jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading edges JSONL: %w", err)
	}

	// Clear existing data
	if _, err := d.db.Exec("DELETE FROM edges"); err != nil {
		return 0, fmt.Errorf("clearing edges table: %w", err)
	}

	// Prepare insert statement
	stmt, err := d.db.Prepare(`
		INSERT INTO edges (source_id, target_id, relationship_type, summary, created_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing edges insert: %w", err)
	}
	defer stmt.Close()

	for _, e := range edges {
		_, err = stmt.Exec(e.SourceID, e.TargetID, e.RelationshipType, e.Summary, e.CreatedAt)
		if err != nil {
			return 0, fmt.Errorf("inserting edge: %w", err)
		}
	}

	return len(edges), nil
}

// InsertEdge inserts a single edge into the database.
func (d *DB) InsertEdge(e edge.Edge) error {
	if err := d.ensureEdgesSchema(); err != nil {
		return err
	}

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO edges (source_id, target_id, relationship_type, summary, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, e.SourceID, e.TargetID, e.RelationshipType, e.Summary, e.CreatedAt)
	return err
}

// GetEdgesBySource returns all edges where the given paper is the source.
func (d *DB) GetEdgesBySource(sourceID string) ([]edge.Edge, error) {
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE source_id = ?
		ORDER BY target_id
	`, "querying edges by source", sourceID)
}

// GetEdgesByTarget returns all edges where the given paper is the target.
func (d *DB) GetEdgesByTarget(targetID string) ([]edge.Edge, error) {
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE target_id = ?
		ORDER BY source_id
	`, "querying edges by target", targetID)
}

// GetEdgesByType returns all edges with the given relationship type.
func (d *DB) GetEdgesByType(relationshipType string) ([]edge.Edge, error) {
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE relationship_type = ?
		ORDER BY source_id, target_id
	`, "querying edges by type", relationshipType)
}

// GetAllEdges returns all edges in the database.
func (d *DB) GetAllEdges() ([]edge.Edge, error) {
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		ORDER BY source_id, target_id, relationship_type
	`, "querying all edges")
}

// GetEdgesByPaper returns all edges involving the given paper (as source or target).
func (d *DB) GetEdgesByPaper(paperID string) ([]edge.Edge, error) {
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE source_id = ? OR target_id = ?
		ORDER BY source_id, target_id, relationship_type
	`, "querying edges by paper", paperID, paperID)
}

// CountEdges returns the total number of edges.
func (d *DB) CountEdges() (int, error) {
	if err := d.ensureEdgesSchema(); err != nil {
		return 0, err
	}

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&count)
	if err != nil {
		// Table might not exist yet (shouldn't happen after ensureEdgesSchema)
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// GetEdgesByProject returns all edges where the project (with prefix) is source or target.
func (d *DB) GetEdgesByProject(projectID string) ([]edge.Edge, error) {
	prefixedID := "project:" + projectID
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE source_id = ? OR target_id = ?
		ORDER BY source_id, target_id, relationship_type
	`, "querying edges by project", prefixedID, prefixedID)
}

// GetConceptsByProject returns edges where a concept links to a project (either direction).
func (d *DB) GetConceptsByProject(projectID string) ([]edge.Edge, error) {
	prefixedID := "project:" + projectID
	return d.queryEdges(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE (source_id = ? AND target_id LIKE 'concept:%')
		   OR (target_id = ? AND source_id LIKE 'concept:%')
		ORDER BY source_id, target_id
	`, "querying concepts by project", prefixedID, prefixedID)
}

// GetPapersByProjectTransitive returns papers linked to concepts that are linked to a project.
// This performs a two-hop query: project ← concept ← paper
func (d *DB) GetPapersByProjectTransitive(projectID string) ([]edge.Edge, error) {
	prefixedID := "project:" + projectID
	// First get concepts linked to this project
	// Then get papers linked to those concepts
	return d.queryEdges(`
		WITH project_concepts AS (
			SELECT
				CASE
					WHEN source_id = ? THEN target_id
					WHEN target_id = ? THEN source_id
				END AS concept_id
			FROM edges
			WHERE (source_id = ? AND target_id LIKE 'concept:%')
			   OR (target_id = ? AND source_id LIKE 'concept:%')
		)
		SELECT e.source_id, e.target_id, e.relationship_type, e.summary, e.created_at
		FROM edges e
		JOIN project_concepts pc ON e.target_id = REPLACE(pc.concept_id, 'concept:', '')
		WHERE e.source_id NOT LIKE '%:%'
		ORDER BY e.source_id, e.target_id
	`, "querying papers by project transitive", prefixedID, prefixedID, prefixedID, prefixedID)
}

// scanEdges scans rows into a slice of edges.
func scanEdges(rows *sql.Rows) ([]edge.Edge, error) {
	var edges []edge.Edge
	for rows.Next() {
		var e edge.Edge
		var createdAt sql.NullString
		err := rows.Scan(&e.SourceID, &e.TargetID, &e.RelationshipType, &e.Summary, &createdAt)
		if err != nil {
			return nil, err
		}
		if createdAt.Valid {
			e.CreatedAt = createdAt.String
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
