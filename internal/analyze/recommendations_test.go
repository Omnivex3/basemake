package analyze

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ─── RecStore: Load/Save ───────────────────────────────────────────────────

func TestRecStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	orig := recStorePath
	recStorePath = func() string { return filepath.Join(dir, "recommendations.json") }
	defer func() { recStorePath = orig }()

	store := &RecStore{
		Recommendations: []Recommendation{
			{
				ID:     "test_table_col",
				Status: "pending",
				Suggestion: IndexSuggestion{
					Table:     "test_table",
					Columns:   []string{"col"},
					CreateSQL: "CREATE INDEX ...",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		LastAnalyzed: time.Now(),
	}

	if err := store.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadRecs()
	if err != nil {
		t.Fatalf("LoadRecs: %v", err)
	}

	if len(loaded.Recommendations) != 1 {
		t.Fatalf("got %d recs, want 1", len(loaded.Recommendations))
	}
	if loaded.Recommendations[0].ID != "test_table_col" {
		t.Errorf("ID = %q, want test_table_col", loaded.Recommendations[0].ID)
	}
	if loaded.Recommendations[0].Status != "pending" {
		t.Errorf("Status = %q, want pending", loaded.Recommendations[0].Status)
	}
}

func TestRecStore_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	orig := recStorePath
	recStorePath = func() string { return filepath.Join(dir, "recommendations.json") }
	defer func() { recStorePath = orig }()

	store, err := LoadRecs()
	if err != nil {
		t.Fatalf("LoadRecs on missing file: %v", err)
	}
	if store == nil {
		t.Fatal("got nil store, expected empty")
	}
	if len(store.Recommendations) != 0 {
		t.Errorf("got %d recs, want 0", len(store.Recommendations))
	}
}

func TestRecStore_SaveCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", ".basemake")
	orig := recStorePath
	recStorePath = func() string { return filepath.Join(dir, "recommendations.json") }
	defer func() { recStorePath = orig }()

	store := &RecStore{}
	if err := store.Save(); err != nil {
		t.Fatalf("Save to nested dir: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}

// ─── RecStore: Merge ───────────────────────────────────────────────────────

func TestRecStore_Merge_NewSuggestion(t *testing.T) {
	store := &RecStore{}
	sugs := []IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "CREATE INDEX idx_orders_status ON orders(status)"},
	}

	store.Merge(sugs)

	if len(store.Recommendations) != 1 {
		t.Fatalf("got %d recs, want 1", len(store.Recommendations))
	}
	if store.Recommendations[0].ID != "orders_status" {
		t.Errorf("ID = %q, want orders_status", store.Recommendations[0].ID)
	}
	if store.Recommendations[0].Status != "pending" {
		t.Errorf("Status = %q, want pending", store.Recommendations[0].Status)
	}
}

func TestRecStore_Merge_Deduplicate(t *testing.T) {
	store := &RecStore{}
	sugs := []IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "CREATE INDEX idx_orders_status ON orders(status)"},
	}

	store.Merge(sugs)
	store.Merge(sugs) // same suggestion again

	if len(store.Recommendations) != 1 {
		t.Fatalf("got %d recs, want 1 (deduplicated)", len(store.Recommendations))
	}
}

func TestRecStore_Merge_UpdateExisting(t *testing.T) {
	store := &RecStore{}
	sugs := []IndexSuggestion{
		{
			Table:          "orders",
			Columns:        []string{"status"},
			CreateSQL:      "CREATE INDEX idx_orders_status ON orders(status)",
			EstImprovement: "Seq Scan → Index Scan (~50%)",
		},
	}

	store.Merge(sugs)

	// Merge with updated estimate
	sugs2 := []IndexSuggestion{
		{
			Table:          "orders",
			Columns:        []string{"status"},
			CreateSQL:      "CREATE INDEX idx_orders_status ON orders(status)",
			EstImprovement: "Seq Scan → Index Scan (~80%)",
		},
	}
	store.Merge(sugs2)

	if len(store.Recommendations) != 1 {
		t.Fatalf("got %d recs, want 1", len(store.Recommendations))
	}
	if store.Recommendations[0].Suggestion.EstImprovement != "Seq Scan → Index Scan (~80%)" {
		t.Errorf("EstImprovement not updated: %q", store.Recommendations[0].Suggestion.EstImprovement)
	}
}

func TestRecStore_Merge_DoesntUpdateApplied(t *testing.T) {
	store := &RecStore{}
	sugs := []IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "CREATE INDEX ..."},
	}
	store.Merge(sugs)
	store.Recommendations[0].Status = "applied"
	store.Recommendations[0].UpdatedAt = time.Now().Add(-24 * time.Hour)

	// Merge same suggestion again — should NOT update applied rec
	store.Merge(sugs)

	if len(store.Recommendations) != 1 {
		t.Fatalf("got %d recs", len(store.Recommendations))
	}
	if store.Recommendations[0].Status != "applied" {
		t.Errorf("Status changed to %q, should still be applied", store.Recommendations[0].Status)
	}
}

func TestRecStore_Merge_PartialIndex(t *testing.T) {
	store := &RecStore{}
	sugs := []IndexSuggestion{
		{
			Table:        "orders",
			Columns:      []string{"status"},
			PartialWhere: "WHERE status = 'pending'",
			CreateSQL:    "CREATE INDEX idx_orders_status_partial ON orders(status) WHERE status = 'pending'",
		},
	}

	store.Merge(sugs)

	if len(store.Recommendations) != 1 {
		t.Fatalf("got %d recs", len(store.Recommendations))
	}
	if store.Recommendations[0].ID != "orders_status_partial" {
		t.Errorf("partial index ID = %q, want orders_status_partial", store.Recommendations[0].ID)
	}
}

func TestRecStore_Merge_MultipleTables(t *testing.T) {
	store := &RecStore{}
	sugs := []IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "..."},
		{Table: "orders", Columns: []string{"total"}, CreateSQL: "..."},
		{Table: "users", Columns: []string{"email"}, CreateSQL: "..."},
	}

	store.Merge(sugs)

	if len(store.Recommendations) != 3 {
		t.Fatalf("got %d recs, want 3", len(store.Recommendations))
	}
}

// ─── RecStore: Apply ───────────────────────────────────────────────────────

func TestRecStore_Apply_Success(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "CREATE INDEX ..."},
	})

	if err := store.Apply("orders_status"); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if store.Recommendations[0].Status != "applied" {
		t.Errorf("Status = %q, want applied", store.Recommendations[0].Status)
	}
	if store.Recommendations[0].ApplyCount != 1 {
		t.Errorf("ApplyCount = %d, want 1", store.Recommendations[0].ApplyCount)
	}
}

func TestRecStore_Apply_AlreadyApplied(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "..."},
	})
	store.Apply("orders_status")

	err := store.Apply("orders_status")
	if err == nil {
		t.Fatal("expected error for double-apply, got nil")
	}
}

func TestRecStore_Apply_NotFound(t *testing.T) {
	store := &RecStore{}
	err := store.Apply("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown ID, got nil")
	}
}

// ─── RecStore: Dismiss ─────────────────────────────────────────────────────

func TestRecStore_Dismiss_Success(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "..."},
	})

	if err := store.Dismiss("orders_status"); err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	if store.Recommendations[0].Status != "dismissed" {
		t.Errorf("Status = %q, want dismissed", store.Recommendations[0].Status)
	}
}

func TestRecStore_Dismiss_AppliedError(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{{Table: "orders", Columns: []string{"status"}, CreateSQL: "..."}})
	store.Apply("orders_status")

	err := store.Dismiss("orders_status")
	if err == nil {
		t.Fatal("expected error dismissing applied rec, got nil")
	}
}

// ─── RecStore: Pending / ByTable ───────────────────────────────────────────

func TestRecStore_Pending(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "..."},
		{Table: "users", Columns: []string{"email"}, CreateSQL: "..."},
	})
	store.Apply("orders_status")

	pending := store.Pending()
	if len(pending) != 1 {
		t.Fatalf("got %d pending, want 1", len(pending))
	}
	if pending[0].ID != "users_email" {
		t.Errorf("pending[0].ID = %q, want users_email", pending[0].ID)
	}
}

func TestRecStore_ByTable(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{
		{Table: "orders", Columns: []string{"status"}, CreateSQL: "..."},
		{Table: "orders", Columns: []string{"total"}, CreateSQL: "..."},
		{Table: "users", Columns: []string{"email"}, CreateSQL: "..."},
	})

	orders := store.ByTable("orders")
	if len(orders) != 2 {
		t.Fatalf("got %d orders recs, want 2", len(orders))
	}

	users := store.ByTable("users")
	if len(users) != 1 {
		t.Fatalf("got %d users recs, want 1", len(users))
	}

	none := store.ByTable("nonexistent")
	if len(none) != 0 {
		t.Errorf("got %d recs for nonexistent table, want 0", len(none))
	}
}

// ─── RecStore: StaleReport ─────────────────────────────────────────────────

func TestRecStore_StaleReport(t *testing.T) {
	store := &RecStore{}
	store.Merge([]IndexSuggestion{
		{Table: "old", Columns: []string{"a"}, CreateSQL: "..."},
		{Table: "recent", Columns: []string{"b"}, CreateSQL: "..."},
		{Table: "applied_old", Columns: []string{"c"}, CreateSQL: "..."},
	})

	// Make first rec 10 days old, second 2 days old
	tenDaysAgo := time.Now().AddDate(0, 0, -10)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	store.Recommendations[0].CreatedAt = tenDaysAgo
	store.Recommendations[1].CreatedAt = twoDaysAgo
	store.Recommendations[2].CreatedAt = tenDaysAgo
	store.Recommendations[2].Status = "applied"

	// 7-day threshold
	stale := store.StaleReport(7)
	if len(stale) != 1 {
		t.Fatalf("got %d stale recs (>7d), want 1 (only the 10-day old pending one)", len(stale))
	}
	if stale[0].ID != "old_a" {
		t.Errorf("stale[0].ID = %q, want old_a", stale[0].ID)
	}

	// 30-day threshold — none
	stale = store.StaleReport(30)
	if len(stale) != 0 {
		t.Errorf("got %d stale recs (>30d), want 0", len(stale))
	}

	// 1-day threshold — both pending
	stale = store.StaleReport(1)
	if len(stale) != 2 {
		t.Errorf("got %d stale recs (>1d), want 2", len(stale))
	}
}

// ─── FormatSuggestions ─────────────────────────────────────────────────────

func TestFormatSuggestions_Empty(t *testing.T) {
	output := FormatSuggestions(nil)
	if output != "" {
		t.Errorf("expected empty, got %q", output)
	}
	output = FormatSuggestions([]IndexSuggestion{})
	if output != "" {
		t.Errorf("expected empty for empty slice, got %q", output)
	}
}

func TestFormatSuggestions_Single(t *testing.T) {
	sugs := []IndexSuggestion{
		{
			Table:          "orders",
			Columns:        []string{"total"},
			EstImprovement: "Seq Scan → Index Scan (~1M rows)",
			Tradeoffs:      []string{"+~3% INSERT overhead", "+~5% UPDATE overhead"},
			Confidence:     "high",
			CreateSQL:      "CREATE INDEX idx_orders_total ON orders(total)",
		},
	}

	output := FormatSuggestions(sugs)
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if !contains(output, "CREATE INDEX") {
		t.Error("missing CREATE INDEX in output")
	}
	if !contains(output, "🟢") {
		t.Error("missing green confidence icon for 'high'")
	}
	if !contains(output, "INSERT overhead") {
		t.Error("missing tradeoff in output")
	}
}

func TestFormatSuggestions_SpeculativeConfidence(t *testing.T) {
	sugs := []IndexSuggestion{
		{
			Table:      "users",
			Columns:    []string{"email"},
			Confidence: "speculative",
			CreateSQL:  "CREATE INDEX idx_users_email ON users(email)",
		},
	}

	output := FormatSuggestions(sugs)
	if !contains(output, "🟡") {
		t.Error("missing yellow icon for speculative confidence")
	}
	if !contains(output, "run ANALYZE") {
		t.Error("missing ANALYZE hint for speculative stats")
	}
}

func TestFormatSuggestions_MediumConfidence(t *testing.T) {
	sugs := []IndexSuggestion{
		{
			Table:      "orders",
			Columns:    []string{"total"},
			Confidence: "medium",
			CreateSQL:  "CREATE INDEX ...",
		},
	}
	output := FormatSuggestions(sugs)
	if !contains(output, "🟠") {
		t.Error("missing orange icon for medium confidence")
	}
}

func TestFormatSuggestions_PartialIndex(t *testing.T) {
	sugs := []IndexSuggestion{
		{
			Table:        "orders",
			Columns:      []string{"status"},
			PartialWhere: "WHERE status = 'pending'",
			CreateSQL:    "CREATE INDEX idx_orders_status ON orders(status) WHERE status = 'pending'",
			Confidence:   "high",
		},
	}
	output := FormatSuggestions(sugs)
	if !contains(output, "📐") {
		t.Error("missing partial index indicator")
	}
	if !contains(output, "WHERE status = 'pending'") {
		t.Error("missing partial WHERE clause in output")
	}
}

func TestFormatSuggestions_Multiple(t *testing.T) {
	sugs := []IndexSuggestion{
		{Table: "a", Columns: []string{"x"}, Confidence: "high", CreateSQL: "CREATE INDEX ..."},
		{Table: "b", Columns: []string{"y"}, Confidence: "high", CreateSQL: "CREATE INDEX ..."},
	}
	output := FormatSuggestions(sugs)
	// Should have numbered entries
	if !contains(output, "1.") || !contains(output, "2.") {
		t.Error("missing numbered entries for multiple suggestions")
	}
}

// ─── buildStatsQuery ───────────────────────────────────────────────────────

func TestBuildStatsQuery_SingleTable(t *testing.T) {
	q := buildStatsQuery([]string{"stress_orders"})
	if !contains(q, "stress_orders") {
		t.Error("missing table name in query")
	}
	if !contains(q, "pg_stats") {
		t.Error("missing pg_stats reference")
	}
	if !contains(q, "most_common_vals") {
		t.Error("missing most_common_vals column")
	}
}

func TestBuildStatsQuery_MultipleTables(t *testing.T) {
	q := buildStatsQuery([]string{"stress_orders", "stress_users"})
	if !contains(q, "stress_orders") || !contains(q, "stress_users") {
		t.Error("missing table names in multi-table query")
	}
}

func TestBuildStatsQuery_AllTables(t *testing.T) {
	q := buildStatsQuery(nil)
	if contains(q, "IN (") {
		t.Error("empty tables should not produce IN clause")
	}
	if !contains(q, "schemaname = 'public'") {
		t.Error("missing schemaname filter")
	}
}

func TestBuildStatsQuery_SQLInjectionSafe(t *testing.T) {
	// Table name with SQL injection attempt
	q := buildStatsQuery([]string{"users; DROP TABLE orders; --"})
	if contains(q, ";") {
		t.Error("semicolons should be escaped")
	}
	if contains(q, "DROP TABLE") {
		t.Error("SQL injection should not pass through")
	}
}

// ─── CollectTablesFromIssues ───────────────────────────────────────────────

func TestCollectTablesFromIssues(t *testing.T) {
	issues := []Issue{
		{TableName: "orders"},
		{TableName: "users"},
		{TableName: "orders"}, // duplicate
	}
	tables := CollectTablesFromIssues(issues)
	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(tables))
	}
	if tables[0] != "orders" || tables[1] != "users" {
		t.Errorf("got %v, want [orders users]", tables)
	}
}

func TestCollectTablesFromIssues_Empty(t *testing.T) {
	tables := CollectTablesFromIssues(nil)
	if len(tables) != 0 {
		t.Errorf("got %d tables for nil, want 0", len(tables))
	}
	tables = CollectTablesFromIssues([]Issue{})
	if len(tables) != 0 {
		t.Errorf("got %d tables for empty, want 0", len(tables))
	}
}

func TestCollectTablesFromIssues_EmptyTableName(t *testing.T) {
	issues := []Issue{
		{TableName: "orders"},
		{TableName: ""}, // empty — should be skipped
	}
	tables := CollectTablesFromIssues(issues)
	if len(tables) != 1 {
		t.Fatalf("got %d tables, want 1", len(tables))
	}
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr &&
		len(s) >= len(substr) &&
		searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
