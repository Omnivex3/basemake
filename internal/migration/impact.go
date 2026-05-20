package migration

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/profile"
)

// AffectedQuery describes one profiled query that would be impacted by a DDL change.
type AffectedQuery struct {
	NormalizedSQL       string
	AvgDurationMs       int64
	LastDurationMs      int64
	RunCount            int
	CurrentScanType     string // e.g., "Index Scan"
	CurrentIndexName    string // e.g., "idx_status"
	TableName           string // e.g., "orders"
	PlanRows            float64
	EstimatedWithoutMs  int64
	Risk                string // "HIGH" or "MEDIUM"
}

// ImpactResult groups affected queries by the DDL change that impacts them.
type ImpactResult struct {
	Change          DDLChange
	AffectedQueries []AffectedQuery
	TotalAffected   int
	HighRiskCount   int
	IsPositive      bool // true for CREATE INDEX (no negative impact)
}

// estRowCost is the estimated ms per row for a sequential scan.
// Used when no stored Seq Scan plan exists for the affected table.
const estRowCost = 0.0015 // 1.5ms per 1000 rows

// Risk thresholds.
const (
	highRunCountThreshold    = 100
	highLatencyMultiplier = 10.0
)

// AnalyzeImpact cross-references DDL changes against all stored query profiles
// and returns the estimated impact for each change.
func AnalyzeImpact(changes []DDLChange, profileDir string) ([]ImpactResult, error) {
	// Load all profiles from disk
	allProfiles, err := loadProfileFiles(profileDir)
	if err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}

	if len(allProfiles) == 0 {
		return nil, nil
	}

	// Build a map of table_name → max PlanRows seen across all profiles.
	// This gives us the estimated full table size for Seq Scan estimation.
	tableMaxRows := buildTableMaxRows(allProfiles)

	var results []ImpactResult

	for _, ch := range changes {
		switch ch.Type {
		case ChangeDropIndex:
			result := analyzeDropIndex(ch, allProfiles, tableMaxRows)
			if result != nil && len(result.AffectedQueries) > 0 {
				results = append(results, *result)
			}
		case ChangeDropTable:
			result := analyzeDropTable(ch, allProfiles)
			if result != nil && len(result.AffectedQueries) > 0 {
				results = append(results, *result)
			}
		case ChangeDropColumn, ChangeAlterColumnType:
			result := analyzeDropColumn(ch, allProfiles, tableMaxRows)
			if result != nil && len(result.AffectedQueries) > 0 {
				results = append(results, *result)
			}
		case ChangeCreateIndex:
			// Positive change — just note it
			results = append(results, ImpactResult{
				Change:     ch,
				IsPositive: true,
			})
		}
	}

	return results, nil
}

// --- Profile loading ---

// loadedProfile pairs a profile with its filename for reference.
type loadedProfile struct {
	Hash    string
	Profile *profile.QueryProfile
}

func loadProfileFiles(dir string) ([]loadedProfile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var profiles []loadedProfile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		hash := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue // skip corrupted files
		}
		var p profile.QueryProfile
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		if len(p.Runs) == 0 {
			continue
		}
		profiles = append(profiles, loadedProfile{
			Hash:    hash,
			Profile: &p,
		})
	}

	return profiles, nil
}

// buildTableMaxRows iterates all profiles and records the maximum PlanRows seen
// per table. This gives us an estimate of full table size for Seq Scan costing.
func buildTableMaxRows(profiles []loadedProfile) map[string]float64 {
	maxRows := make(map[string]float64)

	for _, lp := range profiles {
		latest := lp.Profile.Runs[len(lp.Profile.Runs)-1]
		if latest.PlanText == "" {
			continue
		}
		nodes, err := profile.ExtractPlanNodes(latest.PlanText)
		if err != nil {
			continue
		}
		for _, n := range nodes {
			if n.RelationName != "" && n.PlanRows > maxRows[n.RelationName] {
				maxRows[n.RelationName] = n.PlanRows
			}
		}
	}

	return maxRows
}

// --- Analysis functions ---

func analyzeDropIndex(ch DDLChange, profiles []loadedProfile, tableMaxRows map[string]float64) *ImpactResult {
	result := &ImpactResult{Change: ch}

	for _, lp := range profiles {
		latest := lp.Profile.Runs[len(lp.Profile.Runs)-1]
		if latest.PlanText == "" {
			continue
		}

		nodes, err := profile.ExtractPlanNodes(latest.PlanText)
		if err != nil {
			continue
		}

		// Check if any node references the dropped index
		for _, n := range nodes {
			if n.IndexName == ch.Target {
				aq := buildAffectedQuery(latest, n, tableMaxRows, lp.Profile.Runs)
				result.AffectedQueries = append(result.AffectedQueries, aq)
				break // one match per query is enough
			}
		}
	}

	if len(result.AffectedQueries) == 0 {
		return nil
	}

	result.TotalAffected = len(result.AffectedQueries)
	for _, aq := range result.AffectedQueries {
		if aq.Risk == "HIGH" {
			result.HighRiskCount++
		}
	}
	return result
}

func analyzeDropTable(ch DDLChange, profiles []loadedProfile) *ImpactResult {
	result := &ImpactResult{Change: ch}

	for _, lp := range profiles {
		latest := lp.Profile.Runs[len(lp.Profile.Runs)-1]
		if latest.PlanText == "" {
			continue
		}

		nodes, err := profile.ExtractPlanNodes(latest.PlanText)
		if err != nil {
			continue
		}

		for _, n := range nodes {
			if n.RelationName == ch.Target {
				// For DROP TABLE we don't have a meaningful "without" estimate — the
				// query will fail. Mark as HIGH risk automatically.
				aq := AffectedQuery{
					NormalizedSQL:  latest.NormalizedSQL,
					AvgDurationMs:  avgDuration(lp.Profile.Runs),
					LastDurationMs: latest.DurationMS,
					RunCount:       len(lp.Profile.Runs),
					TableName:      n.RelationName,
					CurrentScanType: n.NodeType,
					Risk:           "HIGH",
				}
				result.AffectedQueries = append(result.AffectedQueries, aq)
				break
			}
		}
	}

	if len(result.AffectedQueries) == 0 {
		return nil
	}

	result.TotalAffected = len(result.AffectedQueries)
	result.HighRiskCount = len(result.AffectedQueries) // all HIGH
	return result
}

func analyzeDropColumn(ch DDLChange, profiles []loadedProfile, tableMaxRows map[string]float64) *ImpactResult {
	result := &ImpactResult{Change: ch}

	for _, lp := range profiles {
		latest := lp.Profile.Runs[len(lp.Profile.Runs)-1]
		if latest.PlanText == "" {
			continue
		}

		nodes, err := profile.ExtractPlanNodes(latest.PlanText)
		if err != nil {
			continue
		}

		// Check if any node references the affected table
		for _, n := range nodes {
			if n.RelationName == ch.Table {
				// The query touches this table — column drop could break it
				// For V0, any query touching the table is flagged. A more precise
				// approach would check if the column is referenced in the query SQL.
				aq := buildAffectedQuery(latest, n, tableMaxRows, lp.Profile.Runs)
				result.AffectedQueries = append(result.AffectedQueries, aq)
				break
			}
		}
	}

	if len(result.AffectedQueries) == 0 {
		return nil
	}

	result.TotalAffected = len(result.AffectedQueries)
	for _, aq := range result.AffectedQueries {
		if aq.Risk == "HIGH" {
			result.HighRiskCount++
		}
	}
	return result
}

// --- Helper functions ---

func buildAffectedQuery(latest profile.QueryRun, node profile.PlanNode, tableMaxRows map[string]float64, runs []profile.QueryRun) AffectedQuery {
	avgMs := avgDuration(runs)

	estimatedWithout := estimateLatencyWithoutIndex(node, tableMaxRows)
	risk := computeRisk(avgMs, estimatedWithout, len(runs))

	scanType := node.NodeType
	if scanType == "" {
		scanType = "unknown"
	}

	return AffectedQuery{
		NormalizedSQL:      latest.NormalizedSQL,
		AvgDurationMs:      avgMs,
		LastDurationMs:     latest.DurationMS,
		RunCount:           len(runs),
		CurrentScanType:    scanType,
		CurrentIndexName:   node.IndexName,
		TableName:          node.RelationName,
		PlanRows:           node.PlanRows,
		EstimatedWithoutMs: estimatedWithout,
		Risk:               risk,
	}
}

func avgDuration(runs []profile.QueryRun) int64 {
	if len(runs) == 0 {
		return 0
	}
	var total int64
	for _, r := range runs {
		total += r.DurationMS
	}
	return total / int64(len(runs))
}

// estimateLatencyWithoutIndex estimates the query execution time (ms) if the
// index were removed and a sequential scan were used instead.
func estimateLatencyWithoutIndex(node profile.PlanNode, tableMaxRows map[string]float64) int64 {
	maxRows, ok := tableMaxRows[node.RelationName]
	if !ok || maxRows == 0 {
		// Fallback: use the node's own PlanRows
		maxRows = node.PlanRows
	}
	if maxRows <= 0 {
		maxRows = 1
	}
	return int64(math.Ceil(maxRows * estRowCost))
}

// computeRisk scores the impact as HIGH or MEDIUM based on run count and
// the estimated latency multiplier relative to current average duration.
func computeRisk(avgDurationMs, estimatedWithoutMs int64, runCount int) string {
	if runCount > highRunCountThreshold {
		return "HIGH"
	}
	if avgDurationMs > 0 && estimatedWithoutMs > 0 {
		multiplier := float64(estimatedWithoutMs) / float64(avgDurationMs)
		if multiplier > highLatencyMultiplier {
			return "HIGH"
		}
	}
	return "MEDIUM"
}

// --- Output formatting ---

// FormatImpactReport produces the exact user-facing output for migration impact.
func FormatImpactReport(results []ImpactResult, profileCount int, profileDir string) string {
	var b strings.Builder

	if profileCount > 0 {
		fmt.Fprintf(&b, "Analyzing migration against %d profiled queries...\n\n", profileCount)
	}

	anyNegative := false

	for _, r := range results {
		if r.IsPositive {
			fmt.Fprintf(&b, "CREATE INDEX %s on %s\n", r.Change.Target, r.Change.Table)
			fmt.Fprintf(&b, "  ✅ This should improve query performance — no risk\n\n")
			continue
		}

		anyNegative = true

		fmt.Fprintf(&b, "%s\n", r.Change.String())
		fmt.Fprintf(&b, "  ⚠️ %d %s affected\n",
			r.TotalAffected,
			pluralQuery(r.TotalAffected),
		)

		for _, aq := range r.AffectedQueries {
			// Query label: use normalized SQL, truncated and capitalized
			label := formatQueryLabel(aq.NormalizedSQL)

			fmt.Fprintf(&b, "\n")
			fmt.Fprintf(&b, "  → %s\n", label)

			// Scan type transition: current → without
			scanLine := fmt.Sprintf("     %s via %s → Seq Scan ",
				aq.CurrentScanType, aq.CurrentIndexName)
			if aq.PlanRows > 0 {
				scanLine += fmt.Sprintf("(~%s rows)", formatNumber(aq.PlanRows))
			}
			fmt.Fprintf(&b, "%s\n", scanLine)

			// Latency comparison
			fmt.Fprintf(&b, "     Avg latency: %s | Estimated: %s | Run count: %d\n",
				formatMs(aq.AvgDurationMs),
				formatMs(aq.EstimatedWithoutMs),
				aq.RunCount,
			)

			// Risk badge
			riskIcon := "MEDIUM"
			if aq.Risk == "HIGH" {
				riskIcon = "HIGH"
			}
			fmt.Fprintf(&b, "     Risk: %s\n", riskIcon)
		}

		// Summary line per change
		if r.HighRiskCount > 0 {
			fmt.Fprintf(&b, "\n  %d %s risk. ",
				r.HighRiskCount,
				pluralChange(r.HighRiskCount),
			)
			fmt.Fprintf(&b, "Run with --approve to proceed anyway.\n")
		}
		fmt.Fprintf(&b, "\n")
	}

	if !anyNegative {
		fmt.Fprintf(&b, "✅ No destructive changes detected that would impact profiled queries.\n")
	}

	return b.String()
}

// HasHighRisk returns true if any result has HIGH risk changes.
func HasHighRisk(results []ImpactResult) bool {
	for _, r := range results {
		if r.HighRiskCount > 0 {
			return true
		}
	}
	return false
}

// formatQueryLabel makes a readable label from normalized SQL.
func formatQueryLabel(normalizedSQL string) string {
	label := strings.TrimSpace(normalizedSQL)
	// Replace multiple spaces
	spaces := regexp.MustCompile(`\s+`)
	label = spaces.ReplaceAllString(label, " ")
	// Uppercase SQL keywords for readability
	label = strings.ToUpper(label[:int(math.Min(float64(len(label)), 1))]) + label[1:]
	if len(label) > 60 {
		label = label[:57] + "..."
	}
	return label
}

func formatMs(ms int64) string {
	switch {
	case ms >= 1000:
		return fmt.Sprintf("%dms", ms)
	case ms >= 1:
		return fmt.Sprintf("%dms", ms)
	default:
		return "<1ms"
	}
}

func formatNumber(n float64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fK", n/1_000)
	default:
		return fmt.Sprintf("%.0f", n)
	}
}

func pluralQuery(n int) string {
	if n == 1 {
		return "query"
	}
	return "queries"
}

func pluralChange(n int) string {
	if n == 1 {
		return "HIGH risk change"
	}
	return "HIGH risk changes"
}

// ProfileDir returns the standard profile directory path.
func ProfileDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".basemake", "profiles")
	}
	return filepath.Join(home, ".basemake", "profiles")
}

// ProfileCount returns the number of profile files in the profile directory.
func ProfileCount(profileDir string) int {
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			count++
		}
	}
	return count
}
