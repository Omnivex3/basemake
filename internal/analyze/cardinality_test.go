package analyze

import (
	"math"
	"testing"
)

// ─── MCV Parser Tests ───────────────────────────────────────────────────────

func TestParseMCV_StringValues(t *testing.T) {
	raw := `{shipped,delivered,cancelled,processing,pending,returned}`
	vals := ParseMCV(raw)
	if len(vals) != 6 {
		t.Fatalf("got %d vals, want 6: %v", len(vals), vals)
	}
	expected := []string{"shipped", "delivered", "cancelled", "processing", "pending", "returned"}
	for i, e := range expected {
		if vals[i] != e {
			t.Errorf("vals[%d] = %q, want %q", i, vals[i], e)
		}
	}
}

func TestParseMCV_QuotedValues(t *testing.T) {
	raw := `{"User_1","User_2","User_3"}`
	vals := ParseMCV(raw)
	if len(vals) != 3 {
		t.Fatalf("got %d vals, want 3", len(vals))
	}
	if vals[0] != "User_1" {
		t.Errorf("vals[0] = %q, want %q", vals[0], "User_1")
	}
}

func TestParseMCV_Empty(t *testing.T) {
	if vals := ParseMCV(""); vals != nil {
		t.Errorf("expected nil, got %v", vals)
	}
	if vals := ParseMCV("{}"); vals != nil {
		t.Errorf("expected nil for {}, got %v", vals)
	}
}

func TestParseMCV_Numeric(t *testing.T) {
	raw := `{8,10,7,9,2,4,3,5,6,1,11}`
	vals := ParseMCV(raw)
	if len(vals) != 11 {
		t.Fatalf("got %d vals, want 11", len(vals))
	}
	if vals[0] != "8" || vals[10] != "11" {
		t.Errorf("unexpected values: %v", vals)
	}
}

func TestParseMCF(t *testing.T) {
	raw := `{0.20266667,0.20026666,0.19863333,0.19806667,0.1007,0.09966667}`
	freqs := ParseMCF(raw)
	if len(freqs) != 6 {
		t.Fatalf("got %d freqs, want 6", len(freqs))
	}
	if math.Abs(freqs[0]-0.20266667) > 0.0001 {
		t.Errorf("freqs[0] = %f, want ~0.2027", freqs[0])
	}
	// Sum should be ~1.0
	sum := 0.0
	for _, f := range freqs {
		sum += f
	}
	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("MCV sum = %f, want ~1.0", sum)
	}
}

func TestParseMCF_Empty(t *testing.T) {
	if freqs := ParseMCF(""); freqs != nil {
		t.Errorf("expected nil, got %v", freqs)
	}
	if freqs := ParseMCF("{}"); freqs != nil {
		t.Errorf("expected nil for {}, got %v", freqs)
	}
}

// ─── Selectivity Tests ──────────────────────────────────────────────────────

func TestSelectivity_MCVValue(t *testing.T) {
	// stress_orders.status: 6 values, evenly distributed
	cs := ColumnStats{
		NDistinct: 6,
		NullFrac:  0,
		MCV:       []string{"shipped", "delivered", "cancelled", "processing", "pending", "returned"},
		MCF:       []float64{0.20266667, 0.20026666, 0.19863333, 0.19806667, 0.1007, 0.09966667},
	}

	sel := cs.Selectivity("pending", 2_000_000)
	if sel < 0.09 || sel > 0.11 {
		t.Errorf("selectivity for 'pending' = %f, want ~0.10", sel)
	}

	rows := cs.EstimateRows("pending", 2_000_000)
	if rows < 180_000 || rows > 220_000 {
		t.Errorf("estimated rows for 'pending' = %d, want ~200K", rows)
	}
}

func TestSelectivity_NonMCVValue(t *testing.T) {
	// stress_products.category: 10 distinct, MCV covers all.
	// A value not in MCV should get 0 remaining spread.
	cs := ColumnStats{
		NDistinct: 10,
		MCV:       []string{"music", "books", "office", "clothing", "food", "furniture", "sports", "tools", "electronics", "toys"},
		MCF:       []float64{0.115166664, 0.112733334, 0.1118, 0.11136667, 0.11136667, 0.11033333, 0.1087, 0.108666666, 0.0552, 0.05466667},
	}

	sel := cs.Selectivity("unknown_category", 500_000)
	if sel != 0 {
		t.Errorf("selectivity for unknown value = %f, want 0 (all values in MCV)", sel)
	}
}

func TestSelectivity_EstimatedReturnsMinusOne(t *testing.T) {
	// n_distinct = -1 means PG estimated it
	cs := ColumnStats{
		NDistinct: -1,
		MCV:       nil,
	}
	if !cs.IsEstimated() {
		t.Error("expected IsEstimated() = true for n_distinct=-1")
	}
	sel := cs.Selectivity("anything", 1000)
	if sel != -1 {
		t.Errorf("selectivity for estimated stats = %f, want -1", sel)
	}
}

func TestSelectivity_FractionalDistinct(t *testing.T) {
	// stress_orders.user_id: n_distinct = -0.2071435 (≈414K distinct in 2M rows)
	cs := ColumnStats{
		NDistinct: -0.2071435,
		MCV:       nil,
		MCF:       nil,
	}

	distinct := cs.ExactDistinct(2_000_000)
	if distinct < 400_000 || distinct > 420_000 {
		t.Errorf("ExactDistinct(2M) = %f, want ~414K", distinct)
	}

	sel := cs.Selectivity("some_user", 2_000_000)
	if sel < 0 {
		t.Error("selectivity should not be -1 for fractional distinct")
	}
	if sel < 0.000001 || sel > 0.00001 {
		t.Errorf("selectivity = %f, want ~0.000005 (1/207K)", sel)
	}
}

// ─── Extract Columns from Filter ────────────────────────────────────────────

func TestExtractColumns_SimpleEquals(t *testing.T) {
	cols := extractColumnsFromFilter("(status = 'pending')")
	if len(cols) != 1 || cols[0] != "status" {
		t.Errorf("got %v, want [status]", cols)
	}
}

func TestExtractColumns_TableQualified(t *testing.T) {
	cols := extractColumnsFromFilter("(orders.status = 'pending')")
	if len(cols) != 1 || cols[0] != "status" {
		t.Errorf("got %v, want [status]", cols)
	}
}

func TestExtractColumns_TypeCast(t *testing.T) {
	cols := extractColumnsFromFilter("(created_at::date = '2026-01-01')")
	if len(cols) != 1 || cols[0] != "created_at" {
		t.Errorf("got %v, want [created_at]", cols)
	}
}

func TestExtractColumns_INClause(t *testing.T) {
	cols := extractColumnsFromFilter("(category IN ('electronics', 'books'))")
	if len(cols) != 1 || cols[0] != "category" {
		t.Errorf("got %v, want [category]", cols)
	}
}

func TestExtractColumns_ISNULL(t *testing.T) {
	cols := extractColumnsFromFilter("shipped_at IS NULL")
	if len(cols) != 1 || cols[0] != "shipped_at" {
		t.Errorf("got %v, want [shipped_at]", cols)
	}
}

func TestExtractColumns_Between(t *testing.T) {
	cols := extractColumnsFromFilter("(price BETWEEN 100 AND 200)")
	if len(cols) != 1 || cols[0] != "price" {
		t.Errorf("got %v, want [price]", cols)
	}
}

func TestExtractColumns_Multiple(t *testing.T) {
	cols := extractColumnsFromFilter("(status = 'active' AND plan = 'pro')")
	if len(cols) != 2 {
		t.Fatalf("got %d cols, want 2: %v", len(cols), cols)
	}
	if cols[0] != "status" || cols[1] != "plan" {
		t.Errorf("got %v, want [status, plan]", cols)
	}
}

func TestExtractColumns_GreaterThan(t *testing.T) {
	cols := extractColumnsFromFilter("(score > 5000)")
	if len(cols) != 1 || cols[0] != "score" {
		t.Errorf("got %v, want [score]", cols)
	}
}

func TestExtractColumns_LIKE(t *testing.T) {
	cols := extractColumnsFromFilter("(name LIKE 'John%')")
	if len(cols) != 1 || cols[0] != "name" {
		t.Errorf("got %v, want [name]", cols)
	}
}

func TestExtractColumns_IN_StringList(t *testing.T) {
	cols := extractColumnsFromFilter("(status IN ('pending', 'processing'))")
	if len(cols) != 1 || cols[0] != "status" {
		t.Errorf("got %v, want [status]", cols)
	}
}

func TestExtractColumns_BETWEEN(t *testing.T) {
	cols := extractColumnsFromFilter("(price BETWEEN 100 AND 200)")
	if len(cols) != 1 || cols[0] != "price" {
		t.Errorf("got %v, want [price]", cols)
	}
}

func TestExtractColumns_DeepNestedParens(t *testing.T) {
	cols := extractColumnsFromFilter("(((status = 'active') AND (plan = 'pro')))")
	if len(cols) != 2 {
		t.Fatalf("got %d cols, want 2: %v", len(cols), cols)
	}
	if cols[0] != "status" || cols[1] != "plan" {
		t.Errorf("got %v, want [status, plan]", cols)
	}
}

func TestExtractColumns_NotEqual(t *testing.T) {
	cols := extractColumnsFromFilter("(status <> 'deleted')")
	if len(cols) != 1 || cols[0] != "status" {
		t.Errorf("got %v, want [status]", cols)
	}
}

func TestExtractColumns_IS_NOT_NULL(t *testing.T) {
	cols := extractColumnsFromFilter("(shipped_at IS NOT NULL)")
	if len(cols) != 1 || cols[0] != "shipped_at" {
		t.Errorf("got %v, want [shipped_at]", cols)
	}
}

func TestExtractColumns_NoOperator(t *testing.T) {
	// No recognizable operator — should return empty
	cols := extractColumnsFromFilter("(1)")
	if len(cols) != 0 {
		t.Errorf("expected no columns for literal-only filter, got %v", cols)
	}
}

func TestExtractColumns_MixedOperators(t *testing.T) {
	// Complex filter with multiple operators
	cols := extractColumnsFromFilter("(total > 100 AND status = 'active' AND created_at >= '2026-01-01')")
	if len(cols) != 3 {
		t.Fatalf("got %d cols, want 3: %v", len(cols), cols)
	}
	// Order may vary, check all present
	expected := map[string]bool{"total": true, "status": true, "created_at": true}
	for _, c := range cols {
		if !expected[c] {
			t.Errorf("unexpected column: %s", c)
		}
	}
}

// ─── Partial Index Detection ────────────────────────────────────────────────

func TestDetectPartialIndex_SelectiveValue(t *testing.T) {
	// stress_orders.status: pending = 10%
	cs := ColumnStats{
		NDistinct: 6,
		MCV:       []string{"shipped", "delivered", "cancelled", "processing", "pending", "returned"},
		MCF:       []float64{0.20, 0.20, 0.20, 0.20, 0.10, 0.10},
	}
	clause := detectPartialIndex("(status = 'pending')", "status", cs, 2_000_000)
	if clause != "WHERE status = 'pending'" {
		t.Errorf("got %q, want \"WHERE status = 'pending'\"", clause)
	}
}

func TestDetectPartialIndex_CommonValue(t *testing.T) {
	// stress_users.plan: free = 60% — too common for partial index
	cs := ColumnStats{
		NDistinct: 3,
		MCV:       []string{"free", "pro", "enterprise"},
		MCF:       []float64{0.60, 0.34, 0.06},
	}
	clause := detectPartialIndex("(plan = 'free')", "plan", cs, 500_000)
	if clause != "" {
		t.Errorf("expected no partial index (60%% is too common), got %q", clause)
	}
}

func TestDetectPartialIndex_RareValue(t *testing.T) {
	// stress_users.plan: enterprise = 6% — good partial candidate
	cs := ColumnStats{
		NDistinct: 3,
		MCV:       []string{"free", "pro", "enterprise"},
		MCF:       []float64{0.60, 0.34, 0.06},
	}
	clause := detectPartialIndex("(plan = 'enterprise')", "plan", cs, 500_000)
	if clause != "WHERE plan = 'enterprise'" {
		t.Errorf("got %q, want \"WHERE plan = 'enterprise'\"", clause)
	}
}

func TestDetectPartialIndex_NoMatch(t *testing.T) {
	cs := ColumnStats{NDistinct: 6}
	clause := detectPartialIndex("(total > 100)", "total", cs, 2_000_000)
	if clause != "" {
		t.Errorf("expected no partial for > operator, got %q", clause)
	}
}

// ─── Trade-off Estimates ────────────────────────────────────────────────────

func TestEstimateTradeoffs(t *testing.T) {
	cs := ColumnStats{AvgWidth: 9}
	tradeoffs := estimateTradeoffs("stress_orders", "status", cs)
	if len(tradeoffs) < 2 {
		t.Fatalf("expected at least 2 tradeoffs (INSERT + UPDATE), got %d: %v", len(tradeoffs), tradeoffs)
	}
	if tradeoffs[0] != "+~3% INSERT overhead on stress_orders" {
		t.Errorf("INSERT tradeoff = %q", tradeoffs[0])
	}
	if tradeoffs[1] != "+~5% UPDATE overhead on stress_orders.status" {
		t.Errorf("UPDATE tradeoff = %q", tradeoffs[1])
	}
}

func TestEstimateTradeoffs_HighNullFrac(t *testing.T) {
	cs := ColumnStats{AvgWidth: 4, NullFrac: 0.7}
	tradeoffs := estimateTradeoffs("orders", "shipped_at", cs)
	hasNullNote := false
	for _, tr := range tradeoffs {
		if tr == "70% NULL values — partial index could skip nulls" {
			hasNullNote = true
		}
	}
	if !hasNullNote {
		t.Errorf("expected null fraction note, got: %v", tradeoffs)
	}
}

// ─── Suggest Indexes from Scan ──────────────────────────────────────────────

func TestSuggestIndexesForScan_WithPartial(t *testing.T) {
	stats := &TableStats{
		Name:      "stress_orders",
		TotalRows: 2_000_000,
		Columns: map[string]ColumnStats{
			"status": {
				NDistinct: 6,
				MCV:       []string{"shipped", "delivered", "cancelled", "processing", "pending", "returned"},
				MCF:       []float64{0.20, 0.20, 0.20, 0.20, 0.10, 0.10},
				AvgWidth:  9,
			},
		},
	}

	suggestions := SuggestIndexesForScan("stress_orders", "(status = 'pending')", 10000, stats)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions, got none")
	}

	sug := suggestions[0]
	if sug.Table != "stress_orders" {
		t.Errorf("table = %q, want stress_orders", sug.Table)
	}
	if sug.PartialWhere != "WHERE status = 'pending'" {
		t.Errorf("partial = %q, want WHERE status = 'pending'", sug.PartialWhere)
	}
	if sug.Confidence != "high" {
		t.Errorf("confidence = %q, want high", sug.Confidence)
	}
	if len(sug.Tradeoffs) == 0 {
		t.Error("expected tradeoffs")
	}
}

func TestSuggestIndexesForScan_SpeculativeStats(t *testing.T) {
	// n_distinct = -1 (estimated, not sampled)
	stats := &TableStats{
		Name:      "stress_users",
		TotalRows: 500_000,
		Columns: map[string]ColumnStats{
			"email": {NDistinct: -1, AvgWidth: 23},
		},
	}

	suggestions := SuggestIndexesForScan("stress_users", "(email = 'test@example.com')", 50000, stats)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions even with speculative stats")
	}
	if suggestions[0].Confidence != "speculative" {
		t.Errorf("confidence = %q, want speculative", suggestions[0].Confidence)
	}
}

func TestSuggestIndexesForScan_NoFilter(t *testing.T) {
	stats := &TableStats{
		Name: "stress_orders",
		Columns: map[string]ColumnStats{
			"status": {NDistinct: 6},
		},
	}
	suggestions := SuggestIndexesForScan("stress_orders", "", 1000, stats)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions without filter, got %d", len(suggestions))
	}
}

func TestSuggestIndexesForScan_NilStats(t *testing.T) {
	suggestions := SuggestIndexesForScan("ghost_table", "(status = 'x')", 1000, nil)
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions for nil stats, got %d", len(suggestions))
	}
}

// ─── Edge Cases ─────────────────────────────────────────────────────────────

func TestIsEstimated(t *testing.T) {
	cs := ColumnStats{NDistinct: -1}
	if !cs.IsEstimated() {
		t.Error("n_distinct=-1 should be estimated")
	}
	cs = ColumnStats{NDistinct: 6}
	if cs.IsEstimated() {
		t.Error("n_distinct=6 should not be estimated")
	}
	cs = ColumnStats{NDistinct: -0.5}
	if cs.IsEstimated() {
		t.Error("n_distinct=-0.5 should not be estimated (fractional)")
	}
}

func TestExactDistinct(t *testing.T) {
	if d := (ColumnStats{NDistinct: 6}).ExactDistinct(100); d != 6 {
		t.Errorf("exact: got %f, want 6", d)
	}
	if d := (ColumnStats{NDistinct: -0.5}).ExactDistinct(100); d != 50 {
		t.Errorf("fractional: got %f, want 50", d)
	}
	if d := (ColumnStats{NDistinct: -1}).ExactDistinct(100); d != 0 {
		t.Errorf("estimated: got %f, want 0", d)
	}
}

func TestExtractColumns_Keywords(t *testing.T) {
	// "and" and "or" should be filtered out
	cols := extractColumnsFromFilter("(status = 'active' AND plan = 'pro')")
	for _, c := range cols {
		if c == "and" || c == "or" {
			t.Errorf("keyword '%s' should not be extracted as column", c)
		}
	}
}

func TestExtractColumns_EmptyFilter(t *testing.T) {
	cols := extractColumnsFromFilter("")
	if len(cols) != 0 {
		t.Errorf("expected empty, got %v", cols)
	}
}
