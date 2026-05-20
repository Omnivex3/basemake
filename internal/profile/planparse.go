package profile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// PlanNode represents a single node in a PostgreSQL EXPLAIN (FORMAT JSON) tree.
type PlanNode struct {
	NodeType     string     `json:"Node Type"`
	RelationName string     `json:"Relation Name"`
	Alias        string     `json:"Alias"`
	Strategy     string     `json:"Strategy"`
	JoinType     string     `json:"Join Type"`
	IndexName    string     `json:"Index Name"`
	IndexCond    string     `json:"Index Cond"`
	Filter       string     `json:"Filter"`
	Plans        []PlanNode `json:"Plans"`
}

type planResult struct {
	Plan PlanNode `json:"Plan"`
}

// ExtractPlanNodes returns all leaf plan nodes (table access nodes) from a
// PostgreSQL JSON explain output. Returns error if plan is not valid JSON.
func ExtractPlanNodes(planJSON string) ([]PlanNode, error) {
	planJSON = strings.TrimSpace(planJSON)
	if planJSON == "" {
		return nil, fmt.Errorf("empty plan")
	}

	var results []planResult
	if err := json.Unmarshal([]byte(planJSON), &results); err != nil {
		// Try single-object format (some PG versions)
		var single planResult
		if err2 := json.Unmarshal([]byte(planJSON), &single); err2 != nil {
			return nil, fmt.Errorf("parse plan JSON: %w", err)
		}
		results = []planResult{single}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no plan data in JSON")
	}

	var nodes []PlanNode
	collectTableAccessNodes(results[0].Plan, &nodes)
	return nodes, nil
}

// collectTableAccessNodes collects all nodes that access a table (have a
// Relation Name). Intermediate nodes like Sort, Hash, Bitmap Index Scan,
// Aggregate, Limit are skipped — they don't tell us about table access patterns.
func collectTableAccessNodes(node PlanNode, nodes *[]PlanNode) {
	if node.RelationName != "" {
		*nodes = append(*nodes, node)
		return // don't descend — children of a table access are index scans, not new tables
	}
	for _, child := range node.Plans {
		collectTableAccessNodes(child, nodes)
	}
}

// PlanHash returns a stable hash of a plan for quick identity comparison.
// Normalizes JSON whitespace first.
func PlanHash(planJSON string) string {
	// Normalize JSON to canonical form
	var raw interface{}
	if err := json.Unmarshal([]byte(planJSON), &raw); err != nil {
		// If not JSON (e.g. SQLite text), hash raw string
		h := sha256.Sum256([]byte(planJSON))
		return fmt.Sprintf("%x", h[:8])
	}
	normalized, _ := json.Marshal(raw)
	h := sha256.Sum256(normalized)
	return fmt.Sprintf("%x", h[:8])
}

// PlanChange describes a structural change in the plan between two runs.
type PlanChange struct {
	OldNodeType  string
	NewNodeType  string
	RelationName string
	IndexName    string // new index name (empty = no index)
	OldIndexName string // previous index name
	IsNew        bool
	IsRemoved    bool
}

// ComparePlans compares leaf nodes between two plan evaluations and returns
// a list of changes. Only reports changes for nodes present in both plans.
func ComparePlans(oldNodes, newNodes []PlanNode) []PlanChange {
	oldByRel := make(map[string]PlanNode)
	seen := make(map[string]bool)
	for _, n := range oldNodes {
		key := n.RelationName
		if key == "" {
			key = fmt.Sprintf("__%s_%d", n.NodeType, len(oldByRel))
		}
		oldByRel[key] = n
	}

	newByRel := make(map[string]PlanNode)
	for _, n := range newNodes {
		key := n.RelationName
		if key == "" {
			key = fmt.Sprintf("__%s_%d", n.NodeType, len(newByRel))
		}
		newByRel[key] = n
	}

	var changes []PlanChange

	for rel, old := range oldByRel {
		seen[rel] = true
		nu, exists := newByRel[rel]
		if !exists {
			changes = append(changes, PlanChange{
				OldNodeType:  old.NodeType,
				RelationName: rel,
				IsRemoved:    true,
			})
			continue
		}
		if old.NodeType != nu.NodeType {
			changes = append(changes, PlanChange{
				OldNodeType:  old.NodeType,
				NewNodeType:  nu.NodeType,
				RelationName: rel,
				OldIndexName: old.IndexName,
				IndexName:    nu.IndexName,
			})
		} else if isIndexScan(nu.NodeType) && old.IndexName != nu.IndexName {
			changes = append(changes, PlanChange{
				OldNodeType:  old.NodeType,
				NewNodeType:  nu.NodeType,
				RelationName: rel,
				OldIndexName: old.IndexName,
				IndexName:    nu.IndexName,
			})
		}
	}

	for rel, nu := range newByRel {
		if !seen[rel] {
			changes = append(changes, PlanChange{
				NewNodeType:  nu.NodeType,
				RelationName: rel,
				IsNew:        true,
			})
		}
	}

	return changes
}

func isIndexScan(nt string) bool {
	return nt == "Index Scan" || nt == "Index Only Scan"
}

// ExplainChange returns a one-line plain English description of a plan change.
// Handles the key patterns that make developers say "oh, I know what to do next."
func ExplainChange(c PlanChange) string {
	// Case: index scan was removed entirely (replaced by non-index access)
	if isIndexScan(c.OldNodeType) && c.OldIndexName != "" && c.OldNodeType != c.NewNodeType {
		switch {
		case c.NewNodeType == "Seq Scan":
			return fmt.Sprintf("The planner stopped using %s on %s. Run ANALYZE.", c.OldIndexName, c.RelationName)
		case c.NewNodeType == "":
			return fmt.Sprintf("Index scan %s on %s removed from plan. The index may have been dropped.", c.OldIndexName, c.RelationName)
		default:
			return fmt.Sprintf("Was using %s on %s. Now using %s.", c.OldIndexName, c.RelationName, c.NewNodeType)
		}
	}

	switch {
	case c.IsNew:
		return fmt.Sprintf("New %s on %s.", c.NewNodeType, c.RelationName)
	case c.IsRemoved:
		if c.OldIndexName != "" {
			return fmt.Sprintf("Index scan %s on %s removed from plan.", c.OldIndexName, c.RelationName)
		}
		return fmt.Sprintf("%s on %s removed from plan.", c.OldNodeType, c.RelationName)
	case c.NewNodeType == "Seq Scan":
		return fmt.Sprintf("Seq Scan replaced %s on %s. Check for missing index.", c.OldNodeType, c.RelationName)
	case c.OldNodeType == "Seq Scan":
		return fmt.Sprintf("Now using %s on %s (was Seq Scan — better).", c.NewNodeType, c.RelationName)
	case c.IndexName != "" && c.OldIndexName != "":
		return fmt.Sprintf("Index changed: %s → %s on %s.", c.OldIndexName, c.IndexName, c.RelationName)
	default:
		return fmt.Sprintf("%s → %s on %s.", c.OldNodeType, c.NewNodeType, c.RelationName)
	}
}
