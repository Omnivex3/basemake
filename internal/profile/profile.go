package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const profilesDir = "profiles"

// QueryRun captures a single execution of a query with its plan and timing.
type QueryRun struct {
	Hash          string    `json:"hash"`
	NormalizedSQL string    `json:"normalized_sql"`
	Timestamp     time.Time `json:"timestamp"`
	DurationMS    int64     `json:"duration_ms"`
	RowsReturned  int64     `json:"rows_returned"`
	PlanText      string    `json:"plan_text"`
	PlanHash      string    `json:"plan_hash"`
	DBFingerprint string    `json:"db_fingerprint"`
}

// QueryProfile stores the history of runs for a single normalized query
// on a specific database.
type QueryProfile struct {
	Runs []QueryRun `json:"runs"`
}

// ProfileDir returns ~/.basemake/profiles/
func ProfileDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".basemake", profilesDir)
	}
	return filepath.Join(home, ".basemake", profilesDir)
}

// ProfilePath returns the file path for a given query hash.
func ProfilePath(hash string) string {
	return filepath.Join(ProfileDir(), hash+".json")
}

// Load retrieves the stored profile for a query hash. Returns an empty
// profile if none exists yet.
func Load(hash string) (*QueryProfile, error) {
	data, err := os.ReadFile(ProfilePath(hash))
	if err != nil {
		if os.IsNotExist(err) {
			return &QueryProfile{Runs: []QueryRun{}}, nil
		}
		return nil, err
	}
	var p QueryProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Save persists the profile to disk.
func Save(hash string, p *QueryProfile) error {
	dir := ProfileDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ProfilePath(hash), data, 0644)
}

// CompareResult holds the user-facing comparison output after profiling a query.
type CompareResult struct {
	RunCount      int
	AvgDurationMS int64
	CurrentRun    *QueryRun
	PreviousRun   *QueryRun
	TimingDelta   string // e.g. "+131% vs avg (124ms)"
	PlanChanged   bool
	Changes       []PlanChange
}

// Compare stores a new query run, compares it against the profile history,
// and returns the comparison. It also persists the new run.
func Compare(hash string, newRun QueryRun) (*CompareResult, error) {
	p, err := Load(hash)
	if err != nil {
		return nil, err
	}

	// Append new run
	p.Runs = append(p.Runs, newRun)
	if err := Save(hash, p); err != nil {
		return nil, err
	}

	r := &CompareResult{
		RunCount:   len(p.Runs),
		CurrentRun: &newRun,
	}

	if len(p.Runs) < 2 {
		return r, nil
	}

	// Previous run is the immediately preceding one
	prev := p.Runs[len(p.Runs)-2]
	r.PreviousRun = &prev

	// Timing: calculate average excluding current run
	var total int64
	for _, run := range p.Runs[:len(p.Runs)-1] {
		total += run.DurationMS
	}
	avg := total / int64(len(p.Runs)-1)
	r.AvgDurationMS = avg

	if avg > 0 {
		delta := float64(newRun.DurationMS-avg) / float64(avg) * 100
		switch {
		case delta > 5:
			r.TimingDelta = fmt.Sprintf("+%.0f%% vs avg (%dms)", delta, avg)
		case delta < -5:
			r.TimingDelta = fmt.Sprintf("%.0f%% vs avg (%dms)", delta, avg)
		default:
			r.TimingDelta = fmt.Sprintf("~%.0f%% vs avg (%dms)", delta, avg)
		}
	}

	// Plan comparison against previous run
	if prev.PlanHash != "" && prev.PlanHash != newRun.PlanHash {
		r.PlanChanged = true
		oldNodes, err := ExtractPlanNodes(prev.PlanText)
		if err != nil {
			// Plan text wasn't valid JSON (SQLite, etc.) — just report hash mismatch
			r.Changes = []PlanChange{{
				OldNodeType:  "different plan",
				RelationName: "",
			}}
			return r, nil
		}
		newNodes, err := ExtractPlanNodes(newRun.PlanText)
		if err != nil {
			return r, nil
		}
		r.Changes = ComparePlans(oldNodes, newNodes)
	}

	return r, nil
}

// FormatComparison renders the comparison result as a short multi-line string
// suitable for printing to stderr.
func FormatComparison(r *CompareResult) string {
	if r == nil {
		return ""
	}

	var b strings.Builder

	// Line 1: summary
	avgStr := ""
	if r.AvgDurationMS > 0 {
		avgStr = fmt.Sprintf(" Avg: %dms.", r.AvgDurationMS)
	}
	runLabel := "times"
	if r.RunCount == 1 {
		runLabel = "time"
	}
	fmt.Fprintf(&b, "⚡ Profiled %d %s.%s", r.RunCount, runLabel, avgStr)

	if r.TimingDelta != "" {
		b.WriteString(" This run: ")
		b.WriteString(r.TimingDelta)
	}
	b.WriteString("\n")

	// Line 2: previous run timestamp
	if r.PreviousRun != nil {
		b.WriteString("   Last run: ")
		b.WriteString(r.PreviousRun.Timestamp.Format("Mon 15:04"))
		b.WriteString("\n")
	}

	// Line 3+: plan changes
	if r.PlanChanged {
		b.WriteString("   ⚠ Plan changed:\n")
		for _, c := range r.Changes {
			b.WriteString("     → ")
			b.WriteString(ExplainChange(c))
			b.WriteString("\n")
		}
	}

	return b.String()
}
