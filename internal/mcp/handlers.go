package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pgvector/pgvector-go"
)

// ToolWithScore represents a tool with its relevance score (matches db.ToolWithScore)
type ToolWithScore struct {
	ID             int64
	Name           string
	Description    string
	InputSchema    *string
	RelevanceScore float64
}

// ToolRepository defines the interface for tool database operations
type ToolRepository interface {
	FindSimilarWithScore(ctx context.Context, embedding pgvector.Vector, minScore float64, limit int) ([]*ToolWithScore, error)
}

// EmbeddingProvider defines the interface for generating embeddings
type EmbeddingProvider interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// ServerDependencies holds the dependencies needed by MCP handlers
type ServerDependencies struct {
	ToolRepo          ToolRepository
	EmbeddingProvider EmbeddingProvider
}

// HandleSearchTools implements the search_tools MCP tool
func (deps *ServerDependencies) HandleSearchTools(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input SearchToolsInput
	inputBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal arguments: %v", err)), nil
	}

	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	// Set defaults
	if input.MaxResults == 0 {
		input.MaxResults = 5
	}
	if input.MinRelevanceScore == 0 {
		input.MinRelevanceScore = 0.7
	}

	// Generate embedding for query
	queryEmbedding, err := deps.EmbeddingProvider.GenerateEmbedding(ctx, input.Query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to generate embedding: %v", err)), nil
	}

	vec := pgvector.NewVector(queryEmbedding)

	// Search for similar tools
	dbTools, err := deps.ToolRepo.FindSimilarWithScore(ctx, vec, input.MinRelevanceScore, input.MaxResults)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Database search failed: %v", err)), nil
	}

	// Convert to search results
	results := make([]ToolSearchResult, 0, len(dbTools))
	for _, dbTool := range dbTools {
		// Use input schema as raw JSON
		var params json.RawMessage
		if dbTool.InputSchema != nil {
			params = json.RawMessage(*dbTool.InputSchema)
		}

		results = append(results, ToolSearchResult{
			Name:           dbTool.Name,
			Description:    dbTool.Description,
			Parameters:     params,
			RelevanceScore: dbTool.RelevanceScore,
		})
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No tools found matching: %q with minimum relevance score of %.2f", input.Query, input.MinRelevanceScore)), nil
	}

	output := SearchToolsOutput{
		Tools: results,
		Query: input.Query,
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal output: %v", err)), nil
	}

	return mcp.NewToolResultText(string(outputJSON)), nil
}

// HandleExecuteTool implements the execute_tool MCP tool
func (deps *ServerDependencies) HandleExecuteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input ExecuteToolInput
	inputBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal arguments: %v", err)), nil
	}

	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
	}

	// Lookup tool handler in executable registry. Exec if it exists.
	handler, exists := GetExecutableTool(input.ToolName)
	if !exists {
		return mcp.NewToolResultError(fmt.Sprintf("Tool not found: %s", input.ToolName)), nil
	}

	result, err := handler(ctx, input.Arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Tool execution failed: %v", err)), nil
	}
	
	output := ExecuteToolOutput{Result: result}
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(outputJSON)), nil
}
