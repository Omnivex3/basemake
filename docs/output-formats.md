# Output Formats

`basemake supports 4 output formats for query results. The format is selected via CLI flags, config file, or defaults to table.

## Format Types

| Format  | Constant         | Selection Method            | Use Case                          |
|---------|------------------|-----------------------------|-----------------------------------|
| Table   | `FormatTable`    | Default / config fallback   | Human-readable terminal output    |
| JSON    | `FormatJSON`     | `--json` flag or config     | Programmatic consumption / APIs   |
| CSV     | `FormatCSV`      | `--csv` flag or config      | Spreadsheets / data import        |
| TSV     | `FormatTSV`      | Code-level constant only    | Unix pipelines / cut/sort/awk     |

Note: TSV exists at the code level but has no CLI flag or config exposure. It's defined in the `display` package but only callable programmatically.

## Table Format

The default output format, styled similarly to `psql` (PostgreSQL CLI).

### Features

- **Column-aligned**: Each column's width is calculated from the widest value (header or data)
- **Header separator**: `-+-` between columns, `---` under each column
- **Right-aligned numbers**: Numeric values (integers, decimals, negative numbers) are right-aligned
- **Left-aligned text**: Text values are left-aligned
- **NULL display**: NULL values are displayed as the string `"NULL"`
- **Footer**: Row count message `(N rows)` or `(N row)` for singular

### Example

```
 id | name  | email
----+-------+------------------
  1 | Alice | alice@example.com
  2 | Bob   | bob@example.com
(2 rows)
```

### Alignment Logic

The `isNumeric()` function determines alignment:

```go
func isNumeric(s string) bool {
    // Empty → false
    // Leading '-' allowed (negative numbers)
    // Single '.' allowed (decimal)
    // All other characters must be digits 0-9
    // Handles: "123", "-42", "3.14", "0"
    // Rejects: "abc", "", "12a", "-"
}
```

- Numeric: right-aligned with `fmt.Sprintf("%*s", width, val)`
- Non-numeric: left-aligned with `fmt.Sprintf("%-*s", width, val)`

### Empty Results

Zero-row results still show headers and separator:

```
 id | name
----+------
(0 rows)
```

### Edge Cases

- NULL values appear as the string "NULL"
- Empty strings appear as empty cells
- Zero columns edge case: `if len(res.Columns) == 0 { return nil }`
- Multi-line cell values are not supported (rendered as a single line)

## JSON Format

Outputs results as a JSON array of objects.

### Structure

```json
[
  {
    "id": "1",
    "name": "Alice",
    "email": "alice@example.com"
  },
  {
    "id": "2",
    "name": "Bob",
    "email": "bob@example.com"
  }
]
```

### Implementation

```go
func printJSON(w io.Writer, res Result) error {
    rows := make([]map[string]string, len(res.Rows))
    for i, row := range res.Rows {
        obj := make(map[string]string, len(res.Columns))
        for j, col := range res.Columns {
            obj[col] = row[j]
        }
        rows[i] = obj
    }
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    return enc.Encode(rows)
}
```

### Characteristics

- **Pretty-printed**: 2-space indentation via `json.Encoder.SetIndent`
- **All values as strings**: Every value (including numbers and NULL strings) is serialized as a string. This is a limitation — actual JSON types (null, number, boolean) are lost.
- **Column names as keys**: The column name is used directly as the JSON object key
- **Empty array**: Zero rows → `[]`
- **Trailing newline**: `json.Encoder.Encode` adds a newline after the JSON

### Limitations

- No null JSON value (NULL becomes `"NULL"` string)
- No numeric JSON types (numbers become `"123"` not `123`)
- No nested objects (all values are flat strings)

## CSV Format

Standard comma-separated values, using Go's `encoding/csv` package.

### Example

```
id,name,email
1,Alice,alice@example.com
2,Bob,bob@example.com
```

### Implementation

```go
func printCSV(w io.Writer, res Result) error {
    cw := csv.NewWriter(w)
    cw.Write(res.Columns)   // Header row
    for _, row := range res.Rows {
        cw.Write(row)       // Data rows
    }
    cw.Flush()
    return cw.Error()
}
```

### Characteristics

- **Header row**: First row is column names
- **Quoting**: Go's `encoding/csv` automatically quotes fields containing commas, quotes, or newlines
- **No row count**: The footer message is NOT appended to CSV output (unlike table format)
- **Row count on stderr**: When CSV format is active, the row count is printed to stderr instead
- **Empty rows**: An empty `[][]string{}` produces only the header line

## TSV Format

Tab-separated values, implemented with simple string joining (no quoting engine).

### Example

```
id\tname\temail
1\tAlice\talice@example.com
2\tBob\tbob@example.com
```

### Implementation

```go
func printTSV(w io.Writer, res Result) error {
    fmt.Fprintln(w, strings.Join(res.Columns, "\t"))
    for _, row := range res.Rows {
        fmt.Fprintln(w, strings.Join(row, "\t"))
    }
    return nil
}
```

### Characteristics

- **No quoting**: Values containing tabs or newlines will break the format
- **No header customization**: Header is always the column names
- **No row count footer**: Like CSV, row count goes to stderr
- **Unix-friendly**: Easy to pipe through `cut`, `awk`, `sort`
- **Not exposed via CLI**: Only usable programmatically through the `display` package

## Row Count Behavior

The row count message `"(N rows)"` is handled differently per format:

| Format | Location | Implementation |
|--------|----------|----------------|
| Table  | Inline (stdout) | Appended as final line in the output |
| JSON   | Stderr          | `fmt.Fprintf(os.Stderr, "\n%s\n", msg)` after output |
| CSV    | Stderr          | Same as JSON |
| TSV    | Stderr          | Same as JSON |

This ensures machine-parsable formats (JSON, CSV, TSV) don't have human-readable footer text mixed into the data.

## Column Type Handling

During query execution, each cell value is converted to string via a type switch:

```go
for i, v := range vals {
    switch val := v.(type) {
    case []byte:
        row[i] = string(val)
    case nil:
        row[i] = "NULL"
    default:
        row[i] = fmt.Sprint(val)
    }
}
```

- `[]byte`: Common for SQLite and MySQL string/text columns
- `nil`: SQL NULL values
- Everything else: `fmt.Sprint()` — handles int64, float64, bool, string, time.Time, etc.
