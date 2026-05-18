package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DynamicKarabo/basemake/internal/analyze"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var checkThreshold string
var checkDryRun bool

var checkCmd = &cobra.Command{
	Use:   "check <sql|file.sql>",
	Short: "CI gate — check query performance, exit with code",
	Long: `Evaluate a query and exit with a CI-friendly code.

Runs EXPLAIN ANALYZE to check for structural issues (seq scans, missing indexes),
then executes the query and compares actual time against a threshold.

Exit codes:
  0  ✅ Pass — query is fast and safe
  1  ❌ Slow — execution exceeded time threshold
  2  🔴 Dangerous — structural issues found (seq scan on large table, missing index)
  3  ⚠ Error — connection failed or query invalid

Examples:
  basemake check "SELECT * FROM users JOIN orders ON ..." --threshold 500ms
  basemake check queries/heavy_report.sql --threshold 2s
  basemake check "SELECT * FROM users" --dry-run            # analyze only
  basemake check "UPDATE accounts SET balance = 0"          # default 1s threshold`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		// Resolve SQL — inline string or file path
		sql, err := readSQL(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Error reading input: %v\n", err)
			os.Exit(3)
			return nil
		}

		// Parse threshold
		threshold, err := time.ParseDuration(checkThreshold)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Invalid threshold: %v\n", err)
			os.Exit(3)
			return nil
		}

		// Get database connection
		conn, err := db.ActiveConnection()
		if err != nil {
			dsn, loadErr := db.LoadDSN()
			if loadErr != nil {
				fmt.Fprintf(os.Stderr, "⚠ No active connection — run 'basemake connect' first\n")
				os.Exit(3)
				return nil
			}
			conn, err = db.Connect(dsn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Reconnect failed: %v\n", err)
				os.Exit(3)
				return nil
			}
		}
		defer conn.Close()

		// Step 1: Structural check via EXPLAIN ANALYZE
		hasCritical := false
		hasWarning := false
		if planJSON, planErr := conn.ExplainJSON(cmd.Context(), sql); planErr == nil {
			if report, parseErr := analyze.ParsePlan(planJSON); parseErr == nil {
				for _, issue := range report.Issues {
					switch issue.Severity {
					case "critical":
						hasCritical = true
						fmt.Fprintf(os.Stderr, "🔴 %s\n", issue.Message)
						if issue.Suggestion != "" {
							fmt.Fprintf(os.Stderr, "   Suggestion: %s\n", issue.Suggestion)
						}
					case "warning":
						hasWarning = true
						fmt.Fprintf(os.Stderr, "🟡 %s\n", issue.Message)
						if issue.Suggestion != "" {
							fmt.Fprintf(os.Stderr, "   Suggestion: %s\n", issue.Suggestion)
						}
					}
				}
			}
		}

		// Step 2: Dry-run only — no execution timing
		if checkDryRun {
			if hasCritical {
				fmt.Fprintf(os.Stderr, "\n❌ DRY-RUN FAILED — structural issues found\n")
				os.Exit(2)
			}
			if hasWarning {
				fmt.Fprintf(os.Stderr, "\n✅ DRY-RUN PASSED WITH WARNINGS\n")
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "\n✅ DRY-RUN PASSED\n")
			os.Exit(0)
			return nil
		}

		// Step 3: Validate SQL before execution
		if _, err := conn.Explain(cmd.Context(), sql); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Invalid SQL: %v\n", err)
			os.Exit(3)
			return nil
		}

		// Step 4: Execute and time
		startTime := time.Now()
		rows, err := conn.Query(cmd.Context(), sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Query failed: %v\n", err)
			os.Exit(3)
			return nil
		}

		// Read all rows
		rowCount := 0
		for rows.Next() {
			rowCount++
		}
		rows.Close()

		elapsed := time.Since(startTime)

		// Step 5: Evaluate results
		if elapsed > threshold {
			ms := elapsed.Milliseconds()
			tms := threshold.Milliseconds()
			rowLabel := "rows"
			if rowCount == 1 {
				rowLabel = "row"
			}
			fmt.Fprintf(os.Stderr, "\n❌ SLOW: %dms (threshold: %dms) — %d %s\n", ms, tms, rowCount, rowLabel)
			os.Exit(1)
		}

		if hasCritical {
			fmt.Fprintf(os.Stderr, "\n❌ DANGEROUS: structural issues found (%dms)\n", elapsed.Milliseconds())
			os.Exit(2)
		}

		if hasWarning {
			rowLabel := "rows"
			if rowCount == 1 {
				rowLabel = "row"
			}
			fmt.Fprintf(os.Stderr, "\n✅ PASS WITH WARNINGS: %dms (threshold: %dms) — %d %s\n",
				elapsed.Milliseconds(), threshold.Milliseconds(), rowCount, rowLabel)
			os.Exit(0)
		}

		rowLabel := "rows"
		if rowCount == 1 {
			rowLabel = "row"
		}
		fmt.Fprintf(os.Stderr, "✅ PASS: %dms (threshold: %dms) — %d %s\n",
			elapsed.Milliseconds(), threshold.Milliseconds(), rowCount, rowLabel)
		return nil
	},
}

// readSQL resolves the input: reads a file if it ends with .sql,
// otherwise treats it as an inline SQL string.
func readSQL(input string) (string, error) {
	if strings.HasSuffix(input, ".sql") {
		data, err := os.ReadFile(input)
		if err != nil {
			// Try resolving relative to current dir or absolute
			abs, absErr := filepath.Abs(input)
			if absErr != nil {
				return "", fmt.Errorf("read file: %w", err)
			}
			data, err = os.ReadFile(abs)
			if err != nil {
				return "", fmt.Errorf("read file %s: %w", abs, err)
			}
		}
		return strings.TrimSpace(string(data)), nil
	}
	return input, nil
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().StringVar(&checkThreshold, "threshold", "1s", "Max query time (e.g. 500ms, 2s)")
	checkCmd.Flags().BoolVar(&checkDryRun, "dry-run", false, "Analyze only — don't execute query")
}
