package cmd

import (
	"context"
	"fmt"
	"time"

	goose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var (
	createType string // "sql" or "go"
)

var migrateCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new migration file",
	Long: `
Create a new migration file in the migrations directory.

By default this creates a SQL migration. Use --type=go to create a Go migration.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		conn, err := openDB(ctx)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer conn.Close()

		if createType == "" {
			createType = "sql"
		}
		if createType != "sql" && createType != "go" {
			return fmt.Errorf("invalid --type=%q (expected sql or go)", createType)
		}

		if err := goose.Create(conn, migrationsDir, name, createType); err != nil {
			return fmt.Errorf("create migration: %w", err)
		}
		return nil
	},
}

func init() {
	migrateCreateCmd.Flags().StringVar(
		&createType,
		"type",
		"sql",
		"Migration type: sql or go",
	)
}
