package server

import "time"

// --- Events ---

// Event represents a single query execution recorded by the team.
type Event struct {
	ID          int64  `json:"id"`
	SQL         string `json:"sql"`
	DurationMs  int64  `json:"duration_ms"`
	PlanJSON    string `json:"plan_json,omitempty"`
	RowsAffected int64  `json:"rows_affected"`
	TableNames  string `json:"table_names,omitempty"`
	BudgetViolations string `json:"budget_violations,omitempty"`
	UserName    string `json:"user_name"`
	Hostname    string `json:"hostname"`
	CreatedAt   string `json:"created_at"`
}

// PushEventRequest is the payload for POST /api/events.
type PushEventRequest struct {
	SQL         string `json:"sql"`
	DurationMs  int64  `json:"duration_ms"`
	PlanJSON    string `json:"plan_json,omitempty"`
	RowsAffected int64  `json:"rows_affected"`
	TableNames  string `json:"table_names,omitempty"`
	BudgetViolations string `json:"budget_violations,omitempty"`
	UserName    string `json:"user_name"`
	Hostname    string `json:"hostname"`
}

// ListEventsResponse is the response for GET /api/events.
type ListEventsResponse struct {
	Events []Event `json:"events"`
	Count  int     `json:"count"`
}

// --- Budget Sync ---

// BudgetSnapshot is a point-in-time copy of budgets pushed to the server.
type BudgetSnapshot struct {
	ID        int64  `json:"id"`
	BudgetsJSON string `json:"budgets_json"`
	UserName  string `json:"user_name"`
	CreatedAt string `json:"created_at"`
}

// SyncBudgetsRequest is the payload for POST /api/budgets/sync.
type SyncBudgetsRequest struct {
	BudgetsJSON string `json:"budgets_json"`
	UserName    string `json:"user_name"`
}

// --- Health ---

// HealthResponse is returned by GET /api/health.
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	EventCount int   `json:"event_count"`
}

// --- Defaults ---

const DefaultPort = 9876

var StartTime = time.Now()

// --- Watch Types ---

// Watch represents a scheduled query monitoring task.
type Watch struct {
	ID          int64  `json:"id"`
	SQL         string `json:"sql"`
	Label       string `json:"label"`
	IntervalSec int    `json:"interval_sec"`
	ThresholdMs int    `json:"threshold_ms"`
	DSN         string `json:"dsn"`
	Enabled     bool   `json:"enabled"`
	EnabledInt  int    `json:"-"`
	CreatedBy   string `json:"created_by"`
	LastRunAt   *string `json:"last_run_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// WatchResult is a single execution result for a watch.
type WatchResult struct {
	ID          int64  `json:"id"`
	WatchID     int64  `json:"watch_id"`
	DurationMs  int64  `json:"duration_ms"`
	RowCount    int    `json:"row_count"`
	ResultHash  string `json:"result_hash,omitempty"`
	Alert       bool   `json:"alert"`
	AlertInt    int    `json:"-"`
	AlertReason string `json:"alert_reason,omitempty"`
	ErrorMsg    string `json:"error_msg,omitempty"`
	ExecutedAt  string `json:"executed_at"`
}

// CreateWatchRequest is the payload for POST /api/watches.
type CreateWatchRequest struct {
	SQL         string `json:"sql"`
	Label       string `json:"label"`
	IntervalSec int    `json:"interval_sec"`
	ThresholdMs int    `json:"threshold_ms"`
	DSN         string `json:"dsn"`
	CreatedBy   string `json:"created_by"`
}
