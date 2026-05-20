package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Server is the basemake team server.
type Server struct {
	store   *Store
	port    int
	version string
	started time.Time
	done    chan struct{}
}

// NewServer creates a new server with the given store and options.
func NewServer(store *Store, port int, version string) *Server {
	return &Server{
		store:   store,
		port:    port,
		version: version,
		started: time.Now(),
		done:    make(chan struct{}),
	}
}

// Shutdown gracefully stops the server, its watch scheduler, and the store.
func (s *Server) Shutdown() {
	close(s.done)
	_ = s.store.Close()
}

// Start runs the HTTP server on the configured port and starts the watch scheduler.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", withCORS(s.handleHealth))
	mux.HandleFunc("/api/events", withCORS(s.handleEvents))
	mux.HandleFunc("/api/events/", withCORS(s.handleEvents))
	mux.HandleFunc("/api/budgets/sync", withCORS(s.handleBudgetsSync))
	mux.HandleFunc("/api/budgets/latest", withCORS(s.handleBudgetsLatest))
	mux.HandleFunc("/api/watches", withCORS(s.handleWatches))
	mux.HandleFunc("/api/watches/", withCORS(s.handleWatchesByID))

	// Root redirect to health
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/api/health", http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(w, r)
	})

	hostname, _ := os.Hostname()

	log.Printf("basemake server starting on %s (pid=%d, host=%s)", addr, os.Getpid(), hostname)
	log.Printf("  API: http://localhost:%d/api/health", s.port)
	log.Printf("  Events: http://localhost:%d/api/events", s.port)
	log.Printf("  Budgets: http://localhost:%d/api/budgets/latest", s.port)
	log.Printf("  Watches: http://localhost:%d/api/watches", s.port)

	// Start watch scheduler in background
	go s.scheduleWatches()

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return srv.ListenAndServe()
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "use GET")
		return
	}

	count, _ := s.store.EventCount()

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:     "ok",
		Version:    s.version,
		Uptime:     time.Since(s.started).Round(time.Second).String(),
		EventCount: count,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listEvents(w, r)
	case http.MethodPost:
		s.pushEvent(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "use GET or POST")
	}
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	events, err := s.store.ListEvents(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if events == nil {
		events = []Event{}
	}

	writeJSON(w, http.StatusOK, ListEventsResponse{
		Events: events,
		Count:  len(events),
	})
}

func (s *Server) pushEvent(w http.ResponseWriter, r *http.Request) {
	var req PushEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.SQL == "" {
		writeError(w, http.StatusBadRequest, "sql is required")
		return
	}

	event := &Event{
		SQL:              req.SQL,
		DurationMs:       req.DurationMs,
		PlanJSON:         req.PlanJSON,
		RowsAffected:     req.RowsAffected,
		TableNames:       req.TableNames,
		BudgetViolations: req.BudgetViolations,
		UserName:         req.UserName,
		Hostname:         req.Hostname,
	}

	id, err := s.store.InsertEvent(event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) handleBudgetsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}

	var req SyncBudgetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.BudgetsJSON == "" {
		writeError(w, http.StatusBadRequest, "budgets_json is required")
		return
	}

	id, err := s.store.SyncBudgets(req.BudgetsJSON, req.UserName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) handleBudgetsLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "use GET")
		return
	}

	bs, err := s.store.LatestBudgets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if bs == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"budgets": nil, "message": "no budgets synced yet"})
		return
	}

	writeJSON(w, http.StatusOK, bs)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// withCORS wraps a handler with permissive CORS headers for local development.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// --- Watch Handlers ---

func (s *Server) handleWatches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listWatches(w, r)
	case http.MethodPost:
		s.createWatch(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "use GET or POST")
	}
}

func (s *Server) handleWatchesByID(w http.ResponseWriter, r *http.Request) {
	// Extract watch ID from path: /api/watches/<id>[/results]
	path := r.URL.Path
	parts := splitPath(path)

	if len(parts) < 3 {
		writeError(w, http.StatusBadRequest, "missing watch ID")
		return
	}

	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch ID")
		return
	}

	// /api/watches/<id>/results
	if len(parts) >= 4 && parts[3] == "results" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "use GET")
			return
		}
		s.listWatchResults(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getWatch(w, r, id)
	case http.MethodDelete:
		s.deleteWatch(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "use GET or DELETE")
	}
}

func (s *Server) listWatches(w http.ResponseWriter, r *http.Request) {
	watches, err := s.store.ListWatches()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if watches == nil {
		watches = []Watch{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"watches": watches,
		"count":   len(watches),
	})
}

func (s *Server) createWatch(w http.ResponseWriter, r *http.Request) {
	var req CreateWatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.SQL == "" {
		writeError(w, http.StatusBadRequest, "sql is required")
		return
	}
	if req.IntervalSec <= 0 {
		req.IntervalSec = 300 // default 5 min
	}
	if req.Label == "" {
		req.Label = truncateLabel(req.SQL)
	}

	watch := &Watch{
		SQL:         req.SQL,
		Label:       req.Label,
		IntervalSec: req.IntervalSec,
		ThresholdMs: req.ThresholdMs,
		DSN:         req.DSN,
		Enabled:     true,
		CreatedBy:   req.CreatedBy,
	}

	id, err := s.store.InsertWatch(watch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) getWatch(w http.ResponseWriter, r *http.Request, id int64) {
	watch, err := s.store.GetWatch(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, watch)
}

func (s *Server) deleteWatch(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.store.DeleteWatch(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) listWatchResults(w http.ResponseWriter, r *http.Request, watchID int64) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := s.store.ListWatchResults(watchID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if results == nil {
		results = []WatchResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

// splitPath splits a URL path into non-empty segments.
func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

func truncateLabel(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

// DefaultDataDir returns the default server data directory.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/basemake-server"
	}
	return filepath.Join(home, ".basemake", "server")
}
