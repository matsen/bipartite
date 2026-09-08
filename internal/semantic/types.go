// Package semantic provides semantic search capabilities for paper abstracts.
package semantic

import "time"

// SemanticIndex holds embeddings for all indexed papers.
type SemanticIndex struct {
	// Version is the format version for compatibility checking.
	// Check against CurrentIndexVersion when loading.
	Version int `json:"version"`

	// Metadata about the index
	ModelName       string    `json:"model_name"`        // e.g., "all-minilm:l6-v2"
	Dimensions      int       `json:"dimensions"`        // 384 for all-minilm
	CreatedAt       time.Time `json:"created_at"`        // When index was built
	PaperCount      int       `json:"paper_count"`       // Number of papers indexed
	SkippedCount    int       `json:"skipped_count"`     // Papers skipped (no/short abstract)
	BuildDurationMs int64     `json:"build_duration_ms"` // Time to build in milliseconds

	// Embeddings map paper IDs to their vector embeddings
	Embeddings map[string][]float32 `json:"-"` // Not included in JSON output
}

// SearchResult represents a paper found by semantic search.
type SearchResult struct {
	PaperID    string  `json:"id"`
	Similarity float32 `json:"similarity"`
}

// BuildStats contains statistics from index building.
type BuildStats struct {
	PapersIndexed int `json:"papers_indexed"`
	PapersSkipped int `json:"papers_skipped"`

	// SkippedNoAbstract and SkippedShortAbstract break PapersSkipped down by
	// cause and sum to it. A paper with no abstract at all counts as
	// no-abstract; one whose abstract is shorter than MinAbstractLength counts
	// as short. The distinction is actionable: a missing abstract can often be
	// fetched, whereas a short one is usually all the source provides.
	SkippedNoAbstract    int `json:"skipped_no_abstract"`
	SkippedShortAbstract int `json:"skipped_short_abstract"`

	Duration       time.Duration `json:"duration"`
	IndexSizeBytes int64         `json:"index_size_bytes"`
}
