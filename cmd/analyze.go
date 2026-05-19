package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DynamicKarabo/basemake/internal/analyze"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var analyzeAll bool

var analyzeCmd = &cobra.Command{
	Use:   "analyze [query]",
	Short: "Analyze query performance with EXPLAIN ANALYZE",
	Long: `Run EXPLAIN ANALYZE on a query and surface performance insights.
Detects sequential scans, missing indexes, row estimate mismatches, and slow nodes.

  basemake analyze "SELECT * FROM orders WHERE created_at > now() - interval '30 days'"
  basemake analyze --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		} else {
			duration := time.Since(start)
			fmt.Fprintf(os.Stderr, "Analysis completed in %v\n\n", duration.Round(time.Millisecond))

			fmt.Println(report.String())

			// Check for index suggestions
			var allSuggestions []analyze.IndexSuggestion
			tables := collectTablesFromReport(report)
			if len(tables) > 0 {
				stats, statsErr := FetchTableStats(ctx, conn, tables)
				if statsErr == nil && stats != nil {
					for _, n := range report.Nodes {
						if (n.NodeType == "Seq Scan" || n.NodeType == "Table Scan") && n.Filter != "" {
							sugs := analyze.SuggestIndexesForScan(n.RelationName, n.Filter, n.PlanRows, stats[n.RelationName])
							allSuggestions = append(allSuggestions, sugs...)
						}
					}
				}
			}

			if idxOutput := analyze.FormatSuggestions(allSuggestions); idxOutput != "" {
				fmt.Println(idxOutput)

				// Persist recommendations
				store, loadErr := analyze.LoadRecs()
				if loadErr == nil {
					store.Merge(allSuggestions)
					if saveErr := store.Save(); saveErr != nil {
						fmt.Fprintf(os.Stderr, "⚠ Could not save recommendations: %v\n", saveErr)
					}
				}
			}

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
		return fmt.Errorf("no cached schema — run 'basemake connect' first: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Analyzing all tables in %s...\n\n", schema.DBName)

	hasIssues := false

	for _, table := range schema.Tables {
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

func init() {
	analyzeCmd.Flags().BoolVar(&analyzeAll, "all", false, "Analyze all cached tables")
}
