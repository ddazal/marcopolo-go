package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/ddazal/marcopolo-go/internal/db"
	goose "github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var (
	migrationsDir     string
	migrationsVerbose bool
)

// migrateCmd is a command group for database migrations.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `
Manage PostgreSQL schema migrations using goose (embedded as a library).

This command group provides subcommands similar to the goose CLI (create, up,
down, status, version), without requiring a globally installed goose binary.`,
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Shared flags across migrate subcommands.
	migrateCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"dir",
		"migrations",
		"Directory containing migration files",
	)
	migrateCmd.PersistentFlags().BoolVar(
		&migrationsVerbose,
		"verbose",
		false,
		"Enable verbose migration logging",
	)

	migrateCmd.AddCommand(migrateCreateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateVersionCmd)
}

// openDB opens a DB connection for migration operations.
// appConfig is loaded once in root initConfig().
func openDB(ctx context.Context) (*sql.DB, error) {
	if appConfig == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return db.Connect(ctx, *appConfig)
}

func newProvider(dbConn *sql.DB) (*goose.Provider, error) {
	fsys := os.DirFS(migrationsDir)
	return goose.NewProvider(
		goose.DialectPostgres,
		dbConn,
		fsys,
		goose.WithVerbose(migrationsVerbose),
	)
}
