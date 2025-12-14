package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the most recently applied migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
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

		if _, err := p.Down(ctx); err != nil {
			return fmt.Errorf("migrate down: %w", err)
		}
		return nil
	},
}
