package display

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Format represents the output format for query results
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
	FormatTSV   Format = "tsv"
)

// Result holds query result data for formatting
type Result struct {
	Columns []string
	Rows    [][]string
	Message string // optional footer message, e.g. "(3 rows)"
}

// Print writes the result in the specified format to the writer
func Print(w io.Writer, res Result, format Format) error {
	switch format {
	case FormatJSON:
		return printJSON(w, res)
	case FormatCSV:
		return printCSV(w, res)
	case FormatTSV:
		return printTSV(w, res)
	default:
		return printTable(w, res)
	}
}

// printTable outputs aligned table format (like psql)
//
//	 id | name
//	----+------
//	  1 | Alice
//	(1 row)
func printTable(w io.Writer, res Result) error {
	if len(res.Columns) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(res.Columns))
	for i, col := range res.Columns {
		widths[i] = len(col)
	}
	for _, row := range res.Rows {
		for i, val := range row {
			if len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	// Print header
	for i, col := range res.Columns {
		if i > 0 {
			fmt.Fprint(w, " | ")
		}
		fmt.Fprintf(w, "%-*s", widths[i], col)
	}
	fmt.Fprintln(w)

	// Print separator
	for i := range res.Columns {
		if i > 0 {
			fmt.Fprint(w, "-+-")
		}
		fmt.Fprint(w, strings.Repeat("-", widths[i]))
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range res.Rows {
		for i, val := range row {
			if i > 0 {
				fmt.Fprint(w, " | ")
			}
			// Left-align text, right-align numbers
			if isNumeric(val) {
				fmt.Fprintf(w, "%*s", widths[i], val)
			} else {
				fmt.Fprintf(w, "%-*s", widths[i], val)
			}
		}
		fmt.Fprintln(w)
	}

	// Footer message
	if res.Message != "" {
		fmt.Fprintln(w, res.Message)
	}

	return nil
}

// printJSON outputs results as a JSON array of objects
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

// printCSV outputs results in CSV format
func printCSV(w io.Writer, res Result) error {
	cw := csv.NewWriter(w)

	// Header
	if err := cw.Write(res.Columns); err != nil {
		return err
	}

	// Rows
	for _, row := range res.Rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}

// printTSV outputs results in tab-separated format
func printTSV(w io.Writer, res Result) error {
	// Header
	fmt.Fprintln(w, strings.Join(res.Columns, "\t"))

	// Rows
	for _, row := range res.Rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return nil
}

// isNumeric checks if a value looks like a number (for alignment)
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	// Allow negative sign
	start := 0
	if s[0] == '-' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	hasDot := false
	for i := start; i < len(s); i++ {
		if s[i] == '.' && !hasDot {
			hasDot = true
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
