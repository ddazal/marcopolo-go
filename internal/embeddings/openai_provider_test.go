package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOpenAIServer creates a test HTTP server that mocks OpenAI API responses
func mockOpenAIServer(t *testing.T, response interface{}, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/embeddings", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}))
}

// mockEmbeddingResponse creates a mock OpenAI embedding API response
func mockEmbeddingResponse(dimensions int) map[string]interface{} {
	embedding := make([]float64, dimensions)
	for i := range embedding {
		embedding[i] = float64(i) / float64(dimensions)
	}

	return map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"object":    "embedding",
				"index":     0,
				"embedding": embedding,
			},
		},
		"model": "text-embedding-3-small",
		"usage": map[string]interface{}{
			"prompt_tokens": 8,
			"total_tokens":  8,
		},
	}
}

func TestOpenAIProvider_GenerateEmbedding(t *testing.T) {
	tests := map[string]struct {
		mockResponse interface{}
		statusCode   int
		text         string
		expectError  bool
		errorMsg     string
		validateFunc func(t *testing.T, result []float32)
	}{
		"successfully generates embedding": {
			mockResponse: mockEmbeddingResponse(1536),
			statusCode:   http.StatusOK,
			text:         "test text",
			expectError:  false,
			validateFunc: func(t *testing.T, result []float32) {
				assert.Len(t, result, 1536)
				// Verify conversion from float64 to float32
				assert.Equal(t, float32(0.0), result[0])
				assert.True(t, result[1] > 0)
			},
		},
		"handles API error": {
			mockResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			},
			statusCode:  http.StatusUnauthorized,
			text:        "test text",
			expectError: true,
			errorMsg:    "openai embedding request failed",
		},
		"handles invalid response with multiple embeddings": {
			mockResponse: map[string]interface{}{
				"object": "list",
				"data": []map[string]interface{}{
					{"object": "embedding", "index": 0, "embedding": make([]float64, 1536)},
					{"object": "embedding", "index": 1, "embedding": make([]float64, 1536)},
				},
			},
			statusCode:  http.StatusOK,
			text:        "test text",
			expectError: true,
			errorMsg:    "expected 1 embedding, got 2",
		},
		"handles empty response": {
			mockResponse: map[string]interface{}{
				"object": "list",
				"data":   []map[string]interface{}{},
			},
			statusCode:  http.StatusOK,
			text:        "test text",
			expectError: true,
			errorMsg:    "expected 1 embedding, got 0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create mock server
			server := mockOpenAIServer(t, tc.mockResponse, tc.statusCode)
			defer server.Close()

			// Create provider with custom base URL
			provider := &OpenAIProvider{
				client: openai.NewClient(
					option.WithAPIKey("test-key"),
					option.WithBaseURL(server.URL),
				),
				model: "text-embedding-3-small",
			}

			// Execute
			ctx := context.Background()
			result, err := provider.GenerateEmbedding(ctx, tc.text)

			// Validate
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tc.validateFunc != nil {
					tc.validateFunc(t, result)
				}
			}
		})
	}
}

func TestOpenAIProvider_GetDimensions(t *testing.T) {
	provider := &OpenAIProvider{
		model: "text-embedding-3-small",
	}

	dimensions := provider.GetDimensions()
	assert.Equal(t, 1536, dimensions)
}

func TestOpenAIProvider_Float64ToFloat32Conversion(t *testing.T) {
	// Test the conversion accuracy in GenerateEmbedding
	mockResp := mockEmbeddingResponse(10)
	server := mockOpenAIServer(t, mockResp, http.StatusOK)
	defer server.Close()

	provider := &OpenAIProvider{
		client: openai.NewClient(
			option.WithAPIKey("test-key"),
			option.WithBaseURL(server.URL),
		),
		model: "text-embedding-3-small",
	}

	ctx := context.Background()
	result, err := provider.GenerateEmbedding(ctx, "test")
	require.NoError(t, err)

	// Verify length
	assert.Len(t, result, 10)

	// Verify conversion accuracy (float32 should be close to float64)
	for i, v := range result {
		expected := float32(float64(i) / 10.0)
		assert.InDelta(t, expected, v, 0.0001)
	}
}
