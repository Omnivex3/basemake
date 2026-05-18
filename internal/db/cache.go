package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TotalColumns returns the total number of columns across all tables
func (s *Schema) TotalColumns() int {
	count := 0
	for _, t := range s.Tables {
		count += len(t.Columns)
	}
	return count
}

// TotalIndexes returns the total number of indexes across all tables
func (s *Schema) TotalIndexes() int {
	count := 0
	for _, t := range s.Tables {
		count += len(t.Indexes)
	}
	return count
}

func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dbai")
}

func cachePath() string {
	return filepath.Join(cacheDir(), "schema.json")
}

// SaveSchema persists the schema to a local JSON cache
func SaveSchema(s *Schema) error {
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema: %w", err)
	}

	if err := os.WriteFile(cachePath(), data, 0644); err != nil {
		return fmt.Errorf("write schema cache: %w", err)
	}

	return nil
}
