package cmd

import (
	"context"
	"fmt"

	"github.com/ddazal/marcopolo-go/internal/db"
	"github.com/ddazal/marcopolo-go/internal/embeddings"
	"github.com/ddazal/marcopolo-go/internal/models"
	"github.com/ddazal/marcopolo-go/internal/tools"
	"github.com/pgvector/pgvector-go"
	"github.com/spf13/cobra"
)

// indexCmd represents the index command
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index tools by generating and storing embeddings",
	Long: `Generate embeddings for all registered tools and store them in the database.

This command:
- Retrieves all registered tools from the tool registry
- Generates embeddings using the configured provider
- Stores tool metadata and embeddings in the database for similarity search

The embedding provider and model can be configured in config.yaml:
  embedding:
    provider: "openai"
    model: "text-embedding-3-small"

Re-running this command is safe - it uses upsert to update existing tools.`,
	RunE: indexTools,
}

func init() {
	rootCmd.AddCommand(indexCmd)
}

func indexTools(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create embedding provider
	embeddingProvider, err := embeddings.NewProvider(*appConfig)
	if err != nil {
		return fmt.Errorf("failed to create embedding provider: %w", err)
	}

	conn, err := openDB(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	repo := db.NewPostgresToolRepository(conn)

	tx, err := conn.Beginx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	allTools := tools.GetAllTools()
	for _, toolDef := range allTools {
		toolDescription, err := tools.DescribeTool(toolDef)
		if err != nil {
			return fmt.Errorf("could not describe tool %q: %w", toolDef.Name, err)
		}

		embedding, err := embeddingProvider.GenerateEmbedding(ctx, toolDescription.Text)
		if err != nil {
			return fmt.Errorf("failed to create embedding for tool %q: %w", toolDef.Name, err)
		}

		// Convert to pgvector
		vec := pgvector.NewVector(embedding)

		tool := models.NewTool(toolDef, toolDescription, vec)
		if err := repo.UpsertTx(ctx, tx, tool); err != nil {
			return fmt.Errorf("failed to upsert tool %q: %w", toolDef.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
