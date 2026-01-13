package semantic

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"github.com/matsen/bipartite/internal/embedding"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// ProgressReporter receives progress updates during index building.
type ProgressReporter interface {
	// OnProgress is called with the current progress.
	OnProgress(current, total int)
}

// ProgressFunc is a function adapter for ProgressReporter.
type ProgressFunc func(current, total int)

// OnProgress implements ProgressReporter.
func (f ProgressFunc) OnProgress(current, total int) {
	f(current, total)
}

// Builder constructs a semantic index from paper abstracts.
type Builder struct {
	provider embedding.Provider
	db       *storage.DB
	progress ProgressReporter
}

// NewBuilder creates a new index builder.
func NewBuilder(provider embedding.Provider, db *storage.DB) *Builder {
	return &Builder{
		provider: provider,
		db:       db,
	}
}

// SetProgressReporter sets the progress reporter for the builder.
func (b *Builder) SetProgressReporter(reporter ProgressReporter) {
	b.progress = reporter
}

// Build creates a semantic index from all papers with abstracts.
func (b *Builder) Build(ctx context.Context, refs []reference.Reference) (*SemanticIndex, *BuildStats, error) {
	startTime := time.Now()

	idx := NewSemanticIndex(b.provider.ModelName(), b.provider.Dimensions())
	stats := &BuildStats{
		SkippedReason: "no_abstract",
	}

	// Clear existing embedding metadata
	if b.db != nil {
		if err := b.db.ClearEmbeddingMetadata(); err != nil {
			return nil, nil, fmt.Errorf("clearing embedding metadata: %w", err)
		}
	}

	total := len(refs)
	processed := 0

	for _, ref := range refs {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		processed++

		// Report progress
		if b.progress != nil {
			b.progress.OnProgress(processed, total)
		}

		// Skip papers without abstracts or with short abstracts
		if ref.Abstract == "" || len(ref.Abstract) < MinAbstractLength {
			stats.PapersSkipped++
			continue
		}

		// Generate embedding
		emb, err := b.provider.Embed(ctx, ref.Abstract)
		if err != nil {
			return nil, nil, fmt.Errorf("embedding paper %s: %w", ref.ID, err)
		}

		// Add to index
		if err := idx.AddEmbedding(ref.ID, emb.Vector); err != nil {
			return nil, nil, fmt.Errorf("adding embedding for %s: %w", ref.ID, err)
		}

		stats.PapersIndexed++

		// Save metadata to database
		if b.db != nil {
			abstractHash := hashAbstract(ref.Abstract)
			meta := storage.EmbeddingMetadata{
				PaperID:      ref.ID,
				ModelName:    b.provider.ModelName(),
				IndexedAt:    time.Now().Unix(),
				AbstractHash: abstractHash,
			}
			if err := b.db.SaveEmbeddingMetadata(meta); err != nil {
				return nil, nil, fmt.Errorf("saving metadata for %s: %w", ref.ID, err)
			}
		}
	}

	idx.PaperCount = stats.PapersIndexed
	idx.SkippedCount = stats.PapersSkipped
	idx.BuildDurationMs = time.Since(startTime).Milliseconds()

	stats.Duration = time.Since(startTime)

	return idx, stats, nil
}

// hashAbstract computes a SHA256 hash of the abstract text.
func hashAbstract(abstract string) string {
	h := sha256.New()
	io.WriteString(h, abstract)
	return fmt.Sprintf("%x", h.Sum(nil))
}
