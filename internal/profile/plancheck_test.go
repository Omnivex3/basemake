package profile

import (
	"context"
	"os"
	"testing"

	"github.com/DynamicKarabo/basemake/internal/db"
)

// mockDB implements the needed parts of db.Database for PlanCheck
type mockDB struct {
	db.Database
	planJSON string
	err      error
}

func (m *mockDB) ExplainNoAnalyze(ctx context.Context, sql string) (string, error) {
	return m.planJSON, m.err
}

func TestPlanCheck_IndexDropped(t *testing.T) {
	// Setup profile
	hash := QueryHash(NormalizeSQL("SELECT * FROM users"))
	p := &QueryProfile{
		Runs: []QueryRun{
			{
				DurationMS: 10,
				PlanText:   readFixture(t, "plan_with_index.json"),
				PlanHash:   PlanHash(readFixture(t, "plan_with_index.json")),
			},
		},
	}
	Save(hash, p)
	defer os.Remove(ProfilePath(hash))

	// Setup current plan returning Seq Scan
	conn := &mockDB{
		planJSON: readFixture(t, "plan_seq_scan.json"),
	}

	warnings := PlanCheck(context.Background(), "SELECT * FROM users", conn)

	if !HasWarnings(warnings) {
		t.Fatal("expected warnings, got none")
	}

	foundIndexDrop := false
	for _, w := range warnings {
		if w.Severity == "warn" && w.Message == "users_email_idx was dropped since the last profile. This query may be slower. Run ANALYZE or recreate the index." {
			foundIndexDrop = true
		}
	}

	if !foundIndexDrop {
		t.Errorf("did not find expected index dropped warning: %v", warnings)
	}
}

func TestPlanCheck_Regression(t *testing.T) {
	hash := QueryHash(NormalizeSQL("SELECT * FROM users"))
	p := &QueryProfile{
		Runs: []QueryRun{
			{DurationMS: 10},
			{DurationMS: 10},
			{DurationMS: 50}, // Last run is 5x slower
		},
	}
	Save(hash, p)
	defer os.Remove(ProfilePath(hash))

	conn := &mockDB{
		planJSON: `[{"Plan": {"Node Type": "Limit", "Plans": []}}]`, // plan doesn't matter for this check
	}

	warnings := PlanCheck(context.Background(), "SELECT * FROM users", conn)

	foundRegression := false
	for _, w := range warnings {
		if w.Severity == "info" && w.Message == "Last run was 5.0x slower than average (50ms vs 10ms avg)" {
			foundRegression = true
		}
	}

	if !foundRegression {
		t.Errorf("did not find expected regression info: %v", warnings)
	}
}
