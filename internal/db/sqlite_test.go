package db

import (
	"context"
	"strings"
	"testing"
)

func TestSQLiteConnect(t *testing.T) {
	d := &sqliteDriver{}
	dbase, err := d.Connect("sqlite:file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dbase.Close()

	if !strings.Contains(dbase.Name(), "SQLite") {
		t.Errorf("Name() = %q, want SQLite", dbase.Name())
	}
}

func TestSQLiteIntrospect(t *testing.T) {
	d := &sqliteDriver{}
	dbase, err := d.Connect("sqlite:file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dbase.Close()

	// Create test tables
	_, err = dbase.(*sqliteDB).conn.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT
		);
		CREATE INDEX idx_users_email ON users(email);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			total REAL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	schema, err := dbase.Introspect(context.Background())
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}

	if len(schema.Tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(schema.Tables))
	}

	// Check users table
	var users TableInfo
	for _, tbl := range schema.Tables {
		if tbl.Name == "users" {
			users = tbl
			break
		}
	}

	if users.Name != "users" {
		t.Fatalf("did not find users table")
	}

	if len(users.Columns) != 3 {
		t.Fatalf("users has %d columns, want 3", len(users.Columns))
	}

	// Check PK detection
	foundPK := false
	for _, c := range users.Columns {
		if c.Name == "id" && c.IsPK {
			foundPK = true
			break
		}
	}
	if !foundPK {
		t.Error("expected id column to be PK")
	}
}

func TestSQLiteQuery(t *testing.T) {
	d := &sqliteDriver{}
	dbase, err := d.Connect("sqlite:file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dbase.Close()

	_, err = dbase.(*sqliteDB).conn.Exec("CREATE TABLE test (id INT, val TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = dbase.(*sqliteDB).conn.Exec("INSERT INTO test VALUES (1, 'hello'), (2, 'world')")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := dbase.Query(context.Background(), "SELECT * FROM test ORDER BY id")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rows.Close()

	cols := rows.Columns()
	if len(cols) != 2 {
		t.Fatalf("got %d columns, want 2", len(cols))
	}

	count := 0
	vals := make([]any, 2)
	ptrs := []any{&vals[0], &vals[1]}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("got %d rows, want 2", count)
	}
}

func TestSQLiteExplain(t *testing.T) {
	d := &sqliteDriver{}
	dbase, err := d.Connect("sqlite:file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dbase.Close()

	_, err = dbase.(*sqliteDB).conn.Exec("CREATE TABLE t (id INT, val TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	plan, err := dbase.Explain(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("Explain: %v", err)
	}

	if !strings.Contains(plan, "SCAN") && !strings.Contains(plan, "scan") {
		t.Errorf("expected SCAN in plan, got: %s", plan)
	}
}

func TestSQLiteExplainJSON(t *testing.T) {
	d := &sqliteDriver{}
	dbase, err := d.Connect("sqlite:file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer dbase.Close()

	_, err = dbase.(*sqliteDB).conn.Exec("CREATE TABLE t (id INT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// SQLite returns text even for ExplainJSON (no JSON format)
	plan, err := dbase.ExplainJSON(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("ExplainJSON: %v", err)
	}
	if len(plan) == 0 {
		t.Error("expected non-empty plan")
	}
}
