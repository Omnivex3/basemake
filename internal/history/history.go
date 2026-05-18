package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Entry represents a single query execution record.
type Entry struct {
	ID                int64     `json:"id"`
	Question          string    `json:"question"`
	SQLGenerated      string    `json:"sql_generated"`
	DatabaseName      string    `json:"database_name"`
	ExecutedAt        time.Time `json:"executed_at"`
	ExecutionTimeMs   float64   `json:"execution_time_ms"`
	RowCount          int       `json:"row_count"`
	WasNaturalLanguage bool     `json:"was_natural_language"`
	ProviderUsed      string    `json:"provider_used,omitempty"`
	ModelUsed         string    `json:"model_used,omitempty"`
}

var db *sql.DB

func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dbai", "history.db")
}

// Init opens or creates the history database.
func Init() error {
	path := dbPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open history db: %w", err)
	}

	// Enable WAL for concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("wal: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS query_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			question TEXT NOT NULL,
			sql_generated TEXT NOT NULL,
			database_name TEXT NOT NULL DEFAULT '',
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			execution_time_ms REAL DEFAULT 0,
			row_count INTEGER DEFAULT 0,
			was_natural_language INTEGER DEFAULT 0,
			provider_used TEXT DEFAULT '',
			model_used TEXT DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_history_time ON query_history(executed_at DESC);
	`); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	return nil
}

// Record saves a query execution to history.
func Record(e Entry) error {
	if db == nil {
		if err := Init(); err != nil {
			return err
		}
	}

	_, err := db.Exec(
		`INSERT INTO query_history (question, sql_generated, database_name, executed_at, execution_time_ms, row_count, was_natural_language, provider_used, model_used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Question, e.SQLGenerated, e.DatabaseName,
		e.ExecutedAt.UTC().Format(time.RFC3339),
		e.ExecutionTimeMs, e.RowCount,
		boolToInt(e.WasNaturalLanguage),
		e.ProviderUsed, e.ModelUsed,
	)
	return err
}

// Recent returns the most recent queries for context compounding.
// Used to prepend recent query patterns to AI prompts.
func Recent(limit int) ([]Entry, error) {
	if db == nil {
		return nil, nil
	}

	rows, err := db.Query(
		`SELECT id, question, sql_generated, database_name, executed_at, execution_time_ms, row_count, was_natural_language, provider_used, model_used
		FROM query_history
		WHERE was_natural_language = 1
		ORDER BY executed_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		var nlInt int
		if err := rows.Scan(&e.ID, &e.Question, &e.SQLGenerated, &e.DatabaseName,
			&ts, &e.ExecutionTimeMs, &e.RowCount, &nlInt,
			&e.ProviderUsed, &e.ModelUsed); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		e.WasNaturalLanguage = nlInt == 1
		t, parseErr := time.Parse(time.RFC3339, ts)
		if parseErr == nil {
			e.ExecutedAt = t
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// List returns the most recent N entries regardless of type.
func List(limit int) ([]Entry, error) {
	if db == nil {
		return nil, nil
	}

	rows, err := db.Query(
		`SELECT id, question, sql_generated, database_name, executed_at, execution_time_ms, row_count, was_natural_language, provider_used, model_used
		FROM query_history
		ORDER BY executed_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		var nlInt int
		if err := rows.Scan(&e.ID, &e.Question, &e.SQLGenerated, &e.DatabaseName,
			&ts, &e.ExecutionTimeMs, &e.RowCount, &nlInt,
			&e.ProviderUsed, &e.ModelUsed); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		e.WasNaturalLanguage = nlInt == 1
		t, parseErr := time.Parse(time.RFC3339, ts)
		if parseErr == nil {
			e.ExecutedAt = t
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// Close closes the history database.
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// BuildPromptWithHistory appends recent query history context to a system prompt.
func BuildPromptWithHistory(schemaPrompt string, historyDepth int) string {
	prompt := `You are a SQL expert. Given the following database schema, convert the user's natural language question into a SQL query.

Rules:
- Generate PostgreSQL-compatible SQL
- Return ONLY the SQL query — no markdown, no backticks, no explanations
- Use proper formatting with newlines
- If the question is ambiguous, make a reasonable assumption and add a comment explaining it`

	// Add recent history for context compounding
	entries, err := Recent(historyDepth)
	if err == nil && len(entries) > 0 {
		prompt += "\n\nRecent queries you've helped with:\n"
		for _, e := range entries {
			prompt += fmt.Sprintf("- Question: %s\n  SQL: %s\n", e.Question, e.SQLGenerated)
		}
	}

	prompt += "\n\nSchema:\n" + schemaPrompt
	return prompt
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
