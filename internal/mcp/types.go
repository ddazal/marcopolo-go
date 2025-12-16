package mcp

import (
	"encoding/json"
)

// SearchToolsInput defines the input for search_tools MCP tool
type SearchToolsInput struct {
	Query             string  `json:"query"`
	MaxResults        int     `json:"max_results,omitempty"`         // default: 5
	MinRelevanceScore float64 `json:"min_relevance_score,omitempty"` // default: 0.7
}

// SearchToolsOutput is the response from search_tools
type SearchToolsOutput struct {
	Tools []ToolSearchResult `json:"tools"`
	Query string             `json:"query"`
}

// ExecuteToolInput defines the input for execute_tool MCP tool
type ExecuteToolInput struct {
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ExecuteToolOutput wraps the result of tool execution
type ExecuteToolOutput struct {
	Result interface{} `json:"result"`
}

// ToolSearchResult represents a tool with its relevance score
type ToolSearchResult struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Parameters     json.RawMessage `json:"parameters,omitempty"`
	RelevanceScore float64         `json:"relevance_score"`
}
