package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type postgresDriver struct{}

func (d *postgresDriver) Scheme() string { return "postgres" }

func (d *postgresDriver) Connect(dsn string) (Database, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	if err := conn.PingContext(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &postgresDB{conn: conn, dsn: dsn}, nil
}

type postgresDB struct {
	conn *sql.DB
	dsn  string
}

func (p *postgresDB) Name() string {
	return fmt.Sprintf("PostgreSQL (%s)", maskDSN(p.dsn))
}

func (p *postgresDB) Dialect() string { return "PostgreSQL" }

func (p *postgresDB) Close() error {
	return p.conn.Close()
}

func (p *postgresDB) Introspect(ctx context.Context) (*Schema, error) {
	s := &Schema{}

	// Get tables + columns
	rows, err := p.conn.QueryContext(ctx, `
		SELECT
			t.table_name,
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			(CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END) as is_pk
		FROM information_schema.tables t
		JOIN information_schema.columns c ON t.table_name = c.table_name AND t.table_schema = c.table_schema
		LEFT JOIN (
			SELECT ku.table_name, ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = 'public'
		) pk ON c.table_name = pk.table_name AND c.column_name = pk.column_name
		WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name, c.ordinal_position
	`)
	if err != nil {
		return nil, fmt.Errorf("introspect tables: %w", err)
	}
	defer rows.Close()

	currentTable := ""
	var table *TableInfo
	tableMap := make(map[string]*TableInfo)
	tableOrder := []string{}

	for rows.Next() {
		var tbl, col, typ, nullable string
		var def *string
		var isPK bool
		if err := rows.Scan(&tbl, &col, &typ, &nullable, &def, &isPK); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}
		if tbl != currentTable {
			table = &TableInfo{Name: tbl}
			tableMap[tbl] = table
			tableOrder = append(tableOrder, tbl)
			currentTable = tbl
		}
		table = tableMap[tbl]
		n := false
		if nullable == "YES" {
			n = true
		}
		d := ""
		if def != nil {
			d = *def
		}
		table.Columns = append(table.Columns, ColumnInfo{
			Name:       col,
			Type:       typ,
			IsPK:       isPK,
			IsNullable: n,
			Default:    d,
		})
	}

	// Build final table list from the ordered map
	for _, name := range tableOrder {
		s.Tables = append(s.Tables, *tableMap[name])
	}

	// Get indexes
	for i, t := range s.Tables {
		idxRows, err := p.conn.QueryContext(ctx, `
			SELECT i.relname as index_name,
			       ix.indisunique as is_unique,
			       a.attname as column_name
			FROM pg_index ix
			JOIN pg_class t ON t.oid = ix.indrelid
			JOIN pg_class i ON i.oid = ix.indexrelid
			JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
			WHERE t.relname = $1
			ORDER BY i.relname, a.attnum
		`, t.Name)
		if err != nil {
			continue
		}

		idxMap := make(map[string]*IndexInfo)
		for idxRows.Next() {
			var name string
			var unique bool
			var col string
			if err := idxRows.Scan(&name, &unique, &col); err != nil {
				continue
			}
			if _, ok := idxMap[name]; !ok {
				idxMap[name] = &IndexInfo{Name: name, Unique: unique}
			}
			idxMap[name].Cols = append(idxMap[name].Cols, col)
		}
		idxRows.Close()

		for _, idx := range idxMap {
			s.Tables[i].Indexes = append(s.Tables[i].Indexes, *idx)
		}
	}

	return s, nil
}

func (p *postgresDB) Query(ctx context.Context, sql string) (*Rows, error) {
	rows, err := p.conn.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("postgres query: %w", err)
	}

	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, fmt.Errorf("postgres columns: %w", err)
	}

	return &Rows{rows: rows, cols: cols}, nil
}
// Explain runs EXPLAIN ANALYZE and returns the raw plan text
// SAFETY: For DML queries (INSERT/UPDATE/DELETE), wraps in BEGIN/ROLLBACK
// to prevent actual data modification.
func (p *postgresDB) Explain(ctx context.Context, query string) (string, error) {
	var plan string
	safeQuery := p.safeExplainSQL(query)
	err := p.conn.QueryRowContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+safeQuery).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("postgres explain: %w", err)
	}
	return plan, nil
}

// ExplainJSON runs EXPLAIN ANALYZE with JSON format for structured analysis
// SAFETY: For DML queries, wraps in BEGIN/ROLLBACK.
func (p *postgresDB) ExplainJSON(ctx context.Context, query string) (string, error) {
	var plan string
	safeQuery := p.safeExplainSQL(query)
	err := p.conn.QueryRowContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+safeQuery).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("postgres explain json: %w", err)
	}
	return plan, nil
}

// ExplainNoAnalyze returns the plan without executing the query.
// Uses EXPLAIN (FORMAT JSON) without ANALYZE — safe for all query types.
func (p *postgresDB) ExplainNoAnalyze(ctx context.Context, query string) (string, error) {
	var plan string
	err := p.conn.QueryRowContext(ctx, "EXPLAIN (FORMAT JSON) "+query).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("postgres explain: %w", err)
	}
	return plan, nil
}

// safeExplainSQL wraps DML queries in a transaction that gets rolled back,
// preventing actual data modification during EXPLAIN ANALYZE.
func (p *postgresDB) safeExplainSQL(query string) string {
	upper := strings.TrimSpace(strings.ToUpper(query))
	for _, prefix := range []string{"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE"} {
		if strings.HasPrefix(upper, prefix) {
			return "BEGIN; " + query + "; ROLLBACK;"
		}
	}
	return query
}

func maskDSN(dsn string) string {
	// Extract host:port from connection string
	parts := strings.Split(dsn, "@")
	if len(parts) > 1 {
		return "***@" + parts[1]
	}
	return dsn
}
