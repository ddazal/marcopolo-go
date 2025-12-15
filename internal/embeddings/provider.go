package embeddings

import "context"

// Provider defines the interface for embedding generation services.
type Provider interface {
	// GenerateEmbedding creates an embedding vector for the given text.
	// Returns a float32 slice representing the embedding.
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GetDimensions returns the dimensionality of embeddings produced by this provider.
	GetDimensions() int
}
