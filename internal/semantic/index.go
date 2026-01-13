package semantic

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Errors returned by semantic index operations.
var (
	ErrIndexNotFound      = errors.New("semantic index not found")
	ErrPaperNotIndexed    = errors.New("paper not in semantic index")
	ErrUnsupportedVersion = errors.New("unsupported index version")
)

const (
	// IndexFileName is the name of the semantic index file.
	IndexFileName = "semantic.gob"

	// MinAbstractLength is the minimum abstract length (in characters) to index.
	// Rationale: ~50 characters = ~10-15 words = minimum for meaningful embeddings.
	// Shorter abstracts lack sufficient semantic content for reliable similarity.
	MinAbstractLength = 50

	// MaxAbstractLength is the maximum abstract length (in characters) to embed.
	// Rationale: nomic-embed-text has 8192 token context window.
	// ~8000 characters ≈ ~2000 tokens, leaving headroom for tokenization overhead.
	// Longer abstracts are truncated to this length before embedding.
	MaxAbstractLength = 8000

	// CurrentIndexVersion is the format version for compatibility checking.
	// Increment this when making breaking changes to the index format.
	CurrentIndexVersion = 1
)

// IndexPath returns the path to the semantic index file.
func IndexPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".bipartite", "cache", IndexFileName)
}

// NewSemanticIndex creates a new empty semantic index.
func NewSemanticIndex(modelName string, dimensions int) *SemanticIndex {
	return &SemanticIndex{
		Version:    CurrentIndexVersion,
		ModelName:  modelName,
		Dimensions: dimensions,
		CreatedAt:  time.Now(),
		Embeddings: make(map[string][]float32),
	}
}

// AddEmbedding adds a paper embedding to the index.
// The PaperCount field is automatically updated to reflect the current number of embeddings.
func (idx *SemanticIndex) AddEmbedding(paperID string, embedding []float32) error {
	if len(embedding) != idx.Dimensions {
		return fmt.Errorf("embedding dimension mismatch: got %d, want %d", len(embedding), idx.Dimensions)
	}
	idx.Embeddings[paperID] = embedding
	idx.PaperCount = len(idx.Embeddings)
	return nil
}

// Save persists the semantic index to disk using GOB encoding.
func (idx *SemanticIndex) Save(repoRoot string) error {
	indexPath := IndexPath(repoRoot)

	// Ensure cache directory exists
	cacheDir := filepath.Dir(indexPath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	// Write to a temp file first, then rename for atomicity
	tempPath := indexPath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	enc := gob.NewEncoder(f)
	if err := enc.Encode(idx); err != nil {
		f.Close()
		os.Remove(tempPath)
		return fmt.Errorf("encoding index: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("closing file: %w", err)
	}

	if err := os.Rename(tempPath, indexPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// Load reads the semantic index from disk.
// Returns ErrUnsupportedVersion if the index was created with an incompatible format.
func Load(repoRoot string) (*SemanticIndex, error) {
	indexPath := IndexPath(repoRoot)

	f, err := os.Open(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrIndexNotFound
		}
		return nil, fmt.Errorf("opening index file: %w", err)
	}
	defer f.Close()

	var idx SemanticIndex
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&idx); err != nil {
		return nil, fmt.Errorf("decoding index: %w", err)
	}

	// Check version compatibility (version 0 means old index without version field)
	if idx.Version != 0 && idx.Version != CurrentIndexVersion {
		return nil, fmt.Errorf("%w: got %d, want %d (rebuild with 'bp index build')",
			ErrUnsupportedVersion, idx.Version, CurrentIndexVersion)
	}

	return &idx, nil
}

// IndexSize returns the size of the index file in bytes.
func IndexSize(repoRoot string) (int64, error) {
	indexPath := IndexPath(repoRoot)
	info, err := os.Stat(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrIndexNotFound
		}
		return 0, err
	}
	return info.Size(), nil
}

// Exists checks if the semantic index file exists.
func Exists(repoRoot string) bool {
	indexPath := IndexPath(repoRoot)
	_, err := os.Stat(indexPath)
	return err == nil
}
