package storage

import (
	"database/sql"
	"fmt"

	"github.com/matsen/bipartite/internal/project"
)

// ensureProjectsSchema ensures the projects schema exists (idempotent via CREATE IF NOT EXISTS).
func (d *DB) ensureProjectsSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			created_at TEXT,
			updated_at TEXT
		);
	`
	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("creating projects schema: %w", err)
	}
	return nil
}

// RebuildProjectsFromJSONL clears the projects table and rebuilds it from a JSONL file.
func (d *DB) RebuildProjectsFromJSONL(jsonlPath string) (int, error) {
	if err := d.ensureProjectsSchema(); err != nil {
		return 0, err
	}

	// Read all projects from JSONL
	projects, err := ReadAllProjects(jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading projects JSONL: %w", err)
	}

	// Clear existing data
	if _, err := d.db.Exec("DELETE FROM projects"); err != nil {
		return 0, fmt.Errorf("clearing projects table: %w", err)
	}

	// Prepare insert statement
	stmt, err := d.db.Prepare(`
		INSERT INTO projects (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing projects insert: %w", err)
	}
	defer stmt.Close()

	for _, p := range projects {
		_, err = stmt.Exec(p.ID, p.Name, nullableStringFromGo(p.Description), p.CreatedAt, p.UpdatedAt)
		if err != nil {
			return 0, fmt.Errorf("inserting project %s: %w", p.ID, err)
		}
	}

	return len(projects), nil
}

// GetProjectByID retrieves a project by its ID.
func (d *DB) GetProjectByID(id string) (*project.Project, error) {
	if err := d.ensureProjectsSchema(); err != nil {
		return nil, err
	}

	row := d.db.QueryRow(`
		SELECT id, name, description, created_at, updated_at
		FROM projects
		WHERE id = ?
	`, id)

	return scanProject(row)
}

// GetAllProjects returns all projects in the database.
func (d *DB) GetAllProjects() ([]project.Project, error) {
	if err := d.ensureProjectsSchema(); err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT id, name, description, created_at, updated_at
		FROM projects
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying projects: %w", err)
	}
	defer rows.Close()

	return scanProjects(rows)
}

// CountProjects returns the total number of projects.
func (d *DB) CountProjects() (int, error) {
	if err := d.ensureProjectsSchema(); err != nil {
		return 0, err
	}

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&count)
	return count, err
}

// scanProject scans a single project from a row.
func scanProject(row *sql.Row) (*project.Project, error) {
	var p project.Project
	var description, createdAt, updatedAt sql.NullString

	err := row.Scan(&p.ID, &p.Name, &description, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	p.Description = description.String
	p.CreatedAt = createdAt.String
	p.UpdatedAt = updatedAt.String
	return &p, nil
}

// scanProjects scans multiple projects from rows.
func scanProjects(rows *sql.Rows) ([]project.Project, error) {
	var projects []project.Project
	for rows.Next() {
		var p project.Project
		var description, createdAt, updatedAt sql.NullString

		if err := rows.Scan(&p.ID, &p.Name, &description, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.Description = description.String
		p.CreatedAt = createdAt.String
		p.UpdatedAt = updatedAt.String
		projects = append(projects, p)
	}
	return projects, rows.Err()
}
