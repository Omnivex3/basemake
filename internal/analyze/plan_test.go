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
