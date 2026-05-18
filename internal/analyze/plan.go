package analyze

import (
	"encoding/json"
	"fmt"
	"strings"
)

// --- PostgreSQL Structures ---

// RawPlan represents the top-level PostgreSQL EXPLAIN JSON structure
type RawPlan struct {
	Plan          RawNode  `json:"Plan"`
	PlanningTime  float64  `json:"Planning Time"`
	ExecutionTime float64  `json:"Execution Time"`
}

// RawNode represents a single node in the PostgreSQL plan tree
type RawNode struct {
	NodeType      string    `json:"Node Type"`
	RelationName  string    `json:"Relation Name"`
	Alias         string    `json:"Alias"`
	StartupCost   float64   `json:"Startup Cost"`
	TotalCost     float64   `json:"Total Cost"`
	PlanRows      float64   `json:"Plan Rows"`
	PlanWidth     int       `json:"Plan Width"`
	ActualStartup float64   `json:"Actual Startup Time"`
	ActualTotal   float64   `json:"Actual Total Time"`
	ActualRows    float64   `json:"Actual Rows"`
	ActualLoops   float64   `json:"Actual Loops"`
	JoinType      string    `json:"Join Type"`
	HashCond      string    `json:"Hash Cond"`
	Filter        string    `json:"Filter"`
	IndexName     string    `json:"Index Name"`
	Plans         []RawNode `json:"Plans"`
}

// --- Unified Structures ---

// FlatNode is a flattened plan node with depth and path info
type FlatNode struct {
	Depth        int
	NodeType     string
	RelationName string
	ActualTotal  float64
	ActualRows   float64
	ActualLoops  float64
	PlanRows     float64
	StartupCost  float64
	TotalCost    float64
	Filter       string
	IndexName    string
	JoinType     string
}

// Issue represents a detected performance issue
type Issue struct {
	Severity   string // "critical", "warning", "info"
	NodeType   string
	TableName  string
	Message    string
	Suggestion string
}

// Report is the complete analysis result
type Report struct {
	Query               string
	Dialect             string
	PlanningTime        float64
	ExecutionTime       float64
	TotalCost           float64
	Nodes               []FlatNode
	Issues              []Issue
	SequentialScans     int
	IndexScans          int
	TotalTableScans     int
	HasRowMismatch      bool
	WorstRowMismatch    float64
	HeaviestNode        string
	HeaviestNodeTime    float64
}

// ParsePlan parses a JSON EXPLAIN string into a Report.
// Auto-detects PostgreSQL vs MySQL format based on JSON structure.
func ParsePlan(jsonPlan string) (*Report, error) {
	trimmed := strings.TrimSpace(jsonPlan)

	// Auto-detect: MySQL uses a single JSON object starting with { containing "query_block"
	// PG uses an array [...] at the top level — even if "query_block" appears in data
	if len(trimmed) > 0 && trimmed[0] == '{' && strings.Contains(trimmed, `"query_block"`) {
		return parseMySQLPlan(jsonPlan)
	}

	// Default: PostgreSQL format (array of plan objects)
	return parsePostgresPlan(jsonPlan)
}

func parsePostgresPlan(jsonPlan string) (*Report, error) {
	var raw []RawPlan
	if err := json.Unmarshal([]byte(jsonPlan), &raw); err != nil {
		return nil, fmt.Errorf("parse postgres plan json: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty postgres plan result")
	}

	top := raw[0]
	report := &Report{
		Dialect:       "PostgreSQL",
		PlanningTime:  top.PlanningTime,
		ExecutionTime: top.ExecutionTime,
		TotalCost:     top.Plan.TotalCost,
	}

	flattenPostgresNode(&top.Plan, 0, report)
	analyzeIssues(report)
	return report, nil
}

func flattenPostgresNode(node *RawNode, depth int, report *Report) {
	fn := FlatNode{
		Depth:        depth,
		NodeType:     node.NodeType,
		RelationName: node.RelationName,
		ActualTotal:  node.ActualTotal,
		ActualRows:   node.ActualRows,
		ActualLoops:  node.ActualLoops,
		PlanRows:     node.PlanRows,
		StartupCost:  node.StartupCost,
		TotalCost:    node.TotalCost,
		Filter:       node.Filter,
		IndexName:    node.IndexName,
		JoinType:     node.JoinType,
	}
	report.Nodes = append(report.Nodes, fn)

	for i := range node.Plans {
		flattenPostgresNode(&node.Plans[i], depth+1, report)
	}
}

func parseMySQLPlan(jsonPlan string) (*Report, error) {
	var raw interface{}
	if err := json.Unmarshal([]byte(jsonPlan), &raw); err != nil {
		return nil, fmt.Errorf("parse mysql plan json: %w", err)
	}

	report := &Report{
		Dialect: "MySQL",
	}

	flattenMySQLNode(raw, 0, report)
	analyzeIssues(report)
	return report, nil
}

func flattenMySQLNode(val interface{}, depth int, report *Report) {
	m, ok := val.(map[string]interface{})
	if !ok {
		if arr, ok := val.([]interface{}); ok {
			for _, item := range arr {
				flattenMySQLNode(item, depth, report)
			}
		}
		return
	}

	// In MySQL JSON EXPLAIN, a "table" or "query_block" represents a node
	if table, ok := m["table"].(map[string]interface{}); ok {
		name, _ := table["table_name"].(string)
		access, _ := table["access_type"].(string)
		rows, _ := table["rows_examined_per_scan"].(float64)

		nodeType := access
		if access == "ALL" {
			nodeType = "Seq Scan"
		} else if access == "index" || access == "range" || access == "ref" || access == "eq_ref" {
			nodeType = "Index Scan"
		}

		report.Nodes = append(report.Nodes, FlatNode{
			Depth:        depth,
			NodeType:     nodeType,
			RelationName: name,
			PlanRows:     rows,
			ActualRows:   rows, // MySQL non-analyze JSON only has estimates
		})
	}

	// Recurse into other potential blocks
	for _, k := range []string{"query_block", "nested_loop", "union_result", "table"} {
		if v, ok := m[k]; ok && k != "table" {
			flattenMySQLNode(v, depth+1, report)
		}
	}

	// Generic recursion for subqueries
	if sub, ok := m["subqueries"].([]interface{}); ok {
		for _, item := range sub {
			flattenMySQLNode(item, depth+1, report)
		}
	}
}

// analyzeIssues walks the flattened plan and detects performance issues
func analyzeIssues(r *Report) {
	for _, n := range r.Nodes {
		// Track scan types
		if n.NodeType == "Seq Scan" {
			r.SequentialScans++
			r.TotalTableScans++
		}
		if strings.HasPrefix(n.NodeType, "Index") || strings.Contains(n.NodeType, "Index") {
			r.IndexScans++
			r.TotalTableScans++
		}

		// Track heaviest node
		if n.ActualTotal > r.HeaviestNodeTime {
			r.HeaviestNodeTime = n.ActualTotal
			r.HeaviestNode = fmt.Sprintf("%s on %s", n.NodeType, n.RelationName)
		}

		// 1. Sequential scans on tables with rows (potential missing index)
		// For MySQL, ActualRows is used as the row estimate since timing isn't available in JSON.
		rowThreshold := 100.0
		if n.NodeType == "Seq Scan" && n.RelationName != "" && n.PlanRows > rowThreshold {
			r.Issues = append(r.Issues, Issue{
				Severity:  "warning",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Sequential scan on %s (%d estimated rows)", n.RelationName, int(n.PlanRows)),
				Suggestion: fmt.Sprintf("Consider adding an index on %s for columns used in WHERE or JOIN conditions", n.RelationName),
			})
		}

		// 2. Row estimate mismatch (PostgreSQL only — MySQL JSON doesn't have actuals)
		if r.Dialect == "PostgreSQL" && n.ActualRows > 0 && n.PlanRows > 0 {
			ratio := n.ActualRows / n.PlanRows
			if ratio > 10 || ratio < 0.1 {
				r.HasRowMismatch = true
				if ratio > r.WorstRowMismatch {
					r.WorstRowMismatch = ratio
				}
				r.Issues = append(r.Issues, Issue{
					Severity:  "warning",
					NodeType:  n.NodeType,
					TableName: n.RelationName,
					Message:   fmt.Sprintf("Row estimate mismatch on %s: actual=%d, estimated=%d (%.1fx off)", n.RelationName, int(n.ActualRows), int(n.PlanRows), ratio),
					Suggestion: "Update table statistics with ANALYZE or adjust default_statistics_target",
				})
			}
		}

		// 3. Expensive filters (PostgreSQL only — MySQL JSON timing not available)
		if r.Dialect == "PostgreSQL" && n.NodeType == "Seq Scan" && n.Filter != "" && n.ActualTotal > 1.0 {
			r.Issues = append(r.Issues, Issue{
				Severity:  "info",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Filter applied on sequential scan: %s (%.1fms)", n.Filter, n.ActualTotal),
				Suggestion: "Consider an index on the filtered column(s)",
			})
		}

		// 4. Nested Loop with many rows (potential missing index)
		if strings.Contains(n.NodeType, "Nested Loop") && n.PlanRows > 1000 {
			r.Issues = append(r.Issues, Issue{
				Severity:  "info",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Nested Loop with %d rows — may benefit from index on inner table", int(n.PlanRows)),
				Suggestion: "Ensure inner table has an index on the join column",
			})
		}

		// 5. Slow individual node (PostgreSQL only — MySQL JSON timing not available)
		if r.Dialect == "PostgreSQL" && n.ActualTotal > 100 && n.NodeType != "" {
			r.Issues = append(r.Issues, Issue{
				Severity:  "critical",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Slow node: %s on %s (%.1fms)", n.NodeType, n.RelationNameOr("unknown"), n.ActualTotal),
				Suggestion: "Investigate this node — consider query rewrite or index strategy",
			})
		}
	}
}

// RelationNameOr returns the relation name or a fallback
func (n FlatNode) RelationNameOr(fallback string) string {
	if n.RelationName != "" {
		return n.RelationName
	}
	return fallback
}

// String returns a human-readable analysis report
func (r *Report) String() string {
	var b strings.Builder

	if r.Dialect == "PostgreSQL" {
		fmt.Fprintf(&b, "Execution Time: %.2f ms\n", r.ExecutionTime)
		fmt.Fprintf(&b, "Planning Time: %.2f ms\n\n", r.PlanningTime)
	} else {
		fmt.Fprintf(&b, "Dialect: %s\n\n", r.Dialect)
	}

	// Summary
	fmt.Fprintf(&b, "Scan Summary:\n")
	fmt.Fprintf(&b, "  Sequential Scans: %d\n", r.SequentialScans)
	fmt.Fprintf(&b, "  Index Scans: %d\n", r.IndexScans)
	if r.HasRowMismatch {
		fmt.Fprintf(&b, "  ⚠ Row Estimate Mismatches: yes (worst: %.1fx)\n", r.WorstRowMismatch)
	}
	if r.HeaviestNode != "" {
		fmt.Fprintf(&b, "  Heaviest Node: %s (%.1fms)\n\n", r.HeaviestNode, r.HeaviestNodeTime)
	}

	// Plan tree
	fmt.Fprintf(&b, "Plan Tree:\n")
	for _, n := range r.Nodes {
		indent := strings.Repeat("  ", n.Depth)
		table := n.RelationName
		if table != "" {
			table = " on " + table
		}
		if r.Dialect == "PostgreSQL" {
			fmt.Fprintf(&b, "%s%s%s (%.1fms, %d rows)\n", indent, n.NodeType, table, n.ActualTotal, int(n.ActualRows))
		} else {
			fmt.Fprintf(&b, "%s%s%s (%d estimated rows)\n", indent, n.NodeType, table, int(n.PlanRows))
		}
	}

	// Issues
	if len(r.Issues) > 0 {
		fmt.Fprintf(&b, "\nIssues:\n")
		for _, iss := range r.Issues {
			icon := "ℹ"
			switch iss.Severity {
			case "critical":
				icon = "🔴"
			case "warning":
				icon = "🟡"
			case "info":
				icon = "ℹ"
			}
			fmt.Fprintf(&b, "%s %s\n", icon, iss.Message)
			fmt.Fprintf(&b, "   → %s\n", iss.Suggestion)
		}
	}

	return b.String()
}
