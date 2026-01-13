package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/edge"
)

// createEdgesSchema creates the edges table and indexes.
func (d *DB) createEdgesSchema() error {
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
	return err
}

// RebuildEdgesFromJSONL clears the edges table and rebuilds it from a JSONL file.
func (d *DB) RebuildEdgesFromJSONL(jsonlPath string) (int, error) {
	// Create schema if needed
	if err := d.createEdgesSchema(); err != nil {
		return 0, fmt.Errorf("creating edges schema: %w", err)
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
	// Create schema if needed
	if err := d.createEdgesSchema(); err != nil {
		return fmt.Errorf("creating edges schema: %w", err)
	}

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO edges (source_id, target_id, relationship_type, summary, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, e.SourceID, e.TargetID, e.RelationshipType, e.Summary, e.CreatedAt)
	return err
}

// GetEdgesBySource returns all edges where the given paper is the source.
func (d *DB) GetEdgesBySource(sourceID string) ([]edge.Edge, error) {
	rows, err := d.db.Query(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE source_id = ?
		ORDER BY target_id
	`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("querying edges by source: %w", err)
	}
	defer rows.Close()

	return scanEdges(rows)
}

// GetEdgesByTarget returns all edges where the given paper is the target.
func (d *DB) GetEdgesByTarget(targetID string) ([]edge.Edge, error) {
	rows, err := d.db.Query(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE target_id = ?
		ORDER BY source_id
	`, targetID)
	if err != nil {
		return nil, fmt.Errorf("querying edges by target: %w", err)
	}
	defer rows.Close()

	return scanEdges(rows)
}

// GetEdgesByType returns all edges with the given relationship type.
func (d *DB) GetEdgesByType(relationshipType string) ([]edge.Edge, error) {
	rows, err := d.db.Query(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE relationship_type = ?
		ORDER BY source_id, target_id
	`, relationshipType)
	if err != nil {
		return nil, fmt.Errorf("querying edges by type: %w", err)
	}
	defer rows.Close()

	return scanEdges(rows)
}

// GetAllEdges returns all edges in the database.
func (d *DB) GetAllEdges() ([]edge.Edge, error) {
	rows, err := d.db.Query(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		ORDER BY source_id, target_id, relationship_type
	`)
	if err != nil {
		return nil, fmt.Errorf("querying all edges: %w", err)
	}
	defer rows.Close()

	return scanEdges(rows)
}

// GetEdgesByPaper returns all edges involving the given paper (as source or target).
func (d *DB) GetEdgesByPaper(paperID string) ([]edge.Edge, error) {
	rows, err := d.db.Query(`
		SELECT source_id, target_id, relationship_type, summary, created_at
		FROM edges
		WHERE source_id = ? OR target_id = ?
		ORDER BY source_id, target_id, relationship_type
	`, paperID, paperID)
	if err != nil {
		return nil, fmt.Errorf("querying edges by paper: %w", err)
	}
	defer rows.Close()

	return scanEdges(rows)
}

// CountEdges returns the total number of edges.
func (d *DB) CountEdges() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&count)
	if err != nil {
		// Table might not exist yet
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
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
