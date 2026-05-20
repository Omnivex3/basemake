package profile

import (
	"context"
	"fmt"

	"github.com/DynamicKarabo/basemake/internal/db"
)

// Warning describes a concern about a query before it executes.
type Warning struct {
	Severity string // "warn" | "info"
	Message  string // one line, plain English
}

// PlanCheck compares a query's current execution plan against its profile
// history and returns warnings about regressions. Fast and silent on the
// happy path — no warnings means the plan is stable and healthy.
//
// Checks in priority order:
//  1. Index was dropped since last profile (warn)
//  2. Plan changed to Seq Scan (warn, skipped if #1 already caught it)
//  3. Last run was 2x+ slower than historical average (info)
//
// PlanCheck runs ExplainNoAnalyze to get the current plan without executing
// the query. It does NOT save the plan to the profile — that's the caller's
// responsibility during execution.
func PlanCheck(ctx context.Context, sql string, conn db.Database) []Warning {
	normSQL := NormalizeSQL(sql)
	hash := QueryHash(normSQL)
	p, err := Load(hash)
	if err != nil || len(p.Runs) == 0 {
		return nil // No history to compare against
	}

	prev := p.Runs[len(p.Runs)-1]

	// Get current plan (non-executing)
	planJSON, err := conn.ExplainNoAnalyze(ctx, sql)
	if err != nil {
		return nil // Can't check — plan unavailable
	}
	currentHash := PlanHash(planJSON)

	var warnings []Warning

	// Checks 1 & 2: plan structure changed
	if prev.PlanHash != "" && prev.PlanHash != currentHash {
		oldNodes, err1 := ExtractPlanNodes(prev.PlanText)
		newNodes, err2 := ExtractPlanNodes(planJSON)
		if err1 == nil && err2 == nil {
			changes := ComparePlans(oldNodes, newNodes)
			for _, c := range changes {
				// Check 1: index was dropped — highest severity
				if isIndexScan(c.OldNodeType) && c.OldIndexName != "" {
					warnings = append(warnings, Warning{
						Severity: "warn",
						Message:  fmt.Sprintf("%s was dropped since the last profile. This query may be slower. Run ANALYZE or recreate the index.", c.OldIndexName),
					})
					continue
				}
				// Check 2: plan changed to Seq Scan (not already covered by #1)
				if c.NewNodeType == "Seq Scan" {
					warnings = append(warnings, Warning{
						Severity: "warn",
						Message:  fmt.Sprintf("Plan changed to Seq Scan on %s. Check for missing index.", c.RelationName),
					})
				}
			}
		}
	}

	// Check 3: historical regression (needs 3+ runs for a meaningful average)
	if len(p.Runs) >= 3 {
		latest := p.Runs[len(p.Runs)-1]
		var total int64
		for _, run := range p.Runs[:len(p.Runs)-1] {
			total += run.DurationMS
		}
		avg := total / int64(len(p.Runs)-1)
		if avg > 0 && latest.DurationMS > avg*2 {
			ratio := float64(latest.DurationMS) / float64(avg)
			warnings = append(warnings, Warning{
				Severity: "info",
				Message:  fmt.Sprintf("Last run was %.1fx slower than average (%dms vs %dms avg)", ratio, latest.DurationMS, avg),
			})
		}
	}

	return warnings
}

// HasWarnings returns true if any warnings have "warn" severity.
func HasWarnings(ww []Warning) bool {
	for _, w := range ww {
		if w.Severity == "warn" {
			return true
		}
	}
	return false
}
