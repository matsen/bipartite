package semantic

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/embedding"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// fixtureDir holds reference-shaped JSON with abstracts, one paper per file.
const fixtureDir = "../../testdata/abstracts"

// stubProvider is a deterministic embedding.Provider for tests: the vector is
// derived from the text, so identical text embeds identically and different
// text embeds differently. It also records every text it was asked to embed.
type stubProvider struct {
	dims     int
	embedded []string
	err      error
}

func newStubProvider() *stubProvider {
	return &stubProvider{dims: 4}
}

func (p *stubProvider) Embed(ctx context.Context, text string) (embedding.Embedding, error) {
	p.embedded = append(p.embedded, text)
	if p.err != nil {
		return embedding.Embedding{}, p.err
	}
	sum := sha256.Sum256([]byte(text))
	vec := make([]float32, p.dims)
	for i := range vec {
		vec[i] = float32(sum[i]) / 255.0
	}
	return embedding.Embedding{Vector: vec}, nil
}

func (p *stubProvider) ModelName() string { return "stub-model" }

func (p *stubProvider) Dimensions() int { return p.dims }

// loadFixture reads one paper from testdata/abstracts.
func loadFixture(t *testing.T, name string) reference.Reference {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}

	var ref reference.Reference
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatalf("unmarshaling fixture %s: %v", name, err)
	}
	return ref
}

// loadFixtures reads the full fixture set: two papers with real abstracts, one
// with an empty abstract, and one whose abstract is below MinAbstractLength.
func loadFixtures(t *testing.T) []reference.Reference {
	t.Helper()

	names := []string{
		"ml_methods.json",
		"phylogenetics.json",
		"no_abstract.json",
		"short_abstract.json",
	}
	refs := make([]reference.Reference, 0, len(names))
	for _, name := range names {
		refs = append(refs, loadFixture(t, name))
	}
	return refs
}

func TestBuild(t *testing.T) {
	refs := loadFixtures(t)

	// Guard the assumptions the assertions below rest on.
	if got := refs[2].Abstract; got != "" {
		t.Fatalf("no_abstract.json should have an empty abstract, got %q", got)
	}
	if n := len(refs[3].Abstract); n == 0 || n >= MinAbstractLength {
		t.Fatalf("short_abstract.json should be nonempty and under %d chars, got %d",
			MinAbstractLength, n)
	}

	provider := newStubProvider()
	idx, stats, err := NewBuilder(provider, nil).Build(context.Background(), refs)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if stats.PapersIndexed != 2 {
		t.Errorf("expected 2 papers indexed, got %d", stats.PapersIndexed)
	}
	// Both the empty and the short abstract are skipped.
	if stats.PapersSkipped != 2 {
		t.Errorf("expected 2 papers skipped, got %d", stats.PapersSkipped)
	}
	if stats.Duration <= 0 {
		t.Error("expected a positive build duration")
	}

	if idx.ModelName != provider.ModelName() {
		t.Errorf("expected model name %q, got %q", provider.ModelName(), idx.ModelName)
	}
	if idx.Dimensions != provider.Dimensions() {
		t.Errorf("expected %d dimensions, got %d", provider.Dimensions(), idx.Dimensions)
	}
	if idx.PaperCount != 2 {
		t.Errorf("expected PaperCount 2, got %d", idx.PaperCount)
	}
	if idx.SkippedCount != 2 {
		t.Errorf("expected SkippedCount 2, got %d", idx.SkippedCount)
	}

	if len(idx.Embeddings) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(idx.Embeddings))
	}
	for _, id := range []string{refs[0].ID, refs[1].ID} {
		if _, ok := idx.Embeddings[id]; !ok {
			t.Errorf("expected %s to be indexed", id)
		}
	}
	for _, id := range []string{refs[2].ID, refs[3].ID} {
		if _, ok := idx.Embeddings[id]; ok {
			t.Errorf("expected %s to be skipped, but it was indexed", id)
		}
	}

	// The two indexed papers have distinct abstracts, so distinct vectors.
	first, second := idx.Embeddings[refs[0].ID], idx.Embeddings[refs[1].ID]
	if equalVectors(first, second) {
		t.Error("expected distinct vectors for the two indexed papers")
	}

	// Only the papers that got past the skip branch were sent to the provider.
	if len(provider.embedded) != 2 {
		t.Errorf("expected 2 Embed calls, got %d", len(provider.embedded))
	}
}

func TestBuildEmptyInput(t *testing.T) {
	idx, stats, err := NewBuilder(newStubProvider(), nil).Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if stats.PapersIndexed != 0 || stats.PapersSkipped != 0 {
		t.Errorf("expected zero counts, got indexed=%d skipped=%d",
			stats.PapersIndexed, stats.PapersSkipped)
	}
	if len(idx.Embeddings) != 0 {
		t.Errorf("expected an empty index, got %d embeddings", len(idx.Embeddings))
	}
}

func TestBuildTruncatesLongAbstract(t *testing.T) {
	long := strings.Repeat("a", MaxAbstractLength+500)
	refs := []reference.Reference{{ID: "long-001", Abstract: long}}

	provider := newStubProvider()
	if _, _, err := NewBuilder(provider, nil).Build(context.Background(), refs); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(provider.embedded) != 1 {
		t.Fatalf("expected 1 Embed call, got %d", len(provider.embedded))
	}
	if got := len(provider.embedded[0]); got != MaxAbstractLength {
		t.Errorf("expected abstract truncated to %d chars, got %d", MaxAbstractLength, got)
	}
}

func TestBuildReportsProgressPerPaperExamined(t *testing.T) {
	refs := loadFixtures(t)

	var currents, totals []int
	builder := NewBuilder(newStubProvider(), nil)
	builder.SetProgressReporter(ProgressFunc(func(current, total int) {
		currents = append(currents, current)
		totals = append(totals, total)
	}))

	if _, _, err := builder.Build(context.Background(), refs); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Progress counts papers examined, so skipped papers report too.
	if len(currents) != len(refs) {
		t.Fatalf("expected %d progress reports, got %d", len(refs), len(currents))
	}
	for i, current := range currents {
		if current != i+1 {
			t.Errorf("progress report %d: expected current %d, got %d", i, i+1, current)
		}
		if totals[i] != len(refs) {
			t.Errorf("progress report %d: expected total %d, got %d", i, len(refs), totals[i])
		}
	}
}

func TestBuildContextCancellation(t *testing.T) {
	refs := loadFixtures(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	provider := newStubProvider()
	_, _, err := NewBuilder(provider, nil).Build(ctx, refs)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(provider.embedded) != 0 {
		t.Errorf("expected no Embed calls after cancellation, got %d", len(provider.embedded))
	}
}

func TestBuildProviderError(t *testing.T) {
	sentinel := errors.New("provider unavailable")
	provider := newStubProvider()
	provider.err = sentinel

	refs := []reference.Reference{loadFixture(t, "ml_methods.json")}
	idx, stats, err := NewBuilder(provider, nil).Build(context.Background(), refs)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the provider error, got %v", err)
	}
	if idx != nil || stats != nil {
		t.Error("expected nil index and stats on error")
	}
}

func TestBuildSavesMetadataToDB(t *testing.T) {
	db := openTestDB(t)

	// A stale row from a previous build must not survive a full rebuild.
	stale := storage.EmbeddingMetadata{
		PaperID:      "stale-001",
		ModelName:    "old-model",
		IndexedAt:    1,
		AbstractHash: "deadbeef",
	}
	if err := db.SaveEmbeddingMetadata(stale); err != nil {
		t.Fatalf("seeding stale metadata: %v", err)
	}

	refs := loadFixtures(t)
	provider := newStubProvider()
	if _, _, err := NewBuilder(provider, db).Build(context.Background(), refs); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	count, err := db.CountEmbeddingMetadata()
	if err != nil {
		t.Fatalf("CountEmbeddingMetadata failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected metadata for 2 papers, got %d", count)
	}

	if got, err := db.GetEmbeddingMetadata(stale.PaperID); err != nil {
		t.Fatalf("GetEmbeddingMetadata failed: %v", err)
	} else if got != nil {
		t.Error("expected stale metadata to be cleared before indexing")
	}

	meta, err := db.GetEmbeddingMetadata(refs[0].ID)
	if err != nil {
		t.Fatalf("GetEmbeddingMetadata failed: %v", err)
	}
	if meta == nil {
		t.Fatalf("expected metadata for %s", refs[0].ID)
	}
	if meta.ModelName != provider.ModelName() {
		t.Errorf("expected model name %q, got %q", provider.ModelName(), meta.ModelName)
	}
	// The hash covers the untruncated abstract.
	if want := hashAbstract(refs[0].Abstract); meta.AbstractHash != want {
		t.Errorf("expected abstract hash %s, got %s", want, meta.AbstractHash)
	}
	if meta.IndexedAt <= 0 {
		t.Errorf("expected a positive IndexedAt, got %d", meta.IndexedAt)
	}
}

func TestHashAbstract(t *testing.T) {
	ml := loadFixture(t, "ml_methods.json")
	phylo := loadFixture(t, "phylogenetics.json")

	if hashAbstract(ml.Abstract) != hashAbstract(ml.Abstract) {
		t.Error("hashAbstract should be stable for identical input")
	}
	if hashAbstract(ml.Abstract) == hashAbstract(phylo.Abstract) {
		t.Error("hashAbstract should differ for different abstracts")
	}
	// SHA256 rendered as hex.
	if got := len(hashAbstract(ml.Abstract)); got != 64 {
		t.Errorf("expected a 64-character hex digest, got %d characters", got)
	}
}

// openTestDB opens a throwaway database in a temp directory.
func openTestDB(t *testing.T) *storage.DB {
	t.Helper()

	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func equalVectors(a, b []float32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
