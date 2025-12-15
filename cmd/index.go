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
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
