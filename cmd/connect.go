package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var (
	connectDetect bool
	connectSave   string
	connectList   bool
)

var connectCmd = &cobra.Command{
	Use:   "connect [dsn]",
	Short: "Connect to a database and introspect schema",
	Long: `Connect to a database and cache its schema locally.
Supports PostgreSQL, MySQL, and SQLite.

  basemake connect postgres://user:***@localhost:5432/mydb
  basemake connect mysql://user:***@localhost:3306/mydb
  basemake connect --detect              # Auto-detect running databases
  basemake connect --save prod <dsn>     # Save as named connection
  basemake connect --list                # Show all saved connections
  basemake use prod                      # Switch to a saved connection`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// --list flag
		if connectList {
			return listConnections()
		}

		// --detect or no args
		if connectDetect || len(args) == 0 {
			return connectWithDetect(cmd)
		}

		dsn := args[0]

		// --save flag: save but don't connect
		if connectSave != "" {
			return saveAndConnect(cmd, connectSave, dsn)
		}

		return connectToDSN(cmd, dsn)
	},
}

func saveAndConnect(cmd *cobra.Command, name, dsn string) error {
	database, err := db.Connect(dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", db.Friendly(err))
	}
	defer database.Close()

	// Introspect and cache schema
	schema, err := database.Introspect(cmd.Context())
	if err != nil {
		return fmt.Errorf("introspect: %w", db.Friendly(err))
	}
	_ = schema.Save()

	// Save as named connection (also sets as active)
	if err := db.SaveConnection(name, dsn); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Could not save connection: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Saved connection %q → %s\n", name, database.Name())
	fmt.Fprintf(os.Stderr, "  Schema: %d tables, %d columns\n", len(schema.Tables), schema.TotalColumns())

	return nil
}

func listConnections() error {
	conns, err := db.ListConnections()
	if err != nil {
		return fmt.Errorf("list connections: %w", err)
	}

	if len(conns) == 0 {
		fmt.Println("No saved connections.")
		fmt.Println("  Save one: basemake connect --save prod postgres://user:***@host/db")
		fmt.Println("  Detect:   basemake connect --detect")
		return nil
	}

	active := db.ActiveConnectionName()
	fmt.Println("Saved connections:")
	fmt.Println()
	for name, dsn := range conns {
		mark := " "
		if name == active {
			mark = "●"
		}
		fmt.Printf("  %s %s  %s\n", mark, name, dsn)
	}
	fmt.Println()
	fmt.Println("  Switch: basemake use <name>")
	return nil
}

func connectWithDetect(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintf(os.Stderr, "  Scanning for databases...\n")
	detected := detectDatabases()

	if len(detected) == 0 {
		fmt.Fprintf(os.Stderr, "  No databases found on localhost.\n")
		fmt.Fprintf(os.Stderr, "  Provide a DSN: basemake connect postgres://user:***@localhost/db\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n")
	for i, d := range detected {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, d.label)
	}
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Which one? [1]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	idx := 0
	fmt.Sscanf(choice, "%d", &idx)
	if idx < 1 || idx > len(detected) {
		fmt.Fprintf(os.Stderr, "  Invalid choice.\n")
		return nil
	}

	return connectToDSN(cmd, detected[idx-1].dsn)
}

func connectToDSN(cmd *cobra.Command, dsn string) error {
	database, err := db.Connect(dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", db.Friendly(err))
	}
	defer database.Close()

	schema, err := database.Introspect(cmd.Context())
	if err != nil {
		return fmt.Errorf("introspect: %w", db.Friendly(err))
	}

	fmt.Fprintf(os.Stderr, "✓ Connected to %s\n", database.Name())
	fmt.Fprintf(os.Stderr, "  Schema loaded: %d tables, %d columns, %d indexes\n\n",
		len(schema.Tables), schema.TotalColumns(), schema.TotalIndexes())

	// Cache schema locally
	if err := schema.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Cache write failed: %v\n", err)
	}

	// Save as default connection
	if err := db.SaveDSN(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Config write failed: %v\n", err)
	}

	// Print table summary
	for _, t := range schema.Tables {
		fmt.Fprintf(os.Stderr, "  %s (%d columns, %d indexes)\n", t.Name, len(t.Columns), len(t.Indexes))
	}

	fmt.Fprintf(os.Stderr, "\n  Next: basemake for interactive mode, or basemake \"ask a question\"\n")

	return nil
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().BoolVar(&connectDetect, "detect", false, "Auto-detect databases on localhost and in Docker")
	connectCmd.Flags().StringVar(&connectSave, "save", "", "Save connection with a name (e.g. --save prod)")
	connectCmd.Flags().BoolVar(&connectList, "list", false, "Show all saved connections")
}
