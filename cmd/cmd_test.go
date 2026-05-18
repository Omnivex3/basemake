package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestLooksLikeSQL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"SELECT * FROM users", true},
		{"select * from users", true},
		{"SELECT id, name FROM orders WHERE created_at > now()", true},
		{"WITH recent AS (SELECT * FROM orders) SELECT * FROM recent", true},
		{"EXPLAIN ANALYZE SELECT * FROM users", true},
		{"INSERT INTO users (name) VALUES ('alice')", true},
		{"UPDATE users SET name = 'bob' WHERE id = 1", true},
		{"DELETE FROM users WHERE id = 1", true},
		{"CREATE TABLE foo (id int)", true},
		{"ALTER TABLE users ADD COLUMN age int", true},
		{"DROP TABLE users", true},
		{"TRUNCATE users", true},
		{"show me users who ordered last month", false},
		{"what are the top 10 products?", false},
		{"how many active users do we have?", false},
		{"", false},
		{"   SELECT id FROM users", true},
		{"\n\tSELECT * FROM t", true},
	}

	for _, tc := range tests {
		got := looksLikeSQL(tc.input)
		if got != tc.want {
			t.Errorf("looksLikeSQL(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDryRunWithSQLInput(t *testing.T) {
	// When --dry-run is set and input is raw SQL,
	// the command should print the SQL and exit without connecting to a DB.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	input := "SELECT 1 AS test"
	rootCmd.SetArgs([]string{"query", "--dry-run", input})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != input {
		t.Errorf("dry-run output = %q, want %q", output, input)
	}

	// stderr should be empty (no connection, no schema load for raw SQL)
	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr: %s", stderr.String())
	}
}

func TestDryRunWithNLInput(t *testing.T) {
	// With --dry-run and NL input, the AI generates SQL but doesn't connect.
	// We just verify no panic/error happens and the output is non-empty SQL.
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	rootCmd.SetArgs([]string{"query", "--dry-run", "show all tables"})

	err := rootCmd.Execute()
	// Expect either success (schema cached or AI placeholder) or "no cached schema"
	if err != nil {
		if !strings.Contains(err.Error(), "no cached schema") {
			t.Fatalf("unexpected error: %v", err)
		}
		// Expected error about missing schema cache
		return
	}

	// If it succeeded, verify SQL output
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		t.Error("expected non-empty SQL output from dry-run")
	}
}
