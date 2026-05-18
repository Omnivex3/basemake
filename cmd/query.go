package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/display"
	"github.com/DynamicKarabo/basemake/internal/history"
	"github.com/spf13/cobra"
)

var queryJSON bool
var queryCSV bool
var queryDryRun bool
var queryExplain bool
var queryNoStream bool

var queryCmd = &cobra.Command{
	Use:   "query [question|sql]",
	Short: "Ask a natural language question about your data",
	Long: `Translate a plain English question into SQL and run it.
Uses your cached schema to generate accurate queries.

  basemake query "show me users who ordered last month"
  basemake query "SELECT * FROM users LIMIT 5"
  basemake query "top 10 products by revenue" --dry-run    # preview SQL
  basemake query "total sales last quarter" --explain       # show plan + results`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		// Load config for defaults
		cfg, cfgErr := config.Load()
		if cfgErr != nil {
			fmt.Fprintf(os.Stderr, "⚠ Could not load config: %v\n", cfgErr)
			cfg = config.DefaultConfig()
		}

		// Determine output format
		var format display.Format
		switch {
		case queryJSON:
			format = display.FormatJSON
		case queryCSV:
			format = display.FormatCSV
		case cfg.OutputFormat == "json":
			format = display.FormatJSON
		case cfg.OutputFormat == "csv":
			format = display.FormatCSV
		default:
			format = display.FormatTable
		}

		// Determine if input is SQL or natural language
		sql := input
		isNL := !looksLikeSQL(sql)

		if isNL {
			s, err := db.LoadSchema()
			if err != nil {
				return fmt.Errorf("load schema: %w", err)
			}

			// Build prompt with recent history for context compounding
			prompt := history.BuildPromptWithHistory(s.SchemaForPrompt(), 5)

			// Get dialect from active connection or config
			dialect := detectDialect()

			if queryNoStream {
				// Blocking mode — wait for full response
				fmt.Fprintf(os.Stderr, "🤖 Generating SQL from: %s\n\n", input)
				sql, err = ai.QuestionToSQL(cmd.Context(), dialect, prompt, input)
				if err != nil {
					return fmt.Errorf("ai: %w", err)
				}
				fmt.Fprintf(os.Stderr, "%s\n\n", sql)
			} else {
				// Streaming mode — print tokens as they arrive
				fmt.Fprintf(os.Stderr, "🤖 Generating SQL...\n\n")
				ch, err := ai.QuestionToSQLStream(cmd.Context(), dialect, prompt, input)
				if err != nil {
					return fmt.Errorf("ai: %w", err)
				}

				var sb strings.Builder
				for token := range ch {
					sb.WriteString(token)
					fmt.Fprint(os.Stderr, token)
				}
				sql = sb.String()
				fmt.Fprintf(os.Stderr, "\n\n")
			}
		}

		// Dry-run: show SQL and exit
		if queryDryRun {
			cmd.Println(sql)
			return nil
		}

		// Find active connection or reconnect
		conn, err := db.ActiveConnection()
		if err != nil {
			dsn, loadErr := db.LoadDSN()
			if loadErr != nil {
				return fmt.Errorf("no active connection — run 'basemake connect' first: %w", err)
			}
			conn, err = db.Connect(dsn)
			if err != nil {
				return fmt.Errorf("reconnect: %w", err)
			}
		}
		defer conn.Close()

		// Validate AI-generated SQL before execution
		if isNL && !queryExplain {
			if err := validateSQL(cmd.Context(), conn, sql); err != nil {
				return fmt.Errorf("generated SQL is invalid — try rephrasing your question:\n  %s\n  %v", sql, err)
			}
		}

		// EXPLAIN mode: show plan then execute
		if queryExplain {
			plan, err := conn.Explain(cmd.Context(), sql)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠ EXPLAIN failed: %v\n\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Execution Plan:\n%s\n\n", plan)
			}
		}

		// Execute with timing
		startTime := time.Now()
		rows, err := conn.Query(cmd.Context(), sql)
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		defer rows.Close()

		elapsed := time.Since(startTime).Seconds() * 1000

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

		// Record in history
		provider, _ := ai.SelectedProvider()
		providerName := ""
		if provider != nil {
			providerName = provider.Name()
		}
		_ = history.Record(history.Entry{
			Question:           input,
			SQLGenerated:       sql,
			DatabaseName:       conn.Name(),
			ExecutedAt:         startTime,
			ExecutionTimeMs:    elapsed,
			RowCount:           len(resultRows),
			WasNaturalLanguage: isNL,
			ProviderUsed:       providerName,
		})

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

// validateSQL checks if a SQL statement is syntactically valid
// by running EXPLAIN on it (which validates syntax without executing).
func validateSQL(ctx context.Context, conn db.Database, sql string) error {
	_, err := conn.Explain(ctx, sql)
	return err
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

// detectDialect tries to determine the database dialect from the active connection,
// falling back to config or env, then defaulting to PostgreSQL.
func detectDialect() string {
	conn, err := db.ActiveConnection()
	if err == nil {
		return conn.Dialect()
	}
	// Try to infer from DSN scheme
	dsn, _ := db.LoadDSN()
	if dsn != "" {
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			return "PostgreSQL"
		}
		if strings.HasPrefix(dsn, "mysql://") {
			return "MySQL"
		}
		if strings.HasPrefix(dsn, "sqlite:") {
			return "SQLite"
		}
	}
	return "PostgreSQL"
}

func init() {
	queryCmd.Flags().BoolVar(&queryJSON, "json", false, "Output results as JSON")
	queryCmd.Flags().BoolVar(&queryCSV, "csv", false, "Output results as CSV")
	queryCmd.Flags().BoolVar(&queryDryRun, "dry-run", false, "Generate SQL but don't execute")
	queryCmd.Flags().BoolVar(&queryExplain, "explain", false, "Show execution plan alongside results")
	queryCmd.Flags().BoolVar(&queryNoStream, "no-stream", false, "Disable streaming AI output (wait for full response)")
}
