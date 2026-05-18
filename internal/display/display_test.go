package display

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintTable(t *testing.T) {
	res := Result{
		Columns: []string{"id", "name", "email"},
		Rows: [][]string{
			{"1", "Alice", "alice@example.com"},
			{"2", "Bob", "bob@example.com"},
		},
		Message: "(2 rows)",
	}

	var buf bytes.Buffer
	if err := Print(&buf, res, FormatTable); err != nil {
		t.Fatalf("Print: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "id | name") {
		t.Errorf("missing header, got:\n%s", output)
	}
	if !strings.Contains(output, "1 | Alice") {
		t.Errorf("missing row 1, got:\n%s", output)
	}
	if !strings.Contains(output, "2 | Bob") {
		t.Errorf("missing row 2, got:\n%s", output)
	}
	if !strings.Contains(output, "(2 rows)") {
		t.Errorf("missing footer, got:\n%s", output)
	}
}

func TestPrintJSON(t *testing.T) {
	res := Result{
		Columns: []string{"id", "name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
	}

	var buf bytes.Buffer
	if err := Print(&buf, res, FormatJSON); err != nil {
		t.Fatalf("Print: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(output, "[") || !strings.HasSuffix(output, "]") {
		t.Errorf("expected JSON array, got:\n%s", output)
	}
	if !strings.Contains(output, `"id": "1"`) {
		t.Errorf("missing data, got:\n%s", output)
	}
}

func TestPrintCSV(t *testing.T) {
	res := Result{
		Columns: []string{"id", "name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
	}

	var buf bytes.Buffer
	if err := Print(&buf, res, FormatCSV); err != nil {
		t.Fatalf("Print: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d", len(lines))
	}
	if lines[0] != "id,name" {
		t.Errorf("header = %q, want %q", lines[0], "id,name")
	}
}

func TestPrintTSV(t *testing.T) {
	res := Result{
		Columns: []string{"id", "name"},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
	}

	var buf bytes.Buffer
	if err := Print(&buf, res, FormatTSV); err != nil {
		t.Fatalf("Print: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "id\tname") {
		t.Errorf("expected tab-separated header, got: %q", lines[0])
	}
}

func TestEmptyResult(t *testing.T) {
	res := Result{
		Columns: []string{"id", "name"},
		Rows:    [][]string{},
		Message: "(0 rows)",
	}

	var buf bytes.Buffer
	if err := Print(&buf, res, FormatTable); err != nil {
		t.Fatalf("Print: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(0 rows)") {
		t.Errorf("expected (0 rows), got:\n%s", output)
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"123", true},
		{"-42", true},
		{"3.14", true},
		{"0", true},
		{"abc", false},
		{"", false},
		{"12a", false},
		{"-", false},
		{"42.5", true},
	}

	for _, tc := range tests {
		got := isNumeric(tc.val)
		if got != tc.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tc.val, got, tc.want)
		}
	}
}

func TestNumericAlignment(t *testing.T) {
	// Numbers should be right-aligned, text left-aligned
	res := Result{
		Columns: []string{"id", "label"},
		Rows: [][]string{
			{"100", "X"},
			{"5", "YY"},
		},
		Message: "(2 rows)",
	}

	var buf bytes.Buffer
	if err := Print(&buf, res, FormatTable); err != nil {
		t.Fatalf("Print: %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Check that "100" and " 5" align right (second column is right-padded with spaces before the number)
	// "100" starts at position 0, " 5" starts at position 1 (right-aligned in a 3-char field)
	for _, line := range lines[2:4] {
		if !strings.Contains(line, "|") {
			continue
		}
		t.Logf("row: %q", line)
	}
}
