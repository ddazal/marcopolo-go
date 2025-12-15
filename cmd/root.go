package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ddazal/marcopolo-go/internal/config"
	"github.com/ddazal/marcopolo-go/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
)

var appConfig *config.Config

// rootCmd is the base command for the marcopolo-go CLI.
var rootCmd = &cobra.Command{
	Use:   "marcopolo-go",
	Short: "MCP server with search tool",
	Long:  "MCP server with search tool",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(fmt.Errorf("could not load config: %w", err))
	}
	appConfig = cfg
}

// openDB opens a DB connection for migration operations.
// appConfig is loaded once in root initConfig().
func openDB(ctx context.Context) (*sqlx.DB, error) {
	if appConfig == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}
	return db.Connect(ctx, *appConfig)
}
