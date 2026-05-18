package cmd

import (
	"fmt"
	"os"

	"github.com/DynamicKarabo/dbai/internal/db"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect [dsn]",
	Short: "Connect to a database and introspect schema",
	Long: `Connect to a database and cache its schema locally.
Supports PostgreSQL and MySQL connection strings.

  dbai connect postgres://user:pass@localhost:5432/mydb
  dbai connect mysql://user:pass@localhost:3306/mydb`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dsn := args[0]

		database, err := db.Connect(dsn)
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer database.Close()

		schema, err := database.Introspect(cmd.Context())
		if err != nil {
			return fmt.Errorf("introspect: %w", err)
		}

		fmt.Fprintf(os.Stderr, "✓ Connected to %s\n", database.Name())
		fmt.Fprintf(os.Stderr, "  Schema loaded: %d tables, %d columns, %d indexes\n\n",
			len(schema.Tables), schema.TotalColumns(), schema.TotalIndexes())

		// Cache schema locally
		if err := schema.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Cache write failed: %v\n", err)
		}

		// Pretty print tables
		for _, t := range schema.Tables {
			fmt.Printf("%s (%d columns, %d indexes)\n", t.Name, len(t.Columns), len(t.Indexes))
			for _, c := range t.Columns {
				fmt.Printf("  ├─ %s %s", c.Name, c.Type)
				if c.IsPK {
					fmt.Print(" [PK]")
				}
				if c.IsNullable {
					fmt.Print(" nullable")
				}
				fmt.Println()
			}
		}

		return nil
	},
}
