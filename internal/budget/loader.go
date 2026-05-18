package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const budgetsDir = ".basemake"
const budgetsFile = "budgets.json"

// LoadBudgets searches upward from startDir for .basemake/budgets.json
// and returns the parsed rules. Returns nil if no budgets file exists.
func LoadBudgets(startDir string) (*BudgetFile, error) {
	dir, err := findBudgetsDir(startDir)
	if err != nil {
		return nil, nil // no budgets file found — not an error
	}

	path := filepath.Join(dir, budgetsDir, budgetsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var bf BudgetFile
	if err := json.Unmarshal(data, &bf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if bf.Version != 1 {
		return nil, fmt.Errorf("unsupported budgets version %d in %s", bf.Version, path)
	}

	return &bf, nil
}

// findBudgetsDir walks up from dir looking for .basemake/budgets.json.
// Returns the directory that contains .basemake/, or an error if not found.
func findBudgetsDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}

	for {
		check := filepath.Join(abs, budgetsDir, budgetsFile)
		if _, err := os.Stat(check); err == nil {
			return abs, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			// Reached root without finding budgets
			return "", fmt.Errorf("no %s found", filepath.Join(budgetsDir, budgetsFile))
		}
		abs = parent
	}
}

// EnsureBudgetsDir creates .basemake/ if it doesn't exist and returns its path.
func EnsureBudgetsDir(dir string) (string, error) {
	p := filepath.Join(dir, budgetsDir)
	if err := os.MkdirAll(p, 0755); err != nil {
		return "", fmt.Errorf("create %s: %w", p, err)
	}
	return p, nil
}

// matchGlob checks if the text matches a simple glob pattern.
// Supports: * (any chars), ? (single char). Case-insensitive.
func matchGlob(pattern, text string) bool {
	px := []rune(strings.ToLower(pattern))
	tx := []rune(strings.ToLower(text))

	pi, ti := 0, 0
	starIdx := -1
	starMatch := -1

	for ti < len(tx) {
		if pi < len(px) && (px[pi] == '?' || px[pi] == tx[ti]) {
			pi++
			ti++
		} else if pi < len(px) && px[pi] == '*' {
			starIdx = pi
			starMatch = ti
			pi++
		} else if starIdx != -1 {
			pi = starIdx + 1
			starMatch++
			ti = starMatch
		} else {
			return false
		}
	}

	for pi < len(px) && px[pi] == '*' {
		pi++
	}

	return pi == len(px)
}

// TemplateBudgets returns a default budgets.json template.
func TemplateBudgets() string {
	return `{
  "version": 1,
  "rules": [
    {
      "type": "table",
      "table": "*",
      "max_seq_rows": 1000,
      "message": "Add an index for large table scans"
    },
    {
      "type": "table",
      "table": "orders",
      "max_seq_rows": 100,
      "message": "Orders table must use index scans — this table powers the revenue dashboard"
    },
    {
      "type": "migration",
      "pattern": "V*__*.sql",
      "threshold": "5s",
      "message": "Migrations must complete in under 5s"
    }
  ]
}
`
}
