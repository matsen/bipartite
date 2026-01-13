package semantic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSemanticIndex(t *testing.T) {
	idx := NewSemanticIndex("test-model", 384)

	if idx.Version != CurrentIndexVersion {
		t.Errorf("expected version %d, got %d", CurrentIndexVersion, idx.Version)
	}
	if idx.ModelName != "test-model" {
		t.Errorf("expected model name 'test-model', got '%s'", idx.ModelName)
	}
	if idx.Dimensions != 384 {
		t.Errorf("expected dimensions 384, got %d", idx.Dimensions)
	}
	if idx.Embeddings == nil {
		t.Error("Embeddings map should be initialized")
	}
	if idx.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestAddEmbedding(t *testing.T) {
	idx := NewSemanticIndex("test-model", 3)

	t.Run("adds embedding successfully", func(t *testing.T) {
		err := idx.AddEmbedding("paper1", []float32{1, 0, 0})
		if err != nil {
			t.Fatalf("AddEmbedding failed: %v", err)
		}

		if !idx.HasPaper("paper1") {
			t.Error("paper should be in index after adding")
		}
		if idx.PaperCount != 1 {
			t.Errorf("expected PaperCount 1, got %d", idx.PaperCount)
		}
	})

	t.Run("updates PaperCount correctly", func(t *testing.T) {
		idx2 := NewSemanticIndex("test-model", 3)
		idx2.AddEmbedding("paper1", []float32{1, 0, 0})
		idx2.AddEmbedding("paper2", []float32{0, 1, 0})
		idx2.AddEmbedding("paper3", []float32{0, 0, 1})

		if idx2.PaperCount != 3 {
			t.Errorf("expected PaperCount 3, got %d", idx2.PaperCount)
		}
	})

	t.Run("rejects dimension mismatch", func(t *testing.T) {
		idx2 := NewSemanticIndex("test-model", 3)
		err := idx2.AddEmbedding("paper1", []float32{1, 0}) // wrong dimensions

		if err == nil {
			t.Error("expected error for dimension mismatch")
		}
	})

	t.Run("overwrites existing embedding", func(t *testing.T) {
		idx2 := NewSemanticIndex("test-model", 3)
		idx2.AddEmbedding("paper1", []float32{1, 0, 0})
		idx2.AddEmbedding("paper1", []float32{0, 1, 0}) // overwrite

		if idx2.PaperCount != 1 {
			t.Errorf("expected PaperCount 1 after overwrite, got %d", idx2.PaperCount)
		}
		// Verify it was actually overwritten
		emb := idx2.Embeddings["paper1"]
		if emb[0] != 0 || emb[1] != 1 {
			t.Error("embedding should have been overwritten")
		}
	})
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temp directory for test
	tmpDir, err := os.MkdirTemp("", "semantic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and populate an index
	idx := NewSemanticIndex("test-model", 3)
	idx.AddEmbedding("paper1", []float32{1, 0, 0})
	idx.AddEmbedding("paper2", []float32{0, 1, 0})
	idx.AddEmbedding("paper3", []float32{0, 0, 1})
	idx.SkippedCount = 5

	// Save
	err = idx.Save(tmpDir)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	indexPath := IndexPath(tmpDir)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index file should exist after Save")
	}

	// Load
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded data
	if loaded.Version != idx.Version {
		t.Errorf("version mismatch: got %d, want %d", loaded.Version, idx.Version)
	}
	if loaded.ModelName != idx.ModelName {
		t.Errorf("model name mismatch: got %s, want %s", loaded.ModelName, idx.ModelName)
	}
	if loaded.Dimensions != idx.Dimensions {
		t.Errorf("dimensions mismatch: got %d, want %d", loaded.Dimensions, idx.Dimensions)
	}
	if loaded.PaperCount != idx.PaperCount {
		t.Errorf("paper count mismatch: got %d, want %d", loaded.PaperCount, idx.PaperCount)
	}
	if loaded.SkippedCount != idx.SkippedCount {
		t.Errorf("skipped count mismatch: got %d, want %d", loaded.SkippedCount, idx.SkippedCount)
	}
	if len(loaded.Embeddings) != len(idx.Embeddings) {
		t.Errorf("embeddings count mismatch: got %d, want %d", len(loaded.Embeddings), len(idx.Embeddings))
	}

	// Verify individual embeddings
	for id, emb := range idx.Embeddings {
		loadedEmb, exists := loaded.Embeddings[id]
		if !exists {
			t.Errorf("missing embedding for %s", id)
			continue
		}
		for i, v := range emb {
			if loadedEmb[i] != v {
				t.Errorf("embedding mismatch for %s at index %d: got %v, want %v", id, i, loadedEmb[i], v)
			}
		}
	}
}

func TestLoad_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "semantic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = Load(tmpDir)
	if err != ErrIndexNotFound {
		t.Errorf("expected ErrIndexNotFound, got %v", err)
	}
}

func TestIndexSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "semantic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and save an index
	idx := NewSemanticIndex("test-model", 3)
	idx.AddEmbedding("paper1", []float32{1, 0, 0})
	idx.Save(tmpDir)

	// Get size
	size, err := IndexSize(tmpDir)
	if err != nil {
		t.Fatalf("IndexSize failed: %v", err)
	}

	if size <= 0 {
		t.Error("index size should be positive")
	}
}

func TestIndexSize_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "semantic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = IndexSize(tmpDir)
	if err != ErrIndexNotFound {
		t.Errorf("expected ErrIndexNotFound, got %v", err)
	}
}

func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "semantic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Before saving
	if Exists(tmpDir) {
		t.Error("Exists should return false before saving")
	}

	// Create and save
	idx := NewSemanticIndex("test-model", 3)
	idx.Save(tmpDir)

	// After saving
	if !Exists(tmpDir) {
		t.Error("Exists should return true after saving")
	}
}

func TestIndexPath(t *testing.T) {
	path := IndexPath("/home/user/repo")
	expected := filepath.Join("/home/user/repo", ".bipartite", "cache", "semantic.gob")
	if path != expected {
		t.Errorf("IndexPath mismatch: got %s, want %s", path, expected)
	}
}
