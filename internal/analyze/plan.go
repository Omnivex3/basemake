package analyze

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// ParsePlan parses a PostgreSQL JSON EXPLAIN string into a Report
func ParsePlan(jsonPlan string) (*Report, error) {
	var raw []RawPlan
	if err := json.Unmarshal([]byte(jsonPlan), &raw); err != nil {
		return nil, fmt.Errorf("parse plan json: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty plan result")
	}

	top := raw[0]
	report := &Report{
		PlanningTime:  top.PlanningTime,
		ExecutionTime: top.ExecutionTime,
		TotalCost:     top.Plan.TotalCost,
	}

	// Flatten plan tree
	flattenNode(&top.Plan, 0, report)

	// Analyze
	analyzeIssues(report)

	return report, nil
}

// flattenNode recursively walks the plan tree and appends flattened nodes
func flattenNode(node *RawNode, depth int, report *Report) {
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
		flattenNode(&node.Plans[i], depth+1, report)
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
		if n.NodeType == "Seq Scan" && n.RelationName != "" && n.ActualRows > 100 {
			r.Issues = append(r.Issues, Issue{
				Severity:  "warning",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Sequential scan on %s (%d rows, %.1fms)", n.RelationName, int(n.ActualRows), n.ActualTotal),
				Suggestion: fmt.Sprintf("Consider adding an index on %s for columns used in WHERE or JOIN conditions", n.RelationName),
			})
		}

		// 2. Row estimate mismatch (accual vs estimated off by 10x+)
		if n.ActualRows > 0 && n.PlanRows > 0 {
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

		// 3. Expensive filters (sequential scan with filter)
		if n.NodeType == "Seq Scan" && n.Filter != "" && n.ActualTotal > 1.0 {
			r.Issues = append(r.Issues, Issue{
				Severity:  "info",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Filter applied on sequential scan: %s (%.1fms)", n.Filter, n.ActualTotal),
				Suggestion: "Consider an index on the filtered column(s)",
			})
		}

		// 4. Nested Loop with many rows (potential missing index)
		if strings.Contains(n.NodeType, "Nested Loop") && n.ActualRows > 1000 {
			r.Issues = append(r.Issues, Issue{
				Severity:  "info",
				NodeType:  n.NodeType,
				TableName: n.RelationName,
				Message:   fmt.Sprintf("Nested Loop with %d rows — may benefit from index on inner table", int(n.ActualRows)),
				Suggestion: "Ensure inner table has an index on the join column",
			})
		}

		// 5. Slow individual node (>100ms)
		if n.ActualTotal > 100 && n.NodeType != "" {
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

	fmt.Fprintf(&b, "Execution Time: %.2f ms\n", r.ExecutionTime)
	fmt.Fprintf(&b, "Planning Time: %.2f ms\n\n", r.PlanningTime)

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
		fmt.Fprintf(&b, "%s%s%s (%.1fms, %d rows)\n", indent, n.NodeType, table, n.ActualTotal, int(n.ActualRows))
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
