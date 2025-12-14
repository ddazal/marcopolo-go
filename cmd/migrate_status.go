package cmd

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  "Print a merged view of migrations from the filesystem and the database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
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

		statuses, err := p.Status(ctx)
		if err != nil {
			return fmt.Errorf("status: %w", err)
		}

		// Tabular output
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tTYPE\tSTATE\tAPPLIED_AT\tPATH")

		for _, st := range statuses {
			if st == nil || st.Source == nil {
				continue
			}

			appliedAt := "-"
			if !st.AppliedAt.IsZero() {
				appliedAt = st.AppliedAt.Format(time.RFC3339)
			}

			fmt.Fprintf(
				w,
				"%d\t%s\t%s\t%s\t%s\n",
				st.Source.Version,
				st.Source.Type,
				st.State,
				appliedAt,
				st.Source.Path,
			)
		}

		return w.Flush()
	},
}
