package observe

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/DynamicKarabo/basemake/internal/profile"
)

const stateFile = "observe_state.json"
const schemaFile = "schema.json"

// observeState tracks what we've already reported so we don't repeat.
type observeState struct {
	LastObservedAt int64  `json:"last_observed_at"` // unix timestamp of last report
	SchemaHash     string `json:"schema_hash"`      // hash of last seen schema.json
}

// Brief checks local profile + schema cache for interesting signals and returns
// a single-line-or-so observation. Returns empty string when there's nothing
// worth reporting. Never makes live database calls — reads only cached state.
func Brief() string {
	st := loadState()
	reported := false
	defer func() {
		if reported {
			st.LastObservedAt = time.Now().Unix()
			saveState(st)
		}
	}()

	// Priority 1: plan changes (most urgent)
	if msg := checkPlanChanges(st); msg != "" {
		reported = true
		return msg
	}

	// Priority 2: slow queries (2x+ slower than average)
	if msg := checkSlowQueries(st); msg != "" {
		reported = true
		return msg
	}

	// Priority 3: schema drift (new tables, dropped columns)
	if msg := checkSchemaDrift(&st); msg != "" {
		reported = true
		return msg
	}

	// Schema hash was stored silently (first run or update without signal)
	saveState(st)
	return ""
}

// ── state ──

func statePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".basemake", stateFile)
	}
	return filepath.Join(home, ".basemake", stateFile)
}

func loadState() observeState {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return observeState{}
	}
	var st observeState
	json.Unmarshal(data, &st)
	return st
}

func saveState(st observeState) {
	dir := filepath.Dir(statePath())
	os.MkdirAll(dir, 0755)
	data, _ := json.Marshal(st)
	os.WriteFile(statePath(), data, 0644)
}

// ── signal: plan changes ──

// checkPlanChanges scans all profiles for a query that changed plans since
// the last observation. Returns a human-readable brief or empty string.
func checkPlanChanges(st observeState) string {
	profiles := loadProfiles()
	if len(profiles) == 0 {
		return ""
	}

	// Sort by newest run first
	sort.Slice(profiles, func(i, j int) bool {
		pi := profiles[i].Runs[len(profiles[i].Runs)-1]
		pj := profiles[j].Runs[len(profiles[j].Runs)-1]
		return pi.Timestamp.After(pj.Timestamp)
	})

	for _, p := range profiles {
		if len(p.Runs) < 2 {
			continue
		}
		latest := p.Runs[len(p.Runs)-1]
		prev := p.Runs[len(p.Runs)-2]

		// Only report if the change is new (after last observation)
		if latest.Timestamp.Unix() <= st.LastObservedAt {
			continue
		}
		if latest.PlanHash == "" || latest.PlanHash == prev.PlanHash {
			continue
		}

		// Attempt detailed node comparison
		oldNodes, err := profile.ExtractPlanNodes(prev.PlanText)
		if err != nil {
			// Fallback: simple hash mismatch
			return "⚠ Query plan changed (" + truncateSQL(latest.NormalizedSQL) + ")"
		}
		newNodes, err := profile.ExtractPlanNodes(latest.PlanText)
		if err != nil {
			continue
		}

		changes := profile.ComparePlans(oldNodes, newNodes)
		if len(changes) > 0 {
			return "⚠ Plan changed: " + profile.ExplainChange(changes[0])
		}
		return "⚠ Query plan changed (" + truncateSQL(latest.NormalizedSQL) + ")"
	}

	return ""
}

// ── signal: slow queries ──

// checkSlowQueries finds queries whose latest run is 2x+ slower than their
// historical average. Returns a brief or empty string.
func checkSlowQueries(st observeState) string {
	profiles := loadProfiles()
	if len(profiles) == 0 {
		return ""
	}

	var best struct {
		ratio   float64
		message string
	}

	for _, p := range profiles {
		if len(p.Runs) < 3 {
			continue // need at least 2 previous runs for a meaningful average
		}
		latest := p.Runs[len(p.Runs)-1]

		if latest.Timestamp.Unix() <= st.LastObservedAt {
			continue
		}
		if latest.DurationMS <= 0 {
			continue
		}

		var total int64
		for _, run := range p.Runs[:len(p.Runs)-1] {
			total += run.DurationMS
		}
		avg := total / int64(len(p.Runs)-1)
		if avg <= 0 {
			continue
		}

		ratio := float64(latest.DurationMS) / float64(avg)
		if ratio >= 2.0 && ratio > best.ratio {
			best.ratio = ratio
			best.message = fmt.Sprintf(
				"⚠ %.1fx slower: %s (%dms vs %dms avg)",
				ratio, truncateSQL(latest.NormalizedSQL), latest.DurationMS, avg,
			)
		}
	}

	return best.message
}

// ── signal: schema drift ──

// checkSchemaDrift compares the cached schema against the last seen state.
// Silently stores the schema hash on first run.
func checkSchemaDrift(st *observeState) string {
	schemaPath := schemaCachePath()
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return ""
	}

	h := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", h[:16])

	if st.SchemaHash == "" {
		// First time — store silently
		st.SchemaHash = hash
		return ""
	}

	if st.SchemaHash != hash {
		st.SchemaHash = hash

		// Attempt a lightweight comparison
		var current, prev struct {
			Tables []struct {
				Name    string `json:"name"`
				Columns []struct {
					Name string `json:"name"`
				} `json:"columns"`
			} `json:"tables"`
		}
		if err := json.Unmarshal(data, &current); err != nil {
			return "⚡ Schema changed"
		}

		schemaPath := schemaCachePath() + ".bak"
		prevData, err := os.ReadFile(schemaPath)
		if err == nil {
			json.Unmarshal(prevData, &prev)
		}

		if diff := diffSchema(&prev, &current); diff != "" {
			return "⚡ " + diff
		}
		return "⚡ Schema changed"
	}

	return ""
}

func diffSchema(prev, current interface{}) string {
	// Placeholder — returns empty for now since we don't reliably have
	// the previous schema on disk. The hash comparison is the primary signal.
	// A proper diff can be added later (parse both schemas, compare tables/columns).
	return ""
}

// ── helpers ──

func loadProfiles() []*profile.QueryProfile {
	dir := profile.ProfileDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var profiles []*profile.QueryProfile
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		hash := entry.Name()[:len(entry.Name())-5]
		p, err := profile.Load(hash)
		if err != nil || len(p.Runs) == 0 {
			continue
		}
		profiles = append(profiles, p)
	}
	return profiles
}

func schemaCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".basemake", schemaFile)
	}
	return filepath.Join(home, ".basemake", schemaFile)
}

// truncateSQL shortens a normalized SQL string for display.
func truncateSQL(s string) string {
	if len(s) <= 60 {
		return s
	}
	return s[:57] + "..."
}
