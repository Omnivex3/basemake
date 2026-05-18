package budget

import (
	"fmt"
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


