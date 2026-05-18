package db

import (
	"testing"
)

func TestDetectDriver(t *testing.T) {
	tests := []struct {
		dsn    string
		scheme string
	}{
		{"postgres://user:pass@localhost:5432/db", "postgres"},
		{"postgres://user@localhost/db", "postgres"},
		{"mysql://user:pass@localhost:3306/db", "mysql"},
		{"mysql://root@localhost/db", "mysql"},
	}

	for _, tc := range tests {
		d, err := detectDriver(tc.dsn)
		if err != nil {
			t.Errorf("detectDriver(%q) error: %v", tc.dsn, err)
			continue
		}
		if d.Scheme() != tc.scheme {
			t.Errorf("detectDriver(%q) = %s, want %s", tc.dsn, d.Scheme(), tc.scheme)
		}
	}
}

func TestDetectDriverUnsupported(t *testing.T) {
	_, err := detectDriver("oracle://user:pass@localhost:1521/db")
	if err == nil {
		t.Error("expected error for unsupported driver, got nil")
	}
}

func TestDetectDriverInvalid(t *testing.T) {
	_, err := detectDriver("not-a-dsn")
	if err == nil {
		t.Error("expected error for invalid DSN, got nil")
	}
}

func TestDetectDriverSQLite(t *testing.T) {
	d, err := detectDriver("sqlite:///tmp/test.db")
	if err != nil {
		t.Fatalf("detectDriver(sqlite) error: %v", err)
	}
	if d.Scheme() != "sqlite" {
		t.Errorf("scheme = %q, want %q", d.Scheme(), "sqlite")
	}
}

func TestDetectDriverPostgresAlias(t *testing.T) {
	d, err := detectDriver("postgresql://user@localhost/db")
	if err != nil {
		t.Fatalf("detectDriver(postgresql) error: %v", err)
	}
	if d.Scheme() != "postgres" {
		t.Errorf("scheme = %q, want %q", d.Scheme(), "postgres")
	}
}
