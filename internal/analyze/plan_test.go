package analyze

import (
	"strings"
	"testing"
)

const samplePlan = `[
  {
    "Plan": {
      "Node Type": "Hash Join",
      "Startup Cost": 100.0,
      "Total Cost": 500.0,
      "Plan Rows": 1000,
      "Plan Width": 40,
      "Actual Startup Time": 0.5,
      "Actual Total Time": 12.3,
      "Actual Rows": 950,
      "Actual Loops": 1,
      "Hash Cond": "(u.id = o.user_id)",
      "Plans": [
        {
          "Node Type": "Seq Scan",
          "Relation Name": "users",
          "Alias": "u",
          "Startup Cost": 0.0,
          "Total Cost": 30.0,
          "Plan Rows": 200,
          "Plan Width": 20,
          "Actual Startup Time": 0.0,
          "Actual Total Time": 0.5,
          "Actual Rows": 150,
          "Actual Loops": 1
        },
        {
          "Node Type": "Seq Scan",
          "Relation Name": "orders",
          "Alias": "o",
          "Startup Cost": 0.0,
          "Total Cost": 50.0,
          "Plan Rows": 5000,
          "Plan Width": 10,
          "Actual Startup Time": 0.0,
          "Actual Total Time": 3.2,
          "Actual Rows": 8000,
          "Actual Loops": 1,
          "Filter": "(created_at > now() - interval '30 days')"
        }
      ]
    },
    "Planning Time": 0.15,
    "Execution Time": 12.5
  }
]`

const samplePlanWithIndex = `[
  {
    "Plan": {
      "Node Type": "Index Scan",
      "Relation Name": "users",
      "Alias": "u",
      "Startup Cost": 0.0,
      "Total Cost": 8.0,
      "Plan Rows": 1,
      "Plan Width": 20,
      "Actual Startup Time": 0.0,
      "Actual Total Time": 0.05,
      "Actual Rows": 1,
      "Actual Loops": 1,
      "Index Name": "users_pkey"
    },
    "Planning Time": 0.1,
    "Execution Time": 0.08
  }
]`

func TestParsePlanBasic(t *testing.T) {
	report, err := ParsePlan(samplePlan)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}

	if report.ExecutionTime != 12.5 {
		t.Errorf("ExecutionTime = %f, want 12.5", report.ExecutionTime)
	}
	if report.PlanningTime != 0.15 {
		t.Errorf("PlanningTime = %f, want 0.15", report.PlanningTime)
	}
	if report.TotalCost != 500.0 {
		t.Errorf("TotalCost = %f, want 500.0", report.TotalCost)
	}
}

func TestParsePlanNodes(t *testing.T) {
	report, err := ParsePlan(samplePlan)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}

	if len(report.Nodes) != 3 {
		t.Fatalf("got %d nodes, want 3 (join + 2 scans)", len(report.Nodes))
	}

	// First node should be Hash Join (root)
	if report.Nodes[0].NodeType != "Hash Join" {
		t.Errorf("node[0].NodeType = %q, want %q", report.Nodes[0].NodeType, "Hash Join")
	}

	// Second and third should be Seq Scans (children)
	if report.Nodes[1].NodeType != "Seq Scan" {
		t.Errorf("node[1].NodeType = %q, want %q", report.Nodes[1].NodeType, "Seq Scan")
	}
	if report.Nodes[2].NodeType != "Seq Scan" {
		t.Errorf("node[2].NodeType = %q, want %q", report.Nodes[2].NodeType, "Seq Scan")
	}
}

func TestParsePlanIssues(t *testing.T) {
	report, err := ParsePlan(samplePlan)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}

	if len(report.Issues) == 0 {
		t.Fatal("expected issues, got none")
	}

	// Should detect sequential scans
	if report.SequentialScans < 2 {
		t.Errorf("expected at least 2 sequential scans, got %d", report.SequentialScans)
	}

	// Should detect the seq scans as issues (they have >100 actual rows)
	hasSeqIssue := false
	for _, iss := range report.Issues {
		if iss.NodeType == "Seq Scan" && iss.TableName == "orders" {
			hasSeqIssue = true
			break
		}
	}
	if !hasSeqIssue {
		t.Error("expected seq scan issue for orders table (high row count)")
	}
}

func TestParsePlanWithIndex(t *testing.T) {
	report, err := ParsePlan(samplePlanWithIndex)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}

	if report.IndexScans != 1 {
		t.Errorf("IndexScans = %d, want 1", report.IndexScans)
	}

	if len(report.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(report.Nodes))
	}

	if report.Nodes[0].IndexName != "users_pkey" {
		t.Errorf("IndexName = %q, want %q", report.Nodes[0].IndexName, "users_pkey")
	}
}

func TestEmptyPlan(t *testing.T) {
	_, err := ParsePlan("[]")
	if err == nil {
		t.Error("expected error for empty plan, got nil")
	}
}

func TestInvalidJSON(t *testing.T) {
	_, err := ParsePlan("{invalid")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

const sampleMySQLPlan = `{
  "query_block": {
    "select_id": 1,
    "cost_info": {
      "query_cost": "105.00"
    },
    "table": {
      "table_name": "users",
      "access_type": "ALL",
      "possible_keys": ["idx_status"],
      "key": "idx_status",
      "key_length": "2",
      "used_key_parts": ["status"],
      "rows_examined_per_scan": 500,
      "rows_produced_per_join": 250,
      "filtered": "50.00",
      "cost_info": {
        "read_cost": "5.00",
        "eval_cost": "25.00",
        "prefix_cost": "30.00",
        "data_read_per_join": "8K"
      },
      "used_columns": ["id", "name", "email", "status"],
      "attached_condition": "(` + "`users`.`status` = 'active'" + `)",
      "nested_loop": [
        {
          "table": {
            "table_name": "orders",
            "access_type": "ref",
            "possible_keys": ["idx_user_id", "idx_created"],
            "key": "idx_user_id",
            "key_length": "4",
            "rows_examined_per_scan": 10,
            "rows_produced_per_join": 250,
            "filtered": "100.00",
            "using_index_condition": true,
            "attached_condition": "(` + "`orders`.`user_id` = `users`.`id`" + `)"
          }
        }
      ]
    }
  }
}`

func TestParsePlanMySQL(t *testing.T) {
	report, err := ParsePlan(sampleMySQLPlan)
	if err != nil {
		t.Fatalf("ParsePlan MySQL: %v", err)
	}

	if report.TotalCost != 105.0 {
		t.Errorf("TotalCost = %f, want 105.0", report.TotalCost)
	}

	if len(report.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2 (table scan + ref lookup)", len(report.Nodes))
	}

	if report.Nodes[0].NodeType != "Table Scan" {
		t.Errorf("node[0].NodeType = %q, want %q", report.Nodes[0].NodeType, "Table Scan")
	}
	if report.Nodes[0].RelationName != "users" {
		t.Errorf("node[0].RelationName = %q, want %q", report.Nodes[0].RelationName, "users")
	}

	if report.Nodes[1].NodeType != "Ref Lookup" {
		t.Errorf("node[1].NodeType = %q, want %q", report.Nodes[1].NodeType, "Ref Lookup")
	}
	if report.Nodes[1].RelationName != "orders" {
		t.Errorf("node[1].RelationName = %q, want %q", report.Nodes[1].RelationName, "orders")
	}

	// Should detect the full table scan as an issue
	if report.SequentialScans != 1 {
		t.Errorf("SequentialScans = %d, want 1", report.SequentialScans)
	}

	hasTableScanIssue := false
	for _, iss := range report.Issues {
		if iss.NodeType == "Table Scan" && iss.TableName == "users" {
			hasTableScanIssue = true
			break
		}
	}
	if !hasTableScanIssue {
		t.Error("expected table scan issue for users (high row count)")
	}
}

func TestReportStringFormatting(t *testing.T) {
	report, err := ParsePlan(samplePlan)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}

	output := report.String()

	// Should include key sections
	if !strings.Contains(output, "Execution Time:") {
		t.Error("missing Execution Time in output")
	}
	if !strings.Contains(output, "Sequential Scans:") {
		t.Error("missing scan summary in output")
	}
	if !strings.Contains(output, "Issues:") {
		t.Error("missing Issues section in output")
	}
	if !strings.Contains(output, "Hash Join") {
		t.Error("missing node types in plan tree output")
	}
}

// ──────────────────────────────────────────────
// MySQL Stress Tests
// ──────────────────────────────────────────────

// 1. Simple full table scan
const mysqlTableScan = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "big_logs", "access_type": "ALL",
    "rows_examined_per_scan": 500000,
    "rows_produced_per_join": 500000,
    "filtered": "10.00",
    "cost_info": { "query_cost": "50000.00" }
  }}
}`

func TestMySQL_TableScan(t *testing.T) {
	r, err := ParsePlan(mysqlTableScan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.SequentialScans != 1 {
		t.Errorf("SequentialScans = %d, want 1", r.SequentialScans)
	}
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1", len(r.Nodes))
	}
	if r.Nodes[0].NodeType != "Table Scan" {
		t.Errorf("NodeType = %q, want %q", r.Nodes[0].NodeType, "Table Scan")
	}
	if r.Nodes[0].PlanRows != 500000 {
		t.Errorf("PlanRows = %f, want 500000", r.Nodes[0].PlanRows)
	}
	// Should flag this as an issue (>100 rows)
	found := false
	for _, iss := range r.Issues {
		if iss.NodeType == "Table Scan" && iss.TableName == "big_logs" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue for table scan on big_logs")
	}
}

// 2. All access type mappings
const mysqlAccessTypes = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "all_types",
    "access_type": "ALL",
    "possible_keys": ["idx_a","idx_b","idx_c"],
    "rows_examined_per_scan": 100,
    "cost_info": { "query_cost": "10.00" },
    "nested_loop": [
      {"table": {"table_name": "t2", "access_type": "ref", "key": "idx_a", "rows_examined_per_scan": 1}},
      {"table": {"table_name": "t3", "access_type": "eq_ref", "key": "PRIMARY", "rows_examined_per_scan": 1}},
      {"table": {"table_name": "t4", "access_type": "range", "key": "idx_b", "rows_examined_per_scan": 50}},
      {"table": {"table_name": "t5", "access_type": "index", "key": "idx_c", "rows_examined_per_scan": 1000}},
      {"table": {"table_name": "t6", "access_type": "const", "rows_examined_per_scan": 0}},
      {"table": {"table_name": "t7", "access_type": "system", "rows_examined_per_scan": 1}},
      {"table": {"table_name": "t8", "access_type": "fulltext", "key": "ft_idx"}}
    ]
  }}
}`

func TestMySQL_AllAccessTypes(t *testing.T) {
	r, err := ParsePlan(mysqlAccessTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 1 table scan + 7 joined = 8 nodes
	if len(r.Nodes) != 8 {
		t.Fatalf("nodes = %d, want 8", len(r.Nodes))
	}
	expected := []struct {
		idx     int
		nodeType string
		table   string
	}{
		{0, "Table Scan", "all_types"},
		{1, "Ref Lookup", "t2"},
		{2, "EQ Ref Lookup", "t3"},
		{3, "Range Scan", "t4"},
		{4, "Index Scan", "t5"},
		{5, "Const Lookup", "t6"},
		{6, "System Lookup", "t7"},
		{7, "Fulltext Search", "t8"},
	}
	for _, exp := range expected {
		n := r.Nodes[exp.idx]
		if n.NodeType != exp.nodeType {
			t.Errorf("node[%d].NodeType = %q, want %q", exp.idx, n.NodeType, exp.nodeType)
		}
		if n.RelationName != exp.table {
			t.Errorf("node[%d].RelationName = %q, want %q", exp.idx, n.RelationName, exp.table)
		}
	}
	if r.SequentialScans != 1 {
		t.Errorf("SequentialScans = %d, want 1", r.SequentialScans)
	}
	if r.IndexScans != 5 {
		t.Errorf("IndexScans = %d, want 5 (ref+eq_ref+range+index+fulltext)", r.IndexScans)
	}
}

// 3. Hash join
const mysqlHashJoin = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "t1", "access_type": "ALL", "rows_examined_per_scan": 1000,
    "cost_info": { "query_cost": "100.00" },
    "hash_join": [
      {"table": {"table_name": "t2", "access_type": "ALL", "rows_examined_per_scan": 500}},
      {"table": {"table_name": "t3", "access_type": "ref", "key": "idx", "rows_examined_per_scan": 10}}
    ]
  }}
}`

func TestMySQL_HashJoin(t *testing.T) {
	r, err := ParsePlan(mysqlHashJoin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3 (t1 + hash children)", len(r.Nodes))
	}
	if r.Nodes[0].NodeType != "Table Scan" || r.Nodes[0].RelationName != "t1" {
		t.Errorf("root node wrong: %s on %s", r.Nodes[0].NodeType, r.Nodes[0].RelationName)
	}
}

// 4. No table query (SELECT 1)
const mysqlNoTable = `{
  "query_block": { "select_id": 1, "table": {
    "access_type": null, "table_name": ""
  }}
}`

func TestMySQL_NoTable(t *testing.T) {
	r, err := ParsePlan(mysqlNoTable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NULL access_type -> empty string -> "No Table"
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1", len(r.Nodes))
	}
	if r.Nodes[0].NodeType != "No Table" {
		t.Errorf("NodeType = %q, want %q", r.Nodes[0].NodeType, "No Table")
	}
}

// 5. Subquery in WHERE (attached_subqueries) — should not crash, should produce sensible output
const mysqlSubqueryInWhere = `{
  "query_block": {
    "select_id": 1,
    "table": {
      "table_name": "orders",
      "access_type": "ALL",
      "rows_examined_per_scan": 10000,
      "cost_info": { "query_cost": "1000.00" },
      "attached_condition": "(` + "`orders`.`user_id` in (select `users`.`id` from `users` where `users`.`status` = 'active')" + `)"
    },
    "subqueries": [
      {
        "query_block": {
          "select_id": 2,
          "table": {
            "table_name": "users",
            "access_type": "ref",
            "key": "idx_status",
            "rows_examined_per_scan": 50,
            "attached_condition": "(` + "`users`.`status` = 'active')" + `"
          }
        }
      }
    ]
  }
}`

func TestMySQL_SubqueryInWhere(t *testing.T) {
	r, err := ParsePlan(mysqlSubqueryInWhere)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should at minimum analyze the outer query
	if r.SequentialScans != 1 {
		t.Errorf("SequentialScans = %d, want 1 (orders table scan)", r.SequentialScans)
	}
	if r.Nodes[0].RelationName != "orders" {
		t.Errorf("outer table = %q, want %q", r.Nodes[0].RelationName, "orders")
	}
	// subqueries are RawMessage — not analyzed deeply, should not crash
	_ = r.String()
}

// 6. Derived table (materialized_from_subquery) — edge case that caught the type mismatch
const mysqlDerivedTable = `{
  "query_block": {
    "select_id": 1,
    "table": {
      "table_name": "<derived2>",
      "access_type": "ALL",
      "rows_examined_per_scan": 0,
      "materialized_from_subquery": {
        "query_block": {
          "select_id": 2,
          "table": {
            "table_name": "users",
            "access_type": "ALL",
            "rows_examined_per_scan": 5000
          }
        }
      }
    }
  }
}`

func TestMySQL_DerivedTable(t *testing.T) {
	r, err := ParsePlan(mysqlDerivedTable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not crash. Outer derived table is present, inner is RawMessage
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1 (outer derived table only)", len(r.Nodes))
	}
}

// 7. UNION
const mysqlUnion = `{
  "query_block": {
    "select_id": 1,
    "table": {
      "table_name": "<union1,2>",
      "access_type": "ALL",
      "rows_examined_per_scan": 0,
      "union_result": {
        "query_block": {
          "select_id": 2,
          "table": {"table_name": "t1", "access_type": "ALL", "rows_examined_per_scan": 100}
        }
      }
    }
  }
}`

func TestMySQL_Union(t *testing.T) {
	r, err := ParsePlan(mysqlUnion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not crash. Union branches are RawMessage, not analyzed
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1 (union wrapper only)", len(r.Nodes))
	}
}

// 8. Index merge
const mysqlIndexMerge = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "t1", "access_type": "index_merge",
    "key": "intersect(idx_a,idx_b)",
    "rows_examined_per_scan": 50,
    "cost_info": { "query_cost": "20.00" }
  }}
}`

func TestMySQL_IndexMerge(t *testing.T) {
	r, err := ParsePlan(mysqlIndexMerge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Nodes[0].NodeType != "Index Merge" {
		t.Errorf("NodeType = %q, want %q", r.Nodes[0].NodeType, "Index Merge")
	}
	if r.IndexScans != 1 {
		t.Errorf("IndexScans = %d, want 1", r.IndexScans)
	}
}

// 9. Null/missing/empty fields — should not panic
const mysqlNullFields = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": null,
    "access_type": null,
    "rows_examined_per_scan": null,
    "rows_produced_per_join": null,
    "filtered": null,
    "cost_info": null,
    "used_columns": null,
    "possible_keys": null,
    "nested_loop": null,
    "hash_join": null
  }}
}`

func TestMySQL_NullFields(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PANIC on null fields: %v", r)
		}
	}()
	r, err := ParsePlan(mysqlNullFields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should produce valid output with an info-level "No Table" issue
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1", len(r.Nodes))
	}
	// Should not produce any critical errors
	for _, iss := range r.Issues {
		if iss.Severity == "critical" {
			t.Errorf("unexpected critical issue: %s", iss.Message)
		}
	}
	_ = r.String()
}

// 10. Empty JSON object — not valid MySQL or PG format
const mysqlEmptyObj = `{}`

func TestMySQL_EmptyJSON(t *testing.T) {
	r, err := ParsePlan(mysqlEmptyObj)
	if err == nil {
		t.Fatal("expected error for empty object, got nil")
	}
	// Should fall through to PG parser which should also fail
	if r != nil {
		t.Errorf("expected nil report on error, got %v", r)
	}
}

// 11. Missing query_block entirely
const mysqlNoQueryBlock = `{"some_other_key": 123}`

func TestMySQL_NoQueryBlock(t *testing.T) {
	r, err := ParsePlan(mysqlNoQueryBlock)
	if err != nil {
		// Both parsers should fail — that's fine
		if r != nil {
			t.Errorf("expected nil report on error")
		}
		return
	}
	// If it somehow succeeded, check it's valid
	if r != nil {
		_ = r.String()
	}
}

// 12. Deeply nested joins (8 levels)
const mysqlDeepNesting = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "l1", "access_type": "ALL", "rows_examined_per_scan": 200,
    "nested_loop": [{"table": {
      "table_name": "l2", "access_type": "ref", "key": "idx", "rows_examined_per_scan": 10,
      "nested_loop": [{"table": {
        "table_name": "l3", "access_type": "eq_ref", "key": "PRIMARY", "rows_examined_per_scan": 1,
        "nested_loop": [{"table": {
          "table_name": "l4", "access_type": "ref", "key": "idx", "rows_examined_per_scan": 5,
          "nested_loop": [{"table": {
            "table_name": "l5", "access_type": "ALL", "rows_examined_per_scan": 500
          }}]
        }}]
      }}]
    }}]
  }}
}`

func TestMySQL_DeepNesting(t *testing.T) {
	r, err := ParsePlan(mysqlDeepNesting)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Nodes) != 5 {
		t.Fatalf("nodes = %d, want 5", len(r.Nodes))
	}
	// Verify depth increases
	for i := 1; i < len(r.Nodes); i++ {
		if r.Nodes[i].Depth <= r.Nodes[i-1].Depth {
			t.Errorf("node[%d].Depth = %d, should be deeper than node[%d].Depth = %d",
				i, r.Nodes[i].Depth, i-1, r.Nodes[i-1].Depth)
		}
	}
	// Should flag both table scans (l1 and l5 have >100 rows)
	scanCount := 0
	for _, n := range r.Nodes {
		if n.NodeType == "Table Scan" && n.PlanRows > 100 {
			scanCount++
		}
	}
	if scanCount != 2 {
		t.Errorf("table scans with >100 rows = %d, want 2 (l1, l5)", scanCount)
	}
}

// 13. Negative/null cost_info
const mysqlNegativeCost = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "t1", "access_type": "ALL",
    "rows_examined_per_scan": 10,
    "cost_info": { "query_cost": "-1.00", "prefix_cost": "-5.00" }
  }}
}`

func TestMySQL_NegativeCost(t *testing.T) {
	r, err := ParsePlan(mysqlNegativeCost)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Negative cost should not produce critical issues
	for _, iss := range r.Issues {
		if iss.Severity == "critical" {
			t.Errorf("unexpected critical issue for negative cost: %s", iss.Message)
		}
	}
}

// 14. Cross-parse: PG plan still works unchanged
func TestMySQL_CrossParsePGUnchanged(t *testing.T) {
	r, err := ParsePlan(samplePlan)
	if err != nil {
		t.Fatalf("PG plan parsing broke: %v", err)
	}
	if r.ExecutionTime != 12.5 {
		t.Errorf("PG ExecutionTime = %f, want 12.5", r.ExecutionTime)
	}
	if len(r.Nodes) != 3 {
		t.Errorf("PG nodes = %d, want 3", len(r.Nodes))
	}
}

// 15. PG plan with verbose output (no regression)
func TestMySQL_CrossParsePGWithIndex(t *testing.T) {
	r, err := ParsePlan(samplePlanWithIndex)
	if err != nil {
		t.Fatalf("PG plan with index broke: %v", err)
	}
	if r.IndexScans != 1 {
		t.Errorf("PG IndexScans = %d, want 1", r.IndexScans)
	}
}
