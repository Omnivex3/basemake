package budget

import (
	"encoding/json"
	"time"
)

// RuleType identifies what kind of budget rule this is.
type RuleType string

const (
	RuleTable     RuleType = "table"
	RuleMigration RuleType = "migration"
	RuleQuery     RuleType = "query"
)

// BudgetFile is the top-level structure for .basemake/budgets.json.
type BudgetFile struct {
	Version int          `json:"version"`
	Rules   []BudgetRule `json:"rules"`
}

// BudgetRule encodes a single database performance policy.
type BudgetRule struct {
	Type RuleType `json:"type"`

	Table  string `json:"table,omitempty"`
	Pattern string `json:"pattern,omitempty"`

	MaxSeqRows   int      `json:"max_seq_rows,omitempty"`
	RequireIndex []string `json:"require_index,omitempty"`
	Threshold    string   `json:"threshold,omitempty"`
	Message      string   `json:"message,omitempty"`
}

// ThresholdDuration parses Threshold as a time.Duration.
func (r BudgetRule) ThresholdDuration() (time.Duration, error) {
	if r.Threshold == "" {
		return 0, nil
	}
	return time.ParseDuration(r.Threshold)
}

// MatchTable returns true if the rule applies to the given table name.
func (r BudgetRule) MatchTable(table string) bool {
	if r.Type != RuleTable {
		return false
	}
	return r.Table == table || r.Table == "*"
}

// MatchMigration returns true if the filename matches the rule's pattern.
func (r BudgetRule) MatchMigration(filename string) bool {
	if r.Type != RuleMigration {
		return false
	}
	return matchGlob(r.Pattern, filename)
}

// MatchQuery returns true if the query text matches the rule's pattern.
func (r BudgetRule) MatchQuery(sql string) bool {
	if r.Type != RuleQuery {
		return false
	}
	if r.Pattern == "" {
		return false
	}
	return matchGlob(r.Pattern, sql)
}

// ValidationResult holds the outcome of checking a rule against a query.
type ValidationResult struct {
	Rule    BudgetRule
	Passed  bool
	Message string
}

// EvaluationReport collects all validation results for a single check.
type EvaluationReport struct {
	Results   []ValidationResult
	HasErrors bool
	// Violations is a flat list of human-readable violation messages.
	Violations []string
}

// MarshalJSON serializes a BudgetFile for display/export.
func (bf *BudgetFile) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(struct {
		Version int          `json:"version"`
		Rules   []BudgetRule `json:"rules"`
	}{
		Version: bf.Version,
		Rules:   bf.Rules,
	}, "", "  ")
}

// UnmarshalJSON deserializes a BudgetFile.
func (bf *BudgetFile) UnmarshalJSON(data []byte) error {
	var raw struct {
		Version int            `json:"version"`
		Rules   []json.RawMessage `json:"rules"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	bf.Version = raw.Version
	for _, r := range raw.Rules {
		var rule BudgetRule
		if err := json.Unmarshal(r, &rule); err != nil {
			return err
		}
		bf.Rules = append(bf.Rules, rule)
	}
	return nil
}
