package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/concept"
)

// ensureConceptsSchema ensures the concepts schema exists (idempotent via CREATE IF NOT EXISTS).
func (d *DB) ensureConceptsSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS concepts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			aliases_json TEXT,
			description TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_concepts_name ON concepts(name);

		CREATE VIRTUAL TABLE IF NOT EXISTS concepts_fts USING fts5(
			id,
			name,
			aliases_text,
			description
		);
	`
	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("creating concepts schema: %w", err)
	}
	return nil
}

// RebuildConceptsFromJSONL clears the concepts tables and rebuilds them from a JSONL file.
func (d *DB) RebuildConceptsFromJSONL(jsonlPath string) (int, error) {
	if err := d.ensureConceptsSchema(); err != nil {
		return 0, err
	}

	// Read all concepts from JSONL
	concepts, err := ReadAllConcepts(jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading concepts JSONL: %w", err)
	}

	// Clear existing data
	if _, err := d.db.Exec("DELETE FROM concepts"); err != nil {
		return 0, fmt.Errorf("clearing concepts table: %w", err)
	}
	if _, err := d.db.Exec("DELETE FROM concepts_fts"); err != nil {
		return 0, fmt.Errorf("clearing concepts_fts table: %w", err)
	}

	// Prepare insert statements
	conceptsStmt, err := d.db.Prepare(`
		INSERT INTO concepts (id, name, aliases_json, description)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing concepts insert: %w", err)
	}
	defer conceptsStmt.Close()

	ftsStmt, err := d.db.Prepare(`
		INSERT INTO concepts_fts (id, name, aliases_text, description)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing concepts_fts insert: %w", err)
	}
	defer ftsStmt.Close()

	for _, c := range concepts {
		// Serialize aliases to JSON
		var aliasesJSON string
		if len(c.Aliases) > 0 {
			aliasesBytes, err := json.Marshal(c.Aliases)
			if err != nil {
				return 0, fmt.Errorf("marshaling aliases for %s: %w", c.ID, err)
			}
			aliasesJSON = string(aliasesBytes)
		}

		// Insert into concepts table
		_, err = conceptsStmt.Exec(c.ID, c.Name, nullableStringFromGo(aliasesJSON), c.Description)
		if err != nil {
			return 0, fmt.Errorf("inserting concept %s: %w", c.ID, err)
		}

		// Build aliases text for FTS (space-joined)
		aliasesText := strings.Join(c.Aliases, " ")

		// Insert into FTS table
		_, err = ftsStmt.Exec(c.ID, c.Name, aliasesText, c.Description)
		if err != nil {
			return 0, fmt.Errorf("inserting concepts_fts for %s: %w", c.ID, err)
		}
	}

	return len(concepts), nil
}

// nullableStringFromGo converts a Go string to sql.NullString.
func nullableStringFromGo(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// GetConceptByID retrieves a concept by its ID.
func (d *DB) GetConceptByID(id string) (*concept.Concept, error) {
	if err := d.ensureConceptsSchema(); err != nil {
		return nil, err
	}

	row := d.db.QueryRow(`
		SELECT id, name, aliases_json, description
		FROM concepts
		WHERE id = ?
	`, id)

	return scanConcept(row)
}

// GetAllConcepts returns all concepts in the database.
func (d *DB) GetAllConcepts() ([]concept.Concept, error) {
	if err := d.ensureConceptsSchema(); err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT id, name, aliases_json, description
		FROM concepts
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying concepts: %w", err)
	}
	defer rows.Close()

	return scanConcepts(rows)
}

// SearchConcepts performs a full-text search on concepts.
func (d *DB) SearchConcepts(query string, limit int) ([]concept.Concept, error) {
	if err := d.ensureConceptsSchema(); err != nil {
		return nil, err
	}

	ftsQuery := prepareFTSQuery(query)

	rows, err := d.db.Query(`
		SELECT c.id, c.name, c.aliases_json, c.description
		FROM concepts c
		WHERE c.id IN (SELECT id FROM concepts_fts WHERE concepts_fts MATCH ?)
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("searching concepts: %w", err)
	}
	defer rows.Close()

	return scanConcepts(rows)
}

// CountConcepts returns the total number of concepts.
func (d *DB) CountConcepts() (int, error) {
	if err := d.ensureConceptsSchema(); err != nil {
		return 0, err
	}

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM concepts").Scan(&count)
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// GetPapersByConcept returns all papers linked to a concept, optionally filtered by relationship type.
func (d *DB) GetPapersByConcept(conceptID string, relationshipType string) ([]PaperConceptEdge, error) {
	if err := d.ensureEdgesSchema(); err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	if relationshipType != "" {
		query = `
			SELECT e.source_id, e.relationship_type, e.summary
			FROM edges e
			WHERE e.target_id = ? AND e.relationship_type = ?
			ORDER BY e.relationship_type, e.source_id
		`
		args = []interface{}{conceptID, relationshipType}
	} else {
		query = `
			SELECT e.source_id, e.relationship_type, e.summary
			FROM edges e
			WHERE e.target_id = ?
			ORDER BY e.relationship_type, e.source_id
		`
		args = []interface{}{conceptID}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying papers by concept: %w", err)
	}
	defer rows.Close()

	var results []PaperConceptEdge
	for rows.Next() {
		var pce PaperConceptEdge
		if err := rows.Scan(&pce.PaperID, &pce.RelationshipType, &pce.Summary); err != nil {
			return nil, err
		}
		results = append(results, pce)
	}

	return results, rows.Err()
}

// GetConceptsByPaper returns all concepts linked to a paper, optionally filtered by relationship type.
func (d *DB) GetConceptsByPaper(paperID string, relationshipType string) ([]PaperConceptEdge, error) {
	if err := d.ensureEdgesSchema(); err != nil {
		return nil, err
	}
	if err := d.ensureConceptsSchema(); err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	if relationshipType != "" {
		query = `
			SELECT e.target_id, e.relationship_type, e.summary
			FROM edges e
			WHERE e.source_id = ? AND e.relationship_type = ?
			  AND e.target_id IN (SELECT id FROM concepts)
			ORDER BY e.relationship_type, e.target_id
		`
		args = []interface{}{paperID, relationshipType}
	} else {
		query = `
			SELECT e.target_id, e.relationship_type, e.summary
			FROM edges e
			WHERE e.source_id = ?
			  AND e.target_id IN (SELECT id FROM concepts)
			ORDER BY e.relationship_type, e.target_id
		`
		args = []interface{}{paperID}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying concepts by paper: %w", err)
	}
	defer rows.Close()

	var results []PaperConceptEdge
	for rows.Next() {
		var pce PaperConceptEdge
		if err := rows.Scan(&pce.ConceptID, &pce.RelationshipType, &pce.Summary); err != nil {
			return nil, err
		}
		results = append(results, pce)
	}

	return results, rows.Err()
}

// CountEdgesByTarget returns the number of edges pointing to a given target.
func (d *DB) CountEdgesByTarget(targetID string) (int, error) {
	if err := d.ensureEdgesSchema(); err != nil {
		return 0, err
	}

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM edges WHERE target_id = ?", targetID).Scan(&count)
	return count, err
}

// PaperConceptEdge represents an edge between a paper and concept in query results.
//
// Field population depends on the query direction:
//   - GetPapersByConcept: PaperID is populated (the paper linking to the concept)
//   - GetConceptsByPaper: ConceptID is populated (the concept linked from the paper)
//
// The omitempty tags ensure only the relevant ID field appears in JSON output.
type PaperConceptEdge struct {
	PaperID          string `json:"paper_id,omitempty"`
	ConceptID        string `json:"concept_id,omitempty"`
	RelationshipType string `json:"relationship_type"`
	Summary          string `json:"summary"`
}

// populateConceptFields deserializes aliasesJSON and description into a concept.
// This is the single source of truth for converting database fields to Concept struct fields.
func populateConceptFields(c *concept.Concept, aliasesJSON, description sql.NullString) error {
	if aliasesJSON.Valid && aliasesJSON.String != "" {
		if err := json.Unmarshal([]byte(aliasesJSON.String), &c.Aliases); err != nil {
			return fmt.Errorf("parsing aliases JSON for %s: %w", c.ID, err)
		}
	}
	c.Description = description.String
	return nil
}

// scanConcept scans a single concept from a row.
func scanConcept(row *sql.Row) (*concept.Concept, error) {
	var c concept.Concept
	var aliasesJSON, description sql.NullString

	err := row.Scan(&c.ID, &c.Name, &aliasesJSON, &description)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := populateConceptFields(&c, aliasesJSON, description); err != nil {
		return nil, err
	}
	return &c, nil
}

// scanConcepts scans multiple concepts from rows.
func scanConcepts(rows *sql.Rows) ([]concept.Concept, error) {
	var concepts []concept.Concept
	for rows.Next() {
		var c concept.Concept
		var aliasesJSON, description sql.NullString

		if err := rows.Scan(&c.ID, &c.Name, &aliasesJSON, &description); err != nil {
			return nil, err
		}
		if err := populateConceptFields(&c, aliasesJSON, description); err != nil {
			return nil, err
		}
		concepts = append(concepts, c)
	}
	return concepts, rows.Err()
}
