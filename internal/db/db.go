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
	ExplainJSON(ctx context.Context, sql string) (string, error)
}

// Schema holds database metadata populated by Introspect
type Schema struct {
	DBName string      `json:"db_name"`
	Tables []TableInfo `json:"tables"`
}

type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
	Indexes []IndexInfo  `json:"indexes"`
}

type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsPK       bool   `json:"is_pk"`
	IsNullable bool   `json:"is_nullable"`
	Default    string `json:"default,omitempty"`
}

type IndexInfo struct {
	Name   string   `json:"name"`
	Unique bool     `json:"unique"`
	Cols   []string `json:"cols"`
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
