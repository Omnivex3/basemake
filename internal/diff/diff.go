// Package diff computes structural schema differences between two databases.
// Use it to detect schema drift between environments (dev vs staging, staging vs prod).
package diff

import (
	"fmt"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/db"
)

// ChangeType describes what kind of schema change was detected.
type ChangeType string

const (
	TableAdded    ChangeType = "table_added"
	TableRemoved  ChangeType = "table_removed"
	ColumnAdded   ChangeType = "column_added"
	ColumnRemoved ChangeType = "column_removed"
	ColumnChanged ChangeType = "column_changed"
	IndexAdded    ChangeType = "index_added"
	IndexRemoved  ChangeType = "index_removed"
)

// Change represents a single schema difference.
type Change struct {
	Type    ChangeType `json:"type"`
	Table   string     `json:"table"`
	Column  string     `json:"column,omitempty"`
	Index   string     `json:"index,omitempty"`
	Old     string     `json:"old,omitempty"`
	New     string     `json:"new,omitempty"`
	Detail  string     `json:"detail,omitempty"`
}

// Report contains the full diff between two schemas.
type Report struct {
	From        string   `json:"from"`
	To          string   `json:"to"`
	Changes     []Change `json:"changes"`
	TablesOnly  bool     `json:"tables_only"`
	TablesAdded int      `json:"tables_added"`
	TablesRemoved int    `json:"tables_removed"`
	ChangesCount int     `json:"changes_count"`
}

// SchemaDiff compares two database schemas and returns a structured diff.
func SchemaDiff(from, to *db.Schema, fromName, toName string) *Report {
	r := &Report{
		From: fromName,
		To:   toName,
	}

	fromMap := make(map[string]*db.TableInfo)
	toMap := make(map[string]*db.TableInfo)

	for i := range from.Tables {
		fromMap[from.Tables[i].Name] = &from.Tables[i]
	}
	for i := range to.Tables {
		toMap[to.Tables[i].Name] = &to.Tables[i]
	}

	// Find added and removed tables, and diff common tables
	for name, toTable := range toMap {
		fromTable, exists := fromMap[name]
		if !exists {
			r.Changes = append(r.Changes, Change{
				Type:   TableAdded,
				Table:  name,
				Detail: fmt.Sprintf("Table %q exists in %s but not in %s", name, toName, fromName),
			})
			r.TablesAdded++
			continue
		}
		// Diff columns
		diffColumns(r, fromTable, toTable)
		// Diff indexes
		diffIndexes(r, fromTable, toTable)
	}

	for name := range fromMap {
		if _, exists := toMap[name]; !exists {
			r.Changes = append(r.Changes, Change{
				Type:   TableRemoved,
				Table:  name,
				Detail: fmt.Sprintf("Table %q exists in %s but not in %s", name, fromName, toName),
			})
			r.TablesRemoved++
		}
	}

	r.ChangesCount = len(r.Changes)
	return r
}

func diffColumns(r *Report, from, to *db.TableInfo) {
	fromCols := make(map[string]*db.ColumnInfo)
	toCols := make(map[string]*db.ColumnInfo)

	for i := range from.Columns {
		fromCols[from.Columns[i].Name] = &from.Columns[i]
	}
	for i := range to.Columns {
		toCols[to.Columns[i].Name] = &to.Columns[i]
	}

	for name, tc := range toCols {
		fc, exists := fromCols[name]
		if !exists {
			r.Changes = append(r.Changes, Change{
				Type:   ColumnAdded,
				Table:  to.Name,
				Column: name,
				Detail: fmt.Sprintf("Column %q.%s added", to.Name, name),
				New:    describeColumn(tc),
			})
			continue
		}

		// Check type changes
		if fc.Type != tc.Type {
			r.Changes = append(r.Changes, Change{
				Type:   ColumnChanged,
				Table:  to.Name,
				Column: name,
				Detail: fmt.Sprintf("Column %q.%s type changed", to.Name, name),
				Old:    describeColumn(fc),
				New:    describeColumn(tc),
			})
			continue
		}

		// Check nullable changes
		if fc.IsNullable != tc.IsNullable {
			r.Changes = append(r.Changes, Change{
				Type:   ColumnChanged,
				Table:  to.Name,
				Column: name,
				Detail: fmt.Sprintf("Column %q.%s nullable changed", to.Name, name),
				Old:    describeColumn(fc),
				New:    describeColumn(tc),
			})
			continue
		}

		// Check PK changes
		if fc.IsPK != tc.IsPK {
			r.Changes = append(r.Changes, Change{
				Type:   ColumnChanged,
				Table:  to.Name,
				Column: name,
				Detail: fmt.Sprintf("Column %q.%s primary key changed", to.Name, name),
				Old:    describeColumn(fc),
				New:    describeColumn(tc),
			})
		}

		// Check default changes (skip empty vs nil)
		if fc.Default != tc.Default && !(fc.Default == "" && tc.Default == "") {
			r.Changes = append(r.Changes, Change{
				Type:   ColumnChanged,
				Table:  to.Name,
				Column: name,
				Detail: fmt.Sprintf("Column %q.%s default changed", to.Name, name),
				Old:    describeColumn(fc),
				New:    describeColumn(tc),
			})
		}
	}

	for name := range fromCols {
		if _, exists := toCols[name]; !exists {
			r.Changes = append(r.Changes, Change{
				Type:   ColumnRemoved,
				Table:  to.Name,
				Column: name,
				Detail: fmt.Sprintf("Column %q.%s removed", to.Name, name),
				Old:    describeColumn(fromCols[name]),
			})
		}
	}
}

func diffIndexes(r *Report, from, to *db.TableInfo) {
	fromIdxs := make(map[string]*db.IndexInfo)
	toIdxs := make(map[string]*db.IndexInfo)

	for i := range from.Indexes {
		fromIdxs[from.Indexes[i].Name] = &from.Indexes[i]
	}
	for i := range to.Indexes {
		toIdxs[to.Indexes[i].Name] = &to.Indexes[i]
	}

	for name, ti := range toIdxs {
		if _, exists := fromIdxs[name]; !exists {
			r.Changes = append(r.Changes, Change{
				Type:  IndexAdded,
				Table: to.Name,
				Index: name,
				New:   describeIndex(ti),
			})
		}
	}

	for name := range fromIdxs {
		if _, exists := toIdxs[name]; !exists {
			r.Changes = append(r.Changes, Change{
				Type:  IndexRemoved,
				Table: to.Name,
				Index: name,
				Old:   describeIndex(fromIdxs[name]),
			})
		}
	}
}

func describeColumn(c *db.ColumnInfo) string {
	parts := []string{c.Type}
	if c.IsPK {
		parts = append(parts, "PK")
	}
	if c.IsNullable {
		parts = append(parts, "nullable")
	}
	if c.Default != "" {
		parts = append(parts, fmt.Sprintf("default=%s", c.Default))
	}
	return strings.Join(parts, " ")
}

func describeIndex(idx *db.IndexInfo) string {
	u := ""
	if idx.Unique {
		u = " unique"
	}
	return fmt.Sprintf("%q%s on (%s)", idx.Name, u, strings.Join(idx.Cols, ", "))
}

// FormatPlain renders the diff report as human-readable plain text.
func FormatPlain(r *Report) string {
	var b strings.Builder

	if r.From != "" && r.To != "" {
		fmt.Fprintf(&b, "Schema diff: %s → %s\n\n", r.From, r.To)
	}

	if len(r.Changes) == 0 {
		b.WriteString("✅ No differences found — schemas are identical.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "%d change(s) detected\n", len(r.Changes))
	fmt.Fprintf(&b, "%s\n\n", strings.Repeat("─", 50))

	// Group by table
	type changeGroup struct {
		Added   []Change
		Removed []Change
		Changed []Change
	}

	groups := make(map[string]*changeGroup)
	tableOrder := []string{}

	for _, c := range r.Changes {
		if _, ok := groups[c.Table]; !ok {
			groups[c.Table] = &changeGroup{}
			tableOrder = append(tableOrder, c.Table)
		}
		g := groups[c.Table]
		switch c.Type {
		case TableAdded:
			g.Added = append(g.Added, c)
		case TableRemoved:
			g.Removed = append(g.Removed, c)
		default:
			g.Changed = append(g.Changed, c)
		}
	}

	for _, table := range tableOrder {
		g := groups[table]
		tableLabel := table
		if table == "" {
			tableLabel = "(indexes)"
		}

		fmt.Fprintf(&b, "  %s:\n", tableLabel)
		for _, c := range g.Added {
			fmt.Fprintf(&b, "    + %s\n", c.Detail)
		}
		for _, c := range g.Removed {
			fmt.Fprintf(&b, "    - %s\n", c.Detail)
		}
		for _, c := range g.Changed {
			fmt.Fprintf(&b, "    ~ %s\n", c.Detail)
			if c.Old != "" || c.New != "" {
				fmt.Fprintf(&b, "        was: %s\n", c.Old)
				fmt.Fprintf(&b, "        now: %s\n", c.New)
			}
		}
	}

	return b.String()
}


