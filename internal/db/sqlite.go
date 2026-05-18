package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type sqliteDriver struct{}

func (d *sqliteDriver) Scheme() string { return "sqlite" }

func (d *sqliteDriver) Connect(dsn string) (Database, error) {
	// Extract file path from DSN
	// Supports: sqlite:///path/to/db.db, sqlite:/path/to/db.db, sqlite:./relative.db
	path := dsn
	for _, prefix := range []string{"sqlite://", "sqlite:"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}

	if path == "" {
		return nil, fmt.Errorf("sqlite: no database path in DSN")
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sqlite wal: %w", err)
	}

	if err := conn.PingContext(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}

	return &sqliteDB{conn: conn, path: path}, nil
}

type sqliteDB struct {
	conn *sql.DB
	path string
}

func (s *sqliteDB) Name() string {
	return fmt.Sprintf("SQLite (%s)", s.path)
}

func (s *sqliteDB) Dialect() string { return "SQLite" }

func (s *sqliteDB) Close() error {
	return s.conn.Close()
}

func (s *sqliteDB) Introspect(ctx context.Context) (*Schema, error) {
	schema := &Schema{DBName: s.path}

	// Get all user tables
	rows, err := s.conn.QueryContext(ctx, `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("sqlite introspect tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("sqlite scan table: %w", err)
		}
		tableNames = append(tableNames, name)
	}

	for _, tblName := range tableNames {
		table := TableInfo{Name: tblName}

		// Get columns via PRAGMA
		colRows, err := s.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", tblName))
		if err != nil {
			continue
		}

		type colInfo struct {
			cid    int
			name   string
			typ    string
			notNull int
			dflt   *string
			pk     int
		}

		for colRows.Next() {
			var ci colInfo
			if err := colRows.Scan(&ci.cid, &ci.name, &ci.typ, &ci.notNull, &ci.dflt, &ci.pk); err != nil {
				colRows.Close()
				continue
			}
			col := ColumnInfo{
				Name:       ci.name,
				Type:       ci.typ,
				IsPK:       ci.pk == 1,
				IsNullable: ci.notNull == 0,
			}
			if ci.dflt != nil {
				col.Default = *ci.dflt
			}
			table.Columns = append(table.Columns, col)
		}
		colRows.Close()

		// Get indexes via PRAGMA
		idxRows, err := s.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%q)", tblName))
		if err != nil {
			schema.Tables = append(schema.Tables, table)
			continue
		}

		type idxInfo struct {
			seq    int
			name   string
			unique int
			origin string
			partial int
		}

		var idxNames []string
		idxUnique := make(map[string]bool)
		for idxRows.Next() {
			var ii idxInfo
			if err := idxRows.Scan(&ii.seq, &ii.name, &ii.unique, &ii.origin, &ii.partial); err != nil {
				continue
			}
			idxNames = append(idxNames, ii.name)
			idxUnique[ii.name] = ii.unique == 1
		}
		idxRows.Close()

		for _, idxName := range idxNames {
			index := IndexInfo{
				Name:   idxName,
				Unique: idxUnique[idxName],
			}

			// Get index columns
			colIdxRows, err := s.conn.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%q)", idxName))
			if err != nil {
				continue
			}

			type idxCol struct {
				seqno int
				cid   int
				name  string
			}
			for colIdxRows.Next() {
				var ic idxCol
				if err := colIdxRows.Scan(&ic.seqno, &ic.cid, &ic.name); err != nil {
					continue
				}
				index.Cols = append(index.Cols, ic.name)
			}
			colIdxRows.Close()

			table.Indexes = append(table.Indexes, index)
		}

		schema.Tables = append(schema.Tables, table)
	}

	return schema, nil
}

func (s *sqliteDB) Query(ctx context.Context, sqlStr string) (*Rows, error) {
	rows, err := s.conn.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite query: %w", err)
	}

	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, fmt.Errorf("sqlite columns: %w", err)
	}

	return &Rows{rows: rows, cols: cols}, nil
}

// Explain returns the query plan as text
func (s *sqliteDB) Explain(ctx context.Context, query string) (string, error) {
	rows, err := s.conn.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query)
	if err != nil {
		return "", fmt.Errorf("sqlite explain: %w", err)
	}
	defer rows.Close()

	// Get columns to determine row structure
	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("sqlite explain columns: %w", err)
	}

	var plan string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", fmt.Errorf("sqlite explain scan: %w", err)
		}

		// Detail is typically in the last column
		detail := ""
		if len(vals) > 0 {
			if v, ok := vals[len(vals)-1].([]byte); ok {
				detail = string(v)
			} else if v, ok := vals[len(vals)-1].(string); ok {
				detail = v
			} else {
				detail = fmt.Sprint(vals[len(vals)-1])
			}
		}

		if plan != "" {
			plan += "\n"
		}
		plan += detail
	}

	if plan == "" {
		return "", fmt.Errorf("sqlite explain: no plan returned")
	}

	return plan, nil
}

func (s *sqliteDB) ExplainJSON(ctx context.Context, query string) (string, error) {
	plan, err := s.Explain(ctx, query)
	if err != nil {
		return "", err
	}
	return plan, nil
}
