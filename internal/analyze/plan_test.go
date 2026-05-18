package analyze

import (
	"fmt"
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

// ──────────────────────────────────────────────
// Advanced Stress Tests
// ──────────────────────────────────────────────

// 16. Concurrent parsing — must be goroutine-safe
func TestMySQL_ConcurrentParse(t *testing.T) {
	plans := []string{mysqlTableScan, mysqlAccessTypes, mysqlHashJoin, mysqlNoTable,
		mysqlSubqueryInWhere, mysqlDerivedTable, mysqlUnion, mysqlIndexMerge,
		mysqlNullFields, mysqlDeepNesting, mysqlNegativeCost}
	iterations := 50
	errs := make(chan error, iterations*len(plans))

	for i := 0; i < iterations; i++ {
		for _, p := range plans {
			go func(plan string) {
				_, err := ParsePlan(plan)
				errs <- err
			}(p)
		}
	}

	for i := 0; i < iterations*len(plans); i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent parse error: %v", err)
		}
	}
}

// 17. Real-world complex plan — 6 tables, joins, filters, subquery, derived
const mysqlComplexRealWorld = `{
  "query_block": {
    "select_id": 1,
    "cost_info": { "query_cost": "1520.00" },
    "table": {
      "table_name": "orders",
      "access_type": "ALL",
      "rows_examined_per_scan": 50000,
      "rows_produced_per_join": 10000,
      "filtered": "20.00",
      "cost_info": { "read_cost": "100.00", "eval_cost": "500.00", "prefix_cost": "600.00" },
      "attached_condition": "(` + "`orders`.`status` = 'pending'" + `)",
      "nested_loop": [
        {"table": {
          "table_name": "users",
          "access_type": "eq_ref",
          "key": "PRIMARY",
          "rows_examined_per_scan": 1,
          "cost_info": { "prefix_cost": "601.00" }
        }},
        {"table": {
          "table_name": "order_items",
          "access_type": "ref",
          "key": "idx_order_id",
          "rows_examined_per_scan": 5,
          "filtered": "90.00",
          "cost_info": { "prefix_cost": "650.00" },
          "nested_loop": [
            {"table": {
              "table_name": "products",
              "access_type": "eq_ref",
              "key": "PRIMARY",
              "rows_examined_per_scan": 1,
              "attached_condition": "(` + "`products`.`stock` > 0)" + `",
              "cost_info": { "prefix_cost": "651.00" }
            }}
          ]
        }},
        {"table": {
          "table_name": "<derived3>",
          "access_type": "ALL",
          "rows_examined_per_scan": 0,
          "cost_info": { "prefix_cost": "1520.00" },
          "materialized_from_subquery": {
            "query_block": {
              "select_id": 3,
              "table": {
                "table_name": "inventory_log",
                "access_type": "ALL",
                "rows_examined_per_scan": 200000,
                "attached_condition": "(` + "`inventory_log`.`action` = 'ship')" + `"
              }
            }
          }
        }}
      ]
    },
    "subqueries": [
      {
        "query_block": {
          "select_id": 2,
          "table": {
            "table_name": "promotions",
            "access_type": "ref",
            "key": "idx_active",
            "rows_examined_per_scan": 3
          }
        }
      }
    ]
  }
}`

func TestMySQL_ComplexRealWorld(t *testing.T) {
	r, err := ParsePlan(mysqlComplexRealWorld)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should parse: orders (table scan) + users (eq_ref) + order_items (ref)
	// + products (eq_ref) + <derived3> (table scan) = 5 nodes
	// subqueries are RawMessage, not flattened
	if len(r.Nodes) != 5 {
		t.Fatalf("nodes = %d, want 5 (4 physical + 1 derived)", len(r.Nodes))
	}
	// Root should be orders table scan
	if r.Nodes[0].RelationName != "orders" || r.Nodes[0].NodeType != "Table Scan" {
		t.Errorf("root = %s on %s, want Table Scan on orders", r.Nodes[0].NodeType, r.Nodes[0].RelationName)
	}
	// Should have 2 table scans (orders + derived3)
	if r.SequentialScans != 2 {
		t.Errorf("SequentialScans = %d, want 2", r.SequentialScans)
	}
	// Should flag warning for orders table scan (50000 rows > 100)
	foundOrdersIssue := false
	for _, iss := range r.Issues {
		if iss.TableName == "orders" && iss.NodeType == "Table Scan" {
			foundOrdersIssue = true
			break
		}
	}
	if !foundOrdersIssue {
		t.Error("expected table scan issue for orders (50000 rows)")
	}
	// Should not crash on String() output
	output := r.String()
	if len(output) < 50 {
		t.Errorf("String() too short: %d chars", len(output))
	}
}

// 18. Report.String() formatting for MySQL
func TestMySQL_ReportStringFormatting(t *testing.T) {
	r, err := ParsePlan(mysqlTableScan)
	if err != nil {
		t.Fatalf("ParsePlan: %v", err)
	}
	output := r.String()
	checks := []string{"Scan Summary", "Sequential Scans:", "Plan Tree:", "big_logs", "Table Scan"}
	for _, c := range checks {
		if !strings.Contains(output, c) {
			t.Errorf("missing %q in MySQL report output", c)
		}
	}
	// Verify no garbage
	if strings.Contains(output, "<nil>") || strings.Contains(output, "%!") {
		t.Error("report contains formatting garbage")
	}

	// Test with indexed plan
	r2, _ := ParsePlan(mysqlIndexMerge)
	output2 := r2.String()
	if !strings.Contains(output2, "Index Merge") {
		t.Error("missing Index Merge in report output")
	}
}

// 19. Deep recursion test — simulate 200-level nested join
func TestMySQL_DeepRecursionLimit(t *testing.T) {
	// Build a deeply nested JSON programmatically
	nested := `{"query_block":{"select_id":1,"table":{"table_name":"t0","access_type":"ALL","rows_examined_per_scan":1`
	for i := 1; i < 200; i++ {
		nested += fmt.Sprintf(`,"nested_loop":[{"table":{"table_name":"t%d","access_type":"ref","key":"idx","rows_examined_per_scan":1`, i)
	}
	// Close all the brackets
	for i := 0; i < 199; i++ {
		nested += "}}]"
	}
	nested += "}}}"

	r, err := ParsePlan(nested)
	if err != nil {
		t.Fatalf("deep recursion parse error: %v", err)
	}
	if len(r.Nodes) != 200 {
		t.Fatalf("nodes = %d, want 200", len(r.Nodes))
	}
	// Verify depth increments for all nodes
	for i := 1; i < len(r.Nodes); i++ {
		if r.Nodes[i].Depth != r.Nodes[i-1].Depth+1 {
			t.Errorf("node[%d].Depth = %d, expected %d", i, r.Nodes[i].Depth, r.Nodes[i-1].Depth+1)
			break
		}
	}
	// Verify no crash on String()
	_ = r.String()
}

// 20. MySQL plan with backtick table names and special characters
const mysqlSpecialChars = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "user's data",
    "access_type": "ALL",
    "rows_examined_per_scan": 1000,
    "attached_condition": "(` + "`user's data`.`status` = 'active' OR `user's data`.`name` LIKE '%test%')" + `"
  }}
}`

func TestMySQL_SpecialChars(t *testing.T) {
	r, err := ParsePlan(mysqlSpecialChars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Nodes[0].RelationName != "user's data" {
		t.Errorf("RelationName = %q, want %q", r.Nodes[0].RelationName, "user's data")
	}
	if !strings.Contains(r.Nodes[0].Filter, "status") {
		t.Errorf("Filter should contain condition, got %q", r.Nodes[0].Filter)
	}
}

// 21. MySQL plan with very large row count (edge of float64)
const mysqlHugeRows = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "huge",
    "access_type": "ALL",
    "rows_examined_per_scan": 1e12,
    "cost_info": { "query_cost": "1e10" }
  }}
}`

func TestMySQL_HugeRows(t *testing.T) {
	r, err := ParsePlan(mysqlHugeRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Nodes[0].PlanRows != 1e12 {
		t.Errorf("PlanRows = %f, want 1e12", r.Nodes[0].PlanRows)
	}
	foundHugeIssue := false
	for _, iss := range r.Issues {
		if iss.NodeType == "Table Scan" && iss.TableName == "huge" {
			foundHugeIssue = true
			break
		}
	}
	if !foundHugeIssue {
		t.Error("expected table scan issue for huge table (1e12 rows)")
	}
}

// 22. MySQL plan with empty nested_loop and hash_join arrays
const mysqlEmptyArrays = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "t1",
    "access_type": "ALL",
    "rows_examined_per_scan": 100,
    "nested_loop": [],
    "hash_join": []
  }}
}`

func TestMySQL_EmptyArrays(t *testing.T) {
	r, err := ParsePlan(mysqlEmptyArrays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1 (empty arrays should be no-ops)", len(r.Nodes))
	}
}

// 23. Non-MySQL plan with "query_block" in data — test auto-detection doesn't false positive
const pgPlanWithQueryBlockText = `[
  {
    "Plan": {
      "Node Type": "Seq Scan",
      "Relation Name": "query_block",
      "Actual Rows": 1,
      "Actual Total Time": 0.1
    },
    "Execution Time": 0.2
  }
]`

func TestMySQL_AutoDetectNoFalsePositive(t *testing.T) {
	r, err := ParsePlan(pgPlanWithQueryBlockText)
	if err != nil {
		t.Fatalf("PG plan with 'query_block' table name should still parse as PG: %v", err)
	}
	// Should parse as PostgreSQL, not MySQL
	if r.ExecutionTime != 0.2 {
		t.Errorf("ExecutionTime = %f, want 0.2 (PG parse)", r.ExecutionTime)
	}
	if len(r.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1", len(r.Nodes))
	}
	if r.Nodes[0].NodeType != "Seq Scan" {
		t.Errorf("NodeType = %q, want %q (PG node type)", r.Nodes[0].NodeType, "Seq Scan")
	}
	if r.Nodes[0].RelationName != "query_block" {
		t.Errorf("RelationName = %q, want %q", r.Nodes[0].RelationName, "query_block")
	}
}

// 24. MySQL plan built incrementally from string (simulating large result concatenation)
func TestMySQL_LargePlanMemory(t *testing.T) {
	// Build a plan with 50 tables in one nested_loop
	plan := `{"query_block":{"select_id":1,"table":{"table_name":"base","access_type":"ALL","rows_examined_per_scan":100,"nested_loop":[`
	for i := 1; i < 50; i++ {
		if i > 1 {
			plan += ","
		}
		plan += fmt.Sprintf(`{"table":{"table_name":"t%d","access_type":"ref","key":"idx","rows_examined_per_scan":1}}`, i)
	}
	plan += `]}}}`

	r, err := ParsePlan(plan)
	if err != nil {
		t.Fatalf("large plan parse error: %v", err)
	}
	if len(r.Nodes) != 50 {
		t.Fatalf("nodes = %d, want 50 (base + 49 joins)", len(r.Nodes))
	}
	// All 49 joined tables should be at depth 1
	for i := 1; i < 50; i++ {
		if r.Nodes[i].Depth != 1 {
			t.Errorf("node[%d].Depth = %d, want 1", i, r.Nodes[i].Depth)
			break
		}
	}
	_ = r.String()
}

// 25. Negative rows count
const mysqlNegativeRows = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "weird", "access_type": "ALL",
    "rows_examined_per_scan": -50
  }}
}`

func TestMySQL_NegativeRows(t *testing.T) {
	r, err := ParsePlan(mysqlNegativeRows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Nodes[0].PlanRows != -50 {
		t.Errorf("PlanRows = %f, want -50", r.Nodes[0].PlanRows)
	}
	for _, iss := range r.Issues {
		if iss.NodeType == "Table Scan" && iss.TableName == "weird" {
			t.Errorf("unexpected issue for negative rows: %s", iss.Message)
		}
	}
}

// ──────────────────────────────────────────────
// Final Round Stress Tests
// ──────────────────────────────────────────────

// 26. Unknown access_type — should fall through to raw string, not crash
const mysqlUnknownAccessType = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "t1", "access_type": "super_fast_quantum_lookup_v2",
    "rows_examined_per_scan": 10
  }}
}`

func TestMySQL_UnknownAccessType(t *testing.T) {
	r, err := ParsePlan(mysqlUnknownAccessType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Nodes[0].NodeType != "super_fast_quantum_lookup_v2" {
		t.Errorf("NodeType = %q, want raw string %q", r.Nodes[0].NodeType, "super_fast_quantum_lookup_v2")
	}
	if r.SequentialScans != 0 {
		t.Errorf("SequentialScans = %d, want 0 (unknown type)", r.SequentialScans)
	}
}

// 27. Null entries in nested_loop array — should skip, not crash
const mysqlNullInNestedLoop = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "t1", "access_type": "ALL", "rows_examined_per_scan": 100,
    "nested_loop": [
      null,
      {"table": {"table_name": "t2", "access_type": "ref", "key": "idx", "rows_examined_per_scan": 1}},
      null,
      {"table": {"table_name": "t3", "access_type": "ref", "key": "idx2", "rows_examined_per_scan": 1}},
      null
    ]
  }}
}`

func TestMySQL_NullInNestedLoop(t *testing.T) {
	r, err := ParsePlan(mysqlNullInNestedLoop)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3 (t1 + t2 + t3, nulls skipped)", len(r.Nodes))
	}
	if r.Nodes[1].RelationName != "t2" || r.Nodes[2].RelationName != "t3" {
		t.Errorf("child tables wrong: got %s, %s", r.Nodes[1].RelationName, r.Nodes[2].RelationName)
	}
}

// 28. Very long table name (MySQL max is 64 chars)
var mysqlLongTableName = `{
  "query_block": { "select_id": 1, "table": {
    "table_name": "` + fmt.Sprintf("t%s", strings.Repeat("a", 63)) + `",
    "access_type": "ALL",
    "rows_examined_per_scan": 100
  }}
}`

func TestMySQL_LongTableName(t *testing.T) {
	r, err := ParsePlan(mysqlLongTableName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedLen := 64
	if len(r.Nodes[0].RelationName) != expectedLen {
		t.Errorf("RelationName length = %d, want %d", len(r.Nodes[0].RelationName), expectedLen)
	}
}

// 29. Report.String() race condition — multiple goroutines
func TestMySQL_ReportStringRace(t *testing.T) {
	r, err := ParsePlan(mysqlComplexRealWorld)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() {
			_ = r.String()
			done <- true
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

// 30. MySQL plan with unexpected extra fields and unusual values
const mysqlMixedTypes = `{
  "query_block": { "select_id": 1, "extra_field_unknown": "foo",
    "table": {
    "table_name": "t1",
    "access_type": "ALL",
    "rows_examined_per_scan": 100,
    "unknown_extra": null,
    "extra_nested": {"a": 1}
  }}
}`

func TestMySQL_MixedTypes(t *testing.T) {
	r, err := ParsePlan(mysqlMixedTypes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Nodes[0].RelationName != "t1" || r.Nodes[0].NodeType != "Table Scan" {
		t.Errorf("basic structure wrong: %s on %s", r.Nodes[0].NodeType, r.Nodes[0].RelationName)
	}
	if r.SequentialScans != 1 {
		t.Errorf("SequentialScans = %d, want 1", r.SequentialScans)
	}
}

// 31. Plan with only cost_info at query_block level, no table-level cost
const mysqlMinimalCost = `{
  "query_block": { "select_id": 1, "cost_info": { "query_cost": "5.00" }, "table": {
    "table_name": "t1", "access_type": "ALL", "rows_examined_per_scan": 100
  }}
}`

func TestMySQL_MinimalCost(t *testing.T) {
	r, err := ParsePlan(mysqlMinimalCost)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.TotalCost != 5.0 {
		t.Errorf("TotalCost = %f, want 5.0", r.TotalCost)
	}
}

// 32. No top-level table node
const mysqlNoTopLevelTable = `{
  "query_block": { "select_id": 1, "cost_info": { "query_cost": "100.00" }
  }
}`

func TestMySQL_NoTopLevelTable(t *testing.T) {
	r, err := ParsePlan(mysqlNoTopLevelTable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.TotalCost != 100.0 {
		t.Errorf("TotalCost = %f, want 100.0", r.TotalCost)
	}
	if len(r.Nodes) != 0 {
		t.Fatalf("nodes = %d, want 0 (no table)", len(r.Nodes))
	}
}

// 33. Consecutive calls with different dialects
func TestMySQL_ConsecutiveDifferentDialects(t *testing.T) {
	r1, err := ParsePlan(samplePlan)
	if err != nil {
		t.Fatalf("PG plan failed: %v", err)
	}
	r2, err := ParsePlan(mysqlTableScan)
	if err != nil {
		t.Fatalf("MySQL plan failed: %v", err)
	}
	r3, err := ParsePlan(samplePlanWithIndex)
	if err != nil {
		t.Fatalf("PG plan second call failed: %v", err)
	}
	if r1.ExecutionTime != 12.5 {
		t.Errorf("r1: PG ExecutionTime = %f, want 12.5", r1.ExecutionTime)
	}
	if r2.SequentialScans != 1 {
		t.Errorf("r2: MySQL SequentialScans = %d, want 1", r2.SequentialScans)
	}
	if r3.IndexScans != 1 {
		t.Errorf("r3: PG IndexScans = %d, want 1", r3.IndexScans)
	}
}
