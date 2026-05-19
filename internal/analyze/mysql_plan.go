package analyze

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// MySQLPlan is the top-level MySQL EXPLAIN FORMAT=JSON wrapper.
type MySQLPlan struct {
	QueryBlock *MySQLQueryBlock `json:"query_block"`
}

// MySQLQueryBlock represents a SELECT query block in MySQL.
type MySQLQueryBlock struct {
	SelectID   int             `json:"select_id"`
	CostInfo   *MySQLCostInfo  `json:"cost_info,omitempty"`
	Table      *MySQLTableNode `json:"table,omitempty"`
	Subqueries json.RawMessage `json:"subqueries,omitempty"` // attached subqueries (IN, EXISTS, etc.)
}

// MySQLCostInfo contains cost information for a query block or table.
type MySQLCostInfo struct {
	QueryCost  string `json:"query_cost,omitempty"`
	ReadCost   string `json:"read_cost,omitempty"`
	EvalCost   string `json:"eval_cost,omitempty"`
	PrefixCost string `json:"prefix_cost,omitempty"`
}

// MySQLTableNode represents a table access in MySQL EXPLAIN.
// Can be a simple table, a joined table, or contain subqueries.
type MySQLTableNode struct {
	TableName           string         `json:"table_name"`
	AccessType          string         `json:"access_type"`
	PossibleKeys        []string       `json:"possible_keys,omitempty"`
	Key                 string         `json:"key,omitempty"`
	KeyLength           string         `json:"key_length,omitempty"`
	UsedKeyParts        []string       `json:"used_key_parts,omitempty"`
	RowsExaminedPerScan float64        `json:"rows_examined_per_scan"`
	RowsProducedPerJoin float64        `json:"rows_produced_per_join"`
	Filtered            string         `json:"filtered,omitempty"`
	CostInfo            *MySQLCostInfo `json:"cost_info,omitempty"`
	UsedColumns         []string       `json:"used_columns,omitempty"`
	AttachedCondition   string         `json:"attached_condition,omitempty"`
	IndexCondition      string         `json:"index_condition,omitempty"`
	UsingIndex          *bool          `json:"using_index,omitempty"`
	UsingIndexCondition *bool          `json:"using_index_condition,omitempty"`
	// materialized_from_subquery and union_result contain query_block objects
	// (not table nodes) — stored as raw JSON for now, not deeply analyzed
	Materialized json.RawMessage  `json:"materialized_from_subquery,omitempty"`
	NestedLoop   []*MySQLJoinNode `json:"nested_loop,omitempty"`
	HashJoin     []*MySQLJoinNode `json:"hash_join,omitempty"`
	UnionResult  json.RawMessage  `json:"union_result,omitempty"`
}

// MySQLJoinNode represents one side of a join in MySQL.
type MySQLJoinNode struct {
	Table *MySQLTableNode `json:"table"`
}

// ParsePlanMySQL parses a MySQL EXPLAIN FORMAT=JSON string into a Report.
func ParsePlanMySQL(jsonPlan string) (*Report, error) {
	var raw MySQLPlan
	if err := json.Unmarshal([]byte(jsonPlan), &raw); err != nil {
		return nil, fmt.Errorf("parse mysql plan json: %w", err)
	}
	if raw.QueryBlock == nil {
		return nil, fmt.Errorf("empty mysql plan result — no query_block found")
	}

	qb := raw.QueryBlock
	report := &Report{}

	// Extract total cost from query_block
	if qb.CostInfo != nil {
		cost, err := strconv.ParseFloat(qb.CostInfo.QueryCost, 64)
		if err == nil {
			report.TotalCost = cost
		}
	}

	// Flatten the table tree
	if qb.Table != nil {
		flattenMySQLTable(qb.Table, 0, report)
	}

	// Analyze for issues
	analyzeMySQLIssues(report)

	return report, nil
}

// flattenMySQLTable recursively walks the MySQL table tree and appends flattened nodes.
func flattenMySQLTable(table *MySQLTableNode, depth int, report *Report) {
	if table == nil {
		return
	}

	nodeType := mysqlAccessTypeToNodeType(table.AccessType)
	indexName := table.Key
	if table.UsingIndex != nil && *table.UsingIndex {
		if indexName == "" {
			indexName = "covering_index"
		}
	}

	filter := table.AttachedCondition
	if table.IndexCondition != "" {
		if filter != "" {
			filter += "; " + table.IndexCondition
		} else {
			filter = table.IndexCondition
		}
	}

	// Get cost
	actualTotal := 0.0
	if table.CostInfo != nil {
		if c, err := strconv.ParseFloat(table.CostInfo.PrefixCost, 64); err == nil {
			actualTotal = c
		}
	}

	// MySQL's FORMAT=JSON has no actual execution timing, so ActualRows stays 0
	fn := FlatNode{
		Depth:        depth,
		NodeType:     nodeType,
		RelationName: table.TableName,
		PlanRows:     table.RowsExaminedPerScan,
		Filter:       filter,
		IndexName:    indexName,
		TotalCost:    actualTotal,
		ActualTotal:  actualTotal, // Estimated cost, not actual time
	}
	report.Nodes = append(report.Nodes, fn)

	// Handle joins — nested loop and hash join children are siblings at same depth
	for _, jn := range table.NestedLoop {
		if jn != nil {
			flattenMySQLTable(jn.Table, depth+1, report)
		}
	}
	for _, jn := range table.HashJoin {
		if jn != nil {
			flattenMySQLTable(jn.Table, depth+1, report)
		}
	}

	// Note: materialized_from_subquery and union_result contain query_block
	// objects (not table nodes) and are not deeply analyzed in v1.
	// They represent subqueries executed separately — an enhancement for future.
}

// mysqlAccessTypeToNodeType maps MySQL access types to our internal node types.
func mysqlAccessTypeToNodeType(accessType string) string {
	switch accessType {
	case "ALL":
		return "Table Scan"
	case "index":
		return "Index Scan"
	case "range":
		return "Range Scan"
	case "ref":
		return "Ref Lookup"
	case "eq_ref":
		return "EQ Ref Lookup"
	case "const":
		return "Const Lookup"
	case "system":
		return "System Lookup"
	case "fulltext":
		return "Fulltext Search"
	case "ref_or_null":
		return "Ref Or Null"
	case "unique_subquery":
		return "Unique Subquery"
	case "index_subquery":
		return "Index Subquery"
	case "index_merge":
		return "Index Merge"
	case "":
		return "No Table"
	default:
		return accessType
	}
}

// analyzeMySQLIssues walks the flattened plan and detects performance issues for MySQL.
func analyzeMySQLIssues(r *Report) {
	for _, n := range r.Nodes {
		// Track scan types
		if n.NodeType == "Table Scan" {
			r.SequentialScans++
			r.TotalTableScans++
		}
		if strings.HasPrefix(n.NodeType, "Index") || strings.Contains(n.NodeType, "Index") ||
			n.NodeType == "Range Scan" || n.NodeType == "Ref Lookup" ||
			n.NodeType == "EQ Ref Lookup" || n.NodeType == "Fulltext Search" {
			r.IndexScans++
			r.TotalTableScans++
		}

		// Track heaviest node
		if n.TotalCost > r.HeaviestNodeTime {
			r.HeaviestNodeTime = n.TotalCost
			r.HeaviestNode = fmt.Sprintf("%s on %s", n.NodeType, n.RelationName)
		}

		// 1. Full table scans (access_type: ALL) with many rows — missing index
		if n.NodeType == "Table Scan" && n.RelationName != "" && n.PlanRows > 100 {
			r.Issues = append(r.Issues, Issue{
				Severity:   "warning",
				NodeType:   n.NodeType,
				TableName:  n.RelationName,
				Message:    fmt.Sprintf("Full table scan on %s (~%d rows examined)", n.RelationName, int(n.PlanRows)),
				Suggestion: fmt.Sprintf("Consider adding an index on %s for columns used in WHERE or JOIN conditions", n.RelationName),
			})
		}

		// 2. Filter with no key used (sequential scan with filter)
		if n.NodeType == "Table Scan" && n.Filter != "" {
			r.Issues = append(r.Issues, Issue{
				Severity:   "info",
				NodeType:   n.NodeType,
				TableName:  n.RelationName,
				Message:    fmt.Sprintf("Filter applied during table scan: %s", n.Filter),
				Suggestion: "Consider an index on the filtered column(s)",
			})
		}

		// 3. Possible index not used
		// Note: we can't detect this from FlatNode alone — the source in MySQLTableNode
		// would need to carry possible_keys. This is a limitation of the flattened format.
		// For now, we detect it at the flattened level by checking if it's a table scan
		// with a filter condition.

		// 4. High cost node
		if n.TotalCost > 100 && n.NodeType != "" {
			r.Issues = append(r.Issues, Issue{
				Severity:   "critical",
				NodeType:   n.NodeType,
				TableName:  n.RelationName,
				Message:    fmt.Sprintf("High cost node: %s on %s (cost %.1f)", n.NodeType, n.RelationNameOr("unknown"), n.TotalCost),
				Suggestion: "Investigate this node — consider query rewrite or index strategy",
			})
		}

		// 5. No table access (SELECT without FROM)
		if n.NodeType == "No Table" {
			r.Issues = append(r.Issues, Issue{
				Severity:   "info",
				NodeType:   n.NodeType,
				TableName:  "",
				Message:    "Query accesses no tables (e.g., SELECT 1)",
				Suggestion: "Remove unnecessary SELECT if not needed",
			})
		}
	}
}
