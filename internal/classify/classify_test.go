package classify

import (
	"testing"
)

func TestClassify_SQLKeywords(t *testing.T) {
	tests := []struct {
		input  string
		intent Intent
	}{
		{"SELECT * FROM users", IntentSQL},
		{"select id, name from orders where status = 'active'", IntentSQL},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", IntentSQL},
		{"EXPLAIN ANALYZE SELECT * FROM orders", IntentSQL},
		{"INSERT INTO users (name) VALUES ('test')", IntentSQL},
		{"UPDATE orders SET status = 'shipped' WHERE id = 1", IntentSQL},
		{"DELETE FROM logs WHERE created_at < '2024-01-01'", IntentSQL},
		{"CREATE INDEX idx_name ON users (name)", IntentSQL},
		{"SHOW TABLES", IntentSQL},
		{"DESCRIBE users", IntentSQL},
	}

	for _, tt := range tests {
		got := Classify(tt.input)
		if got != tt.intent {
			t.Errorf("Classify(%q) = %v, want %v", tt.input, got, tt.intent)
		}
	}
}

func TestClassify_DataRequests(t *testing.T) {
	tests := []struct {
		input  string
		intent Intent
	}{
		{"show me all users", IntentSQL},
		{"list active orders", IntentSQL},
		{"find customers in New York", IntentSQL},
		{"count orders by status", IntentSQL},
		{"get me the latest transactions", IntentSQL},
		{"give me all products under $50", IntentSQL},
		{"fetch recent logs", IntentSQL},
	}

	for _, tt := range tests {
		got := Classify(tt.input)
		if got != tt.intent {
			t.Errorf("Classify(%q) = %v, want %v", tt.input, got, tt.intent)
		}
	}
}

func TestClassify_AnalysisRequests(t *testing.T) {
	tests := []struct {
		input  string
		intent Intent
	}{
		{"why is my dashboard slow?", IntentAgent},
		{"what changed since last deploy?", IntentAgent},
		{"is this query normal?", IntentAgent},
		{"explain this query plan", IntentAgent},
		{"analyze the performance of orders query", IntentAgent},
		{"what queries are slow right now", IntentAgent},
		{"debug why this is taking so long", IntentAgent},
		{"compare query performance between yesterday and today", IntentAgent},
		{"root cause of the regression", IntentAgent},
		{"check if there's schema drift", IntentAgent},
	}

	for _, tt := range tests {
		got := Classify(tt.input)
		if got != tt.intent {
			t.Errorf("Classify(%q) = %v, want %v", tt.input, got, tt.intent)
		}
	}
}

func TestClassify_Empty(t *testing.T) {
	if got := Classify(""); got != IntentSQL {
		t.Errorf("Classify('') = %v, want IntentSQL", got)
	}
}

func TestClassify_ShortDefault(t *testing.T) {
	// Short non-obvious queries default to SQL (fast path)
	if got := Classify("how many users?"); got != IntentSQL {
		t.Errorf("Classify('how many users?') = %v, want IntentSQL", got)
	}

	// Longer analysis question
	if got := Classify("how many users have we onboarded this month and what's the trend compared to last month?"); got != IntentAgent {
		t.Errorf("Classify(long analysis question) = %v, want IntentAgent", got)
	}
}
