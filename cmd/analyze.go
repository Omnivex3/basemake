package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DynamicKarabo/dbai/internal/analyze"
	"github.com/DynamicKarabo/dbai/internal/db"
	"github.com/spf13/cobra"
)

var analyzeAll bool

var analyzeCmd = &cobra.Command{
	Use:   "analyze [query]",
	Short: "Analyze query performance with EXPLAIN ANALYZE",
	Long: `Run EXPLAIN ANALYZE on a query and surface performance insights.
Detects sequential scans, missing indexes, row estimate mismatches, and slow nodes.

  dbai analyze "SELECT * FROM orders WHERE created_at > now() - interval '30 days'"
  dbai analyze --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		defer conn.Close()

		if analyzeAll {
			return runAnalyzeAll(cmd.Context(), conn)
		}
		if len(args) == 0 {
			return fmt.Errorf("provide a query to analyze, or use --all to analyze cached queries")
		}
		return analyzeQuery(cmd.Context(), conn, args[0])
	},
}

func analyzeQuery(ctx context.Context, conn db.Database, query string) error {
	fmt.Fprintf(os.Stderr, "Running EXPLAIN ANALYZE...\n")

	start := time.Now()

	// Try JSON explain first (PostgreSQL), fall back to text
	jsonPlan, err := conn.ExplainJSON(ctx, query)
	if err == nil {
		report, parseErr := analyze.ParsePlan(jsonPlan)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "⚠ Could not parse plan: %v\n", parseErr)
			// Fall through to text explain below
		} else {
			duration := time.Since(start)
			fmt.Fprintf(os.Stderr, "Analysis completed in %v\n\n", duration.Round(time.Millisecond))

			// Print the full analysis
			fmt.Println(report.String())
			return nil
		}
	}

	// Fallback to text explain
	plan, err := conn.Explain(ctx, query)
	if err != nil {
		return fmt.Errorf("explain: %w", err)
	}

	duration := time.Since(start)
	fmt.Fprintf(os.Stderr, "EXPLAIN completed in %v\n\n", duration.Round(time.Millisecond))
	fmt.Println(plan)

	return nil
}

func runAnalyzeAll(ctx context.Context, conn db.Database) error {
	schema, err := db.LoadSchema()
	if err != nil {
		return fmt.Errorf("no cached schema — run 'dbai connect' first: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Analyzing all tables in %s...\n\n", schema.DBName)

	hasIssues := false

	for _, table := range schema.Tables {
		// Generate a sample query for each table
		sampleQuery := fmt.Sprintf("SELECT * FROM %s LIMIT 1000", table.Name)
		fmt.Fprintf(os.Stderr, "── %s ──\n", table.Name)

		plan, err := conn.ExplainJSON(ctx, sampleQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Could not analyze: %v\n\n", err)
			continue
		}

		report, err := analyze.ParsePlan(plan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Parse error: %v\n\n", err)
			continue
		}

		// Only show if there are issues
		if len(report.Issues) > 0 {
			hasIssues = true
			fmt.Printf("%s (%.0fms):\n", table.Name, report.ExecutionTime)
			for _, iss := range report.Issues {
				icon := "ℹ"
				switch iss.Severity {
				case "critical":
					icon = "🔴"
				case "warning":
					icon = "🟡"
				}
				fmt.Printf("  %s %s\n", icon, iss.Message)
			}
			fmt.Println()
		} else {
			fmt.Fprintf(os.Stderr, "  ✅ No issues found\n\n")
		}
	}

	if !hasIssues {
		fmt.Println("✅ No issues found across any table")
	}

	return nil
}

// init registers commands — called from root.go's init
// Note: we keep this separate so root.go can call analyzeCmd directly
// and we register flags here.
func init() {
	// No need to add to rootCmd — root.go already does that via AddCommand
	analyzeCmd.Flags().BoolVar(&analyzeAll, "all", false, "Analyze all cached tables")
}
