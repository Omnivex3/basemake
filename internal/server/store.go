package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver (already a dependency)
)

// Store provides SQLite-backed persistence for the server.
type Store struct {
	db *sql.DB
}

// NewStore opens or creates the SQLite database at the given path.
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// WAL mode for concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("migrate store: %w", err)
	}

	return store, nil
}

// migrate creates tables on first run.
func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		sql         TEXT NOT NULL,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		plan_json   TEXT,
		rows_affected INTEGER DEFAULT 0,
		table_names TEXT,
		budget_violations TEXT,
		user_name   TEXT NOT NULL DEFAULT '',
		hostname    TEXT NOT NULL DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC);

	CREATE TABLE IF NOT EXISTS budgets (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		budgets_json TEXT NOT NULL,
		user_name    TEXT NOT NULL DEFAULT '',
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// InsertEvent stores a new query event.
func (s *Store) InsertEvent(e *Event) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO events (sql, duration_ms, plan_json, rows_affected, table_names, budget_violations, user_name, hostname)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.SQL, e.DurationMs, e.PlanJSON, e.RowsAffected, e.TableNames, e.BudgetViolations, e.UserName, e.Hostname,
	)
	if err != nil {
		return 0, fmt.Errorf("insert event: %w", err)
	}
	return result.LastInsertId()
}

// ListEvents returns the most recent events, ordered by creation time descending.
func (s *Store) ListEvents(limit, offset int) ([]Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT id, sql, duration_ms, COALESCE(plan_json, ''), rows_affected,
		        COALESCE(table_names, ''), COALESCE(budget_violations, ''),
		        user_name, hostname, created_at
		 FROM events
		 ORDER BY created_at DESC, id DESC
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID, &e.SQL, &e.DurationMs, &e.PlanJSON,
			&e.RowsAffected, &e.TableNames, &e.BudgetViolations,
			&e.UserName, &e.Hostname, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// EventCount returns the total number of events.
func (s *Store) EventCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	return count, err
}

// SyncBudgets stores a new budget snapshot.
func (s *Store) SyncBudgets(budgetsJSON, userName string) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO budgets (budgets_json, user_name) VALUES (?, ?)`,
		budgetsJSON, userName,
	)
	if err != nil {
		return 0, fmt.Errorf("sync budgets: %w", err)
	}
	return result.LastInsertId()
}

// LatestBudgets returns the most recent budget snapshot.
func (s *Store) LatestBudgets() (*BudgetSnapshot, error) {
	var bs BudgetSnapshot
	err := s.db.QueryRow(
		`SELECT id, budgets_json, user_name, created_at
		 FROM budgets
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1`,
	).Scan(&bs.ID, &bs.BudgetsJSON, &bs.UserName, &bs.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest budgets: %w", err)
	}
	return &bs, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
