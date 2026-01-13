// Package embedding provides vector embedding generation for text.
package embedding

// Embedding represents a vector embedding of text.
type Embedding struct {
	Vector []float32 // The embedding vector (e.g., 384 dimensions for all-minilm)
}

// Dimensions returns the dimensionality of the embedding.
func (e Embedding) Dimensions() int {
	return len(e.Vector)
}
