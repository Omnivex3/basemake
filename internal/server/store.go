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

	CREATE TABLE IF NOT EXISTS watches (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		sql          TEXT NOT NULL,
		label        TEXT NOT NULL DEFAULT '',
		interval_sec INTEGER NOT NULL DEFAULT 300,
		threshold_ms INTEGER NOT NULL DEFAULT 0,
		dsn          TEXT NOT NULL DEFAULT '',
		enabled      INTEGER NOT NULL DEFAULT 1,
		created_by   TEXT NOT NULL DEFAULT '',
		last_run_at  DATETIME,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS watch_results (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		watch_id     INTEGER NOT NULL REFERENCES watches(id),
		duration_ms  INTEGER NOT NULL DEFAULT 0,
		row_count    INTEGER DEFAULT 0,
		result_hash  TEXT,
		alert        INTEGER NOT NULL DEFAULT 0,
		alert_reason TEXT,
		error_msg    TEXT,
		executed_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_watch_results_watch ON watch_results(watch_id, executed_at DESC);
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

// --- Watch Methods ---

// InsertWatch creates a new watch and returns its ID.
func (s *Store) InsertWatch(w *Watch) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO watches (sql, label, interval_sec, threshold_ms, dsn, enabled, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.SQL, w.Label, w.IntervalSec, w.ThresholdMs, w.DSN, boolToInt(w.Enabled), w.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("insert watch: %w", err)
	}
	return result.LastInsertId()
}

// ListWatches returns all watches ordered by creation time.
func (s *Store) ListWatches() ([]Watch, error) {
	rows, err := s.db.Query(
		`SELECT id, sql, COALESCE(label, ''), interval_sec, threshold_ms,
		        COALESCE(dsn, ''), enabled, COALESCE(created_by, ''),
		        last_run_at, created_at
		 FROM watches
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list watches: %w", err)
	}
	defer rows.Close()

	var watches []Watch
	for rows.Next() {
		var w Watch
		if err := rows.Scan(
			&w.ID, &w.SQL, &w.Label, &w.IntervalSec, &w.ThresholdMs,
			&w.DSN, &w.EnabledInt, &w.CreatedBy, &w.LastRunAt, &w.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watch: %w", err)
		}
		w.Enabled = w.EnabledInt == 1
		watches = append(watches, w)
	}
	return watches, rows.Err()
}

// ListActiveWatches returns all enabled watches where next run is due.
func (s *Store) ListActiveWatches() ([]Watch, error) {
	rows, err := s.db.Query(
		`SELECT id, sql, COALESCE(label, ''), interval_sec, threshold_ms,
		        COALESCE(dsn, ''), enabled, COALESCE(created_by, ''),
		        last_run_at, created_at
		 FROM watches
		 WHERE enabled = 1
		 ORDER BY COALESCE(last_run_at, '1970-01-01') ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active watches: %w", err)
	}
	defer rows.Close()

	var watches []Watch
	for rows.Next() {
		var w Watch
		if err := rows.Scan(
			&w.ID, &w.SQL, &w.Label, &w.IntervalSec, &w.ThresholdMs,
			&w.DSN, &w.EnabledInt, &w.CreatedBy, &w.LastRunAt, &w.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watch: %w", err)
		}
		w.Enabled = w.EnabledInt == 1
		watches = append(watches, w)
	}
	return watches, rows.Err()
}

// GetWatch returns a single watch by ID.
func (s *Store) GetWatch(id int64) (*Watch, error) {
	var w Watch
	err := s.db.QueryRow(
		`SELECT id, sql, COALESCE(label, ''), interval_sec, threshold_ms,
		        COALESCE(dsn, ''), enabled, COALESCE(created_by, ''),
		        last_run_at, created_at
		 FROM watches WHERE id = ?`, id,
	).Scan(&w.ID, &w.SQL, &w.Label, &w.IntervalSec, &w.ThresholdMs,
		&w.DSN, &w.EnabledInt, &w.CreatedBy, &w.LastRunAt, &w.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("watch %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get watch: %w", err)
	}
	w.Enabled = w.EnabledInt == 1
	return &w, nil
}

// UpdateWatchEnabled enables or disables a watch.
func (s *Store) UpdateWatchEnabled(id int64, enabled bool) error {
	result, err := s.db.Exec(`UPDATE watches SET enabled = ? WHERE id = ?`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("update watch: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("watch %d not found", id)
	}
	return nil
}

// DeleteWatch removes a watch and its results.
func (s *Store) DeleteWatch(id int64) error {
	_, err := s.db.Exec(`DELETE FROM watch_results WHERE watch_id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete watch results: %w", err)
	}
	result, err := s.db.Exec(`DELETE FROM watches WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete watch: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("watch %d not found", id)
	}
	return nil
}

// UpdateWatchLastRun updates the last_run_at timestamp for a watch.
func (s *Store) UpdateWatchLastRun(id int64) error {
	_, err := s.db.Exec(`UPDATE watches SET last_run_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// InsertWatchResult stores the result of a watch execution.
func (s *Store) InsertWatchResult(wr *WatchResult) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO watch_results (watch_id, duration_ms, row_count, result_hash, alert, alert_reason, error_msg)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		wr.WatchID, wr.DurationMs, wr.RowCount, wr.ResultHash, boolToInt(wr.Alert), wr.AlertReason, wr.ErrorMsg,
	)
	if err != nil {
		return 0, fmt.Errorf("insert watch result: %w", err)
	}
	return result.LastInsertId()
}

// ListWatchResults returns recent results for a watch.
func (s *Store) ListWatchResults(watchID int64, limit int) ([]WatchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, watch_id, duration_ms, row_count, COALESCE(result_hash, ''),
		        alert, COALESCE(alert_reason, ''), COALESCE(error_msg, ''), executed_at
		 FROM watch_results
		 WHERE watch_id = ?
		 ORDER BY executed_at DESC
		 LIMIT ?`, watchID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list watch results: %w", err)
	}
	defer rows.Close()

	var results []WatchResult
	for rows.Next() {
		var wr WatchResult
		if err := rows.Scan(
			&wr.ID, &wr.WatchID, &wr.DurationMs, &wr.RowCount,
			&wr.ResultHash, &wr.AlertInt, &wr.AlertReason, &wr.ErrorMsg, &wr.ExecutedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watch result: %w", err)
		}
		wr.Alert = wr.AlertInt == 1
		results = append(results, wr)
	}
	return results, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
