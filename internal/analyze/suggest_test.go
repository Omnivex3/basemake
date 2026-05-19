package analyze

import (
	"testing"
)

func TestExtractColumns_GreaterThanWithCast(t *testing.T) {
	// Match the actual EXPLAIN output format
	cols := extractColumnsFromFilter("(total > '500'::numeric)")
	if len(cols) != 1 || cols[0] != "total" {
		t.Errorf("got %v, want [total]", cols)
	}
}

func TestExtractColumns_StressOrdersFilter(t *testing.T) {
	// The actual filter from basemake analyze on stressdb
	filters := []struct {
		filter string
		cols   []string
	}{
		{"(total > '500'::numeric)", []string{"total"}},
		{"(status = 'pending')", []string{"status"}},
		{"(score > 8000)", []string{"score"}},
	}
	for _, tc := range filters {
		cols := extractColumnsFromFilter(tc.filter)
		if len(cols) != len(tc.cols) {
			t.Errorf("filter %q: got %v, want %v", tc.filter, cols, tc.cols)
			continue
		}
		for i, c := range cols {
			if c != tc.cols[i] {
				t.Errorf("filter %q: cols[%d] = %q, want %q", tc.filter, i, c, tc.cols[i])
			}
		}
	}
}

func TestSuggestIndexesForScan_StressOrdersTotal(t *testing.T) {
	stats := &TableStats{
		Name:      "stress_orders",
		TotalRows: 2_000_000,
		Columns: map[string]ColumnStats{
			"total": {
				NDistinct: 95924,
				NullFrac:  0,
				AvgWidth:  6,
			},
		},
	}

	sugs := SuggestIndexesForScan("stress_orders", "(total > '500'::numeric)", 1018608, stats)
	if len(sugs) == 0 {
		t.Fatal("expected suggestions for total > 500, got none")
	}
	if sugs[0].Columns[0] != "total" {
		t.Errorf("column = %q, want total", sugs[0].Columns[0])
	}
	if sugs[0].CreateSQL != "CREATE INDEX idx_stress_orders_total ON stress_orders(total)" {
		t.Errorf("sql = %q", sugs[0].CreateSQL)
	}
}
