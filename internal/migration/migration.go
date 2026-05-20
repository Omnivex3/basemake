// Package migration detects DDL changes in SQL migration files and cross-references
// them against query profiles to estimate the performance impact of schema changes.
package migration

import (
	"fmt"
	"regexp"
	"strings"
)

// ChangeType identifies the kind of DDL change detected.
type ChangeType string

const (
	ChangeDropIndex      ChangeType = "drop_index"
	ChangeCreateIndex    ChangeType = "create_index"
	ChangeDropColumn     ChangeType = "drop_column"
	ChangeAlterColumnType ChangeType = "alter_column_type"
	ChangeDropTable      ChangeType = "drop_table"
)

// DDLChange represents a single DDL operation detected in a migration file.
type DDLChange struct {
	Type   ChangeType
	Target string // name of the affected object (index, column, table)
	Table  string // containing table (empty for DROP TABLE)
	Schema string // schema if qualified
}

// String returns a one-line label like "DROP INDEX idx_status on orders".
func (c DDLChange) String() string {
	switch c.Type {
	case ChangeDropIndex:
		if c.Table != "" {
			return fmt.Sprintf("DROP INDEX %s on %s", c.Target, c.Table)
		}
		return fmt.Sprintf("DROP INDEX %s", c.Target)
	case ChangeCreateIndex:
		return fmt.Sprintf("CREATE INDEX %s on %s", c.Target, c.Table)
	case ChangeDropColumn:
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", c.Table, c.Target)
	case ChangeAlterColumnType:
		return fmt.Sprintf("ALTER TABLE %s ALTER %s TYPE", c.Table, c.Target)
	case ChangeDropTable:
		return fmt.Sprintf("DROP TABLE %s", c.Target)
	}
	return "unknown change"
}

// ParseMigration extracts all DDL changes from a migration SQL string that
// could affect query execution performance.
func ParseMigration(sql string) ([]DDLChange, error) {
	stmts := splitStatements(sql)
	var changes []DDLChange

	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if c := matchDropIndex(stmt); c != nil {
			changes = append(changes, *c)
		} else if c := matchCreateIndex(stmt); c != nil {
			changes = append(changes, *c)
		} else if c := matchDropColumn(stmt); c != nil {
			changes = append(changes, *c)
		} else if c := matchAlterColumnType(stmt); c != nil {
			changes = append(changes, *c)
		} else if c := matchDropTable(stmt); c != nil {
			changes = append(changes, *c)
		}
	}
	return changes, nil
}

// splitStatements splits SQL into individual statements at semicolons,
// respecting dollar-quoted strings to avoid splitting inside function bodies.
func splitStatements(sql string) []string {
	var stmts []string
	var buf strings.Builder
	inDollar := false
	dollarTag := ""

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		// Enter dollar-quoted string: $$ or $tag$
		if ch == '$' && !inDollar {
			j := i + 1
			start := j
			for j < len(sql) && sql[j] != '$' {
				j++
			}
			if j < len(sql) && sql[j] == '$' {
				dollarTag = sql[start:j]
				inDollar = true
				buf.WriteByte(ch)
				continue
			}
		}

		// Exit dollar-quoted string
		if inDollar && ch == '$' {
			endTagLen := len(dollarTag) + 1 // $ + tag + $
			if i+endTagLen <= len(sql) && sql[i:i+endTagLen] == "$"+dollarTag+"$" {
				inDollar = false
				dollarTag = ""
			}
		}

		// Split on semicolons outside dollar quotes
		if !inDollar && ch == ';' {
			stmts = append(stmts, buf.String())
			buf.Reset()
			continue
		}

		buf.WriteByte(ch)
	}

	// Last statement (may not end with semicolon)
	if buf.Len() > 0 {
		stmts = append(stmts, buf.String())
	}

	return stmts
}

// --- Regex patterns for DDL matching ---

var (
	// DROP INDEX [CONCURRENTLY] [IF EXISTS] [schema.]name [ON [schema.]table] [CASCADE|RESTRICT]
	dropIndexRE = regexp.MustCompile(`(?i)drop\s+index\s+(?:concurrently\s+)?(?:if\s+exists\s+)?(?:(\w+)\.)?(\w+)(?:\s+on\s+(?:(\w+)\.)?(\w+))?(?:\s+(?:cascade|restrict))?`)

	// CREATE [UNIQUE] INDEX [CONCURRENTLY] [IF NOT EXISTS] [schema.]name ON [schema.]table ...
	createIndexRE = regexp.MustCompile(`(?i)create\s+(?:unique\s+)?index\s+(?:concurrently\s+)?(?:if\s+not\s+exists\s+)?(?:(\w+)\.)?(\w+)\s+on\s+(?:only\s+)?(?:(\w+)\.)?(\w+)`)

	// ALTER TABLE [ONLY] [schema.]table DROP [COLUMN] [IF EXISTS] column
	dropColumnRE = regexp.MustCompile(`(?i)alter\s+table\s+(?:only\s+)?(?:(\w+)\.)?(\w+)\s+drop\s+(?:column\s+)?(?:if\s+exists\s+)?(\w+)`)

	// ALTER TABLE [ONLY] [schema.]table ALTER [COLUMN] column [SET DATA] TYPE ...
	alterColumnTypeRE = regexp.MustCompile(`(?i)alter\s+table\s+(?:only\s+)?(?:(\w+)\.)?(\w+)\s+alter\s+(?:column\s+)?(\w+)\s+(?:set\s+data\s+)?type\s+`)

	// DROP TABLE [IF EXISTS] [schema.]table [CASCADE|RESTRICT]
	dropTableRE = regexp.MustCompile(`(?i)drop\s+table\s+(?:if\s+exists\s+)?(?:(\w+)\.)?(\w+?)(?:\s+(?:cascade|restrict))?$`)
)

func matchDropIndex(stmt string) *DDLChange {
	m := dropIndexRE.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	schema := m[1]
	name := m[2]
	table := m[4] // ON table
	if table == "" {
		table = m[3] // if no "ON", m[3] is empty
	}
	return &DDLChange{Type: ChangeDropIndex, Target: name, Table: table, Schema: schema}
}

func matchCreateIndex(stmt string) *DDLChange {
	m := createIndexRE.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	table := m[4]
	if table == "" {
		table = m[3]
	}
	return &DDLChange{Type: ChangeCreateIndex, Target: m[2], Table: table, Schema: m[1]}
}

func matchDropColumn(stmt string) *DDLChange {
	m := dropColumnRE.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	return &DDLChange{Type: ChangeDropColumn, Target: m[3], Table: m[2], Schema: m[1]}
}

func matchAlterColumnType(stmt string) *DDLChange {
	m := alterColumnTypeRE.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	return &DDLChange{Type: ChangeAlterColumnType, Target: m[3], Table: m[2], Schema: m[1]}
}

func matchDropTable(stmt string) *DDLChange {
	m := dropTableRE.FindStringSubmatch(stmt)
	if m == nil {
		return nil
	}
	return &DDLChange{Type: ChangeDropTable, Target: m[2], Schema: m[1]}
}
