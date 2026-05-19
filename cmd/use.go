package cmd

import (
	"fmt"

	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a saved database connection",
	Long: `Switch to a previously saved database connection and reload schema.

  basemake use prod          # Switch to "prod" connection
  basemake use staging       # Switch to "staging" connection
  basemake use               # Show current connection

Connections are saved with: basemake connect --save <name> <dsn>`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Show current
			active := db.ActiveConnectionName()
			dsn, err := db.LoadDSN()
			if err != nil {
				fmt.Println("No active connection.")
				fmt.Println("  Connect: basemake connect --detect")
				fmt.Println("  Save:    basemake connect --save prod postgres://user:***/db")
				return nil
			}
			// Mask the DSN for display
			masked := dsn
			if len(dsn) > 60 {
				masked = dsn[:57] + "..."
			}
			fmt.Printf("Active connection: %s\n", active)
			fmt.Printf("  %s\n", masked)
			fmt.Println()
			fmt.Println("  Use:     basemake use prod")
			fmt.Println("  List:    basemake connect --list")
			return nil
		}

		name := args[0]
		if err := db.SetActiveConnection(name); err != nil {
			return fmt.Errorf("switch: %w", err)
		}

		// Load the DSN and connect to cache schema
		dsn, err := db.LoadDSN()
		if err == nil {
			conn, err := db.Connect(dsn)
			if err == nil {
				schema, err := conn.Introspect(cmd.Context())
				if err == nil {
					_ = schema.Save()
				}
				conn.Close()
				fmt.Printf("✓ Switched to %s (%d tables)\n", name, len(schema.Tables))
				return nil
			}
		}

		fmt.Printf("✓ Switched to %s\n", name)
		fmt.Println("  Run a query to connect and cache schema.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
