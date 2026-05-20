package classify

import (
	"strings"
	"unicode"
)

// Intent represents the type of user input.
type Intent int

const (
	IntentSQL    Intent = iota // Raw SQL or data request — fast path
	IntentAgent                // Analysis, debugging, explanation — agent path
)

// String returns a human-readable name for the intent.
func (i Intent) String() string {
	switch i {
	case IntentSQL:
		return "sql"
	case IntentAgent:
		return "agent"
	default:
		return "unknown"
	}
}

// analysisKeywords are words that signal the user is asking for analysis or debugging.
var analysisKeywords = []string{
	"why", "what's wrong", "what changed", "is this",
	"explain", "analyze", "debug", "slow", "regression",
	"anomaly", "different", "compare", "problem", "issue",
	"what happened", "bottleneck", "performance", "drift",
	"did this query", "how is", "investigate", "check",
	"root cause", "what's going on", "how come", "profile",
	"budget", "dashboard", "plan", "normal",
}

// Classify determines whether user input should go to the fast path (SQL gen)
// or the agent path (tool loop).
//
// It uses a lightweight keyword heuristic — no model call required.
//
// Returns IntentSQL when:
//   - Input starts with a SQL keyword (SELECT, WITH, EXPLAIN, INSERT, etc.)
//   - Input looks like a direct data request
//
// Returns IntentAgent when:
//   - Input contains analysis/debugging keywords
//   - Input is a question about database state or performance
func Classify(input string) Intent {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return IntentSQL
	}

	// Check if it looks like raw SQL (starts with a SQL keyword and structurally is SQL)
	firstWord := extractFirstWord(trimmed)
	upperWord := strings.ToUpper(firstWord)

	sqlKeywords := map[string]bool{
		"SELECT": true, "WITH": true, "INSERT": true,
		"UPDATE": true, "DELETE": true, "CREATE": true, "ALTER": true,
		"DROP": true, "TRUNCATE": true, "SET": true, "SHOW": true,
		"DESCRIBE": true, "BEGIN": true, "COMMIT": true, "ROLLBACK": true,
		"GRANT": true, "REVOKE": true, "VACUUM": true,
		"REINDEX": true, "CALL": true, "DO": true,
	}

	if sqlKeywords[upperWord] {
		return IntentSQL
	}

	// EXPLAIN / ANALYZE are ambiguous — check if followed by a SQL statement
	if upperWord == "EXPLAIN" || upperWord == "ANALYZE" {
		after := strings.TrimSpace(strings.TrimPrefix(trimmed, firstWord))
		nextWord := strings.ToUpper(extractFirstWord(after))
		if sqlKeywords[nextWord] || nextWord == "ANALYZE" {
			return IntentSQL
		}
		// "explain this query plan" → agent path
		return IntentAgent
	}

	// Check for data request patterns (fast path)
	lower := strings.ToLower(trimmed)
	if startsWithPattern(lower, []string{"show me", "list", "find", "count", "get", "give me", "fetch"}) {
		return IntentSQL
	}

	// Check for analysis keywords (agent path)
	for _, kw := range analysisKeywords {
		if strings.Contains(lower, kw) {
			return IntentAgent
		}
	}

	// Default: short queries that look like data requests go fast path
	if len(trimmed) < 60 {
		return IntentSQL
	}

	// Longer natural language questions default to agent
	return IntentAgent
}

// extractFirstWord returns the first word in a string.
func extractFirstWord(s string) string {
	s = strings.TrimSpace(s)
	idx := strings.IndexFunc(s, unicode.IsSpace)
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// startsWithPattern checks if s starts with any of the given patterns.
func startsWithPattern(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
