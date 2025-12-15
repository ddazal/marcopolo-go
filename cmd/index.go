package cmd

import (
	"context"
	"fmt"

	"github.com/ddazal/marcopolo-go/internal/db"
	"github.com/ddazal/marcopolo-go/internal/models"
	"github.com/ddazal/marcopolo-go/internal/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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
	client := openai.NewClient(option.WithAPIKey(appConfig.OpenAIAPIKey))

	ctx := context.Background()
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
		// Generate tool description
		toolDescription, err := tools.DescribeTool(toolDef)
		if err != nil {
			return fmt.Errorf("could not describe tool %q: %w", toolDef.Name, err)
		}

		// Generate embedding using OpenAI
		resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: []string{toolDescription.Text},
			},
			Model:          openai.EmbeddingModelTextEmbedding3Small,
			EncodingFormat: "float",
		})
		if err != nil {
			return fmt.Errorf("failed to create embedding for tool %q: %w", toolDef.Name, err)
		}

		if len(resp.Data) != 1 {
			return fmt.Errorf("expected 1 embedding, got %d", len(resp.Data))
		}

		// Convert embedding
		emb64 := resp.Data[0].Embedding // []float64
		emb32 := Float64To32(emb64)     // []float32
		vec := pgvector.NewVector(emb32)

		// Create tool entity
		tool := models.NewTool(toolDef, toolDescription, vec)

		// Use repository to persist (upsert for idempotency)
		if err := repo.UpsertTx(ctx, tx, tool); err != nil {
			return fmt.Errorf("failed to upsert tool %q: %w", toolDef.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func Float64To32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(x)
	}
	return out
}
