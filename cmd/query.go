package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/dbai/internal/ai"
	"github.com/DynamicKarabo/dbai/internal/db"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [question|sql]",
	Short: "Ask a natural language question about your data",
	Long: `Translate a plain English question into SQL and run it.
Uses your cached schema to generate accurate queries.

  dbai query "show me users who ordered last month"
  dbai query "SELECT * FROM users LIMIT 5"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
	// Find active connection or reconnect
		conn, err := db.ActiveConnection()
		if err != nil {
			dsn, loadErr := db.LoadDSN()
			if loadErr != nil {
				return fmt.Errorf("no active connection — run 'dbai connect' first: %w", err)
			}
			conn, err = db.Connect(dsn)
			if err != nil {
				return fmt.Errorf("reconnect: %w", err)
			}
		}

		// Determine if input is SQL or natural language
		sql := input
		isNL := !looksLikeSQL(sql)

		if isNL {
			s, err := db.LoadSchema()
			if err != nil {
				return fmt.Errorf("load schema: %w", err)
			}

			fmt.Fprintf(os.Stderr, "🤖 Generating SQL from: %s\n\n", input)
			sql, err = ai.QuestionToSQL(cmd.Context(), s.SchemaForPrompt(), input)
			if err != nil {
				return fmt.Errorf("ai: %w", err)
			}
			fmt.Fprintf(os.Stderr, "%s\n\n", sql)
		}

		// Execute
		rows, err := conn.Query(cmd.Context(), sql)
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		defer rows.Close()

		cols := rows.Columns()
		fmt.Fprintf(os.Stderr, "→ %d columns\n", len(cols))

		// Print header
		for i, c := range cols {
			if i > 0 {
				fmt.Print("\t")
			}
			fmt.Print(c)
		}
		fmt.Println()

		// Print rows
		rowCount := 0
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for rows.Next() {
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			for i, v := range vals {
				if i > 0 {
					fmt.Print("\t")
				}
				switch val := v.(type) {
				case []byte:
					fmt.Print(string(val))
				case nil:
					fmt.Print("NULL")
				default:
					fmt.Print(val)
				}
			}
			fmt.Println()
			rowCount++
		}

		plural := "rows"
		if rowCount == 1 {
			plural = "row"
		}
		fmt.Fprintf(os.Stderr, "\n✓ %d %s\n", rowCount, plural)
		return nil
	},
}

func looksLikeSQL(s string) bool {
	trimmed := ""
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\n' {
			trimmed += string(c)
			if len(trimmed) >= 10 {
				break
			}
		}
	}
	upper := strings.ToUpper(trimmed)
	keywords := []string{"SELECT", "WITH", "EXPLAIN", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE"}
	for _, kw := range keywords {
		if len(upper) >= len(kw) && upper[:len(kw)] == kw {
			return true
		}
	}
	return false
}

func init() {
	queryCmd.Flags().Bool("dry-run", false, "Generate SQL but don't execute")
}
