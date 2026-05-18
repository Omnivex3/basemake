package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Info mirrors db.TableInfo for the cached schema
type Info struct {
	DBName  string     `json:"db_name"`
	Tables  []Table    `json:"tables"`
	Version string     `json:"version"`
}

type Table struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
	Indexes []Index  `json:"indexes"`
}

type Column struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsPK       bool   `json:"is_pk"`
	IsNullable bool   `json:"is_nullable"`
	Default    string `json:"default,omitempty"`
}

type Index struct {
	Name   string   `json:"name"`
	Unique bool     `json:"unique"`
	Cols   []string `json:"cols"`
}

type Schema struct {
	info Info
}

func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dbai")
}

func cachePath() string {
	return filepath.Join(cacheDir(), "schema.json")
}

// Load reads the cached schema from disk
func Load() (*Schema, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, fmt.Errorf("no cached schema — run 'dbai connect' first: %w", err)
	}

	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse schema cache: %w", err)
	}

	return &Schema{info: info}, nil
}

// SchemaForPrompt returns a compact schema description for AI prompts
func (s *Schema) SchemaForPrompt() string {
	result := fmt.Sprintf("Database: %s\n\nTables:\n", s.info.DBName)
	for _, t := range s.info.Tables {
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
				result += fmt.Sprintf("      - %s on (%s)%s\n", idx.Name, joinCols(idx.Cols), u)
			}
		}
	}
	return result
}

func joinCols(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
