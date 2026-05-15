package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matsen/bipartite/internal/ncbi"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// fakeConverter is a stand-in for *ncbi.Client used in backfill tests. It
// records every Convert() call and returns a fixed response shaped by the
// recordsByRequestedID map.
type fakeConverter struct {
	calls   int
	inputs  [][]ncbi.Input
	respond func(inputs []ncbi.Input) ([]ncbi.Record, error)
}

func (f *fakeConverter) Convert(ctx context.Context, inputs []ncbi.Input) ([]ncbi.Record, error) {
	f.calls++
	f.inputs = append(f.inputs, inputs)
	if f.respond != nil {
		return f.respond(inputs)
	}
	return nil, nil
}

// writeTempRefs serializes refs to a JSONL file under t.TempDir() and returns
// the path.
func writeTempRefs(t *testing.T, refs []reference.Reference) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "refs.jsonl")
	if err := storage.WriteAll(path, refs); err != nil {
		t.Fatalf("WriteAll: %v", err)
	}
	return path
}

// readRawLines returns the raw bytes of each non-empty line. Used to verify
// byte-identical preservation of un-modified refs.
func readRawLines(t *testing.T, path string) [][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		// Copy because scanner reuses its buffer.
		cp := make([]byte, len(b))
		copy(cp, b)
		lines = append(lines, cp)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return lines
}

func TestBackfill_NotInPMCIsNotAnError(t *testing.T) {
	// One ref has a DOI; NCBI returns a per-record error for it ("not in PMC").
	// Expect: PMCID stays empty, the ref is counted in Queried but not Found,
	// and the file is not written (because Found == 0).
	ref := reference.Reference{
		ID:    "Smith2024-aa",
		DOI:   "10.99999/fake",
		Title: "Foo",
	}
	path := writeTempRefs(t, []reference.Reference{ref})

	stat0, _ := os.Stat(path)
	// Give the FS time to register an mtime that's distinguishable from any
	// post-test write — macOS sometimes returns the same second-resolution
	// mtime on rapid writes.
	time.Sleep(10 * time.Millisecond)

	fc := &fakeConverter{
		respond: func(inputs []ncbi.Input) ([]ncbi.Record, error) {
			return []ncbi.Record{
				{
					RequestedID: "10.99999/fake",
					DOI:         "10.99999/fake",
					Status:      "error",
					ErrMsg:      "Identifier not found in PMC",
				},
			}, nil
		},
	}

	summary, err := backfillPMCIDs(context.Background(), path, fc, BackfillOptions{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if summary.Queried != 1 {
		t.Errorf("expected Queried=1, got %d", summary.Queried)
	}
	if summary.Found != 0 {
		t.Errorf("expected Found=0, got %d", summary.Found)
	}

	updated, err := storage.ReadAll(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if updated[0].PMCID != "" {
		t.Errorf("expected PMCID empty, got %q", updated[0].PMCID)
	}

	// File should not have been rewritten (Found == 0).
	stat1, _ := os.Stat(path)
	if !stat1.ModTime().Equal(stat0.ModTime()) {
		t.Errorf("file mtime changed despite Found=0: %v → %v", stat0.ModTime(), stat1.ModTime())
	}
}

func TestBackfill_PreservesOtherFieldsByteIdentical(t *testing.T) {
	// Start with a ref carrying many populated fields. After backfill, the
	// only on-disk change must be the PMCID being added. Other lines in the
	// file (if any) must be byte-identical.
	target := reference.Reference{
		ID:        "Smith2024-aa",
		DOI:       "10.1038/foo",
		Title:     "Foo",
		Note:      "bar",
		Tags:      []string{"x", "y"},
		Authors:   []reference.Author{{First: "Jane", Last: "Smith"}},
		Venue:     "Nature",
		Published: reference.PublicationDate{Year: 2024, Month: 6, Day: 1},
		Source:    reference.ImportSource{Type: "manual", ID: "abc"},
	}
	bystander := reference.Reference{
		ID:    "NoChange2023-zz",
		DOI:   "",
		Title: "Bystander",
	}
	path := writeTempRefs(t, []reference.Reference{target, bystander})

	preLines := readRawLines(t, path)

	fc := &fakeConverter{
		respond: func(inputs []ncbi.Input) ([]ncbi.Record, error) {
			return []ncbi.Record{
				{RequestedID: "10.1038/foo", DOI: "10.1038/foo", PMCID: "PMC123", PMID: 99},
			}, nil
		},
	}

	if _, err := backfillPMCIDs(context.Background(), path, fc, BackfillOptions{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	postLines := readRawLines(t, path)
	if len(postLines) != len(preLines) {
		t.Fatalf("line count changed: %d → %d", len(preLines), len(postLines))
	}

	// The bystander (second line) must be byte-identical.
	if string(postLines[1]) != string(preLines[1]) {
		t.Errorf("bystander line changed:\n  pre:  %s\n  post: %s", preLines[1], postLines[1])
	}

	// The target line must differ only by PMCID being set; verify by JSON
	// equality of everything else.
	var pre, post reference.Reference
	if err := json.Unmarshal(preLines[0], &pre); err != nil {
		t.Fatalf("unmarshal pre: %v", err)
	}
	if err := json.Unmarshal(postLines[0], &post); err != nil {
		t.Fatalf("unmarshal post: %v", err)
	}
	if post.PMCID != "PMC123" {
		t.Errorf("expected PMCID=PMC123, got %q", post.PMCID)
	}
	post.PMCID = ""
	if !equalReferences(pre, post) {
		t.Errorf("other fields changed.\n  pre:  %+v\n  post: %+v", pre, post)
	}
}

func equalReferences(a, b reference.Reference) bool {
	// json round-trip equality is enough here, and avoids dragging in reflect.DeepEqual semantics
	// for nil-vs-empty-slice differences that matter on disk but not in memory.
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func TestBackfill_Idempotent(t *testing.T) {
	// After backfill writes a PMCID, a second run on the same file must query zero refs.
	ref := reference.Reference{ID: "Smith2024-aa", DOI: "10.1038/foo"}
	path := writeTempRefs(t, []reference.Reference{ref})

	fc := &fakeConverter{
		respond: func(inputs []ncbi.Input) ([]ncbi.Record, error) {
			return []ncbi.Record{
				{RequestedID: "10.1038/foo", DOI: "10.1038/foo", PMCID: "PMC777"},
			}, nil
		},
	}

	if _, err := backfillPMCIDs(context.Background(), path, fc, BackfillOptions{}); err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	if fc.calls != 1 {
		t.Errorf("expected 1 NCBI call on first run, got %d", fc.calls)
	}

	summary, err := backfillPMCIDs(context.Background(), path, fc, BackfillOptions{})
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if summary.Queried != 0 {
		t.Errorf("expected Queried=0 on second run, got %d", summary.Queried)
	}
	if fc.calls != 1 {
		t.Errorf("expected NCBI call count unchanged at 1, got %d", fc.calls)
	}
}

func TestBackfill_DryRunDoesNotWrite(t *testing.T) {
	ref := reference.Reference{ID: "Smith2024-aa", DOI: "10.1038/foo"}
	path := writeTempRefs(t, []reference.Reference{ref})

	stat0, _ := os.Stat(path)
	time.Sleep(10 * time.Millisecond)

	fc := &fakeConverter{
		respond: func(inputs []ncbi.Input) ([]ncbi.Record, error) {
			return []ncbi.Record{
				{RequestedID: "10.1038/foo", DOI: "10.1038/foo", PMCID: "PMC777"},
			}, nil
		},
	}

	summary, err := backfillPMCIDs(context.Background(), path, fc, BackfillOptions{DryRun: true})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if summary.Found != 1 {
		t.Errorf("expected Found=1 in dry-run report, got %d", summary.Found)
	}

	updated, err := storage.ReadAll(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if updated[0].PMCID != "" {
		t.Errorf("dry-run wrote PMCID to file: %q", updated[0].PMCID)
	}
	stat1, _ := os.Stat(path)
	if !stat1.ModTime().Equal(stat0.ModTime()) {
		t.Errorf("dry-run changed file mtime: %v → %v", stat0.ModTime(), stat1.ModTime())
	}
}

func TestSelectBackfillCandidates(t *testing.T) {
	refs := []reference.Reference{
		{ID: "has-doi-no-pmcid", DOI: "10.1/A"},
		{ID: "has-pmid-no-pmcid", PMID: "12345"},
		{ID: "has-pmcid-already", DOI: "10.1/B", PMCID: "PMC42"},
		{ID: "neither", Title: "Skip me"},
		{ID: "tag-match", DOI: "10.1/C", Tags: []string{"immunology"}},
		{ID: "tag-no-match", DOI: "10.1/D", Tags: []string{"phylo"}},
	}

	t.Run("no tag", func(t *testing.T) {
		got, summary := selectBackfillCandidates(refs, "", 0)
		if summary.Scanned != 6 {
			t.Errorf("Scanned=%d, want 6", summary.Scanned)
		}
		if summary.NoConvertible != 1 {
			t.Errorf("NoConvertible=%d, want 1", summary.NoConvertible)
		}
		// Expect: has-doi, has-pmid, tag-match, tag-no-match  (skips has-pmcid, neither)
		if len(got) != 4 {
			t.Errorf("len(candidates)=%d, want 4", len(got))
		}
	})

	t.Run("with tag", func(t *testing.T) {
		got, _ := selectBackfillCandidates(refs, "immun", 0)
		if len(got) != 1 || refs[got[0]].ID != "tag-match" {
			t.Errorf("expected only tag-match, got %v", got)
		}
	})

	t.Run("limit", func(t *testing.T) {
		got, _ := selectBackfillCandidates(refs, "", 2)
		if len(got) != 2 {
			t.Errorf("len(candidates)=%d with limit=2, want 2", len(got))
		}
	})
}

func TestCandidateToInput_PrefersDOI(t *testing.T) {
	// When a ref has both DOI and PMID, DOI wins because NCBI's DOI lookup is
	// more reliable. The PMID is kept on the ref untouched.
	in := candidateToInput(reference.Reference{DOI: "10.1/A", PMID: "12345"})
	if in.Type != ncbi.IDTypeDOI {
		t.Errorf("Type=%v, want DOI", in.Type)
	}
	if in.ID != "10.1/A" {
		t.Errorf("ID=%q, want 10.1/A", in.ID)
	}
}

func TestCandidateToInput_PMIDFallback(t *testing.T) {
	in := candidateToInput(reference.Reference{PMID: "12345"})
	if in.Type != ncbi.IDTypePMID {
		t.Errorf("Type=%v, want PMID", in.Type)
	}
	if in.ID != "12345" {
		t.Errorf("ID=%q, want 12345", in.ID)
	}
}

func TestBackfill_PropagatesAPIError(t *testing.T) {
	// HTTP/API errors from the converter must bubble up to the caller with
	// the failing batch's IDs intact, not be silently dropped.
	ref := reference.Reference{ID: "x", DOI: "10.1/A"}
	path := writeTempRefs(t, []reference.Reference{ref})

	want := &ncbi.APIError{StatusCode: 400, Code: "invalid_dois", Message: "bad", BatchIDs: []string{"10.1/A"}}
	fc := &fakeConverter{
		respond: func(inputs []ncbi.Input) ([]ncbi.Record, error) {
			return nil, want
		},
	}

	_, err := backfillPMCIDs(context.Background(), path, fc, BackfillOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *ncbi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *ncbi.APIError in chain, got %v", err)
	}
	if !strings.Contains(err.Error(), "10.1/A") {
		t.Errorf("error should include failing batch ID: %v", err)
	}
}
