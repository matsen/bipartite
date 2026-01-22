package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestReadAll_EmptyFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	// Create empty file
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	f.Close()

	refs, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("ReadAll() returned %d refs, want 0", len(refs))
	}
}

func TestReadAll_NonExistentFile(t *testing.T) {
	refs, err := ReadAll("/nonexistent/path/refs.jsonl")
	if err != nil {
		t.Fatalf("ReadAll() error = %v (should return nil for nonexistent file)", err)
	}
	if refs != nil && len(refs) != 0 {
		t.Errorf("ReadAll() returned %v, want nil or empty slice", refs)
	}
}

func TestReadAll_SingleRef(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	content := `{"id":"Smith2026","doi":"10.1234/test","title":"Test Paper","authors":[{"first":"John","last":"Smith"}],"published":{"year":2026},"source":{"type":"manual","id":""}}`
	if err := os.WriteFile(path, []byte(content+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	refs, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("ReadAll() returned %d refs, want 1", len(refs))
	}

	ref := refs[0]
	if ref.ID != "Smith2026" {
		t.Errorf("ID = %q, want Smith2026", ref.ID)
	}
	if ref.DOI != "10.1234/test" {
		t.Errorf("DOI = %q, want 10.1234/test", ref.DOI)
	}
	if ref.Title != "Test Paper" {
		t.Errorf("Title = %q, want Test Paper", ref.Title)
	}
}

func TestReadAll_MultipleRefs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	lines := []string{
		`{"id":"A2026","title":"Paper A","authors":[{"last":"A"}],"published":{"year":2026},"source":{"type":"manual"}}`,
		`{"id":"B2025","title":"Paper B","authors":[{"last":"B"}],"published":{"year":2025},"source":{"type":"manual"}}`,
		`{"id":"C2024","title":"Paper C","authors":[{"last":"C"}],"published":{"year":2024},"source":{"type":"manual"}}`,
	}

	content := ""
	for _, line := range lines {
		content += line + "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	refs, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(refs) != 3 {
		t.Fatalf("ReadAll() returned %d refs, want 3", len(refs))
	}

	// Check order is preserved
	if refs[0].ID != "A2026" || refs[1].ID != "B2025" || refs[2].ID != "C2024" {
		t.Errorf("ReadAll() returned refs in wrong order: %v, %v, %v", refs[0].ID, refs[1].ID, refs[2].ID)
	}
}

func TestReadAll_SkipsEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	content := `{"id":"A","title":"A","authors":[{"last":"A"}],"published":{"year":2026},"source":{"type":"manual"}}

{"id":"B","title":"B","authors":[{"last":"B"}],"published":{"year":2025},"source":{"type":"manual"}}
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	refs, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("ReadAll() returned %d refs, want 2", len(refs))
	}
}

func TestReadAll_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	content := `{"id":"valid","title":"Valid","authors":[{"last":"V"}],"published":{"year":2026},"source":{"type":"manual"}}
not valid json
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := ReadAll(path)
	if err == nil {
		t.Error("ReadAll() expected error for invalid JSON")
	}
}

func TestAppend(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	ref := reference.Reference{
		ID:        "Test2026",
		Title:     "Test Paper",
		Authors:   []reference.Author{{First: "Test", Last: "Author"}},
		Published: reference.PublicationDate{Year: 2026},
		Source:    reference.ImportSource{Type: "manual"},
	}

	// Append to new file
	if err := Append(path, ref); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Verify by reading back
	refs, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("After Append(), got %d refs, want 1", len(refs))
	}
	if refs[0].ID != "Test2026" {
		t.Errorf("After Append(), ID = %q, want Test2026", refs[0].ID)
	}
}

func TestAppend_MultipleRefs(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	refs := []reference.Reference{
		{ID: "A", Title: "A", Authors: []reference.Author{{Last: "A"}}, Published: reference.PublicationDate{Year: 2026}, Source: reference.ImportSource{Type: "manual"}},
		{ID: "B", Title: "B", Authors: []reference.Author{{Last: "B"}}, Published: reference.PublicationDate{Year: 2025}, Source: reference.ImportSource{Type: "manual"}},
	}

	for _, ref := range refs {
		if err := Append(path, ref); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	// Verify
	read, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(read) != 2 {
		t.Errorf("After 2 Appends, got %d refs, want 2", len(read))
	}
}

func TestWriteAll(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	refs := []reference.Reference{
		{ID: "A", Title: "A", Authors: []reference.Author{{Last: "A"}}, Published: reference.PublicationDate{Year: 2026}, Source: reference.ImportSource{Type: "manual"}},
		{ID: "B", Title: "B", Authors: []reference.Author{{Last: "B"}}, Published: reference.PublicationDate{Year: 2025}, Source: reference.ImportSource{Type: "manual"}},
	}

	if err := WriteAll(path, refs); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	// Verify
	read, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(read) != 2 {
		t.Fatalf("After WriteAll(), got %d refs, want 2", len(read))
	}
	if read[0].ID != "A" || read[1].ID != "B" {
		t.Errorf("WriteAll() refs in wrong order or wrong IDs")
	}
}

func TestWriteAll_Overwrites(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	// Write initial refs
	initial := []reference.Reference{
		{ID: "Old1", Title: "Old1", Authors: []reference.Author{{Last: "O"}}, Published: reference.PublicationDate{Year: 2020}, Source: reference.ImportSource{Type: "manual"}},
		{ID: "Old2", Title: "Old2", Authors: []reference.Author{{Last: "O"}}, Published: reference.PublicationDate{Year: 2020}, Source: reference.ImportSource{Type: "manual"}},
	}
	if err := WriteAll(path, initial); err != nil {
		t.Fatalf("Initial WriteAll() error = %v", err)
	}

	// Overwrite with new refs
	updated := []reference.Reference{
		{ID: "New1", Title: "New1", Authors: []reference.Author{{Last: "N"}}, Published: reference.PublicationDate{Year: 2026}, Source: reference.ImportSource{Type: "manual"}},
	}
	if err := WriteAll(path, updated); err != nil {
		t.Fatalf("Second WriteAll() error = %v", err)
	}

	// Verify old refs are gone
	read, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(read) != 1 {
		t.Fatalf("After overwrite, got %d refs, want 1", len(read))
	}
	if read[0].ID != "New1" {
		t.Errorf("After overwrite, ID = %q, want New1", read[0].ID)
	}
}

func TestFindByDOI(t *testing.T) {
	refs := []reference.Reference{
		{ID: "A", DOI: "10.1234/a"},
		{ID: "B", DOI: "10.1234/b"},
		{ID: "C", DOI: ""},
	}

	tests := []struct {
		doi     string
		wantIdx int
		wantOK  bool
	}{
		{"10.1234/a", 0, true},
		{"10.1234/b", 1, true},
		{"10.1234/c", -1, false},
		{"", -1, false}, // Empty DOI always returns not found
	}

	for _, tt := range tests {
		t.Run(tt.doi, func(t *testing.T) {
			idx, ok := FindByDOI(refs, tt.doi)
			if idx != tt.wantIdx || ok != tt.wantOK {
				t.Errorf("FindByDOI(%q) = (%d, %v), want (%d, %v)", tt.doi, idx, ok, tt.wantIdx, tt.wantOK)
			}
		})
	}
}

func TestFindByID(t *testing.T) {
	refs := []reference.Reference{
		{ID: "Smith2026"},
		{ID: "Jones2025"},
		{ID: "Brown2024"},
	}

	tests := []struct {
		id      string
		wantIdx int
		wantOK  bool
	}{
		{"Smith2026", 0, true},
		{"Jones2025", 1, true},
		{"Brown2024", 2, true},
		{"NotFound", -1, false},
		{"", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			idx, ok := FindByID(refs, tt.id)
			if idx != tt.wantIdx || ok != tt.wantOK {
				t.Errorf("FindByID(%q) = (%d, %v), want (%d, %v)", tt.id, idx, ok, tt.wantIdx, tt.wantOK)
			}
		})
	}
}

func TestFindBySourceID(t *testing.T) {
	refs := []reference.Reference{
		{ID: "A", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-1"}},
		{ID: "B", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-2"}},
		{ID: "C", Source: reference.ImportSource{Type: "s2", ID: "s2-id-1"}},
		{ID: "D", Source: reference.ImportSource{Type: "manual", ID: ""}},
	}

	tests := []struct {
		name       string
		sourceType string
		sourceID   string
		wantIdx    int
		wantOK     bool
	}{
		{"matches paperpile source", "paperpile", "pp-uuid-1", 0, true},
		{"matches different paperpile source", "paperpile", "pp-uuid-2", 1, true},
		{"matches s2 source", "s2", "s2-id-1", 2, true},
		{"wrong type for source ID", "s2", "pp-uuid-1", -1, false},
		{"not found source ID", "paperpile", "not-found", -1, false},
		{"empty source ID returns not found", "paperpile", "", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, ok := FindBySourceID(refs, tt.sourceType, tt.sourceID)
			if idx != tt.wantIdx || ok != tt.wantOK {
				t.Errorf("FindBySourceID(%q, %q) = (%d, %v), want (%d, %v)",
					tt.sourceType, tt.sourceID, idx, ok, tt.wantIdx, tt.wantOK)
			}
		})
	}
}

func TestGenerateUniqueID(t *testing.T) {
	tests := []struct {
		name     string
		existing []reference.Reference
		baseID   string
		want     string
	}{
		{
			name:     "no conflict",
			existing: []reference.Reference{},
			baseID:   "Smith2026",
			want:     "Smith2026",
		},
		{
			name:     "single conflict",
			existing: []reference.Reference{{ID: "Smith2026"}},
			baseID:   "Smith2026",
			want:     "Smith2026-2",
		},
		{
			name:     "multiple conflicts",
			existing: []reference.Reference{{ID: "Smith2026"}, {ID: "Smith2026-2"}, {ID: "Smith2026-3"}},
			baseID:   "Smith2026",
			want:     "Smith2026-4",
		},
		{
			name:     "conflict with different ID",
			existing: []reference.Reference{{ID: "Jones2025"}},
			baseID:   "Smith2026",
			want:     "Smith2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUniqueID(tt.existing, tt.baseID)
			if got != tt.want {
				t.Errorf("GenerateUniqueID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRoundTrip_CompleteReference(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "refs.jsonl")

	// Create a reference with all fields populated
	original := reference.Reference{
		ID:       "Complete2026-ab",
		DOI:      "10.1234/complete",
		Title:    "A Complete Reference",
		Abstract: "This is a complete abstract with special chars: α β γ",
		Venue:    "Journal of Testing",
		Authors: []reference.Author{
			{First: "John", Last: "Smith", ORCID: "0000-0001-2345-6789"},
			{First: "Jane", Last: "Doe"},
		},
		Published: reference.PublicationDate{
			Year:  2026,
			Month: 6,
			Day:   15,
		},
		PDFPath:         "Papers/complete.pdf",
		SupplementPaths: []string{"Papers/supp1.pdf", "Papers/supp2.pdf"},
		Source: reference.ImportSource{
			Type: "paperpile",
			ID:   "abc-123",
		},
		Supersedes: "10.1234/old",
	}

	// Write and read back
	if err := WriteAll(path, []reference.Reference{original}); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	read, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(read) != 1 {
		t.Fatalf("ReadAll() returned %d refs, want 1", len(read))
	}

	got := read[0]

	// Verify all fields
	if got.ID != original.ID {
		t.Errorf("ID = %q, want %q", got.ID, original.ID)
	}
	if got.DOI != original.DOI {
		t.Errorf("DOI = %q, want %q", got.DOI, original.DOI)
	}
	if got.Title != original.Title {
		t.Errorf("Title = %q, want %q", got.Title, original.Title)
	}
	if got.Abstract != original.Abstract {
		t.Errorf("Abstract = %q, want %q", got.Abstract, original.Abstract)
	}
	if got.Venue != original.Venue {
		t.Errorf("Venue = %q, want %q", got.Venue, original.Venue)
	}
	if len(got.Authors) != len(original.Authors) {
		t.Fatalf("Authors len = %d, want %d", len(got.Authors), len(original.Authors))
	}
	for i, a := range original.Authors {
		if got.Authors[i] != a {
			t.Errorf("Authors[%d] = %+v, want %+v", i, got.Authors[i], a)
		}
	}
	if got.Published != original.Published {
		t.Errorf("Published = %+v, want %+v", got.Published, original.Published)
	}
	if got.PDFPath != original.PDFPath {
		t.Errorf("PDFPath = %q, want %q", got.PDFPath, original.PDFPath)
	}
	if len(got.SupplementPaths) != len(original.SupplementPaths) {
		t.Fatalf("SupplementPaths len = %d, want %d", len(got.SupplementPaths), len(original.SupplementPaths))
	}
	for i, p := range original.SupplementPaths {
		if got.SupplementPaths[i] != p {
			t.Errorf("SupplementPaths[%d] = %q, want %q", i, got.SupplementPaths[i], p)
		}
	}
	if got.Source != original.Source {
		t.Errorf("Source = %+v, want %+v", got.Source, original.Source)
	}
	if got.Supersedes != original.Supersedes {
		t.Errorf("Supersedes = %q, want %q", got.Supersedes, original.Supersedes)
	}
}
