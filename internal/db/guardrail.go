package db

import (
	"fmt"
	"regexp"
	"strings"
)

// ── SELECT * Guardrail ──

// selectStarRE matches SELECT * or SELECT table.* patterns (case-insensitive).
// It aims for the outermost SELECT — subqueries and CTEs are not matched.
var selectStarRE = regexp.MustCompile(`(?i)\bSELECT\s+(?:ALL\s+|DISTINCT\s+)?\*`)

// selectStarPrefixRE matches SELECT followed by a column list ending in .*
// e.g. SELECT col1, col2, t.* FROM ...
var selectStarPrefixRE = regexp.MustCompile(`(?i)\bSELECT\s+.+?\b\w+\.\*\s`)

// GuardrailResult describes the outcome of a guardrail check.
type GuardrailResult struct {
	SQL     string // possibly modified SQL
	Warning string // human-readable warning (empty = no issue)
	Blocked bool   // true = execution should be prevented
	Rewrote bool   // true = SQL was modified
}

// GuardrailSelectStar checks a SQL query for SELECT * patterns and applies
// row-count-aware protections. Returns the (possibly rewritten) SQL and a result.
//
// Tiers (based on the largest table's estimated row count for the *d table):
//
//	< 10K  → rewrite to explicit columns, log what was done
//	10K–1M → warn + rewrite
//	> 1M   → block with suggestion
//	unknown → warn once, pass through
func GuardrailSelectStar(sql string, schema *Schema) GuardrailResult {
	if schema == nil || !selectStarRE.MatchString(sql) {
		return GuardrailResult{SQL: sql}
	}

	// Find which top-level table is being SELECT *'d
	tableName := extractTableFromSQL(sql)
	if tableName == "" {
		// Complex query — can't determine the table, warn generically
		if selectStarRE.MatchString(sql) {
			return GuardrailResult{
				SQL:     sql,
				Warning: "⚠️ Query uses SELECT * — consider specifying columns explicitly for better performance and clarity.",
			}
		}
		return GuardrailResult{SQL: sql}
	}

	// Look up the table in the schema
	var table *TableInfo
	for i := range schema.Tables {
		if strings.EqualFold(schema.Tables[i].Name, tableName) {
			table = &schema.Tables[i]
			break
		}
	}

	if table == nil {
		// Table not in schema cache — generic warning
		return GuardrailResult{
			SQL:     sql,
			Warning: fmt.Sprintf("⚠️ Query uses SELECT * on %q — consider specifying columns explicitly.", tableName),
		}
	}

	rows := table.EstimatedRows
	colCount := len(table.Columns)

	// Try to rewrite SELECT * to explicit columns
	rewrittenSQL := rewriteSelectStar(sql, table)

	switch {
	case rows == 0:
		// Unknown row count — warn but allow
		warning := fmt.Sprintf("⚠️ Query uses SELECT * on %q (%d columns). Row count unknown — consider explicit columns.", tableName, colCount)
		if rewrittenSQL != sql {
			warning = fmt.Sprintf("ℹ️ Expanded SELECT * on %q to %d explicit columns.", tableName, colCount)
		}
		return GuardrailResult{
			SQL:     rewrittenSQL,
			Warning: warning,
			Rewrote: rewrittenSQL != sql,
		}

	case rows < 10000:
		// Small table — rewrite with a note
		warning := fmt.Sprintf("ℹ️ Expanded SELECT * on %q (~%d rows) to %d explicit columns.", tableName, rows, colCount)
		return GuardrailResult{
			SQL:     rewrittenSQL,
			Warning: warning,
			Rewrote: rewrittenSQL != sql,
		}

	case rows < 1_000_000:
		// Medium table — warn + rewrite
		warning := fmt.Sprintf("⚠️ SELECT * on %q (~%d rows, %d columns). Consider selecting only needed columns for performance.", tableName, rows, colCount)
		return GuardrailResult{
			SQL:     rewrittenSQL,
			Warning: warning,
			Rewrote: rewrittenSQL != sql,
		}

	default:
		// Large table — block
		return GuardrailResult{
			SQL:     sql,
			Warning: fmt.Sprintf("🚫 SELECT * on %q (~%d rows) blocked. Add explicit column names or a LIMIT clause:\n  SELECT col1, col2, ... FROM %s LIMIT 100", tableName, rows, tableName),
			Blocked: true,
		}
	}
}

// extractTableFromSQL does a best-effort extraction of the first FROM-clause table.
// Handles: SELECT * FROM table, SELECT * FROM table AS t, SELECT * FROM "table"
func extractTableFromSQL(sql string) string {
	// Normalize: strip string literals and comments for simpler matching
	clean := stripStringLiterals(sql)
	clean = stripComments(clean)

	// Look for FROM clause after the first SELECT
	re := regexp.MustCompile(`(?i)\bFROM\s+[` + "`" + `"'\[]?(\w+)[` + "`" + `"'\]]?`)
	matches := re.FindStringSubmatch(clean)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// rewriteSelectStar replaces the first SELECT * with explicit column names.
// For simple SELECT * FROM table, replaces * with the column list.
// For SELECT t.* FROM table t, replaces t.* with t.col1, t.col2, ...
func rewriteSelectStar(sql string, table *TableInfo) string {
	if len(table.Columns) == 0 {
		return sql
	}

	colNames := make([]string, len(table.Columns))
	for i, c := range table.Columns {
		colNames[i] = c.Name
	}
	explicitCols := strings.Join(colNames, ", ")

	// Replace SELECT * with explicit columns
	re := regexp.MustCompile(`(?i)(\bSELECT\s+(?:ALL\s+|DISTINCT\s+)?)\*\b`)
	modified := re.ReplaceAllString(sql, "${1}"+explicitCols)

	// If no change, try SELECT table.*
	if modified == sql {
		re2 := regexp.MustCompile(`(?i)(\w+)\.\*`)
		modified = re2.ReplaceAllStringFunc(sql, func(match string) string {
			parts := strings.SplitN(match, ".", 2)
			if len(parts) != 2 {
				return match
			}
			prefix := parts[0]
			prefixedCols := make([]string, len(table.Columns))
			for i, c := range table.Columns {
				prefixedCols[i] = prefix + "." + c.Name
			}
			return strings.Join(prefixedCols, ", ")
		})
	}

	return modified
}

// ── SQL helpers ──

// stripStringLiterals removes content inside single-quoted strings.
func stripStringLiterals(sql string) string {
	var out strings.Builder
	inStr := false
	escaped := false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if escaped {
			out.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\'' {
			inStr = !inStr
			out.WriteByte(' ')
			continue
		}
		if ch == '\\' && inStr {
			escaped = true
			continue
		}
		if inStr {
			out.WriteByte(' ')
		} else {
			out.WriteByte(ch)
		}
	}
	return out.String()
}

// stripComments removes SQL comments (-- and /* */).
func stripComments(sql string) string {
	// Remove single-line comments
	re := regexp.MustCompile(`(?m)--.*$`)
	s := re.ReplaceAllString(sql, "")

	// Remove block comments
	re2 := regexp.MustCompile(`(?s)/\*.*?\*/`)
	s = re2.ReplaceAllString(s, "")
	return s
}
