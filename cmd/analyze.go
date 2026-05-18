package cmd

import (
	"fmt"
	"os"

	"github.com/DynamicKarabo/dbai/internal/db"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [query]",
	Short: "Run EXPLAIN ANALYZE on a query",
	Long: `Execute EXPLAIN ANALYZE on a query and surface performance insights.
Highlights sequential scans, missing indexes, and expensive operations.

  dbai analyze "SELECT * FROM orders WHERE created_at > now() - interval '30 days'"
  dbai analyze --all`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		conn, err := db.ActiveConnection()
		if err != nil {
			return fmt.Errorf("no active connection — run 'dbai connect' first: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Running EXPLAIN ANALYZE...\n")

		plan, err := conn.Explain(cmd.Context(), query)
		if err != nil {
			return fmt.Errorf("explain: %w", err)
		}

		fmt.Fprintf(os.Stderr, "\nExecution Plan:\n")
		fmt.Println(plan)

		// TODO: Parse plan for insights
		// - Sequential scans
		// - Missing indexes
		// - Expensive joins
		// - Row estimate mismatches

		return nil
	},
}

func init() {
	analyzeCmd.Flags().Bool("all", false, "Analyze all cached queries")
}
