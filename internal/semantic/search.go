package semantic

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Errors returned by search operations.
var (
	ErrEmptyIndex        = errors.New("index has no embeddings")
	ErrDimensionMismatch = errors.New("query dimensions don't match index dimensions")
	ErrNegativeLimit     = errors.New("limit cannot be negative")
)

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1 and 1, where:
//   - 1.0 = vectors point in identical direction
//   - 0.0 = vectors are orthogonal (no similarity)
//   - -1.0 = vectors point in opposite directions
//
// IMPORTANT: Returns 0 for invalid inputs (mismatched lengths, empty vectors,
// or zero-magnitude vectors). Since 0 is also a valid similarity score for
// orthogonal vectors, callers MUST validate input dimensions before calling
// this function to distinguish between "orthogonal" and "invalid input".
// The Search() and FindSimilar() methods perform this validation automatically.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denominator := float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))
	if denominator == 0 {
		return 0
	}

	return dot / denominator
}

// paperFilter determines whether a paper should be included in results.
type paperFilter func(paperID string, similarity float32) bool

// findMatchingPapers calculates similarity scores, filters results, and sorts by similarity.
// This is the core ranking algorithm used by Search() and FindSimilar().
// Results are sorted by similarity (highest first).
func (idx *SemanticIndex) findMatchingPapers(query []float32, shouldInclude paperFilter) []SearchResult {
	results := make([]SearchResult, 0, len(idx.Embeddings))
	for paperID, embedding := range idx.Embeddings {
		sim := CosineSimilarity(query, embedding)
		if shouldInclude(paperID, sim) {
			results = append(results, SearchResult{
				PaperID:    paperID,
				Similarity: sim,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	return results
}

// applyLimit truncates results to the specified limit.
func applyLimit(results []SearchResult, limit int) []SearchResult {
	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

// Search finds papers similar to a query embedding.
// Results are sorted by similarity (highest first) and filtered by threshold.
// Returns an error if the index is empty, query dimensions don't match, or limit is negative.
func (idx *SemanticIndex) Search(query []float32, limit int, threshold float32) ([]SearchResult, error) {
	if idx.Embeddings == nil || len(idx.Embeddings) == 0 {
		return nil, ErrEmptyIndex
	}
	if len(query) != idx.Dimensions {
		return nil, fmt.Errorf("%w: got %d, want %d", ErrDimensionMismatch, len(query), idx.Dimensions)
	}
	if limit < 0 {
		return nil, ErrNegativeLimit
	}

	results := idx.findMatchingPapers(query, func(_ string, sim float32) bool {
		return sim >= threshold
	})

	return applyLimit(results, limit), nil
}

// FindSimilar finds papers similar to a given paper by ID.
// The source paper is excluded from results.
func (idx *SemanticIndex) FindSimilar(paperID string, limit int) ([]SearchResult, error) {
	embedding, exists := idx.Embeddings[paperID]
	if !exists {
		return nil, ErrPaperNotIndexed
	}
	if limit < 0 {
		return nil, ErrNegativeLimit
	}

	results := idx.findMatchingPapers(embedding, func(id string, _ float32) bool {
		return id != paperID
	})

	return applyLimit(results, limit), nil
}

// HasPaper checks if a paper is in the index.
func (idx *SemanticIndex) HasPaper(paperID string) bool {
	_, exists := idx.Embeddings[paperID]
	return exists
}
