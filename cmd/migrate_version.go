package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var migrateVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print current DB migration version",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		conn, err := openDB(ctx)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer conn.Close()

		p, err := newProvider(conn)
		if err != nil {
			return fmt.Errorf("init provider: %w", err)
		}

		v, err := p.GetDBVersion(ctx)
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), v)
		return nil
	},
}
