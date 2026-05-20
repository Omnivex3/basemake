package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return string(data)
}

func TestExtractPlanNodes(t *testing.T) {
	planJSON := readFixture(t, "plan_with_index.json")
	nodes, err := ExtractPlanNodes(planJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].NodeType != "Index Scan" || nodes[0].RelationName != "users" {
		t.Errorf("unexpected node: %+v", nodes[0])
	}
}

func TestComparePlans_IndexDroppedToSeqScan(t *testing.T) {
	oldJSON := readFixture(t, "plan_with_index.json")
	newJSON := readFixture(t, "plan_seq_scan.json")

	oldNodes, _ := ExtractPlanNodes(oldJSON)
	newNodes, _ := ExtractPlanNodes(newJSON)

	changes := ComparePlans(oldNodes, newNodes)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.OldNodeType != "Index Scan" || change.NewNodeType != "Seq Scan" {
		t.Errorf("expected Index Scan -> Seq Scan, got %s -> %s", change.OldNodeType, change.NewNodeType)
	}
	if change.RelationName != "users" {
		t.Errorf("expected relation users, got %s", change.RelationName)
	}

	explanation := ExplainChange(change)
	expected := "The planner stopped using users_email_idx on users. Run ANALYZE."
	if explanation != expected {
		t.Errorf("expected %q, got %q", expected, explanation)
	}
}

func TestComparePlans_IndexChanged(t *testing.T) {
	oldJSON := readFixture(t, "plan_with_index.json")
	newJSON := readFixture(t, "plan_different_index.json")

	oldNodes, _ := ExtractPlanNodes(oldJSON)
	newNodes, _ := ExtractPlanNodes(newJSON)

	changes := ComparePlans(oldNodes, newNodes)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if change.OldIndexName != "users_email_idx" || change.IndexName != "users_name_idx" {
		t.Errorf("expected users_email_idx -> users_name_idx, got %s -> %s", change.OldIndexName, change.IndexName)
	}

	explanation := ExplainChange(change)
	expected := "Index changed: users_email_idx → users_name_idx on users."
	if explanation != expected {
		t.Errorf("expected %q, got %q", expected, explanation)
	}
}

func TestComparePlans_NewNode(t *testing.T) {
	oldJSON := `[{"Plan": {"Node Type": "Limit", "Plans": []}}]` // no relation
	newJSON := readFixture(t, "plan_seq_scan.json")

	oldNodes, _ := ExtractPlanNodes(oldJSON)
	newNodes, _ := ExtractPlanNodes(newJSON)

	changes := ComparePlans(oldNodes, newNodes)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if !change.IsNew {
		t.Errorf("expected new node change")
	}

	explanation := ExplainChange(change)
	expected := "New Seq Scan on users."
	if explanation != expected {
		t.Errorf("expected %q, got %q", expected, explanation)
	}
}

func TestComparePlans_RemovedNode(t *testing.T) {
	oldJSON := readFixture(t, "plan_seq_scan.json")
	newJSON := `[{"Plan": {"Node Type": "Limit", "Plans": []}}]` // no relation

	oldNodes, _ := ExtractPlanNodes(oldJSON)
	newNodes, _ := ExtractPlanNodes(newJSON)

	changes := ComparePlans(oldNodes, newNodes)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	change := changes[0]
	if !change.IsRemoved {
		t.Errorf("expected removed node change")
	}

	explanation := ExplainChange(change)
	expected := "Seq Scan on users removed from plan."
	if explanation != expected {
		t.Errorf("expected %q, got %q", expected, explanation)
	}
}
