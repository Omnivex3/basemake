package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/dbai/internal/ai"
	"github.com/DynamicKarabo/dbai/internal/db"
	"github.com/DynamicKarabo/dbai/internal/display"
	"github.com/spf13/cobra"
)

var queryJSON bool
var queryCSV bool

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

		// Determine output format
		var format display.Format
		switch {
		case queryJSON:
			format = display.FormatJSON
		case queryCSV:
			format = display.FormatCSV
		default:
			format = display.FormatTable
		}

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

		// Collect all rows
		var resultRows [][]string
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for rows.Next() {
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			row := make([]string, len(cols))
			for i, v := range vals {
				switch val := v.(type) {
				case []byte:
					row[i] = string(val)
				case nil:
					row[i] = "NULL"
				default:
					row[i] = fmt.Sprint(val)
				}
			}
			resultRows = append(resultRows, row)
		}

		// Build row count message
		plural := "rows"
		if len(resultRows) == 1 {
			plural = "row"
		}
		msg := fmt.Sprintf("(%d %s)", len(resultRows), plural)

		// Print results
		res := display.Result{
			Columns: cols,
			Rows:    resultRows,
			Message: msg,
		}

		if err := display.Print(os.Stdout, res, format); err != nil {
			return fmt.Errorf("print: %w", err)
		}

		// Row count on stderr for non-table formats
		if format != display.FormatTable {
			fmt.Fprintf(os.Stderr, "\n%s\n", msg)
		}

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
	queryCmd.Flags().BoolVar(&queryJSON, "json", false, "Output results as JSON")
	queryCmd.Flags().BoolVar(&queryCSV, "csv", false, "Output results as CSV")
	queryCmd.Flags().Bool("dry-run", false, "Generate SQL but don't execute")
}
