package cmd

import (
	"context"
	"fmt"

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

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	allTools := tools.GetAllTools()
	for _, tool := range allTools {
		toolDescription, err := tools.DescribeTool(tool)
		if err != nil {
			return fmt.Errorf("could not describe tool %q: %w", tool.Name, err)
		}
		resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: []string{toolDescription.Text},
			},
			Model:          openai.EmbeddingModelTextEmbedding3Small,
			EncodingFormat: "float",
		})
		if err != nil {
			return fmt.Errorf("failed to create embedding for tool %q: %w", tool.Name, err)
		}

		if len(resp.Data) != 1 {
			return fmt.Errorf("expected 1 embedding, got %d", len(resp.Data))
		}

		emb64 := resp.Data[0].Embedding // []float64
		emb32 := Float64To32(emb64)     // []float32
		vec := pgvector.NewVector(emb32)

		_, err = tx.ExecContext(
			ctx,
			"INSERT INTO tools (name, description, embedding) VALUES ($1, $2, $3)",
			tool.Name,
			toolDescription.Text,
			vec,
		)
		if err != nil {
			return fmt.Errorf("failed to insert embedding for tool %q: %w", tool.Name, err)
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
