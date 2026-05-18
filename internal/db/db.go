package db

import (
	"context"
)

// Rows holds query result rows for iteration
type Rows struct {
	rows interface {
		Next() bool
		Scan(dest ...any) error
		Close() error
	}
	cols []string
}

func (r *Rows) Next() bool {
	return r.rows.Next()
}

func (r *Rows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *Rows) Close() error {
	return r.rows.Close()
}

func (r *Rows) Columns() []string {
	return r.cols
}

// Database defines the interface all supported databases must implement
type Database interface {
	Name() string
	Close() error
	Introspect(ctx context.Context) (*Schema, error)
	Query(ctx context.Context, sql string) (*Rows, error)
	Explain(ctx context.Context, sql string) (string, error)
}

// Schema holds database metadata populated by Introspect
type Schema struct {
	DBName string
	Tables []TableInfo
}

type TableInfo struct {
	Name    string
	Columns []ColumnInfo
	Indexes []IndexInfo
}

type ColumnInfo struct {
	Name       string
	Type       string
	IsPK       bool
	IsNullable bool
	Default    string
}

type IndexInfo struct {
	Name   string
	Unique bool
	Cols   []string
}

// active holds the current database connection
var active Database

// ActiveConnection returns the current connection
func ActiveConnection() (Database, error) {
	if active == nil {
		return nil, ErrNoConnection
	}
	return active, nil
}

// Connect establishes a new database connection from a DSN
func Connect(dsn string) (Database, error) {
	driver, err := detectDriver(dsn)
	if err != nil {
		return nil, err
	}

	db, err := driver.Connect(dsn)
	if err != nil {
		return nil, err
	}

	active = db
	return db, nil
}
