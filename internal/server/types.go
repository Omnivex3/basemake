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
