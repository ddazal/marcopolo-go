package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer *server.MCPServer
	deps      *ServerDependencies
}

// NewServer creates and configures an MCP server with the provided dependencies
func NewServer(deps *ServerDependencies) *Server {
	mcpServer := server.NewMCPServer(
		"marcopolo-go",
		"1.0.0",
		server.WithLogging(),
	)

	// search_tools
	searchToolDef := mcp.NewTool(
		"search_tools",
		mcp.WithDescription("Search for tools using semantic similarity based on a natural language query. Returns tool definitions with relevance scores."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural language description of the tool you're looking for. Include specific details from the user's request (locations, names, IDs, etc.).")),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 5)")),
		mcp.WithNumber("min_relevance_score",
			mcp.Description("Minimum relevance score threshold 0-1 (default: 0.7)")),
	)
	mcpServer.AddTool(searchToolDef, deps.HandleSearchTools)

	// execute_tool
	executeToolDef := mcp.NewTool(
		"execute_tool",
		mcp.WithDescription("Execute a specific tool by name with provided parameters. Use this after finding a tool with search_tools."),
		mcp.WithString("tool_name",
			mcp.Required(),
			mcp.Description("Name of the tool to execute (obtained from search_tools)")),
		mcp.WithObject("arguments",
			mcp.Required(),
			mcp.Description("Arguments to pass to the tool as a JSON object matching the tool's parameter schema")),
	)
	mcpServer.AddTool(executeToolDef, deps.HandleExecuteTool)

	return &Server{
		mcpServer: mcpServer,
		deps:      deps,
	}
}

// Serve starts the MCP server using stdio transport
func (s *Server) Serve(_ context.Context) error {
	return server.ServeStdio(s.mcpServer)
}
