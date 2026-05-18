package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// SchemaForPrompt returns a compact schema description for AI prompts
func (s *Schema) SchemaForPrompt() string {
	result := fmt.Sprintf("Database: %s\n\nTables:\n", s.DBName)
	for _, t := range s.Tables {
		result += fmt.Sprintf("  %s:\n", t.Name)
		for _, c := range t.Columns {
			pk := ""
			if c.IsPK {
				pk = " [PK]"
			}
			nullable := ""
			if c.IsNullable {
				nullable = " nullable"
			}
			result += fmt.Sprintf("    - %s %s%s%s\n", c.Name, c.Type, pk, nullable)
		}
		if len(t.Indexes) > 0 {
			result += "    Indexes:\n"
			for _, idx := range t.Indexes {
				u := ""
				if idx.Unique {
					u = " (unique)"
				}
				result += fmt.Sprintf("      - %s on (%s)%s\n", idx.Name, strings.Join(idx.Cols, ", "), u)
			}
		}
	}
	return result
}

func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake")
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

// LoadSchema reads the cached schema from disk
func LoadSchema() (*Schema, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, fmt.Errorf("no cached schema — run 'basemake connect' first: %w", err)
	}

	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema cache: %w", err)
	}

	return &s, nil
}

// Save persists the schema to local cache
func (s *Schema) Save() error {
	return SaveSchema(s)
}
