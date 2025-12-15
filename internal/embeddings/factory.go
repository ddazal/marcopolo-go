package embeddings

import (
	"fmt"

	"github.com/ddazal/marcopolo-go/internal/config"
)

// NewProvider creates an embedding provider based on configuration.
func NewProvider(cfg config.Config) (Provider, error) {
	switch cfg.Embedding.Provider {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required for OpenAI provider")
		}
		return NewOpenAIProvider(cfg.OpenAIAPIKey, cfg.Embedding.Model), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Embedding.Provider)
	}
}
