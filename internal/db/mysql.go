package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type mysqlDriver struct{}

func (d *mysqlDriver) Scheme() string { return "mysql" }

func (d *mysqlDriver) Connect(dsn string) (Database, error) {
	nativeDSN := dsn

	// Convert mysql://user:pass@host:3306/db to user:pass@tcp(host:3306)/db
	if strings.HasPrefix(dsn, "mysql://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			user := ""
			if parsed.User != nil {
				user = parsed.User.String()
			}
			host := parsed.Host // includes port if present
			if host == "" {
				host = "127.0.0.1:3306"
			} else if !strings.Contains(host, ":") {
				host = host + ":3306"
			}
			dbName := strings.TrimPrefix(parsed.Path, "/")
			params := parsed.RawQuery

			nativeDSN = fmt.Sprintf("%s@tcp(%s)/%s", user, host, dbName)
			if params != "" {
				nativeDSN += "?" + params
			}
		}
	}

	conn, err := sql.Open("mysql", nativeDSN)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}
	if err := conn.PingContext(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	return &mysqlDB{conn: conn, dsn: dsn}, nil
}

type mysqlDB struct {
	conn *sql.DB
	dsn  string
}

func (m *mysqlDB) Name() string {
	return fmt.Sprintf("MySQL (%s)", maskDSN(m.dsn))
}

func (m *mysqlDB) Dialect() string { return "MySQL" }

func (m *mysqlDB) Close() error {
	return m.conn.Close()
}

func (m *mysqlDB) Introspect(ctx context.Context) (*Schema, error) {
	s := &Schema{
		DBName: m.dsn,
	}

	rows, err := m.conn.QueryContext(ctx, `
		SELECT
			t.TABLE_NAME,
			c.COLUMN_NAME,
			c.COLUMN_TYPE,
			c.IS_NULLABLE,
			c.COLUMN_DEFAULT,
			(CASE WHEN c.COLUMN_KEY = 'PRI' THEN true ELSE false END) as is_pk
		FROM information_schema.TABLES t
		JOIN information_schema.COLUMNS c ON t.TABLE_NAME = c.TABLE_NAME AND t.TABLE_SCHEMA = c.TABLE_SCHEMA
		WHERE t.TABLE_SCHEMA = DATABASE() AND t.TABLE_TYPE = 'BASE TABLE'
		ORDER BY t.TABLE_NAME, c.ORDINAL_POSITION
	`)
	if err != nil {
		return nil, fmt.Errorf("introspect mysql: %w", err)
	}
	defer rows.Close()

	currentTable := ""
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
			t := &TableInfo{Name: tbl}
			tableMap[tbl] = t
			tableOrder = append(tableOrder, tbl)
			currentTable = tbl
		}
		table := tableMap[tbl]
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
		idxRows, err := m.conn.QueryContext(ctx, `
			SELECT INDEX_NAME, NON_UNIQUE, COLUMN_NAME
			FROM information_schema.STATISTICS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
			ORDER BY INDEX_NAME, SEQ_IN_INDEX
		`, t.Name)
		if err != nil {
			continue
		}

		idxMap := make(map[string]*IndexInfo)
		for idxRows.Next() {
			var name string
			var nonUnique bool
			var col string
			if err := idxRows.Scan(&name, &nonUnique, &col); err != nil {
				continue
			}
			if _, ok := idxMap[name]; !ok {
				idxMap[name] = &IndexInfo{Name: name, Unique: !nonUnique}
			}
			idxMap[name].Cols = append(idxMap[name].Cols, col)
		}
		idxRows.Close()

		for _, idx := range idxMap {
			s.Tables[i].Indexes = append(s.Tables[i].Indexes, *idx)
		}
	}

	// Get foreign keys
	for i, t := range s.Tables {
		fkRows, err := m.conn.QueryContext(ctx, `
			SELECT COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE()
				AND TABLE_NAME = ?
				AND REFERENCED_TABLE_NAME IS NOT NULL
		`, t.Name)
		if err != nil {
			continue
		}

		for fkRows.Next() {
			var col, refTbl, refCol string
			if err := fkRows.Scan(&col, &refTbl, &refCol); err != nil {
				continue
			}
			s.Tables[i].ForeignKeys = append(s.Tables[i].ForeignKeys, ForeignKeyInfo{
				Column:    col,
				RefTable:  refTbl,
				RefColumn: refCol,
			})
		}
		fkRows.Close()
	}

	// Get estimated row counts from information_schema
	for i, t := range s.Tables {
		err := m.conn.QueryRowContext(ctx,
			"SELECT TABLE_ROWS FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", t.Name).Scan(&s.Tables[i].EstimatedRows)
		if err != nil {
			s.Tables[i].EstimatedRows = 0
		}
	}

	return s, nil
}

func (m *mysqlDB) Query(ctx context.Context, query string) (*Rows, error) {
	rows, err := m.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mysql query: %w", err)
	}

	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		return nil, fmt.Errorf("mysql columns: %w", err)
	}

	return &Rows{rows: rows, cols: cols}, nil
}

func (m *mysqlDB) Explain(ctx context.Context, query string) (string, error) {
	tx, err := m.conn.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("mysql begin tx: %w", err)
	}
	defer tx.Rollback() // always rollback — safe for EXPLAIN

	var plan string
	err = tx.QueryRowContext(ctx, "EXPLAIN ANALYZE "+query).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("mysql explain: %w", err)
	}
	return plan, nil
}

// ExplainJSON runs EXPLAIN FORMAT=JSON for structured analysis.
// MySQL supports FORMAT=JSON since 5.6.
// Note: MySQL's EXPLAIN FORMAT=JSON does not currently support ANALYZE (timing),
// so actual execution metrics will be zero.
func (m *mysqlDB) ExplainJSON(ctx context.Context, query string) (string, error) {
	var plan string
	err := m.conn.QueryRowContext(ctx, "EXPLAIN FORMAT=JSON "+query).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("mysql explain json: %w", err)
	}
	return plan, nil
}

func (m *mysqlDB) ExplainNoAnalyze(ctx context.Context, query string) (string, error) {
	var plan string
	err := m.conn.QueryRowContext(ctx, "EXPLAIN FORMAT=JSON "+query).Scan(&plan)
	if err != nil {
		return "", fmt.Errorf("mysql explain: %w", err)
	}
	return plan, nil
}
