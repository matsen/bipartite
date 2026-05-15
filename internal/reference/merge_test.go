package reference

import (
	"encoding/json"
	"testing"
)

func TestMergeUpdate_PreservesExternalIdentifiers(t *testing.T) {
	// Real case: refs.jsonl has a PMCID from `bip ncbi backfill`. A
	// Paperpile re-import sends the same ref with no PMCID field. The merge
	// must keep the existing PMCID.
	existing := Reference{
		ID:    "Smith2024-aa",
		DOI:   "10.1038/foo",
		Title: "Foo",
		PMCID: "PMC123",
		PMID:  "456",
		S2ID:  "abc",
	}
	incoming := Reference{
		ID:     "Smith2024-aa",
		DOI:    "10.1038/foo",
		Title:  "Foo (updated title)",
		Source: ImportSource{Type: "paperpile", ID: "uuid-1"},
		// No PMCID/PMID/S2ID — Paperpile doesn't carry these.
	}

	got := MergeUpdate(existing, incoming)
	if got.PMCID != "PMC123" {
		t.Errorf("PMCID lost: got %q, want PMC123", got.PMCID)
	}
	if got.PMID != "456" {
		t.Errorf("PMID lost: got %q, want 456", got.PMID)
	}
	if got.S2ID != "abc" {
		t.Errorf("S2ID lost: got %q, want abc", got.S2ID)
	}
	if got.Title != "Foo (updated title)" {
		t.Errorf("incoming Title should win when non-empty: got %q", got.Title)
	}
	if got.Source.Type != "paperpile" {
		t.Errorf("Source should be taken from incoming: got %+v", got.Source)
	}
}

func TestMergeUpdate_IncomingOverridesWhenNonZero(t *testing.T) {
	// If both existing and incoming have a value for a field, the incoming
	// wins. This is the standard "update" behavior — the new import is more
	// recent and the user presumably edited it intentionally.
	existing := Reference{
		ID:       "x",
		Title:    "old title",
		Note:     "old note",
		Abstract: "old abstract",
		Tags:     []string{"old"},
	}
	incoming := Reference{
		ID:       "x",
		Title:    "new title",
		Note:     "new note",
		Abstract: "new abstract",
		Tags:     []string{"new1", "new2"},
		Source:   ImportSource{Type: "paperpile", ID: "uuid"},
	}

	got := MergeUpdate(existing, incoming)
	if got.Title != "new title" || got.Note != "new note" || got.Abstract != "new abstract" {
		t.Errorf("incoming should override existing for non-empty fields: %+v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "new1" {
		t.Errorf("incoming Tags should override: got %v", got.Tags)
	}
}

func TestMergeUpdate_IDAndSourceAlwaysFromIncoming(t *testing.T) {
	// ID and Source describe the import event, not the ref's content. They
	// must always come from the incoming side, even when the existing values
	// look "more complete."
	existing := Reference{
		ID:     "OldKey2020-ab",
		Source: ImportSource{Type: "manual", ID: "manual-1"},
	}
	incoming := Reference{
		ID:     "NewKey2024-cd",
		Source: ImportSource{Type: "paperpile", ID: "uuid-new"},
	}

	got := MergeUpdate(existing, incoming)
	if got.ID != "NewKey2024-cd" {
		t.Errorf("ID should be from incoming: got %q", got.ID)
	}
	if got.Source.Type != "paperpile" || got.Source.ID != "uuid-new" {
		t.Errorf("Source should be from incoming: got %+v", got.Source)
	}
}

func TestMergeUpdate_PreservesPDFAndSupplementPaths(t *testing.T) {
	// PDFPath is often added or updated post-import (e.g., by `bip s2 linkpub`)
	// or set by Paperpile itself. Either way, if incoming doesn't carry it,
	// preserve.
	existing := Reference{
		ID:              "x",
		PDFPath:         "Smith/2024/Smith2024-aa.pdf",
		SupplementPaths: []string{"Smith/2024/Smith2024-aa-supp.pdf"},
	}
	incoming := Reference{
		ID:     "x",
		Source: ImportSource{Type: "paperpile", ID: "uuid"},
	}

	got := MergeUpdate(existing, incoming)
	if got.PDFPath != "Smith/2024/Smith2024-aa.pdf" {
		t.Errorf("PDFPath lost: got %q", got.PDFPath)
	}
	if len(got.SupplementPaths) != 1 {
		t.Errorf("SupplementPaths lost: got %v", got.SupplementPaths)
	}
}

func TestMergeUpdate_PreservesYearOnUndatedImport(t *testing.T) {
	// A Paperpile entry without a year imports as `Published.Year: 0` (with
	// a `paperpile:incomplete` tag). The merge must not blow away an existing
	// real year — that would be a regression on data quality.
	existing := Reference{
		ID:        "x",
		Published: PublicationDate{Year: 2020, Month: 6, Day: 15},
	}
	incoming := Reference{
		ID:     "x",
		Source: ImportSource{Type: "paperpile", ID: "uuid"},
		// Year: 0
	}

	got := MergeUpdate(existing, incoming)
	if got.Published.Year != 2020 || got.Published.Month != 6 || got.Published.Day != 15 {
		t.Errorf("Published date lost: got %+v", got.Published)
	}
}

func TestMergeUpdate_NoChangeWhenIncomingMatchesExisting(t *testing.T) {
	// Stability check: if existing and incoming are equal, the merge result
	// must be byte-identical when marshaled. Guards against accidental
	// transformations (e.g., normalizing slice ordering).
	r := Reference{
		ID:        "x",
		DOI:       "10.1/A",
		Title:     "Foo",
		Tags:      []string{"a", "b"},
		Authors:   []Author{{First: "J", Last: "S"}},
		Published: PublicationDate{Year: 2024},
		Source:    ImportSource{Type: "paperpile", ID: "uuid"},
		PMCID:     "PMC1",
	}
	got := MergeUpdate(r, r)
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(got)
	if string(a) != string(b) {
		t.Errorf("identity merge changed bytes:\n  in:  %s\n  out: %s", a, b)
	}
}
