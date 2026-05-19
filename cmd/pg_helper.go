package cmd

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/DynamicKarabo/basemake/internal/analyze"
	"github.com/DynamicKarabo/basemake/internal/db"
)

// queryPgStats runs a pg_stats query via the db connection and returns rows.
// Adapts db.Database to analyze.PgStatsCallback.
func queryPgStats(ctx context.Context, conn db.Database) analyze.PgStatsCallback {
	return func(ctx context.Context, sqlStr string) (analyze.PgStatsRows, error) {
		rows, err := conn.Query(ctx, sqlStr)
		if err != nil {
			return nil, err
		}
		return &pgStatsRowAdapter{rows: rows}, nil
	}
}

// pgStatsRowAdapter adapts db.Rows to analyze.PgStatsRows interface.
type pgStatsRowAdapter struct {
	rows interface {
		Next() bool
		Scan(dest ...any) error
		Close() error
	}
}

func (a *pgStatsRowAdapter) Next() bool             { return a.rows.Next() }
func (a *pgStatsRowAdapter) Scan(dest ...any) error { return a.rows.Scan(dest...) }
func (a *pgStatsRowAdapter) Close() error           { return a.rows.Close() }

// FetchTableStats is a convenience wrapper around analyze.FetchPgStats.
func FetchTableStats(ctx context.Context, conn db.Database, tables []string) (map[string]*analyze.TableStats, error) {
	if conn.Dialect() != "PostgreSQL" {
		return nil, nil
	}
	if len(tables) == 0 {
		return nil, nil
	}
	return analyze.FetchPgStats(ctx, queryPgStats(ctx, conn), tables)
}

// collectTablesFromReport extracts table names from scan issues in a report.
func collectTablesFromReport(report *analyze.Report) []string {
	return analyze.CollectTablesFromIssues(report.Issues)
}

// RunSQL runs a SQL query via the db connection.
// Used for index apply and other mutation operations.
func RunSQL(ctx context.Context, conn db.Database, sqlStr string) error {
	// For PostgreSQL, use db's Query method
	rows, err := conn.Query(ctx, sqlStr)
	if err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}
	defer func() {
		// Drain and close
		for rows.Next() {
			var buf sql.NullString
			rows.Scan(&buf)
		}
		rows.Close()
	}()
	return nil
}
