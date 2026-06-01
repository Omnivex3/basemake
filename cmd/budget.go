package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/budget"
	"github.com/DynamicKarabo/basemake/internal/license"
	"github.com/spf13/cobra"
)

var (
	budgetTable     string
	budgetPattern   string
	budgetMaxSeq    int
	budgetRequire   []string
	budgetThreshold string
	budgetMessage   string
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Database performance policy as code",
	Long: `Manage database performance budgets — policy that travels with your code.

Budgets are stored in .basemake/budgets.json and can be checked into version control.
They define performance expectations for tables, migrations, and queries.

  basemake budget init                           # Create a template budgets.json
  basemake budget set --table orders --max-seq-rows 1000
  basemake budget set --pattern "V*__*.sql" --threshold 5s
  basemake budget list                           # Show all active rules
  basemake budget diff                           # Compare with committed version`,
}

var budgetInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a template .basemake/budgets.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := findBudgetsRoot()
		if err != nil {
			dir, _ = os.Getwd()
		}

		bmDir, err := budget.EnsureBudgetsDir(dir)
		if err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		path := filepath.Join(bmDir, "budgets.json")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("budgets.json already exists at %s", path)
		}

		if err := os.WriteFile(path, []byte(budget.TemplateBudgets()), 0600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}

		fmt.Printf("✅ Created %s\n", path)
		fmt.Println("  Edit this file to define your team's database performance policy.")
		fmt.Println("  Check it into version control alongside your migrations.")
		return nil
	},
}

var budgetSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Add or update a budget rule",
	Example: `  basemake budget set --table orders --max-seq-rows 1000
  basemake budget set --table users --require-index email,status
  basemake budget set --pattern "V*__*.sql" --threshold 5s
  basemake budget set --pattern "SELECT * FROM" --threshold 1s`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !requireLicense(license.FeatureBudget) {
			return fmt.Errorf("license required for budget feature")
		}
		dir, err := findBudgetsRoot()
		if err != nil {
			return fmt.Errorf("no .basemake/budgets.json found — run 'basemake budget init' first")
		}

		bf, err := budget.LoadBudgets(dir)
		if err != nil {
			return fmt.Errorf("load budgets: %w", err)
		}

		rule := budget.BudgetRule{
			Message: budgetMessage,
		}

		if budgetTable != "" {
			rule.Type = budget.RuleTable
			rule.Table = budgetTable
			rule.MaxSeqRows = budgetMaxSeq
			if len(budgetRequire) > 0 {
				rule.RequireIndex = budgetRequire
			}
		} else if budgetPattern != "" {
			// Determine type by context: if --migration was used, it's migration
			// We're inferring from the fact that --pattern is used vs --table
			// Actually use flags separately. Let's check which flag was set.
			rule.Type = budget.RuleQuery
			rule.Pattern = budgetPattern
		} else {
			return fmt.Errorf("specify --table or --pattern")
		}

		if budgetThreshold != "" {
			rule.Threshold = budgetThreshold
		}

		bf.Rules = append(bf.Rules, rule)

		if err := saveBudgets(dir, bf); err != nil {
			return err
		}

		fmt.Printf("✅ Budget rule added for %s\n", describeRule(rule))
		return nil
	},
}

var budgetSetTableCmd = &cobra.Command{
	Use:   "table <name>",
	Short: "Set a budget rule for a specific table",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := findBudgetsRoot()
		if err != nil {
			return fmt.Errorf("no .basemake/budgets.json found — run 'basemake budget init' first")
		}

		bf, err := budget.LoadBudgets(dir)
		if err != nil {
			return fmt.Errorf("load budgets: %w", err)
		}

		rule := budget.BudgetRule{
			Type:         budget.RuleTable,
			Table:        args[0],
			MaxSeqRows:   budgetMaxSeq,
			RequireIndex: budgetRequire,
			Threshold:    budgetThreshold,
			Message:      budgetMessage,
		}

		bf.Rules = append(bf.Rules, rule)

		if err := saveBudgets(dir, bf); err != nil {
			return err
		}

		fmt.Printf("✅ Budget rule added for table `%s`\n", rule.Table)
		return nil
	},
}

var budgetSetMigrationCmd = &cobra.Command{
	Use:   "migration <pattern>",
	Short: "Set a budget rule for migration files matching a pattern",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := findBudgetsRoot()
		if err != nil {
			return fmt.Errorf("no .basemake/budgets.json found — run 'basemake budget init' first")
		}

		bf, err := budget.LoadBudgets(dir)
		if err != nil {
			return fmt.Errorf("load budgets: %w", err)
		}

		rule := budget.BudgetRule{
			Type:      budget.RuleMigration,
			Pattern:   args[0],
			Threshold: budgetThreshold,
			Message:   budgetMessage,
		}

		bf.Rules = append(bf.Rules, rule)

		if err := saveBudgets(dir, bf); err != nil {
			return err
		}

		fmt.Printf("✅ Budget rule added for migrations matching `%s`\n", rule.Pattern)
		return nil
	},
}

var budgetListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all budget rules",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := findBudgetsRoot()
		if err != nil {
			fmt.Println("No .basemake/budgets.json found.")
			fmt.Println("  Run 'basemake budget init' to create one.")
			return nil
		}

		bf, err := budget.LoadBudgets(dir)
		if err != nil {
			return fmt.Errorf("load budgets: %w", err)
		}

		if len(bf.Rules) == 0 {
			fmt.Println("No budget rules defined.")
			return nil
		}

		fmt.Printf("Budgets from %s/.basemake/budgets.json:\n\n", filepath.Base(dir))
		for i, rule := range bf.Rules {
			fmt.Printf("  %d. [%s] %s\n", i+1, rule.Type, describeRule(rule))
		}
		return nil
	},
}

var budgetDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show unstaged changes to budgets.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := findBudgetsRoot()
		if err != nil {
			fmt.Println("No .basemake/budgets.json found.")
			return nil
		}

		path := filepath.Join(dir, ".basemake", "budgets.json")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		// Show current content
		fmt.Printf("📋 %s:\n\n", path)
		fmt.Println(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(budgetCmd)
	budgetCmd.AddCommand(budgetInitCmd)
	budgetCmd.AddCommand(budgetSetCmd)
	budgetCmd.AddCommand(budgetSetTableCmd)
	budgetCmd.AddCommand(budgetSetMigrationCmd)
	budgetCmd.AddCommand(budgetListCmd)
	budgetCmd.AddCommand(budgetDiffCmd)

	// Flags for budget set
	budgetSetCmd.Flags().StringVar(&budgetTable, "table", "", "Table name (e.g. orders)")
	budgetSetCmd.Flags().StringVar(&budgetPattern, "pattern", "", "Query pattern glob")
	budgetSetCmd.Flags().IntVar(&budgetMaxSeq, "max-seq-rows", 0, "Max sequential scan rows")
	budgetSetCmd.Flags().StringSliceVar(&budgetRequire, "require-index", nil, "Required index columns (comma-separated)")
	budgetSetCmd.Flags().StringVar(&budgetThreshold, "threshold", "", "Max execution time (e.g. 500ms)")
	budgetSetCmd.Flags().StringVar(&budgetMessage, "message", "", "Human-readable policy explanation")

	// Flags for budget set table
	budgetSetTableCmd.Flags().IntVar(&budgetMaxSeq, "max-seq-rows", 0, "Max sequential scan rows")
	budgetSetTableCmd.Flags().StringSliceVar(&budgetRequire, "require-index", nil, "Required index columns")
	budgetSetTableCmd.Flags().StringVar(&budgetThreshold, "threshold", "", "Max execution time")
	budgetSetTableCmd.Flags().StringVar(&budgetMessage, "message", "", "Policy explanation")

	// Flags for budget set migration
	budgetSetMigrationCmd.Flags().StringVar(&budgetThreshold, "threshold", "", "Max execution time")
	budgetSetMigrationCmd.Flags().StringVar(&budgetMessage, "message", "", "Policy explanation")
}

// findBudgetsRoot searches from the current working directory upward.
func findBudgetsRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}

	for {
		check := filepath.Join(abs, ".basemake", "budgets.json")
		if _, err := os.Stat(check); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("no .basemake/budgets.json found")
		}
		abs = parent
	}
}

// saveBudgets writes the budget file, keeping the existing structure.
func saveBudgets(dir string, bf *budget.BudgetFile) error {
	path := filepath.Join(dir, ".basemake", "budgets.json")

	// Pretty-print with indentation
	data, err := json.MarshalIndent(bf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// describeRule returns a human-readable summary of a budget rule.
func describeRule(r budget.BudgetRule) string {
	parts := []string{}
	switch r.Type {
	case budget.RuleTable:
		parts = append(parts, fmt.Sprintf("table=%s", r.Table))
		if r.MaxSeqRows > 0 {
			parts = append(parts, fmt.Sprintf("max_seq_rows=%d", r.MaxSeqRows))
		}
		if len(r.RequireIndex) > 0 {
			parts = append(parts, fmt.Sprintf("require_index=[%s]", strings.Join(r.RequireIndex, ", ")))
		}
	case budget.RuleMigration:
		parts = append(parts, fmt.Sprintf("pattern=%s", r.Pattern))
	case budget.RuleQuery:
		parts = append(parts, fmt.Sprintf("pattern=%s", r.Pattern))
	}
	if r.Threshold != "" {
		parts = append(parts, fmt.Sprintf("threshold=%s", r.Threshold))
	}
	if r.Message != "" {
		parts = append(parts, fmt.Sprintf("msg=%s", r.Message))
	}
	return strings.Join(parts, ", ")
}
