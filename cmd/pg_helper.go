package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/analyze"
	"github.com/DynamicKarabo/basemake/internal/db"
)

// FetchTableStats queries pg_stats for the given tables and returns structured stats.
func FetchTableStats(ctx context.Context, conn db.Database, tables []string) (map[string]*analyze.TableStats, error) {
	if conn.Dialect() != "PostgreSQL" {
		return nil, nil // pg_stats is PostgreSQL-only
	}

	if len(tables) == 0 {
		return nil, nil
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, t := range tables {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}

	query := buildStatsQuery(unique)
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pg_stats query: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*analyze.TableStats)
	for rows.Next() {
		var tabName, colName string
		var nDistinct float64
		var nullFrac float64
		var avgWidth int
		var corr float64
		var mcvRaw, mcfRaw *string

		if err := rows.Scan(&tabName, &colName, &nDistinct, &nullFrac, &avgWidth, &corr, &mcvRaw, &mcfRaw); err != nil {
			return nil, fmt.Errorf("pg_stats scan: %w", err)
		}

		st, ok := result[tabName]
		if !ok {
			st = &analyze.TableStats{Name: tabName, Columns: make(map[string]analyze.ColumnStats)}
			result[tabName] = st
		}

		cs := analyze.ColumnStats{
			Table:     tabName,
			Column:    colName,
			NDistinct: nDistinct,
			NullFrac:  nullFrac,
			AvgWidth:  avgWidth,
			Corr:      corr,
		}

		if mcvRaw != nil && *mcvRaw != "" {
			cs.MCV = analyze.ParseMCV(*mcvRaw)
		}
		if mcfRaw != nil && *mcfRaw != "" {
			cs.MCF = analyze.ParseMCF(*mcfRaw)
		}

		st.Columns[colName] = cs
	}

	// Also get row counts for each table
	// We can estimate from pg_class or run a quick count
	for _, tbl := range unique {
		st, ok := result[tbl]
		if !ok {
			continue
		}
		// Try to get row count from pg_class
		row, err := conn.Query(ctx,
			fmt.Sprintf("SELECT reltuples::bigint FROM pg_class WHERE relname = '%s'", strings.ReplaceAll(tbl, "'", "''")))
		if err == nil {
			if row.Next() {
				var count int64
				if err := row.Scan(&count); err == nil {
					st.TotalRows = count
				}
			}
			row.Close()
		}
	}

	return result, nil
}

func buildStatsQuery(tables []string) string {
	quoted := make([]string, len(tables))
	for i, t := range tables {
		quoted[i] = "'" + strings.ReplaceAll(t, "'", "''") + "'"
	}
	return fmt.Sprintf(`SELECT tablename, attname, n_distinct, null_frac, avg_width, correlation,
						   most_common_vals::text, most_common_freqs::text
					FROM pg_stats
					WHERE schemaname = 'public'
					  AND tablename IN (%s)
					ORDER BY tablename, attname`, strings.Join(quoted, ","))
}

// FormatSuggestions renders index suggestions as a formatted string.
func FormatSuggestions(suggestions []analyze.IndexSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n💡 Index Suggestions\n\n")
	for i, sug := range suggestions {
		confidenceIcon := "🟢"
		switch sug.Confidence {
		case "speculative":
			confidenceIcon = "🟡"
		case "medium":
			confidenceIcon = "🟠"
		}
		b.WriteString(fmt.Sprintf("  %d. %s %s\n", i+1, confidenceIcon, sug.CreateSQL))
		b.WriteString(fmt.Sprintf("     📊 %s\n", sug.EstImprovement))
		for _, tr := range sug.Tradeoffs {
			b.WriteString(fmt.Sprintf("     ⚠ %s\n", tr))
		}
		if sug.PartialWhere != "" {
			b.WriteString(fmt.Sprintf("     📐 Partial index: %s\n", sug.PartialWhere))
		}
		if sug.Confidence == "speculative" {
			b.WriteString("     🟡 Stats are estimated (run ANALYZE for better accuracy)\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// collectTablesFromReport extracts table names from scan issues in a report.
func collectTablesFromReport(report *analyze.Report) []string {
	seen := make(map[string]bool)
	var tables []string
	for _, iss := range report.Issues {
		if iss.TableName != "" && !seen[iss.TableName] {
			seen[iss.TableName] = true
			tables = append(tables, iss.TableName)
		}
	}
	return tables
}
