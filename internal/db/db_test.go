package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaJSONRoundTrip(t *testing.T) {
	s := &Schema{
		DBName: "testdb",
		Tables: []TableInfo{
			{
				Name: "users",
				Columns: []ColumnInfo{
					{Name: "id", Type: "integer", IsPK: true},
					{Name: "email", Type: "text", IsNullable: true},
				},
				Indexes: []IndexInfo{
					{Name: "users_pkey", Unique: true, Cols: []string{"id"}},
				},
			},
		},
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Schema
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.DBName != "testdb" {
		t.Errorf("DBName = %q, want %q", got.DBName, "testdb")
	}
	if len(got.Tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(got.Tables))
	}
	if got.Tables[0].Name != "users" {
		t.Errorf("table name = %q, want %q", got.Tables[0].Name, "users")
	}
	if len(got.Tables[0].Columns) != 2 {
		t.Fatalf("got %d columns, want 2", len(got.Tables[0].Columns))
	}
	if !got.Tables[0].Columns[0].IsPK {
		t.Error("id column should be PK")
	}
}

func TestTotalColumnsAndIndexes(t *testing.T) {
	s := &Schema{
		Tables: []TableInfo{
			{
				Name: "a",
				Columns: []ColumnInfo{
					{Name: "id"}, {Name: "name"},
				},
				Indexes: []IndexInfo{
					{Name: "a_pkey", Cols: []string{"id"}},
				},
			},
			{
				Name: "b",
				Columns: []ColumnInfo{
					{Name: "id"}, {Name: "val"},
				},
			},
		},
	}

	if got := s.TotalColumns(); got != 4 {
		t.Errorf("TotalColumns = %d, want 4", got)
	}
	if got := s.TotalIndexes(); got != 1 {
		t.Errorf("TotalIndexes = %d, want 1", got)
	}
}

func TestSchemaForPrompt(t *testing.T) {
	s := &Schema{
		DBName: "mydb",
		Tables: []TableInfo{
			{
				Name: "users",
				Columns: []ColumnInfo{
					{Name: "id", Type: "integer", IsPK: true},
					{Name: "name", Type: "text", IsNullable: true},
				},
				Indexes: []IndexInfo{
					{Name: "users_pkey", Unique: true, Cols: []string{"id"}},
				},
			},
		},
	}

	prompt := s.SchemaForPrompt()

	if !contains(prompt, "Database: mydb") {
		t.Error("missing database name in prompt")
	}
	if !contains(prompt, "users:") {
		t.Error("missing table name in prompt")
	}
	if !contains(prompt, "[PK]") {
		t.Error("missing PK marker in prompt")
	}
	if !contains(prompt, "(unique)") {
		t.Error("missing unique marker in prompt")
	}
}

func TestSaveAndLoadSchema(t *testing.T) {
	tmp := t.TempDir()

	// Override cache dir by setting HOME
	t.Setenv("HOME", tmp)

	s := &Schema{
		DBName: "test",
		Tables: []TableInfo{
			{Name: "t1", Columns: []ColumnInfo{{Name: "id", Type: "int"}}},
		},
	}

	if err := SaveSchema(s); err != nil {
		t.Fatalf("SaveSchema: %v", err)
	}

	// Verify file exists
	cacheFile := filepath.Join(tmp, ".basemake", "schema.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatalf("cache file not created at %s", cacheFile)
	}

	loaded, err := LoadSchema()
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	if loaded.DBName != "test" {
		t.Errorf("DBName = %q, want %q", loaded.DBName, "test")
	}
	if len(loaded.Tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(loaded.Tables))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
