package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

// setupTestDB creates a test database and JSONL file with test data
func setupTestDB(t *testing.T) (*DB, string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "refs.jsonl")

	// Create test references
	refs := []reference.Reference{
		{
			ID:       "Smith2026-ab",
			DOI:      "10.1234/smith",
			Title:    "Machine Learning in Biology",
			Abstract: "This paper discusses machine learning applications.",
			Venue:    "Nature",
			Authors: []reference.Author{
				{First: "John", Last: "Smith", ORCID: "0000-0001-2345-6789"},
				{First: "Jane", Last: "Doe"},
			},
			Published: reference.PublicationDate{Year: 2026, Month: 3, Day: 15},
			PDFPath:   "Papers/smith.pdf",
			Source:    reference.ImportSource{Type: "paperpile", ID: "abc123"},
		},
		{
			ID:       "Jones2025-cd",
			DOI:      "10.1234/jones",
			Title:    "Deep Learning for Protein Structure",
			Abstract: "A study of deep learning methods for proteins.",
			Venue:    "Science",
			Authors: []reference.Author{
				{First: "Alice", Last: "Jones"},
			},
			Published: reference.PublicationDate{Year: 2025, Month: 6},
			PDFPath:   "Papers/jones.pdf",
			Source:    reference.ImportSource{Type: "paperpile", ID: "def456"},
		},
		{
			ID:       "Brown2024-ef",
			DOI:      "10.1234/brown",
			Title:    "Statistical Methods in Genomics",
			Abstract: "Statistical approaches for genomic analysis.",
			Venue:    "PLOS Computational Biology",
			Authors: []reference.Author{
				{First: "Bob", Last: "Brown"},
				{First: "Carol", Last: "White"},
			},
			Published:       reference.PublicationDate{Year: 2024},
			PDFPath:         "Papers/brown.pdf",
			SupplementPaths: []string{"Papers/brown_supp.pdf"},
			Source:          reference.ImportSource{Type: "paperpile", ID: "ghi789"},
		},
	}

	// Write JSONL file
	if err := WriteAll(jsonlPath, refs); err != nil {
		t.Fatalf("Failed to write test JSONL: %v", err)
	}

	// Open database
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Rebuild from JSONL
	if _, err := db.RebuildFromJSONL(jsonlPath); err != nil {
		db.Close()
		t.Fatalf("Failed to rebuild DB: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, tmpDir, cleanup
}

func TestOpenDB_CreatesSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("OpenDB() did not create database file")
	}
}

func TestDB_RebuildFromJSONL(t *testing.T) {
	db, tmpDir, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := db.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}

	// Test rebuild overwrites
	jsonlPath := filepath.Join(tmpDir, "refs.jsonl")
	newRefs := []reference.Reference{
		{
			ID:        "New2026",
			Title:     "New Paper",
			Authors:   []reference.Author{{Last: "New"}},
			Published: reference.PublicationDate{Year: 2026},
			Source:    reference.ImportSource{Type: "manual"},
		},
	}
	if err := WriteAll(jsonlPath, newRefs); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	rebuilt, err := db.RebuildFromJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("RebuildFromJSONL() error = %v", err)
	}
	if rebuilt != 1 {
		t.Errorf("RebuildFromJSONL() = %d, want 1", rebuilt)
	}

	count, _ = db.Count()
	if count != 1 {
		t.Errorf("After rebuild, Count() = %d, want 1", count)
	}
}

func TestDB_GetByID(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		id        string
		wantFound bool
		wantTitle string
	}{
		{"Smith2026-ab", true, "Machine Learning in Biology"},
		{"Jones2025-cd", true, "Deep Learning for Protein Structure"},
		{"NotFound", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ref, err := db.GetByID(tt.id)
			if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if tt.wantFound {
				if ref == nil {
					t.Error("GetByID() returned nil, want ref")
					return
				}
				if ref.Title != tt.wantTitle {
					t.Errorf("GetByID() title = %q, want %q", ref.Title, tt.wantTitle)
				}
			} else {
				if ref != nil {
					t.Errorf("GetByID() returned %+v, want nil", ref)
				}
			}
		})
	}
}

func TestDB_GetByID_FullReference(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	ref, err := db.GetByID("Smith2026-ab")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if ref == nil {
		t.Fatal("GetByID() returned nil")
	}

	// Verify all fields
	if ref.ID != "Smith2026-ab" {
		t.Errorf("ID = %q, want Smith2026-ab", ref.ID)
	}
	if ref.DOI != "10.1234/smith" {
		t.Errorf("DOI = %q, want 10.1234/smith", ref.DOI)
	}
	if ref.Title != "Machine Learning in Biology" {
		t.Errorf("Title = %q, want Machine Learning in Biology", ref.Title)
	}
	if len(ref.Authors) != 2 {
		t.Fatalf("Authors len = %d, want 2", len(ref.Authors))
	}
	if ref.Authors[0].First != "John" || ref.Authors[0].Last != "Smith" {
		t.Errorf("Authors[0] = %+v, want John Smith", ref.Authors[0])
	}
	if ref.Authors[0].ORCID != "0000-0001-2345-6789" {
		t.Errorf("Authors[0].ORCID = %q, want 0000-0001-2345-6789", ref.Authors[0].ORCID)
	}
	if ref.Published.Year != 2026 || ref.Published.Month != 3 || ref.Published.Day != 15 {
		t.Errorf("Published = %+v, want Year:2026 Month:3 Day:15", ref.Published)
	}
	if ref.Source.Type != "paperpile" || ref.Source.ID != "abc123" {
		t.Errorf("Source = %+v, want paperpile/abc123", ref.Source)
	}
}

func TestDB_Search(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		query   string
		limit   int
		wantIDs []string
		wantMin int // Minimum expected results (for flexibility)
	}{
		// Title search
		{"machine learning", 10, nil, 1},
		{"deep learning", 10, nil, 1},
		{"statistical", 10, nil, 1},

		// Abstract search
		{"protein", 10, nil, 1},
		{"genomic", 10, nil, 1},

		// Author search
		{"Smith", 10, nil, 1},
		{"Jones", 10, nil, 1},

		// Year search
		{"2026", 10, []string{"Smith2026-ab"}, 1},
		{"2025", 10, []string{"Jones2025-cd"}, 1},
		{"2024", 10, []string{"Brown2024-ef"}, 1},

		// Combined author and year search
		{"Smith 2026", 10, []string{"Smith2026-ab"}, 1},
		{"Jones 2025", 10, []string{"Jones2025-cd"}, 1},

		// No results
		{"nonexistent query xyz", 10, nil, 0},

		// Limit
		{"learning", 1, nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			refs, err := db.Search(tt.query, tt.limit)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if len(refs) < tt.wantMin {
				t.Errorf("Search(%q) returned %d results, want at least %d", tt.query, len(refs), tt.wantMin)
			}

			if tt.wantIDs != nil {
				for _, wantID := range tt.wantIDs {
					found := false
					for _, ref := range refs {
						if ref.ID == wantID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Search(%q) missing expected ID %q", tt.query, wantID)
					}
				}
			}
		})
	}
}

func TestDB_SearchField(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// Author search
	refs, err := db.SearchField("author", "Smith", 10)
	if err != nil {
		t.Fatalf("SearchField(author) error = %v", err)
	}
	if len(refs) < 1 {
		t.Error("SearchField(author, Smith) returned no results")
	}

	// Title search
	refs, err = db.SearchField("title", "Machine", 10)
	if err != nil {
		t.Fatalf("SearchField(title) error = %v", err)
	}
	if len(refs) < 1 {
		t.Error("SearchField(title, Machine) returned no results")
	}

	// Invalid field
	_, err = db.SearchField("invalid", "test", 10)
	if err == nil {
		t.Error("SearchField(invalid) should return error")
	}
}

func TestDB_SearchWithFilters(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		filters SearchFilters
		limit   int
		wantIDs []string
		wantMin int
	}{
		{
			name:    "keyword only",
			filters: SearchFilters{Keyword: "machine learning"},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "single author",
			filters: SearchFilters{Authors: []string{"Smith"}},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "single author prefix matching",
			filters: SearchFilters{Authors: []string{"Jo"}}, // Should match John and Jones
			limit:   10,
			wantMin: 2,
		},
		{
			name:    "multiple authors (AND logic)",
			filters: SearchFilters{Authors: []string{"Smith", "Doe"}},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "year exact",
			filters: SearchFilters{YearFrom: 2025, YearTo: 2025},
			limit:   10,
			wantIDs: []string{"Jones2025-cd"},
			wantMin: 1,
		},
		{
			name:    "year range",
			filters: SearchFilters{YearFrom: 2024, YearTo: 2025},
			limit:   10,
			wantMin: 2,
		},
		{
			name:    "year from only (open-ended)",
			filters: SearchFilters{YearFrom: 2025},
			limit:   10,
			wantMin: 2, // 2025 and 2026
		},
		{
			name:    "year to only (open-ended)",
			filters: SearchFilters{YearTo: 2025},
			limit:   10,
			wantMin: 2, // 2024 and 2025
		},
		{
			name:    "author and year combined",
			filters: SearchFilters{Authors: []string{"Smith"}, YearFrom: 2026, YearTo: 2026},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "keyword and author combined",
			filters: SearchFilters{Keyword: "deep learning", Authors: []string{"Jones"}},
			limit:   10,
			wantIDs: []string{"Jones2025-cd"},
			wantMin: 1,
		},
		{
			name:    "all filters combined",
			filters: SearchFilters{Keyword: "protein", Authors: []string{"Jones"}, YearFrom: 2025, YearTo: 2025},
			limit:   10,
			wantIDs: []string{"Jones2025-cd"},
			wantMin: 1,
		},
		{
			name:    "no matches",
			filters: SearchFilters{Authors: []string{"NonexistentAuthor"}},
			limit:   10,
			wantMin: 0,
		},
		{
			name:    "year only - no keywords or authors",
			filters: SearchFilters{YearFrom: 2026, YearTo: 2026},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "title search",
			filters: SearchFilters{Title: "Machine Learning"},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "venue filter",
			filters: SearchFilters{Venue: "Nature"},
			limit:   10,
			wantIDs: []string{"Smith2026-ab"},
			wantMin: 1,
		},
		{
			name:    "venue partial match",
			filters: SearchFilters{Venue: "PLOS"},
			limit:   10,
			wantIDs: []string{"Brown2024-ef"},
			wantMin: 1,
		},
		{
			name:    "DOI exact match",
			filters: SearchFilters{DOI: "10.1234/jones"},
			limit:   10,
			wantIDs: []string{"Jones2025-cd"},
			wantMin: 1,
		},
		{
			name:    "DOI no match",
			filters: SearchFilters{DOI: "10.1234/nonexistent"},
			limit:   10,
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := db.SearchWithFilters(tt.filters, tt.limit)
			if err != nil {
				t.Fatalf("SearchWithFilters() error = %v", err)
			}

			if len(refs) < tt.wantMin {
				t.Errorf("SearchWithFilters() returned %d results, want at least %d", len(refs), tt.wantMin)
			}

			if tt.wantIDs != nil {
				for _, wantID := range tt.wantIDs {
					found := false
					for _, ref := range refs {
						if ref.ID == wantID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("SearchWithFilters() missing expected ID %q", wantID)
					}
				}
			}
		})
	}
}

func TestDB_ListAll(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	// List all
	refs, err := db.ListAll(0)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(refs) != 3 {
		t.Errorf("ListAll(0) returned %d refs, want 3", len(refs))
	}

	// With limit
	refs, err = db.ListAll(2)
	if err != nil {
		t.Fatalf("ListAll(2) error = %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("ListAll(2) returned %d refs, want 2", len(refs))
	}

	// Limit greater than count
	refs, err = db.ListAll(100)
	if err != nil {
		t.Fatalf("ListAll(100) error = %v", err)
	}
	if len(refs) != 3 {
		t.Errorf("ListAll(100) returned %d refs, want 3", len(refs))
	}
}

func TestDB_Count(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	count, err := db.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestDB_SupplementPaths(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	ref, err := db.GetByID("Brown2024-ef")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if ref == nil {
		t.Fatal("GetByID() returned nil")
	}

	if len(ref.SupplementPaths) != 1 {
		t.Fatalf("SupplementPaths len = %d, want 1", len(ref.SupplementPaths))
	}
	if ref.SupplementPaths[0] != "Papers/brown_supp.pdf" {
		t.Errorf("SupplementPaths[0] = %q, want Papers/brown_supp.pdf", ref.SupplementPaths[0])
	}
}

func TestDB_EmptyJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	jsonlPath := filepath.Join(tmpDir, "refs.jsonl")

	// Create empty JSONL
	if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty JSONL: %v", err)
	}

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildFromJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("RebuildFromJSONL() error = %v", err)
	}
	if count != 0 {
		t.Errorf("RebuildFromJSONL() = %d, want 0", count)
	}

	dbCount, _ := db.Count()
	if dbCount != 0 {
		t.Errorf("Count() = %d, want 0", dbCount)
	}
}

func TestPrepareFTSQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"two words", "two words"},
		{"  spaces  ", "spaces"},               // Trimmed
		{"", ""},                               // Empty stays empty
		{`with "quotes"`, `"with ""quotes"""`}, // Quotes escaped
		{"special*chars", `"special*chars"`},   // Special chars quoted
		{"term:colon", `"term:colon"`},         // Colon quoted
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := prepareFTSQuery(tt.input)
			if got != tt.want {
				t.Errorf("prepareFTSQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDB_Close(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}

	// Close should not error
	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Operations after close should fail
	_, err = db.Count()
	if err == nil {
		t.Error("Operations after Close() should fail")
	}
}
