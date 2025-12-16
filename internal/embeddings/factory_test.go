package embeddings

import (
	"testing"

	"github.com/ddazal/marcopolo-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	tests := map[string]struct {
		config       config.Config
		expectError  bool
		errorMsg     string
		providerType interface{}
	}{
		"creates openai provider successfully": {
			config: config.Config{
				OpenAIAPIKey: "test-key",
				Embedding: config.EmbeddingConfig{
					Provider: "openai",
					Model:    "text-embedding-3-small",
				},
			},
			expectError:  false,
			providerType: &OpenAIProvider{},
		},
		"returns error for missing api key": {
			config: config.Config{
				OpenAIAPIKey: "",
				Embedding: config.EmbeddingConfig{
					Provider: "openai",
					Model:    "text-embedding-3-small",
				},
			},
			expectError: true,
			errorMsg:    "openai_api_key is required",
		},
		"returns error for unsupported provider": {
			config: config.Config{
				OpenAIAPIKey: "test-key",
				Embedding: config.EmbeddingConfig{
					Provider: "anthropic",
					Model:    "some-model",
				},
			},
			expectError: true,
			errorMsg:    "unsupported embedding provider: anthropic",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			provider, err := NewProvider(tc.config)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
				assert.IsType(t, tc.providerType, provider)
			}
		})
	}
}
