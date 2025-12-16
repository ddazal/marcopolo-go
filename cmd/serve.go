package cmd

import (
	"context"
	"fmt"

	"github.com/ddazal/marcopolo-go/internal/db"
	"github.com/ddazal/marcopolo-go/internal/embeddings"
	"github.com/ddazal/marcopolo-go/internal/mcp"
	"github.com/pgvector/pgvector-go"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server with tool search and execution",
	Long: `Starts an MCP (Model Context Protocol) server that exposes tools for:

- Searching available tools using semantic similarity
- Executing tools with provided parameters

The server communicates over stdio and can be used with MCP-compatible clients.

Configuration:
- Database connection for tool search
- Embedding provider for query vectorization
- Tool registry for execution

Example usage:
  marcopolo-go serve

The server will start and listen for MCP requests on stdin/stdout.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

// toolRepositoryAdapter adapts db.ToolRepository to mcp.ToolRepository
type toolRepositoryAdapter struct {
	repo db.ToolRepository
}

func (a *toolRepositoryAdapter) FindSimilarWithScore(ctx context.Context, embedding pgvector.Vector, minScore float64, limit int) ([]*mcp.ToolWithScore, error) {
	dbTools, err := a.repo.FindSimilarWithScore(ctx, embedding, minScore, limit)
	if err != nil {
		return nil, err
	}

	// Convert db.ToolWithScore to mcp.ToolWithScore
	mcpTools := make([]*mcp.ToolWithScore, len(dbTools))
	for i, dbTool := range dbTools {
		mcpTools[i] = &mcp.ToolWithScore{
			ID:             dbTool.ID,
			Name:           dbTool.Name,
			Description:    dbTool.Description,
			InputSchema:    dbTool.InputSchema,
			RelevanceScore: dbTool.RelevanceScore,
		}
	}

	return mcpTools, nil
}

func runServe(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	conn, err := openDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	embProvider, err := embeddings.NewProvider(*appConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedding provider: %w", err)
	}

	// Create repository and adapt it
	dbRepo := db.NewPostgresToolRepository(conn)
	mcpRepo := &toolRepositoryAdapter{repo: dbRepo}

	server := mcp.NewServer(&mcp.ServerDependencies{
		ToolRepo:          mcpRepo,
		EmbeddingProvider: embProvider,
	})

	return server.Serve(ctx)
}
