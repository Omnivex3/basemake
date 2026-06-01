# SQL Guardrails & Schema Optimization

This document covers the safety guardrails and schema optimization features in basemake — the `SELECT *` guardrail, query-aware schema truncation, FK context injection, and how these are wired into both the TUI and CLI execution paths.

---

## 1. SELECT * Guardrail (`GuardrailSelectStar`)

The `SELECT *` guardrail prevents accidental full-table scans by intercepting `SELECT *` and `SELECT t.*` patterns before query execution. It uses **estimated row counts** from the schema cache to apply graduated enforcement.

### Source

`internal/db/guardrail.go`

### Detection

The guardrail uses a regex to match `SELECT *` and `SELECT t.*` patterns (case-insensitive):

```go
var selectStarRE = regexp.MustCompile(`(?i)\bSELECT\s+(?:ALL\s+|DISTINCT\s+)?\*`)
```

It extracts the top-level FROM-clause table name via `extractTableFromSQL()` — a best-effort parser that handles `SELECT * FROM table`, `SELECT * FROM table AS t`, and quoted identifiers. Subqueries and CTEs are not matched (the regex only targets the outermost SELECT).

### Three Tiers

| Row Count | Action | Behavior |
|-----------|--------|----------|
| **< 10K** (small) | Rewrite + note | Expands `*` to explicit column names; logs a blue ℹ message |
| **10K – 1M** (medium) | Warn + rewrite | Expands `*` and displays a yellow ⚠ warning about performance |
| **> 1M** (large) | Block | Query is **not executed**; prints a red 🚫 error with a suggestion like `SELECT col1, col2, ... FROM table LIMIT 100` |
| **Unknown** (0) | Warn + pass through | Warns generically; if the table has columns, rewrites anyway |

### Rewrite Engine

The `rewriteSelectStar()` function handles two patterns:

1. **`SELECT *`** — replaced with `SELECT col1, col2, ...` using the column names from `TableInfo.Columns`
2. **`SELECT t.*`** — replaced with `SELECT t.col1, t.col2, ...` using the alias prefix

If the table has no cached columns, the original SQL passes through unchanged.

### Estimated Rows Sources

The row count source depends on the database dialect:

| Dialect | Source | Query |
|---------|--------|-------|
| **PostgreSQL** | `pg_class.reltuples` | `SELECT reltuples::bigint FROM pg_class WHERE relname = $1` |
| **MySQL** | `information_schema.TABLES.TABLE_ROWS` | `SELECT TABLE_ROWS FROM information_schema.TABLES WHERE ...` |
| **SQLite** | — | Returns `0` (SQLite has no built-in row count estimate) |

When `EstimatedRows` is `0`, the guardrail treats it as "unknown" — it warns but allows execution.

---

## 2. Schema Truncation (`SchemaForPrompt`)

The schema is too large to fit in an LLM context window for databases with many tables and columns. `SchemaForPrompt()` in `internal/db/cache.go` performs **query-aware truncation** to stay within a ~2000 token budget (~8000 characters).

### Two-Stage Filtering

When the user asks a natural language question (e.g., "show me users who ordered last month"), the truncation runs two stages:

**Stage 1 — Keyword Matching**

1. Tokenize the question: split on non-alphanumeric chars, filter out stop words (the, show, me, get, all, etc.)
2. Score each table:
   - Table name matches a keyword → +2 points
   - Any column name matches a keyword → +1 point per keyword

**Stage 2 — FK Expansion**

Once keyword-matched tables are identified, the expander adds FK-related tables:

- **Forward FK**: If table `A` has a foreign key referencing table `B`, and `A` matched, `B` is included
- **Reverse FK**: If table `C` has a foreign key referencing a matched table, `C` is included

This ensures JOIN-relevant tables are always present in the prompt, even if they didn't match query keywords directly.

### Sorting and Budget

Tables are rendered in this order:
1. Highest keyword score first (most relevant)
2. FK-expanded tables next (join neighbors)
3. Alphabetical within same priority

Each table block includes:

```text
  users:
    - id integer [PK]
    - email text nullable
    - name text nullable
    Foreign Keys:
      - role_id → roles.id
    Indexes:
      - users_pkey on (id) (unique)
```

If a table doesn't fit within the 8000-character budget, it's omitted. A note is appended:

```
    ... and N additional table(s) omitted. Ask about a specific table for details.
```

### Empty Question Fallback

When no question is provided (e.g., `.schema` command), tables are included in their natural order (as returned by introspection) up to the budget, with the same truncation note.

### Token Estimation

```go
func EstimateTokens(text string) int {
    return len(text) / 4  // ~4 chars per token for English text
}
```

---

## 3. FK Context in Schema Prompts

Foreign key relationships are a critical signal for NL→SQL accuracy. The prompt rendering always includes FK information when it exists.

### FK Rendering in `renderTableBlock()`

```text
    Foreign Keys:
      - role_id → roles.id
```

This format (`column → ref_table.ref_column`) is compact and unambiguous for LLMs. The FK data comes from `ForeignKeyInfo`:

```go
type ForeignKeyInfo struct {
    Column    string `json:"column"`     // local column name
    RefTable  string `json:"ref_table"`  // referenced table name
    RefColumn string `json:"ref_column"` // referenced column name
}
```

### FK Data Collection per Dialect

| Dialect | Source Query |
|---------|-------------|
| **PostgreSQL** | `information_schema.table_constraints` + `key_column_usage` + `constraint_column_usage` |
| **MySQL** | `information_schema.KEY_COLUMN_USAGE WHERE REFERENCED_TABLE_NAME IS NOT NULL` |
| **SQLite** | `PRAGMA foreign_key_list(table)` |

### Why FK Context Matters

Without FK context, the AI might:
- Generate `JOIN` clauses on unrelated columns
- Miss available join paths entirely
- Generate cross-joins instead of proper equi-joins

With FK context, the AI sees the exact relationships needed for accurate multi-table queries.

---

## 4. Core Data Structures

### `TableInfo`

```go
type TableInfo struct {
    Name          string           `json:"name"`
    Columns       []ColumnInfo     `json:"columns"`
    Indexes       []IndexInfo      `json:"indexes"`
    ForeignKeys   []ForeignKeyInfo `json:"foreign_keys"`
    EstimatedRows int64            `json:"estimated_rows"`
}
```

### `ColumnInfo`

```go
type ColumnInfo struct {
    Name       string `json:"name"`
    Type       string `json:"type"`
    IsPK       bool   `json:"is_pk"`
    IsNullable bool   `json:"is_nullable"`
    Default    string `json:"default,omitempty"`
}
```

### `ForeignKeyInfo`

```go
type ForeignKeyInfo struct {
    Column    string `json:"column"`
    RefTable  string `json:"ref_table"`
    RefColumn string `json:"ref_column"`
}
```

### `Schema`

```go
type Schema struct {
    DBName string      `json:"db_name"`
    Tables []TableInfo `json:"tables"`
}
```

### `GuardrailResult`

```go
type GuardrailResult struct {
    SQL     string // possibly modified SQL
    Warning string // human-readable warning (empty = no issue)
    Blocked bool   // true = execution should be prevented
    Rewrote bool   // true = SQL was modified
}
```

---

## 5. Guardrail Wiring

The guardrail is invoked in both execution paths:

### CLI Path (`cmd/query.go`)

After AI generation (or direct SQL input), `GuardrailSelectStar` runs before query execution:

```go
// cmd/query.go lines 102-112
if s, err := db.LoadSchema(); err == nil {
    result := db.GuardrailSelectStar(sql, s)
    if result.Warning != "" {
        fmt.Fprintf(os.Stderr, "%s\n\n", result.Warning)
    }
    if result.Blocked {
        return fmt.Errorf("query blocked by guardrail")
    }
    sql = result.SQL
}
```

### TUI Path (`internal/tui/repl.go`)

The TUI has two execution functions:

1. **`execQueryWithCtx()`** — runs user-entered SQL directly (guardrail is **not** applied here, since user-typed SQL is intentional)
2. **`validateAndExecSQL()`** — runs AI-generated SQL with validation + guardrails:

```go
// internal/tui/repl.go lines 1102-1112
if schema, err := db.LoadSchema(); err == nil {
    result := db.GuardrailSelectStar(sqlStr, schema)
    if result.Blocked {
        return queryResultMsg{err: fmt.Errorf("%s", result.Warning)}
    }
    if result.Warning != "" {
        (&m).addMessage(msgCmd, insightBubble(result.Warning))
    }
    sqlStr = result.SQL
}
```

In the TUI, warnings appear as styled insight bubbles rather than plain stderr output.

### Flow Diagram

```text
User Input (NL or SQL)
    │
    ├── Is SQL? → execQueryWithCtx (no guardrail)
    │
    └── Is NL? → AI generates SQL
                    │
                    ▼
              validateAndExecSQL
                    │
                    ├── EXPLAIN validation (syntax check)
                    ├── GuardrailSelectStar (SELECT * check)
                    │       │
                    │       ├── Warning → display to user
                    │       ├── Rewrite → modify SQL
                    │       └── Blocked → return error
                    │
                    ▼
              conn.Query() → execute
```

---

## 7. Schema Cache

The schema is cached locally as JSON to avoid repeated introspection:

| Detail | Value |
|--------|-------|
| **Cache path** | `~/.basemake/schema.json` |
| **Format** | JSON (indented) |
| **Trigger** | Automatic on `basemake connect` |
| **Refresh** | Manual via `basemake connect --refresh` or by deleting `~/.basemake/schema.json` |

```go
func cachePath() string {
    return filepath.Join(homeDir, ".basemake", "schema.json")
}
```

---

## 8. Appendix: Complete GuardrailResult Handling

### GuardrailResult struct fields

| Field | Type | Description |
|-------|------|-------------|
| `SQL` | `string` | The (possibly modified) SQL. If `Rewrote` is true, this contains the rewritten SQL with explicit columns. If `Blocked`, this is the original unmodified SQL. |
| `Warning` | `string` | Human-readable message. Empty string means no issues. |
| `Blocked` | `bool` | When true, the caller should **not** execute the query. The warning contains an explanation. |
| `Rewrote` | `bool` | When true, the SQL was modified (SELECT * expanded to explicit columns). |

### Caller Contract

| Scenario | `Blocked` | `Rewrote` | `SQL` | Action |
|----------|-----------|-----------|-------|--------|
| No SELECT * found | `false` | `false` | Original | Execute as-is |
| Small table (<10K) | `false` | `true` | Rewritten | Execute rewritten |
| Medium table (10K–1M) | `false` | `true` | Rewritten | Display warning, execute rewritten |
| Large table (>1M) | **`true`** | `false` | Original | **Do not execute**, show error |
| Unknown row count | `false` | `true` | Rewritten | Execute rewritten with warning |
| Complex query (can't extract table) | `false` | `false` | Original | Execute with generic warning |
| Table not in schema cache | `false` | `false` | Original | Execute with table-specific warning |
