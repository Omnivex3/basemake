package cmd

import (
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
