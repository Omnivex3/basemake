package cmd

import (
	"fmt"

	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from the current database",
	Long: `Close the active database connection and clear cached schema.
Saved connections are preserved — use 'basemake use <name>' to reconnect.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db.ClearActiveConnection()
		if err := db.ClearActive(); err != nil {
			return fmt.Errorf("disconnect: %w", err)
		}
		if err := db.ClearSchemaCache(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  ⚠ Could not clear schema cache: %v\n", err)
		}
		fmt.Println("✓ Disconnected")
		fmt.Println("  Run 'basemake connect' or 'basemake use <name>' to reconnect.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}
