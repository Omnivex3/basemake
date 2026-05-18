package budget

import (
	"fmt"
	"strings"
)

// ScanInfo describes a sequential scan discovered in the query plan.
type ScanInfo struct {
	Table    string
	RowCount int
}

// EvaluateCheck checks a single SQL query against all applicable budget rules.
// It takes table scan information from the plan analysis to determine violations.
func EvaluateCheck(sql string, scans []ScanInfo, bf *BudgetFile) *EvaluationReport {
	report := &EvaluationReport{}

	if bf == nil {
		return report
	}

	for _, rule := range bf.Rules {
		result := ValidationResult{Rule: rule, Passed: true}

		switch rule.Type {
		case RuleTable:
			// Find matching scans
			for _, scan := range scans {
				if !rule.MatchTable(scan.Table) {
					continue
				}
				// Check max_seq_rows
				if rule.MaxSeqRows > 0 && scan.RowCount > rule.MaxSeqRows {
					result.Passed = false
					result.Message = rule.Message
					if result.Message == "" {
						result.Message = fmt.Sprintf("Table %s exceeds max_seq_rows budget (%d > %d)",
							scan.Table, scan.RowCount, rule.MaxSeqRows)
					}
					break
				}
			}
			// If rule exists but no matching scan, it passes
			if result.Passed {
				continue
			}

		case RuleMigration:
			if !rule.MatchMigration(sql) {
				continue
			}
			result.Passed = true // migration patterns just carry threshold

		case RuleQuery:
			if !rule.MatchQuery(sql) {
				continue
			}
			result.Passed = true // query patterns just carry threshold
		}

		report.Results = append(report.Results, result)
		if !result.Passed {
			report.HasErrors = true
		}
	}

	return report
}

// EffectiveThreshold returns the most specific threshold from budgets
// for the given query and input filename, or empty string if none is set.
func EffectiveThreshold(sql, filename string, bf *BudgetFile) string {
	if bf == nil {
		return ""
	}

	// Priority: migration patterns > query patterns > table rules > nothing
	closest := ""

	for _, rule := range bf.Rules {
		if rule.Threshold == "" {
			continue
		}

		switch rule.Type {
		case RuleMigration:
			if rule.MatchMigration(filename) {
				closest = rule.Threshold
			}
		case RuleQuery:
			if rule.MatchQuery(sql) {
				if closest == "" {
					closest = rule.Threshold
				}
			}
		case RuleTable:
			// Table rules provide fallback threshold
			if rule.Table == "*" && closest == "" {
				closest = rule.Threshold
			}
		}
	}

	return closest
}

// FormatReport returns a human-readable summary of budget violations.
func (r *EvaluationReport) FormatReport() string {
	if r == nil || len(r.Results) == 0 {
		return ""
	}

	var b strings.Builder
	for _, vr := range r.Results {
		if vr.Passed {
			continue
		}
		b.WriteString(fmt.Sprintf("🔴 Budget violation: %s\n", vr.Message))
		if vr.Rule.MaxSeqRows > 0 {
			b.WriteString(fmt.Sprintf("   max_seq_rows: %d\n", vr.Rule.MaxSeqRows))
		}
		if len(vr.Rule.RequireIndex) > 0 {
			b.WriteString(fmt.Sprintf("   require_index: %s\n", strings.Join(vr.Rule.RequireIndex, ", ")))
		}
	}
	return b.String()
}

// ExtractTablesFromSQL attempts to identify table names referenced in a SQL query.
// This is a best-effort parser — it looks for patterns like "FROM table", "JOIN table", "UPDATE table".
func ExtractTablesFromSQL(sql string) []string {
	upper := strings.ToUpper(sql)
	tables := []string{}
	seen := map[string]bool{}

	// Very simple extraction: find words after FROM, JOIN, UPDATE, INTO
	keywords := []string{"FROM ", "JOIN ", "UPDATE ", "INTO "}

	parts := []string{upper}
	for _, kw := range keywords {
		var newParts []string
		for _, p := range parts {
			split := strings.Split(p, kw)
			for i, s := range split {
				if i > 0 {
					// The next word after the keyword is likely a table name
					fields := strings.Fields(s)
					if len(fields) > 0 {
						name := strings.Trim(fields[0], "`\"'[];(),")
						if name != "" && !isSQLKeyword(name) && !seen[name] {
							seen[name] = true
							tables = append(tables, name)
						}
					}
				}
				newParts = append(newParts, s)
			}
		}
		parts = newParts
	}

	return tables
}

// Common SQL keywords to filter out from table name extraction.
var sqlKeywords = map[string]bool{
	"SELECT": true, "WHERE": true, "AND": true, "OR": true, "NOT": true,
	"IN": true, "ON": true, "AS": true, "SET": true, "VALUES": true,
	"GROUP": true, "ORDER": true, "BY": true, "HAVING": true, "LIMIT": true,
	"OFFSET": true, "UNION": true, "ALL": true, "DISTINCT": true,
	"LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true, "FULL": true,
	"CROSS": true, "NATURAL": true, "USING": true, "EXISTS": true,
	"BETWEEN": true, "LIKE": true, "IS": true, "NULL": true, "TRUE": true,
	"FALSE": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "ASC": true, "DESC": true, "WITH": true,
}

func isSQLKeyword(word string) bool {
	return sqlKeywords[word]
}
