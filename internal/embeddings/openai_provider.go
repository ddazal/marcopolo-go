package embeddings

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// OpenAIProvider implements the Provider interface using OpenAI's embedding API.
type OpenAIProvider struct {
	client openai.Client
	model  string
}

// NewOpenAIProvider creates a new OpenAI embedding provider.
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// GenerateEmbedding creates an embedding vector for the given text using OpenAI's API.
func (p *OpenAIProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	resp, err := p.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: []string{text},
		},
		Model:          openai.EmbeddingModel(p.model),
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, fmt.Errorf("openai embedding request failed: %w", err)
	}

	if len(resp.Data) != 1 {
		return nil, fmt.Errorf("expected 1 embedding, got %d", len(resp.Data))
	}

	// Convert []float64 to []float32
	emb64 := resp.Data[0].Embedding
	emb32 := make([]float32, len(emb64))
	for i, v := range emb64 {
		emb32[i] = float32(v)
	}

	return emb32, nil
}

// GetDimensions returns the dimensionality of embeddings produced by this provider.
func (p *OpenAIProvider) GetDimensions() int {
	// text-embedding-3-small: 1536 dimensions
	// text-embedding-3-large: 3072 dimensions
	// For simplicity, return 1536 (matches current migration)
	// TODO: Make this dynamic based on model
	return 1536
}
