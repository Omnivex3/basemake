package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ─── Recommendation Persistence ─────────────────────────────────────────────

// Recommendation wraps an IndexSuggestion with metadata for persistence.
type Recommendation struct {
	ID         string          `json:"id"`
	Suggestion IndexSuggestion `json:"suggestion"`
	Status     string          `json:"status"` // "pending", "applied", "dismissed"
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ApplyCount int             `json:"apply_count,omitempty"`
	Notes      string          `json:"notes,omitempty"`
}

// RecStore is the persisted list of recommendations.
type RecStore struct {
	Recommendations []Recommendation `json:"recommendations"`
	LastAnalyzed    time.Time        `json:"last_analyzed"` // when analyze --all was last run
	LastTable       string           `json:"last_table,omitempty"`
}

func recStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake", "recommendations.json")
}

// LoadRecs loads recommendations from disk, returning an empty store if not found.
func LoadRecs() (*RecStore, error) {
	path := recStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RecStore{}, nil
		}
		return nil, fmt.Errorf("read recommendations: %w", err)
	}
	var store RecStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parse recommendations: %w", err)
	}
	return &store, nil
}

// Save persists the recommendation store to disk.
func (s *RecStore) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal recommendations: %w", err)
	}
	path := recStorePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write recommendations: %w", err)
	}
	return nil
}

// Merge adds new suggestions, updating existing ones by table+column+partial match.
func (s *RecStore) Merge(suggestions []IndexSuggestion) {
	now := time.Now()
	s.LastAnalyzed = now

	for _, sug := range suggestions {
		// Generate a stable ID
		id := sug.Table + "_" + sug.Columns[0]
		if sug.PartialWhere != "" {
			id += "_partial"
		}

		// Check if already exists
		found := false
		for i, rec := range s.Recommendations {
			if rec.ID == id {
				// Update: refresh suggestion data, keep status unless it was applied
				if rec.Status != "applied" {
					s.Recommendations[i].Suggestion = sug
					s.Recommendations[i].UpdatedAt = now
					s.Recommendations[i].Status = "pending"
				}
				found = true
				break
			}
		}

		if !found {
			s.Recommendations = append(s.Recommendations, Recommendation{
				ID:         id,
				Suggestion: sug,
				Status:     "pending",
				CreatedAt:  now,
				UpdatedAt:  now,
			})
		}
	}
}

// Apply marks a recommendation as applied.
func (s *RecStore) Apply(id string) error {
	for i, rec := range s.Recommendations {
		if rec.ID == id {
			if rec.Status == "applied" {
				return fmt.Errorf("recommendation %q already applied", id)
			}
			s.Recommendations[i].Status = "applied"
			s.Recommendations[i].ApplyCount++
			s.Recommendations[i].UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("recommendation %q not found", id)
}

// Dismiss marks a recommendation as dismissed.
func (s *RecStore) Dismiss(id string) error {
	for i, rec := range s.Recommendations {
		if rec.ID == id {
			if rec.Status == "applied" {
				return fmt.Errorf("cannot dismiss an applied recommendation")
			}
			s.Recommendations[i].Status = "dismissed"
			s.Recommendations[i].UpdatedAt = time.Now()
			return s.Save()
		}
	}
	return fmt.Errorf("recommendation %q not found", id)
}

// StaleReport returns recommendations that are > N days old and still pending.
func (s *RecStore) StaleReport(days int) []Recommendation {
	var stale []Recommendation
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, rec := range s.Recommendations {
		if rec.Status == "pending" && rec.CreatedAt.Before(cutoff) {
			stale = append(stale, rec)
		}
	}
	return stale
}

// Pending returns all pending (non-dismissed, non-applied) recommendations.
func (s *RecStore) Pending() []Recommendation {
	var pending []Recommendation
	for _, rec := range s.Recommendations {
		if rec.Status == "pending" {
			pending = append(pending, rec)
		}
	}
	return pending
}

// ByTable returns recommendations filtered by table name.
func (s *RecStore) ByTable(table string) []Recommendation {
	var filtered []Recommendation
	for _, rec := range s.Recommendations {
		if rec.Suggestion.Table == table {
			filtered = append(filtered, rec)
		}
	}
	return filtered
}
