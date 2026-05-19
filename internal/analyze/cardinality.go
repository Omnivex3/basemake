package analyze

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// ─── Column Statistics ──────────────────────────────────────────────────────

// ColumnStats holds the pg_stats data for a single column.
type ColumnStats struct {
	Table     string    `json:"table"`
	Column    string    `json:"column"`
	NDistinct float64   `json:"n_distinct"` // positive = exact, negative = fraction, -1 = estimated
	NullFrac  float64   `json:"null_frac"`
	AvgWidth  int       `json:"avg_width"`
	Corr      float64   `json:"correlation"`
	MCV       []string  `json:"most_common_vals,omitempty"`
	MCF       []float64 `json:"most_common_freqs,omitempty"`
}

// IsEstimated returns true when n_distinct = -1 (PG estimated via sampling).
func (c ColumnStats) IsEstimated() bool {
	return c.NDistinct == -1
}

// ExactDistinct returns the estimated number of distinct values.
// > 0: exact count. < 0 but != -1: fraction of total rows. = -1: 0 (caller checks IsEstimated).
func (c ColumnStats) ExactDistinct(totalRows int64) float64 {
	if c.NDistinct > 0 {
		return c.NDistinct
	}
	if c.NDistinct < 0 && c.NDistinct != -1 {
		return float64(totalRows) * (-c.NDistinct)
	}
	return 0
}

// Selectivity returns estimated fraction of rows matching an equality value.
// Uses MCV if available, otherwise spreads remaining frequency across non-MCV values.
// Returns -1 when stats are speculative (n_distinct=-1).
func (c ColumnStats) Selectivity(value string, totalRows int64) float64 {
	if c.IsEstimated() {
		return -1
	}

	// MCV lookup
	for i, v := range c.MCV {
		if v == value {
			return c.MCF[i]
		}
	}

	distinct := c.ExactDistinct(totalRows)
	if distinct <= 0 {
		return -1
	}

	sumMCV := 0.0
	for _, f := range c.MCF {
		sumMCV += f
	}
	remaining := 1.0 - sumMCV
	remainingDistinct := distinct - float64(len(c.MCV))
	if remainingDistinct <= 0 {
		return 0
	}
	return remaining / remainingDistinct
}

// EstimateRows returns row count estimate for a value match. -1 when speculative.
func (c ColumnStats) EstimateRows(value string, totalRows int64) int64 {
	sel := c.Selectivity(value, totalRows)
	if sel < 0 {
		return -1
	}
	return int64(math.Round(float64(totalRows) * sel))
}

// ─── Table Stats ────────────────────────────────────────────────────────────

type TableStats struct {
	Name      string                 `json:"name"`
	TotalRows int64                  `json:"total_rows"`
	Columns   map[string]ColumnStats `json:"columns"`
}

// ─── Index Suggestion ───────────────────────────────────────────────────────

type IndexSuggestion struct {
	Table          string   `json:"table"`
	Columns        []string `json:"columns"`
	PartialWhere   string   `json:"partial_where,omitempty"`
	EstImprovement string   `json:"est_improvement"`
	Tradeoffs      []string `json:"tradeoffs"`
	Confidence     string   `json:"confidence"` // "high", "medium", "speculative"
	Reason         string   `json:"reason"`
	CreateSQL      string   `json:"create_sql"`
}

// SuggestIndexesForScan generates index suggestions for a Seq Scan node.
func SuggestIndexesForScan(table, filter string, planRows float64, stats *TableStats) []IndexSuggestion {
	if stats == nil {
		return nil
	}

	var suggestions []IndexSuggestion
	if filter == "" {
		return nil
	}

	cols := extractColumnsFromFilter(filter)
	for _, col := range cols {
		cs, ok := stats.Columns[col]
		if !ok {
			continue
		}

		confidence := "high"
		if cs.IsEstimated() {
			confidence = "speculative"
		} else if len(cs.MCV) > 0 {
			// Has MCV data — high confidence for matching values
			mcvCoverage := 0.0
			for _, f := range cs.MCF {
				mcvCoverage += f
			}
			if mcvCoverage < 0.3 {
				confidence = "medium" // MCV doesn't cover most values
			}
		}

		improvement := "Seq Scan → Index Scan"
		if planRows > 0 {
			improvement = fmt.Sprintf("Seq Scan → Index Scan (~%d rows scanned)", int(planRows))
		}

		partialClause := detectPartialIndex(filter, col, cs, stats.TotalRows)
		cols := []string{col}
		createSQL := fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s(%s)", stats.Name, col, stats.Name, col)
		if partialClause != "" {
			createSQL += " " + partialClause
		}

		entry := IndexSuggestion{
			Table:          stats.Name,
			Columns:        cols,
			PartialWhere:   partialClause,
			EstImprovement: improvement,
			Confidence:     confidence,
			Reason:         fmt.Sprintf("Seq Scan on %s filtering by: %s", stats.Name, filter),
			Tradeoffs:      estimateTradeoffs(stats.Name, col, cs),
			CreateSQL:      createSQL,
		}
		suggestions = append(suggestions, entry)
	}

	return suggestions
}

// ─── Filter Parsing ─────────────────────────────────────────────────────────

func extractColumnsFromFilter(filter string) []string {
	var cols []string
	seen := make(map[string]bool)

	if filter == "" {
		return nil
	}

	// First, split by AND/OR (case-insensitive) to isolate individual predicates
	// This prevents operator splitting from mixing terms
	predicates := splitPredicates(filter)

	for _, pred := range predicates {
		pred = strings.TrimSpace(pred)
		if pred == "" {
			continue
		}
		// Find the first SQL operator in this predicate
		col := extractColumnFromPredicate(pred)
		if col != "" && !seen[col] {
			cols = append(cols, col)
			seen[col] = true
		}
	}

	return cols
}

// splitPredicates splits a filter expression into individual conditions
// by AND/OR boundaries. Strips outer wrapping parens first.
func splitPredicates(filter string) []string {
	// Strip outer wrapping parens
	for strings.HasPrefix(filter, "(") && strings.HasSuffix(filter, ")") {
		inner := filter[1 : len(filter)-1]
		// Only strip if the parens are matching (count open/close)
		d := 0
		matching := true
		for i := 0; i < len(inner); i++ {
			switch inner[i] {
			case '(':
				d++
			case ')':
				if d == 0 {
					matching = false
				} else {
					d--
				}
			}
		}
		if matching && d == 0 {
			filter = inner
		} else {
			break
		}
	}

	upper := strings.ToUpper(filter)
	var result []string
	start := 0
	depth := 0

	for i := 0; i < len(upper); i++ {
		switch upper[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				// Check for AND (with word boundaries)
				if i+3 < len(upper) && upper[i:i+3] == "AND" &&
					(i == 0 || !isIdentByte(upper[i-1])) &&
					(i+3 >= len(upper) || !isIdentByte(upper[i+3])) {
					result = append(result, filter[start:i])
					i += 2 // skip "ND"
					start = i + 1
					continue
				}
				// Check for OR (with word boundaries)
				if i+2 < len(upper) && upper[i:i+2] == "OR" &&
					(i == 0 || !isIdentByte(upper[i-1])) &&
					(i+2 >= len(upper) || !isIdentByte(upper[i+2])) {
					result = append(result, filter[start:i])
					i += 1 // skip 'R'
					start = i + 1
					continue
				}
			}
		}
	}

	if start < len(filter) {
		result = append(result, filter[start:])
	}
	return result
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// extractColumnFromPredicate finds the column name in a single predicate.
// e.g., "status = 'active'" → "status", "(total > 100)" → "total"
func extractColumnFromPredicate(pred string) string {
	// Remove surrounding parens
	for strings.HasPrefix(pred, "(") {
		pred = strings.TrimPrefix(pred, "(")
	}
	for strings.HasSuffix(pred, ")") {
		pred = strings.TrimSuffix(pred, ")")
	}
	pred = strings.TrimSpace(pred)
	if pred == "" {
		return ""
	}

	// Find the first operator position
	ops := []string{" = ", " IN ", " > ", " < ", " >= ", " <= ", " <> ", " != ", " IS ", " LIKE ", " BETWEEN "}
	firstOpIdx := -1
	for _, op := range ops {
		idx := strings.Index(pred, op)
		if idx >= 0 && (firstOpIdx < 0 || idx < firstOpIdx) {
			firstOpIdx = idx
		}
	}

	if firstOpIdx < 0 {
		return ""
	}

	candidate := pred[:firstOpIdx]
	candidate = strings.TrimSpace(candidate)

	// Strip table qualifier
	if dot := strings.LastIndex(candidate, "."); dot >= 0 {
		candidate = candidate[dot+1:]
	}
	// Strip type cast
	if colon := strings.Index(candidate, "::"); colon >= 0 {
		candidate = candidate[:colon]
	}
	candidate = strings.TrimSpace(candidate)

	if candidate != "" && isIdent(candidate) {
		return candidate
	}
	return ""
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_') {
			return false
		}
	}
	return !isKeyword(s)
}

var sqlKeywords = map[string]bool{
	"select": true, "from": true, "where": true, "and": true, "or": true,
	"not": true, "in": true, "is": true, "null": true, "like": true,
	"between": true, "true": true, "false": true, "as": true, "on": true,
}

func isKeyword(s string) bool {
	return sqlKeywords[strings.ToLower(s)]
}

// ─── Partial Index Detection ────────────────────────────────────────────────

func detectPartialIndex(filter, column string, cs ColumnStats, totalRows int64) string {
	// Look for: col = '<value>' with appropriate selectivity
	idx := strings.Index(filter, column+" =")
	if idx < 0 {
		return ""
	}
	rest := filter[idx+len(column)+2:]
	rest = strings.TrimSpace(rest)

	value := ""
	if strings.HasPrefix(rest, "'") {
		end := strings.Index(rest[1:], "'")
		if end >= 0 {
			value = rest[1 : end+1]
		}
	} else if strings.HasPrefix(rest, "\"") {
		end := strings.Index(rest[1:], "\"")
		if end >= 0 {
			value = rest[1 : end+1]
		}
	} else {
		// Numeric or boolean literal — read until space/paren/comma
		for i := 0; i < len(rest); i++ {
			ch := rest[i]
			if ch == ' ' || ch == ')' || ch == ',' || ch == ';' {
				value = rest[:i]
				break
			}
		}
		if value == "" {
			value = rest
		}
	}

	if value == "" {
		return ""
	}

	// Check selectivity — partial index is worthwhile when value matches < 30% of rows
	sel := cs.Selectivity(value, totalRows)
	if sel >= 0 && sel < 0.30 {
		return fmt.Sprintf("WHERE %s = '%s'", column, value)
	}

	return ""
}

// ─── Trade-offs ─────────────────────────────────────────────────────────────

func estimateTradeoffs(table, column string, cs ColumnStats) []string {
	var t []string
	pct := int(math.Ceil(float64(cs.AvgWidth) / 10.0))
	if pct < 3 {
		pct = 3
	}
	if pct > 25 {
		pct = 25
	}
	t = append(t, fmt.Sprintf("+~%d%% INSERT overhead on %s", pct, table))
	if cs.AvgWidth > 4 {
		t = append(t, fmt.Sprintf("+~%d%% UPDATE overhead on %s.%s", pct+2, table, column))
	}
	if cs.NullFrac > 0.5 {
		t = append(t, fmt.Sprintf("%.0f%% NULL values — partial index could skip nulls", cs.NullFrac*100))
	}
	return t
}

// ─── MCV Parsers ────────────────────────────────────────────────────────────

func ParseMCV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")

	var vals []string
	var cur strings.Builder
	inQuotes := false
	escaped := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			cur.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && ch == ',' {
			vals = append(vals, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(ch)
	}
	if cur.Len() > 0 {
		vals = append(vals, cur.String())
	}
	return vals
}

func ParseMCF(raw string) []float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")

	parts := strings.Split(raw, ",")
	freqs := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var f float64
		if _, err := fmt.Sscanf(p, "%f", &f); err == nil {
			freqs = append(freqs, f)
		}
	}
	return freqs
}

// ─── PG Stats Fetcher ───────────────────────────────────────────────────────

// PgStatsCallback is a function that queries the database and returns raw rows.
type PgStatsCallback func(ctx context.Context, sql string) (PgStatsRows, error)

// PgStatsRows is the scan interface for pg_stats query results.
type PgStatsRows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
}

// FetchPgStats queries pg_stats via the callback and returns structured stats.
func FetchPgStats(ctx context.Context, callback PgStatsCallback, tables []string) (map[string]*TableStats, error) {
	query := buildStatsQuery(tables)
	rows, err := callback(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pg_stats query: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*TableStats)
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
			st = &TableStats{Name: tabName, Columns: make(map[string]ColumnStats)}
			result[tabName] = st
		}

		cs := ColumnStats{
			Table:     tabName,
			Column:    colName,
			NDistinct: nDistinct,
			NullFrac:  nullFrac,
			AvgWidth:  avgWidth,
			Corr:      corr,
		}

		if mcvRaw != nil && *mcvRaw != "" {
			cs.MCV = ParseMCV(*mcvRaw)
		}
		if mcfRaw != nil && *mcfRaw != "" {
			cs.MCF = ParseMCF(*mcfRaw)
		}

		st.Columns[colName] = cs
	}
	if err := rows.Close(); err != nil {
		return result, err
	}

	for _, tbl := range tables {
		st, ok := result[tbl]
		if !ok {
			continue
		}
		row, err := callback(ctx,
			fmt.Sprintf("SELECT reltuples::bigint FROM pg_class WHERE relname = '%s'",
				strings.ReplaceAll(tbl, "'", "''")))
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
	if len(tables) == 0 {
		return `SELECT tablename, attname, n_distinct, null_frac, avg_width, correlation,
					   most_common_vals::text, most_common_freqs::text
				FROM pg_stats
				WHERE schemaname = 'public'
				ORDER BY tablename, attname`
	}
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
func FormatSuggestions(suggestions []IndexSuggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n  💡 Index Suggestions\n\n")
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

// CollectTablesFromIssues extracts unique table names from issues.
func CollectTablesFromIssues(issues []Issue) []string {
	seen := make(map[string]bool)
	var tables []string
	for _, iss := range issues {
		if iss.TableName != "" && !seen[iss.TableName] {
			seen[iss.TableName] = true
			tables = append(tables, iss.TableName)
		}
	}
	return tables
}
