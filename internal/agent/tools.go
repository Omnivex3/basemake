package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/observe"
	"github.com/DynamicKarabo/basemake/internal/profile"
)

// toolSpec holds the definition and implementation for a single tool.
type toolSpec struct {
	Name        string
	Description string
	InputSchema inputSchema
	Execute     func(ctx context.Context, input map[string]any) (string, error)
}

// agentTools returns the 4 tool definitions for the agent loop.
func agentTools() []toolSpec {
	return []toolSpec{
		{
			Name:        "get_schema",
			Description: "Get database schema. Call this first for any question about tables, columns, or relationships.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propertySchema{
					"tables": {
						Type:        "array",
						Items:       &itemRef{Type: "string"},
						Description: "Optional table names to filter. Returns all tables if omitted.",
					},
				},
			},
			Execute: toolGetSchema,
		},
		{
			Name:        "get_profiles",
			Description: "Get query performance history. Call when asked about slow queries, regressions, or performance trends.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propertySchema{
					"limit": {
						Type:        "number",
						Description: "Number of recent profiles to return (default 10, max 50).",
					},
				},
			},
			Execute: toolGetProfiles,
		},
		{
			Name:        "run_explain",
			Description: "Run EXPLAIN on a SQL query. Call when you need to analyze a specific query's execution plan.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]propertySchema{
					"sql": {
						Type:        "string",
						Description: "The SQL query to explain.",
					},
				},
				Required: []string{"sql"},
			},
			Execute: toolRunExplain,
		},
		{
			Name:        "get_observations",
			Description: "Get current database observations — plan changes, slow query alerts, schema drift. Call FIRST when asked what's wrong or what changed.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]propertySchema{},
			},
			Execute: toolGetObservations,
		},
	}
}

// ── Tool implementations ──

func toolGetSchema(ctx context.Context, input map[string]any) (string, error) {
	schema, err := db.LoadSchema()
	if err != nil {
		return "", fmt.Errorf("load schema: %w", err)
	}
	if schema == nil {
		return "No schema cached. Connect to a database first with `basemake connect`.", nil
	}

	// Filter tables if requested
	var tables []string
	if raw, ok := input["tables"]; ok {
		if list, ok := raw.([]any); ok {
			for _, v := range list {
				if s, ok := v.(string); ok {
					tables = append(tables, s)
				}
			}
		}
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("Database: %s\n", schema.DBName))
	out.WriteString(fmt.Sprintf("Tables: %d\n", len(schema.Tables)))

	for _, t := range schema.Tables {
		if len(tables) > 0 && !contains(tables, t.Name) {
			continue
		}

		rowCount := ""
		if t.EstimatedRows > 0 {
			rowCount = fmt.Sprintf(" (~%d rows)", t.EstimatedRows)
		}
		out.WriteString(fmt.Sprintf("\n  %s%s\n", t.Name, rowCount))
		out.WriteString("    Columns:\n")
		for _, c := range t.Columns {
			nullable := ""
			if c.IsNullable {
				nullable = " nullable"
			}
			pk := ""
			if c.IsPK {
				pk = " [PK]"
			}
			def := ""
			if c.Default != "" {
				def = fmt.Sprintf(" default=%s", c.Default)
			}
			out.WriteString(fmt.Sprintf("      - %s %s%s%s%s\n", c.Name, c.Type, pk, nullable, def))
		}
		if len(t.Indexes) > 0 {
			out.WriteString("    Indexes:\n")
			for _, idx := range t.Indexes {
				unique := ""
				if idx.Unique {
					unique = " (unique)"
				}
				out.WriteString(fmt.Sprintf("      - %s on (%s)%s\n", idx.Name, strings.Join(idx.Cols, ", "), unique))
			}
		}
		if len(t.ForeignKeys) > 0 {
			out.WriteString("    Foreign Keys:\n")
			for _, fk := range t.ForeignKeys {
				out.WriteString(fmt.Sprintf("      - %s → %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn))
			}
		}
	}

	return out.String(), nil
}

func toolGetProfiles(ctx context.Context, input map[string]any) (string, error) {
	limit := 10
	if raw, ok := input["limit"]; ok {
		if f, ok := raw.(float64); ok {
			limit = int(f)
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 50 {
		limit = 50
	}

	dir := profile.ProfileDir()

	type namedProfile struct {
		hash          string
		normalizedSQL string
		runs          int
		avgDuration   int64
		lastRun       profile.QueryRun
	}

	var profiles []namedProfile

	// Walk directory recursively to handle scoped profile directories
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var p profile.QueryProfile
		if err := json.Unmarshal(data, &p); err != nil || len(p.Runs) == 0 {
			return nil
		}
		last := p.Runs[len(p.Runs)-1]
		var total int64
		for _, r := range p.Runs {
			total += r.DurationMS
		}
		profiles = append(profiles, namedProfile{
			hash:          strings.TrimSuffix(filepath.Base(path), ".json"),
			normalizedSQL: last.NormalizedSQL,
			runs:          len(p.Runs),
			avgDuration:   total / int64(len(p.Runs)),
			lastRun:       last,
		})
		return nil
	})

	// Sort by most recent run
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].lastRun.Timestamp.After(profiles[j].lastRun.Timestamp)
	})

	if len(profiles) > limit {
		profiles = profiles[:limit]
	}

	if len(profiles) == 0 {
		return "No profiles found. Run queries with `--explain` to build profiles.", nil
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("Recent profiles: %d queries, showing %d\n\n", len(profiles), len(profiles)))
	for _, p := range profiles {
		out.WriteString(fmt.Sprintf("Query: %s\n", p.normalizedSQL))
		out.WriteString(fmt.Sprintf("  Runs: %d, Avg: %dms, Last: %dms (%s)\n",
			p.runs, p.avgDuration, p.lastRun.DurationMS,
			p.lastRun.Timestamp.Format("2006-01-02 15:04")))
		if p.lastRun.RowsReturned > 0 {
			out.WriteString(fmt.Sprintf("  Rows: %d\n", p.lastRun.RowsReturned))
		}
		if p.lastRun.PlanHash != "" {
			out.WriteString(fmt.Sprintf("  Plan hash: %s\n", p.lastRun.PlanHash))
		}
		out.WriteString("\n")
	}

	return out.String(), nil
}

func toolRunExplain(ctx context.Context, input map[string]any) (string, error) {
	sql, ok := input["sql"].(string)
	if !ok || strings.TrimSpace(sql) == "" {
		return "", fmt.Errorf("'sql' parameter is required and must be a string")
	}

	conn, err := db.ActiveConnection()
	if err != nil {
		return "", fmt.Errorf("no active database connection: %w", err)
	}

	plan, err := conn.ExplainNoAnalyze(ctx, sql)
	if err != nil {
		return "", fmt.Errorf("explain failed: %w", err)
	}

	return fmt.Sprintf("EXPLAIN for: %s\n\n%s", sql, plan), nil
}

func toolGetObservations(ctx context.Context, input map[string]any) (string, error) {
	obs := observe.Brief()
	if obs == "" {
		return "No observations. Everything looks normal — no plan changes, slow queries, or schema drift detected.", nil
	}
	return obs, nil
}

// ── Helpers ──

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
