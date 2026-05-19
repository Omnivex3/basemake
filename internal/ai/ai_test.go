package ai

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	// ~4 chars per token
	short := "SELECT 1"
	if n := EstimateTokens(short); n < 1 || n > 3 {
		t.Errorf("EstimateTokens(%q) = %d, want ~2", short, n)
	}

	long := "SELECT * FROM users WHERE created_at > now() - interval '30 days'"
	if n := EstimateTokens(long); n < 5 || n > 25 {
		t.Errorf("EstimateTokens(long) = %d, want ~15", n)
	}

	empty := ""
	if n := EstimateTokens(empty); n != 0 {
		t.Errorf("EstimateTokens(empty) = %d, want 0", n)
	}
}

func TestEstimateCost_KnownModel(t *testing.T) {
	cost := EstimateCost("gpt-4", 500, 200) // 500 input + 200 output tokens
	if cost == "?" {
		t.Error("expected known cost for gpt-4")
	}
	if cost == "" {
		t.Error("expected non-empty cost string")
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	cost := EstimateCost("unknown-model-9000", 100, 100)
	if cost != "?" {
		t.Errorf("expected ?, got %q", cost)
	}
}

func TestEstimateCost_TinyTokens(t *testing.T) {
	cost := EstimateCost("gpt-4o-mini", 10, 5)
	if cost != "<$0.001" {
		t.Errorf("expected <$0.001 for tiny tokens, got %q", cost)
	}
}

func TestModelPricing_KnownModels(t *testing.T) {
	known := []string{"gpt-4", "gpt-4o", "gpt-4o-mini", "claude-sonnet-4-20250514", "claude-3-haiku", "claude-3-opus"}
	for _, model := range known {
		if _, ok := modelPricing[model]; !ok {
			t.Errorf("missing pricing for %s", model)
		}
	}
}

func TestEstimateCost_OutputZeroTokens(t *testing.T) {
	cost := EstimateCost("gpt-4", 1000, 0)
	if cost == "?" {
		t.Error("expected known cost for gpt-4 with zero output")
	}
}
