# Database Drivers

`basemake supports 3 database engines through a common `Database` interface. Each driver is a self-contained implementation in the `internal/db` package.

## Database Interface

Defined in `internal/db/db.go`:

```go
type Database interface {
    Name() string                    // Human-readable name (e.g., "PostgreSQL (***@host:port/db)")
    Close() error                    // Close the underlying sql.DB connection
    Introspect(ctx context.Context) (*Schema, error)  // Full metadata scan
    Query(ctx context.Context, sql string) (*Rows, error)  // Execute and return rows
    Explain(ctx context.Context, sql string) (string, error)  // Execution plan as text
    ExplainJSON(ctx context.Context, sql string) (string, error)  // JSON-format plan
}
```

## Connection Management

### Driver Detection

The `detectDriver()` function in `internal/db/driver.go` matches DSNs against registered drivers:

```go
func detectDriver(dsn string) (driverConnector, error) {
    // Check each registered driver's Scheme() prefix
    // "postgres://..." → postgresDriver
    // "mysql://..." → mysqlDriver
    // "sqlite://..." or "sqlite:" → sqliteDriver
    // "postgresql://..." → postgresDriver (alias)
    // Everything else → ErrUnsupported
}
```

### Active Connection Pattern

`basemake maintains a global active connection (`var active Database` in `db.go`). Commands check this before attempting a new connection:

```go
func ActiveConnection() (Database, error) {
    if active == nil {
        return nil, ErrNoConnection
    }
    return active, nil
}
```

When a command needs a connection but none is active, it falls back to:
1. `BASEMAKE_DSN` env var → `db.Connect(dsn)`
2. `config.DefaultDSN` → `db.Connect(dsn)`
3. `db.LoadDSN()` (legacy file) → `db.Connect(dsn)`
4. Error: "no active connection — run 'basemake connect' first"

---

## PostgreSQL Driver

**Import path:** `github.com/lib/pq`  
**File:** `internal/db/postgres.go`

### DSN Format

```
postgres://username:password@host:port/dbname?sslmode=disable
postgresql://username@host/dbname   (alias, maps to same driver)
```

### Connection

```go
conn, err := sql.Open("postgres", dsn)
// Ping to verify
```

Opens a TCP connection to the PostgreSQL server. No special connection parameters are set.

### Introspection

Uses PostgreSQL's `information_schema` and `pg_catalog`:

**Tables & Columns:**
```sql
SELECT t.table_name, c.column_name, c.data_type, c.is_nullable,
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
```

**Indexes:**
```sql
SELECT i.relname as index_name, ix.indisunique as is_unique, a.attname as column_name
FROM pg_index ix
JOIN pg_class t ON t.oid = ix.indrelid
JOIN pg_class i ON i.oid = ix.indexrelid
JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
WHERE t.relname = $1
ORDER BY i.relname, a.attnum
```

### Query Execution

Standard `db.QueryContext()` with column metadata extraction via `rows.Columns()`.

### EXPLAIN

**Text format** (`Explain`):
```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) <query>
```

**JSON format** (`ExplainJSON`):
```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) <query>
```

PostgreSQL is the **only** driver that supports JSON-format EXPLAIN, which enables the full analysis engine (plan parsing, issue detection, plan tree visualization).

### DSN Masking

```go
func maskDSN(dsn string) string {
    // "postgres://user:pass@host:5432/db" → "***@host:5432/db"
    parts := strings.Split(dsn, "@")
    if len(parts) > 1 {
        return "***@" + parts[1]
    }
    return dsn
}
```

Used in `Name()` output and displayed connection messages.

---

## MySQL Driver

**Import path:** `github.com/go-sql-driver/mysql`  
**File:** `internal/db/mysql.go`

### DSN Format

```
mysql://username:password@tcp(host:port)/dbname
mysql://username@tcp(localhost:3306)/dbname
```

### Connection

```go
conn, err := sql.Open("mysql", dsn)
```

### Introspection

Uses MySQL's `information_schema`:

**Tables & Columns:**
```sql
SELECT t.TABLE_NAME, c.COLUMN_NAME, c.COLUMN_TYPE, c.IS_NULLABLE,
       c.COLUMN_DEFAULT,
       (CASE WHEN c.COLUMN_KEY = 'PRI' THEN true ELSE false END) as is_pk
FROM information_schema.TABLES t
JOIN information_schema.COLUMNS c ON t.TABLE_NAME = c.TABLE_NAME AND t.TABLE_SCHEMA = c.TABLE_SCHEMA
WHERE t.TABLE_SCHEMA = DATABASE() AND t.TABLE_TYPE = 'BASE TABLE'
ORDER BY t.TABLE_NAME, c.ORDINAL_POSITION
```

**Indexes:**
```sql
SELECT INDEX_NAME, NON_UNIQUE, COLUMN_NAME
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
ORDER BY INDEX_NAME, SEQ_IN_INDEX
```

Note: MySQL's `NON_UNIQUE` flag is inverted — `NON_UNIQUE = 0` means unique index.

### EXPLAIN

**Text format** (`Explain`):
```sql
EXPLAIN ANALYZE <query>
```

**JSON format** (`ExplainJSON`):  
**Not supported.** MySQL's EXPLAIN ANALYZE returns text only. `ExplainJSON()` returns an error: `"mysql explain json: MySQL does not support JSON format EXPLAIN"`. The `analyze` command catches this and falls back to text display.

### Limitations vs PostgreSQL

- No JSON plan format → no structured plan analysis
- No issue detection (the analyze command shows raw text)
- `DBName` in schema is set to the full DSN string (not extracted cleanly)
- No `pg_catalog`-style index column mapping (uses SEQ_IN_INDEX for ordering)

---

## SQLite Driver

**Import path:** `modernc.org/sqlite` (pure Go, no CGo)  
**File:** `internal/db/sqlite.go`

### DSN Format

```
sqlite:/path/to/database.db
sqlite:///path/to/database.db
sqlite:./relative.db
sqlite:file::memory:?mode=memory&cache=shared    (in-memory, for testing)
```

### DSN Parsing

```go
path := dsn
for _, prefix := range []string{"sqlite://", "sqlite:"} {
    if strings.HasPrefix(path, prefix) {
        path = strings.TrimPrefix(path, prefix)
        break
    }
}
```

Supports both `sqlite://` (URL-style) and `sqlite:` (short form). The path is used directly for `sql.Open("sqlite", path)`.

### WAL Mode

On connection, WAL journal mode is enabled:

```go
conn.Exec("PRAGMA journal_mode=WAL")
```

This improves concurrent read performance and is safe for read-heavy workloads.

### Introspection

Uses SQLite's `sqlite_master` table and PRAGMA functions:

**Tables:**
```sql
SELECT name FROM sqlite_master
WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
ORDER BY name
```

**Columns** (via `PRAGMA table_info(name)`):
Returns: `cid, name, type, notnull, dflt_value, pk`

**Indexes** (via `PRAGMA index_list(name)`):
Returns: `seq, name, unique, origin, partial`

**Index columns** (via `PRAGMA index_info(name)`):
Returns: `seqno, cid, name`

### Query Execution

Same pattern as PostgreSQL/MySQL — `QueryContext()` with column metadata.

### EXPLAIN

**Text format** (`Explain`):
```sql
EXPLAIN QUERY PLAN <query>
```

SQLite's `EXPLAIN QUERY PLAN` returns a multi-row result with detail in the last column. The driver concatenates rows separated by newlines.

**JSON format** (`ExplainJSON`):  
Falls back to text format. Returns the same output as `Explain()` — no JSON support, but the command-level fallback in `analyzeQuery()` still works (displays text).

### Test Coverage

The SQLite driver has the most comprehensive test suite (4 test functions):
- `TestSQLiteConnect` — Connection lifecycle
- `TestSQLiteIntrospect` — Full schema introspection with PK/index detection
- `TestSQLiteQuery` — Query execution with row iteration
- `TestSQLiteExplain` — EXPLAIN QUERY PLAN output verification
- `TestSQLiteExplainJSON` — Fallback behavior for JSON explain

### Performance Note

Since `modernc.org/sqlite` is pure Go, it's approximately 2-3x slower than CGo-based SQLite for bulk operations. For CLI query workloads (reading result sets), the difference is negligible.

---

## Schema Types

```go
type Schema struct {
    DBName string      `json:"db_name"`       // Database name
    Tables []TableInfo `json:"tables"`         // All user tables
}

type TableInfo struct {
    Name    string       `json:"name"`          // Table name
    Columns []ColumnInfo `json:"columns"`       // Column definitions
    Indexes []IndexInfo  `json:"indexes"`       // Index definitions
}

type ColumnInfo struct {
    Name       string `json:"name"`              // Column name
    Type       string `json:"type"`              // Data type (DB-specific)
    IsPK       bool   `json:"is_pk"`             // Is primary key?
    IsNullable bool   `json:"is_nullable"`       // Can be NULL?
    Default    string `json:"default,omitempty"` // Default value expression
}

type IndexInfo struct {
    Name   string   `json:"name"`    // Index name
    Unique bool     `json:"unique"`  // Is unique constraint?
    Cols   []string `json:"cols"`    // Column names in order
}
```

### Schema Helper Methods

```go
func (s *Schema) TotalColumns() int    // Sum of all columns across all tables
func (s *Schema) TotalIndexes() int    // Sum of all indexes across all tables
func (s *Schema) SchemaForPrompt() string  // Compact text for AI prompt
```

## Schema Cache

- Written to `/root/.basemake/schema.json` (or `$HOME/.basemake/schema.json`)
- Used by NL→SQL generation and `analyze --all`
- Cleared and re-fetched on each `basemake connect`

## Rows Type

The custom `Rows` struct wraps `database/sql.Rows` with column metadata:

```go
type Rows struct {
    rows interface {
        Next() bool
        Scan(dest ...any) error
        Close() error
    }
    cols []string
}
```

This abstraction exists so the `Database` interface can return a uniform type regardless of the underlying driver.

## Error Types

Defined in `internal/db/errors.go`:

```go
var ErrNoConnection = fmt.Errorf("no active connection")
var ErrUnsupported = fmt.Errorf("unsupported database driver")
```

- `ErrNoConnection`: Returned by `ActiveConnection()` when no connection has been made
- `ErrUnsupported`: Returned by `detectDriver()` when the DSN scheme doesn't match any registered driver
