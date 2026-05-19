package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"time"

	"github.com/DynamicKarabo/basemake/internal/db"
)

// scheduleWatches runs in a background goroutine, polling active watches
// on their configured intervals and recording results.
func (s *Server) scheduleWatches() {
	// Wait a moment for the server to start
	time.Sleep(2 * time.Second)
	log.Printf("[watcher] Starting watch scheduler...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkWatches()
		case <-s.done:
			log.Printf("[watcher] Shutting down watch scheduler")
			return
		}
	}
}

func (s *Server) checkWatches() {
	watches, err := s.store.ListActiveWatches()
	if err != nil {
		log.Printf("[watcher] Error listing watches: %v", err)
		return
	}

	now := time.Now()

	for _, w := range watches {
		// Check if this watch is due
		if !isDue(&w, now) {
			continue
		}

		go s.executeWatch(&w)
	}
}

// isDue checks if a watch is due to run based on its interval and last run time.
func isDue(w *Watch, now time.Time) bool {
	if w.LastRunAt == nil || *w.LastRunAt == "" {
		return true // never run
	}

	lastRun, err := time.Parse("2006-01-02 15:04:05", *w.LastRunAt)
	if err != nil {
		// Try parsing with T separator (some SQLite drivers use this)
		lastRun, err = time.Parse("2006-01-02T15:04:05", *w.LastRunAt)
		if err != nil {
			return true // can't parse, run anyway
		}
	}

	nextRun := lastRun.Add(time.Duration(w.IntervalSec) * time.Second)
	return now.After(nextRun) || now.Equal(nextRun)
}

// executeWatch runs a single watch query and records the result.
func (s *Server) executeWatch(w *Watch) {
	start := time.Now()

	var durationMs int64
	var rowCount int
	var resultHash string
	var alert bool
	var alertReason string
	var errorMsg string

	// Determine which database to query
	dsn := w.DSN
	if dsn == "" {
		// Use the DSN from the active connection stored in config
		dsn, _ = db.LoadDSN()
	}

	if dsn == "" {
		log.Printf("[watcher] Watch %d (%s): no DSN configured — skipping", w.ID, w.Label)
		return
	}

	// Open connection and execute query using centralized internal/db
	conn, err := db.Connect(dsn)
	if err != nil {
		errorMsg = fmt.Sprintf("connect: %v", err)
		alert = true
		alertReason = errorMsg
		log.Printf("[watcher] Watch %d (%s): %s", w.ID, w.Label, errorMsg)
		s.recordWatchResult(w.ID, durationMs, rowCount, resultHash, alert, alertReason, errorMsg)
		return
	}
	defer conn.Close()

	rows, err := conn.Query(context.Background(), w.SQL)
	if err != nil {
		errorMsg = fmt.Sprintf("query: %v", err)
		alert = true
		alertReason = errorMsg
		log.Printf("[watcher] Watch %d (%s): %s", w.ID, w.Label, errorMsg)
		s.recordWatchResult(w.ID, durationMs, rowCount, resultHash, alert, alertReason, errorMsg)
		return
	}
	defer rows.Close()

	// Read rows to count and hash
	cols := rows.Columns()
	rowCount = 0
	hasher := sha256.New()
	scanBuf := make([]interface{}, len(cols))
	scanPtrs := make([]interface{}, len(cols))

	for rows.Next() {
		for i := range scanBuf {
			scanPtrs[i] = &scanBuf[i]
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			continue
		}
		rowCount++
		for _, v := range scanBuf {
			switch val := v.(type) {
			case []byte:
				hasher.Write(val)
			case string:
				hasher.Write([]byte(val))
			default:
				hasher.Write([]byte(fmt.Sprint(val)))
			}
		}
		hasher.Write([]byte("\n"))
	}

	resultHash = fmt.Sprintf("%x", hasher.Sum(nil)[:8])
	durationMs = time.Since(start).Milliseconds()

	// Check threshold alert
	if w.ThresholdMs > 0 && durationMs > int64(w.ThresholdMs) {
		alert = true
		alertReason = fmt.Sprintf("slow: %dms (threshold: %dms)", durationMs, w.ThresholdMs)
		log.Printf("[watcher] ⚠️ Watch %d (%s): %s", w.ID, w.Label, alertReason)
	}

	// Check for data regression: compare with previous result hash
	if !alert && w.ThresholdMs == 0 {
		prevResults, _ := s.store.ListWatchResults(w.ID, 2)
		if len(prevResults) >= 2 {
			// Compare current hash with most recent successful run
			prevHash := prevResults[0].ResultHash
			// Find the previous non-alert result for comparison
			for _, pr := range prevResults {
				if !pr.Alert && pr.ResultHash != "" && pr.ID != prevResults[0].ID {
					prevHash = pr.ResultHash
					break
				}
			}
			if resultHash != prevHash {
				alert = true
				alertReason = fmt.Sprintf("result changed: hash %s → %s", prevHash, resultHash)
				log.Printf("[watcher] ⚠️ Watch %d (%s): %s", w.ID, w.Label, alertReason)
			}
		}
	}

	s.recordWatchResult(w.ID, durationMs, rowCount, resultHash, alert, alertReason, errorMsg)
	_ = s.store.UpdateWatchLastRun(w.ID)

	if !alert {
		log.Printf("[watcher] Watch %d (%s): %d rows in %dms ✅", w.ID, w.Label, rowCount, durationMs)
	}
}

func (s *Server) recordWatchResult(watchID int64, durationMs int64, rowCount int, resultHash string, alert bool, alertReason, errorMsg string) {
	_, err := s.store.InsertWatchResult(&WatchResult{
		WatchID:     watchID,
		DurationMs:  durationMs,
		RowCount:    rowCount,
		ResultHash:  resultHash,
		Alert:       alert,
		AlertReason: alertReason,
		ErrorMsg:    errorMsg,
	})
	if err != nil {
		log.Printf("[watcher] Error recording result for watch %d: %v", watchID, err)
	}
}
