package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection.
type DB struct {
	db *sql.DB
}

// selectRefFields contains the standard field list for SELECT queries.
const selectRefFields = `id, doi, title, abstract, venue,
	pub_year, pub_month, pub_day,
	pdf_path, source_type, source_id, supersedes,
	authors_json, supplement_paths_json,
	pmid, pmcid, arxiv_id, s2_id`

// OpenDB opens or creates a SQLite database at the given path.
func OpenDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set pragmas for better performance
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes

	// Create schema if needed
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// createSchema creates the database schema if it doesn't exist.
func createSchema(db *sql.DB) error {
	schema := `
		-- Main references table
		CREATE TABLE IF NOT EXISTS refs (
			id TEXT PRIMARY KEY,
			doi TEXT,
			title TEXT NOT NULL,
			abstract TEXT,
			venue TEXT,
			pub_year INTEGER NOT NULL,
			pub_month INTEGER,
			pub_day INTEGER,
			pdf_path TEXT,
			source_type TEXT NOT NULL,
			source_id TEXT,
			supersedes TEXT,
			authors_json TEXT NOT NULL,
			supplement_paths_json TEXT,
			pmid TEXT,
			pmcid TEXT,
			arxiv_id TEXT,
			s2_id TEXT
		);

		-- Index for DOI lookups
		CREATE INDEX IF NOT EXISTS idx_refs_doi ON refs(doi) WHERE doi IS NOT NULL AND doi != '';

		-- Full-text search virtual table (standalone, not external content)
		CREATE VIRTUAL TABLE IF NOT EXISTS refs_fts USING fts5(
			id,
			title,
			abstract,
			authors_text,
			pub_year
		);

		-- Embedding metadata for semantic index staleness detection (Phase II)
		CREATE TABLE IF NOT EXISTS embedding_metadata (
			paper_id TEXT PRIMARY KEY,
			model_name TEXT NOT NULL,
			indexed_at INTEGER NOT NULL,
			abstract_hash TEXT NOT NULL
		);
	`

	_, err := db.Exec(schema)
	return err
}

// RebuildFromJSONL clears the database and rebuilds it from a JSONL file.
func (d *DB) RebuildFromJSONL(jsonlPath string) (int, error) {
	// Read all references from JSONL
	refs, err := ReadAll(jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading JSONL: %w", err)
	}

	// Clear existing data
	if _, err := d.db.Exec("DELETE FROM refs"); err != nil {
		return 0, fmt.Errorf("clearing refs table: %w", err)
	}
	if _, err := d.db.Exec("DELETE FROM refs_fts"); err != nil {
		return 0, fmt.Errorf("clearing refs_fts table: %w", err)
	}

	// Prepare statements
	refsStmt, err := d.db.Prepare(`
		INSERT INTO refs (
			id, doi, title, abstract, venue,
			pub_year, pub_month, pub_day,
			pdf_path, source_type, source_id, supersedes,
			authors_json, supplement_paths_json,
			pmid, pmcid, arxiv_id, s2_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing refs insert: %w", err)
	}
	defer refsStmt.Close()

	ftsStmt, err := d.db.Prepare(`
		INSERT INTO refs_fts (id, title, abstract, authors_text, pub_year)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("preparing fts insert: %w", err)
	}
	defer ftsStmt.Close()

	for _, ref := range refs {
		authorsJSON, err := json.Marshal(ref.Authors)
		if err != nil {
			return 0, fmt.Errorf("marshaling authors for %s: %w", ref.ID, err)
		}
		var supplementJSON []byte
		if len(ref.SupplementPaths) > 0 {
			supplementJSON, err = json.Marshal(ref.SupplementPaths)
			if err != nil {
				return 0, fmt.Errorf("marshaling supplement paths for %s: %w", ref.ID, err)
			}
		}

		// Insert into refs table
		_, err = refsStmt.Exec(
			ref.ID, ref.DOI, ref.Title, ref.Abstract, ref.Venue,
			ref.Published.Year, ref.Published.Month, ref.Published.Day,
			ref.PDFPath, ref.Source.Type, ref.Source.ID, ref.Supersedes,
			string(authorsJSON), nullableString(supplementJSON),
			nullableStringValue(ref.PMID), nullableStringValue(ref.PMCID),
			nullableStringValue(ref.ArXivID), nullableStringValue(ref.S2ID),
		)
		if err != nil {
			return 0, fmt.Errorf("inserting ref %s: %w", ref.ID, err)
		}

		// Build authors text for FTS
		authorsText := formatAuthorsText(ref.Authors)

		// Insert into FTS table
		_, err = ftsStmt.Exec(ref.ID, ref.Title, ref.Abstract, authorsText, strconv.Itoa(ref.Published.Year))
		if err != nil {
			return 0, fmt.Errorf("inserting fts for %s: %w", ref.ID, err)
		}
	}

	return len(refs), nil
}

// formatAuthorsText creates a searchable text representation of authors.
func formatAuthorsText(authors []reference.Author) string {
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

// GetByID retrieves a reference by its ID.
func (d *DB) GetByID(id string) (*reference.Reference, error) {
	row := d.db.QueryRow(`SELECT `+selectRefFields+` FROM refs WHERE id = ?`, id)
	return scanReference(row)
}

// Search performs a full-text search and returns matching references.
func (d *DB) Search(query string, limit int) ([]reference.Reference, error) {
	// Escape special FTS5 characters and prepare query
	ftsQuery := prepareFTSQuery(query)

	rows, err := d.db.Query(`
		SELECT `+selectRefFields+`
		FROM refs
		WHERE id IN (SELECT id FROM refs_fts WHERE refs_fts MATCH ?)
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("searching: %w", err)
	}
	defer rows.Close()

	return scanReferences(rows)
}

// SearchField performs a search on a specific field.
func (d *DB) SearchField(field, value string, limit int) ([]reference.Reference, error) {
	var ftsQuery string

	switch field {
	case "author":
		ftsQuery = "authors_text:" + prepareFTSQuery(value)
	case "title":
		ftsQuery = "title:" + prepareFTSQuery(value)
	default:
		return nil, fmt.Errorf("unknown search field: %s", field)
	}

	rows, err := d.db.Query(`
		SELECT `+selectRefFields+`
		FROM refs
		WHERE id IN (SELECT id FROM refs_fts WHERE refs_fts MATCH ?)
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("searching %s: %w", field, err)
	}
	defer rows.Close()

	return scanReferences(rows)
}

// SearchFilters contains optional filters for SearchWithFilters.
//
// MAINTAINER NOTE: This filter-based approach works well for ~6-8 filters.
// Consider refactoring if:
//   - This struct exceeds 10 fields
//   - SearchWithFilters exceeds 80 lines
//   - You need complex logic (OR between different field types, negation)
//   - Filter interaction bugs become common
//
// The current approach mixes FTS5 (text search) and SQL WHERE (exact/range).
// Both support OR natively, so adding same-type ORs is straightforward.
type SearchFilters struct {
	Keyword  string   // General keyword search across all fields
	Authors  []string // Author names to search for (AND logic, fuzzy prefix matching)
	YearFrom int      // Minimum publication year (0 = no minimum)
	YearTo   int      // Maximum publication year (0 = no maximum)
	Title    string   // Search in title only (FTS)
	Venue    string   // Filter by venue (SQL LIKE, case-insensitive)
	DOI      string   // Exact DOI match (SQL)
}

// SearchWithFilters performs a search with multiple optional filters.
// Returns references matching ALL specified criteria (AND logic).
func (d *DB) SearchWithFilters(filters SearchFilters, limit int) ([]reference.Reference, error) {
	var ftsTerms []string
	var args []interface{}

	// Build FTS query parts (text-based searches)
	if filters.Keyword != "" {
		ftsTerms = append(ftsTerms, prepareFTSQuery(filters.Keyword))
	}
	if filters.Title != "" {
		ftsTerms = append(ftsTerms, "title:"+prepareFTSQuery(filters.Title))
	}
	for _, author := range filters.Authors {
		if author != "" {
			ftsTerms = append(ftsTerms, "authors_text:"+prepareAuthorQuery(author))
		}
	}

	// Build the query
	var query string
	if len(ftsTerms) > 0 {
		ftsQuery := strings.Join(ftsTerms, " AND ")
		query = `SELECT ` + selectRefFields + `
			FROM refs
			WHERE id IN (SELECT id FROM refs_fts WHERE refs_fts MATCH ?)`
		args = append(args, ftsQuery)
	} else {
		query = `SELECT ` + selectRefFields + ` FROM refs WHERE 1=1`
	}

	// SQL-based filters (exact/range matches)
	if filters.YearFrom > 0 {
		query += " AND pub_year >= ?"
		args = append(args, filters.YearFrom)
	}
	if filters.YearTo > 0 {
		query += " AND pub_year <= ?"
		args = append(args, filters.YearTo)
	}
	if filters.Venue != "" {
		query += " AND venue LIKE ?"
		args = append(args, "%"+filters.Venue+"%")
	}
	if filters.DOI != "" {
		query += " AND doi = ?"
		args = append(args, filters.DOI)
	}

	query += " LIMIT ?"
	args = append(args, limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("searching with filters: %w", err)
	}
	defer rows.Close()

	return scanReferences(rows)
}

// prepareAuthorQuery prepares an author name for FTS5 search with prefix matching.
// It adds a wildcard (*) to enable fuzzy matching (e.g., "Tim" matches "Timothy").
func prepareAuthorQuery(author string) string {
	author = strings.TrimSpace(author)
	if author == "" {
		return author
	}

	// Split into parts for multi-word names
	parts := strings.Fields(author)
	var terms []string
	for _, part := range parts {
		// Escape special characters and add prefix wildcard
		escaped := strings.ReplaceAll(part, "\"", "\"\"")
		// Add * for prefix matching
		terms = append(terms, "\""+escaped+"\"*")
	}

	// Use OR for multi-word author queries (match any part)
	return "(" + strings.Join(terms, " OR ") + ")"
}

// ListAll returns all references, optionally limited.
func (d *DB) ListAll(limit int) ([]reference.Reference, error) {
	query := `SELECT ` + selectRefFields + ` FROM refs ORDER BY id`
	var args []interface{}

	if limit > 0 {
		query += " LIMIT ?"
		args = []interface{}{limit}
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing refs: %w", err)
	}
	defer rows.Close()

	return scanReferences(rows)
}

// Count returns the total number of references.
func (d *DB) Count() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM refs").Scan(&count)
	return count, err
}

// scanner interface for sql.Row and sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanReference(s scanner) (*reference.Reference, error) {
	var ref reference.Reference
	var authorsJSON, supplementJSON sql.NullString
	var doi, abstract, venue, pdfPath, sourceID, supersedes sql.NullString
	var pmid, pmcid, arxivID, s2id sql.NullString
	var pubMonth, pubDay sql.NullInt64

	err := s.Scan(
		&ref.ID, &doi, &ref.Title, &abstract, &venue,
		&ref.Published.Year, &pubMonth, &pubDay,
		&pdfPath, &ref.Source.Type, &sourceID, &supersedes,
		&authorsJSON, &supplementJSON,
		&pmid, &pmcid, &arxivID, &s2id,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	ref.DOI = doi.String
	ref.Abstract = abstract.String
	ref.Venue = venue.String
	ref.PDFPath = pdfPath.String
	ref.Source.ID = sourceID.String
	ref.Supersedes = supersedes.String
	ref.PMID = pmid.String
	ref.PMCID = pmcid.String
	ref.ArXivID = arxivID.String
	ref.S2ID = s2id.String

	if pubMonth.Valid {
		ref.Published.Month = int(pubMonth.Int64)
	}
	if pubDay.Valid {
		ref.Published.Day = int(pubDay.Int64)
	}

	// Parse JSON fields
	if authorsJSON.Valid {
		if err := json.Unmarshal([]byte(authorsJSON.String), &ref.Authors); err != nil {
			return nil, fmt.Errorf("parsing authors JSON for %s: %w", ref.ID, err)
		}
	}
	if supplementJSON.Valid && supplementJSON.String != "" {
		if err := json.Unmarshal([]byte(supplementJSON.String), &ref.SupplementPaths); err != nil {
			return nil, fmt.Errorf("parsing supplement paths JSON for %s: %w", ref.ID, err)
		}
	}

	return &ref, nil
}

func scanReferences(rows *sql.Rows) ([]reference.Reference, error) {
	var refs []reference.Reference
	for rows.Next() {
		ref, err := scanReference(rows)
		if err != nil {
			return nil, err
		}
		if ref != nil {
			refs = append(refs, *ref)
		}
	}
	return refs, rows.Err()
}

func nullableString(b []byte) sql.NullString {
	if len(b) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// nullableStringValue converts a string to sql.NullString, treating empty as NULL.
func nullableStringValue(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// prepareFTSQuery escapes special characters for FTS5 queries.
func prepareFTSQuery(query string) string {
	// For simple queries, just quote the terms
	// FTS5 uses double quotes for phrase matching
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}

	// If query contains special chars, quote it
	if strings.ContainsAny(query, "\"*+-:(){}[]^~") {
		// Escape internal quotes and wrap in quotes
		query = strings.ReplaceAll(query, "\"", "\"\"")
		return "\"" + query + "\""
	}

	return query
}

// EmbeddingMetadata represents embedding metadata stored in the database.
type EmbeddingMetadata struct {
	PaperID      string
	ModelName    string
	IndexedAt    int64  // Unix timestamp
	AbstractHash string // SHA256 of abstract
}

// SaveEmbeddingMetadata saves or updates embedding metadata for a paper.
func (d *DB) SaveEmbeddingMetadata(meta EmbeddingMetadata) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO embedding_metadata (paper_id, model_name, indexed_at, abstract_hash)
		VALUES (?, ?, ?, ?)
	`, meta.PaperID, meta.ModelName, meta.IndexedAt, meta.AbstractHash)
	return err
}

// GetEmbeddingMetadata retrieves embedding metadata for a paper.
func (d *DB) GetEmbeddingMetadata(paperID string) (*EmbeddingMetadata, error) {
	var meta EmbeddingMetadata
	err := d.db.QueryRow(`
		SELECT paper_id, model_name, indexed_at, abstract_hash
		FROM embedding_metadata
		WHERE paper_id = ?
	`, paperID).Scan(&meta.PaperID, &meta.ModelName, &meta.IndexedAt, &meta.AbstractHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

// ClearEmbeddingMetadata removes all embedding metadata.
func (d *DB) ClearEmbeddingMetadata() error {
	_, err := d.db.Exec("DELETE FROM embedding_metadata")
	return err
}

// CountEmbeddingMetadata returns the number of papers with embedding metadata.
func (d *DB) CountEmbeddingMetadata() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM embedding_metadata").Scan(&count)
	return count, err
}

// CountPapersWithAbstract returns the number of papers that have abstracts.
func (d *DB) CountPapersWithAbstract(minLength int) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM refs WHERE abstract IS NOT NULL AND LENGTH(abstract) >= ?", minLength).Scan(&count)
	return count, err
}

// ListPaperIDsWithAbstract returns IDs of papers that have abstracts of sufficient length.
func (d *DB) ListPaperIDsWithAbstract(minLength int) ([]string, error) {
	rows, err := d.db.Query("SELECT id FROM refs WHERE abstract IS NOT NULL AND LENGTH(abstract) >= ?", minLength)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
