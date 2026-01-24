package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/matsen/bipartite/internal/repo"
)

// ensureReposSchema ensures the repos schema exists (idempotent via CREATE IF NOT EXISTS).
func (d *DB) ensureReposSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS repos (
			id TEXT PRIMARY KEY,
			project TEXT NOT NULL,
			type TEXT NOT NULL CHECK (type IN ('github', 'manual')),
			name TEXT NOT NULL,
			github_url TEXT,
			description TEXT,
			topics_json TEXT,
			language TEXT,
			created_at TEXT,
			updated_at TEXT,
			UNIQUE(github_url)
		);

		CREATE INDEX IF NOT EXISTS idx_repos_project ON repos(project);
		CREATE INDEX IF NOT EXISTS idx_repos_github_url ON repos(github_url) WHERE github_url IS NOT NULL;
	`
	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("creating repos schema: %w", err)
	}
	return nil
}

// RebuildReposFromJSONL clears the repos table and rebuilds it from a JSONL file.
func (d *DB) RebuildReposFromJSONL(jsonlPath string) (int, error) {
	if err := d.ensureReposSchema(); err != nil {
		return 0, err
	}

	// Read all repos from JSONL
	repos, err := ReadAllRepos(jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading repos JSONL: %w", err)
	}

	// Clear existing data
	if _, err := d.db.Exec("DELETE FROM repos"); err != nil {
		return 0, fmt.Errorf("clearing repos table: %w", err)
	}

	// Prepare insert statement
	stmt, err := d.db.Prepare(`
		INSERT INTO repos (id, project, type, name, github_url, description, topics_json, language, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing repos insert: %w", err)
	}
	defer stmt.Close()

	for _, r := range repos {
		// Serialize topics to JSON
		var topicsJSON string
		if len(r.Topics) > 0 {
			topicsBytes, err := json.Marshal(r.Topics)
			if err != nil {
				return 0, fmt.Errorf("marshaling topics for %s: %w", r.ID, err)
			}
			topicsJSON = string(topicsBytes)
		}

		_, err = stmt.Exec(
			r.ID, r.Project, r.Type, r.Name,
			nullableStringFromGo(r.GitHubURL),
			nullableStringFromGo(r.Description),
			nullableStringFromGo(topicsJSON),
			nullableStringFromGo(r.Language),
			r.CreatedAt, r.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf("inserting repo %s: %w", r.ID, err)
		}
	}

	return len(repos), nil
}

// GetRepoByID retrieves a repo by its ID.
func (d *DB) GetRepoByID(id string) (*repo.Repo, error) {
	if err := d.ensureReposSchema(); err != nil {
		return nil, err
	}

	row := d.db.QueryRow(`
		SELECT id, project, type, name, github_url, description, topics_json, language, created_at, updated_at
		FROM repos
		WHERE id = ?
	`, id)

	return scanRepo(row)
}

// GetAllRepos returns all repos in the database.
func (d *DB) GetAllRepos() ([]repo.Repo, error) {
	if err := d.ensureReposSchema(); err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT id, project, type, name, github_url, description, topics_json, language, created_at, updated_at
		FROM repos
		ORDER BY project, id
	`)
	if err != nil {
		return nil, fmt.Errorf("querying repos: %w", err)
	}
	defer rows.Close()

	return scanRepos(rows)
}

// GetReposByProject returns all repos belonging to a project.
func (d *DB) GetReposByProject(projectID string) ([]repo.Repo, error) {
	if err := d.ensureReposSchema(); err != nil {
		return nil, err
	}

	rows, err := d.db.Query(`
		SELECT id, project, type, name, github_url, description, topics_json, language, created_at, updated_at
		FROM repos
		WHERE project = ?
		ORDER BY id
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("querying repos by project: %w", err)
	}
	defer rows.Close()

	return scanRepos(rows)
}

// CountRepos returns the total number of repos.
func (d *DB) CountRepos() (int, error) {
	if err := d.ensureReposSchema(); err != nil {
		return 0, err
	}

	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM repos").Scan(&count)
	return count, err
}

// scanRepo scans a single repo from a row.
func scanRepo(row *sql.Row) (*repo.Repo, error) {
	var r repo.Repo
	var githubURL, description, topicsJSON, language, createdAt, updatedAt sql.NullString

	err := row.Scan(&r.ID, &r.Project, &r.Type, &r.Name, &githubURL, &description, &topicsJSON, &language, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	r.GitHubURL = githubURL.String
	r.Description = description.String
	r.Language = language.String
	r.CreatedAt = createdAt.String
	r.UpdatedAt = updatedAt.String

	if topicsJSON.Valid && topicsJSON.String != "" {
		if err := json.Unmarshal([]byte(topicsJSON.String), &r.Topics); err != nil {
			return nil, fmt.Errorf("parsing topics JSON for %s: %w", r.ID, err)
		}
	}

	return &r, nil
}

// scanRepos scans multiple repos from rows.
func scanRepos(rows *sql.Rows) ([]repo.Repo, error) {
	var repos []repo.Repo
	for rows.Next() {
		var r repo.Repo
		var githubURL, description, topicsJSON, language, createdAt, updatedAt sql.NullString

		if err := rows.Scan(&r.ID, &r.Project, &r.Type, &r.Name, &githubURL, &description, &topicsJSON, &language, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		r.GitHubURL = githubURL.String
		r.Description = description.String
		r.Language = language.String
		r.CreatedAt = createdAt.String
		r.UpdatedAt = updatedAt.String

		if topicsJSON.Valid && topicsJSON.String != "" {
			if err := json.Unmarshal([]byte(topicsJSON.String), &r.Topics); err != nil {
				return nil, fmt.Errorf("parsing topics JSON for %s: %w", r.ID, err)
			}
		}

		repos = append(repos, r)
	}
	return repos, rows.Err()
}
