package semantic

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0},
			b:        []float32{0, 1},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0},
			b:        []float32{-1, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{1, 1},
			b:        []float32{1, 0},
			expected: 0.7071067, // cos(45 degrees)
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			a:        []float32{1, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "zero vector a",
			a:        []float32{0, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "zero vector b",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 0, 0},
			expected: 0.0,
		},
		{
			name:     "normalized vectors",
			a:        []float32{0.6, 0.8},
			b:        []float32{0.6, 0.8},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(got-tt.expected)) > 0.0001 {
				t.Errorf("CosineSimilarity(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity_Commutative(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}

	ab := CosineSimilarity(a, b)
	ba := CosineSimilarity(b, a)

	if math.Abs(float64(ab-ba)) > 0.0001 {
		t.Errorf("CosineSimilarity is not commutative: (%v, %v) = %v, (%v, %v) = %v",
			a, b, ab, b, a, ba)
	}
}

func TestSearch(t *testing.T) {
	idx := NewSemanticIndex("test-model", 3)
	idx.AddEmbedding("paper1", []float32{1, 0, 0})
	idx.AddEmbedding("paper2", []float32{0.9, 0.1, 0})
	idx.AddEmbedding("paper3", []float32{0, 1, 0})
	idx.AddEmbedding("paper4", []float32{0, 0, 1})

	t.Run("finds similar papers", func(t *testing.T) {
		query := []float32{1, 0, 0}
		results, err := idx.Search(query, 10, 0.0)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 4 {
			t.Errorf("expected 4 results, got %d", len(results))
		}

		// First result should be paper1 (exact match)
		if results[0].PaperID != "paper1" {
			t.Errorf("expected paper1 as top result, got %s", results[0].PaperID)
		}
		if math.Abs(float64(results[0].Similarity-1.0)) > 0.0001 {
			t.Errorf("expected similarity 1.0 for paper1, got %v", results[0].Similarity)
		}

		// Second should be paper2 (very similar)
		if results[1].PaperID != "paper2" {
			t.Errorf("expected paper2 as second result, got %s", results[1].PaperID)
		}
	})

	t.Run("respects threshold", func(t *testing.T) {
		query := []float32{1, 0, 0}
		results, err := idx.Search(query, 10, 0.9)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Only paper1 (1.0) and paper2 (~0.99) should be above 0.9 threshold
		if len(results) != 2 {
			t.Errorf("expected 2 results above threshold 0.9, got %d", len(results))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		query := []float32{1, 0, 0}
		results, err := idx.Search(query, 2, 0.0)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results with limit=2, got %d", len(results))
		}
	})

	t.Run("returns sorted results", func(t *testing.T) {
		query := []float32{1, 0, 0}
		results, err := idx.Search(query, 10, 0.0)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		for i := 1; i < len(results); i++ {
			if results[i].Similarity > results[i-1].Similarity {
				t.Errorf("results not sorted: result[%d].Similarity (%v) > result[%d].Similarity (%v)",
					i, results[i].Similarity, i-1, results[i-1].Similarity)
			}
		}
	})
}

func TestSearch_Errors(t *testing.T) {
	t.Run("empty index returns error", func(t *testing.T) {
		idx := NewSemanticIndex("test-model", 3)
		query := []float32{1, 0, 0}

		_, err := idx.Search(query, 10, 0.0)
		if err != ErrEmptyIndex {
			t.Errorf("expected ErrEmptyIndex, got %v", err)
		}
	})

	t.Run("dimension mismatch returns error", func(t *testing.T) {
		idx := NewSemanticIndex("test-model", 3)
		idx.AddEmbedding("paper1", []float32{1, 0, 0})

		query := []float32{1, 0} // wrong dimensions
		_, err := idx.Search(query, 10, 0.0)
		if err == nil {
			t.Error("expected error for dimension mismatch")
		}
	})

	t.Run("negative limit returns error", func(t *testing.T) {
		idx := NewSemanticIndex("test-model", 3)
		idx.AddEmbedding("paper1", []float32{1, 0, 0})

		query := []float32{1, 0, 0}
		_, err := idx.Search(query, -1, 0.0)
		if err != ErrNegativeLimit {
			t.Errorf("expected ErrNegativeLimit, got %v", err)
		}
	})

	t.Run("zero limit returns all results", func(t *testing.T) {
		idx := NewSemanticIndex("test-model", 3)
		idx.AddEmbedding("paper1", []float32{1, 0, 0})
		idx.AddEmbedding("paper2", []float32{0, 1, 0})

		query := []float32{1, 0, 0}
		results, err := idx.Search(query, 0, 0.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results with limit=0, got %d", len(results))
		}
	})
}

func TestFindSimilar(t *testing.T) {
	idx := NewSemanticIndex("test-model", 3)
	idx.AddEmbedding("paper1", []float32{1, 0, 0})
	idx.AddEmbedding("paper2", []float32{0.9, 0.1, 0})
	idx.AddEmbedding("paper3", []float32{0, 1, 0})

	t.Run("excludes source paper", func(t *testing.T) {
		results, err := idx.FindSimilar("paper1", 10)
		if err != nil {
			t.Fatalf("FindSimilar failed: %v", err)
		}

		for _, r := range results {
			if r.PaperID == "paper1" {
				t.Error("source paper should be excluded from results")
			}
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results (excluding source), got %d", len(results))
		}
	})

	t.Run("returns sorted results", func(t *testing.T) {
		results, err := idx.FindSimilar("paper1", 10)
		if err != nil {
			t.Fatalf("FindSimilar failed: %v", err)
		}

		// paper2 should be more similar to paper1 than paper3
		if results[0].PaperID != "paper2" {
			t.Errorf("expected paper2 as most similar, got %s", results[0].PaperID)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		results, err := idx.FindSimilar("paper1", 1)
		if err != nil {
			t.Fatalf("FindSimilar failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result with limit=1, got %d", len(results))
		}
	})

	t.Run("paper not in index returns error", func(t *testing.T) {
		_, err := idx.FindSimilar("nonexistent", 10)
		if err != ErrPaperNotIndexed {
			t.Errorf("expected ErrPaperNotIndexed, got %v", err)
		}
	})

	t.Run("negative limit returns error", func(t *testing.T) {
		_, err := idx.FindSimilar("paper1", -1)
		if err != ErrNegativeLimit {
			t.Errorf("expected ErrNegativeLimit, got %v", err)
		}
	})
}

func TestHasPaper(t *testing.T) {
	idx := NewSemanticIndex("test-model", 3)
	idx.AddEmbedding("paper1", []float32{1, 0, 0})

	if !idx.HasPaper("paper1") {
		t.Error("HasPaper should return true for existing paper")
	}

	if idx.HasPaper("nonexistent") {
		t.Error("HasPaper should return false for non-existing paper")
	}
}
