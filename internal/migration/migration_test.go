package migration

import (
	"testing"
)

func TestParseMigration_DropIndex(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		want    string // DDLChange.String()
		wantErr bool
	}{
		{
			name: "simple drop index",
			sql:  "DROP INDEX idx_status;",
			want: "DROP INDEX idx_status",
		},
		{
			name: "drop index if exists",
			sql:  "DROP INDEX IF EXISTS idx_status;",
			want: "DROP INDEX idx_status",
		},
		{
			name: "drop index on table (mysql style)",
			sql:  "DROP INDEX idx_status ON orders;",
			want: "DROP INDEX idx_status on orders",
		},
		{
			name: "drop index concurrently",
			sql:  "DROP INDEX CONCURRENTLY idx_status;",
			want: "DROP INDEX idx_status",
		},
		{
			name: "create index",
			sql:  "CREATE INDEX idx_status ON orders (status);",
			want: "CREATE INDEX idx_status on orders",
		},
		{
			name: "create unique index",
			sql:  "CREATE UNIQUE INDEX idx_email ON users (email);",
			want: "CREATE INDEX idx_email on users",
		},
		{
			name: "drop column",
			sql:  "ALTER TABLE orders DROP COLUMN legacy_flag;",
			want: "ALTER TABLE orders DROP COLUMN legacy_flag",
		},
		{
			name: "alter column type",
			sql:  "ALTER TABLE orders ALTER COLUMN status TYPE integer;",
			want: "ALTER TABLE orders ALTER status TYPE",
		},
		{
			name: "drop table",
			sql:  "DROP TABLE legacy_archive;",
			want: "DROP TABLE legacy_archive",
		},
		{
			name: "no DDL in SELECT",
			sql:  "SELECT * FROM orders;",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := ParseMigration(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMigration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == "" {
				if len(changes) != 0 {
					t.Errorf("ParseMigration() = %v, want no changes", changes)
				}
				return
			}

			if len(changes) == 0 {
				t.Errorf("ParseMigration() = no changes, want %q", tt.want)
				return
			}

			got := changes[0].String()
			if got != tt.want {
				t.Errorf("ParseMigration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMigration_MultiStatement(t *testing.T) {
	sql := `-- Migration: drop old index
DROP INDEX IF EXISTS idx_status;
CREATE INDEX idx_status_covering ON orders (status, created_at);`

	changes, err := ParseMigration(sql)
	if err != nil {
		t.Fatalf("ParseMigration() error = %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("ParseMigration() = %d changes, want 2", len(changes))
	}
	if changes[0].Type != ChangeDropIndex {
		t.Errorf("change[0].Type = %v, want ChangeDropIndex", changes[0].Type)
	}
	if changes[1].Type != ChangeCreateIndex {
		t.Errorf("change[1].Type = %v, want ChangeCreateIndex", changes[1].Type)
	}
}
