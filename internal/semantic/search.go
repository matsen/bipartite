package semantic

import (
	"math"
	"sort"
)

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 means identical direction.
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

// Search finds papers similar to a query embedding.
// Results are sorted by similarity (highest first) and filtered by threshold.
func (idx *SemanticIndex) Search(query []float32, limit int, threshold float32) []SearchResult {
	if idx.Embeddings == nil || len(query) != idx.Dimensions {
		return nil
	}

	results := make([]SearchResult, 0, len(idx.Embeddings))
	for paperID, embedding := range idx.Embeddings {
		sim := CosineSimilarity(query, embedding)
		if sim >= threshold {
			results = append(results, SearchResult{
				PaperID:    paperID,
				Similarity: sim,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// FindSimilar finds papers similar to a given paper by ID.
// The source paper is excluded from results.
func (idx *SemanticIndex) FindSimilar(paperID string, limit int) ([]SearchResult, error) {
	embedding, exists := idx.Embeddings[paperID]
	if !exists {
		return nil, ErrPaperNotIndexed
	}

	results := make([]SearchResult, 0, len(idx.Embeddings)-1)
	for id, emb := range idx.Embeddings {
		if id == paperID {
			continue // Skip the source paper
		}
		sim := CosineSimilarity(embedding, emb)
		results = append(results, SearchResult{
			PaperID:    id,
			Similarity: sim,
		})
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// HasPaper checks if a paper is in the index.
func (idx *SemanticIndex) HasPaper(paperID string) bool {
	_, exists := idx.Embeddings[paperID]
	return exists
}
