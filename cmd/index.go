package cmd

import (
	"fmt"
	"os"

	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index recommendations (Pro)",
	Long: `Analyze and suggest database indexes based on query patterns and statistics.

Index recommendations help you identify missing indexes, partial index opportunities,
and potential performance improvements for your queries.

  basemake index list           # Show pending recommendations

Note: Index recommendations appear automatically in 'basemake analyze' output.`,
}

var indexListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show pending index recommendations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := db.ActiveConnection()
		if err != nil {
			return fmt.Errorf("no active connection — run 'basemake connect' first")
		}
		defer conn.Close()

		fmt.Fprintf(os.Stderr, "Run 'basemake analyze <query>' to generate index recommendations.\n")
		fmt.Fprintf(os.Stderr, "Recommendations appear automatically in the analysis output.\n\n")
		fmt.Println("💡 Example:  basemake analyze \"SELECT * FROM orders WHERE status = 'pending'\"")
		fmt.Println("💡 Scan all:  basemake analyze --all")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.AddCommand(indexListCmd)
}
