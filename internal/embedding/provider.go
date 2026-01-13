package embedding

import "context"

// Provider generates embeddings from text.
type Provider interface {
	// Embed generates an embedding for the given text.
	Embed(ctx context.Context, text string) (Embedding, error)

	// ModelName returns the name of the embedding model.
	ModelName() string

	// Dimensions returns the expected vector dimensions.
	Dimensions() int
}
