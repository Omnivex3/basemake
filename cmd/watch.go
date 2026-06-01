package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DynamicKarabo/basemake/internal/license"
	"github.com/DynamicKarabo/basemake/internal/server"
	"github.com/spf13/cobra"
)

var (
	watchServerURL string
	watchEvery     string
	watchThreshold string
	watchLabel     string
	watchUser      string
	watchDSN       string
	watchLimit     int
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor a query on a schedule, alert on regression",
	Long: `Schedule a query to run periodically and alert when results change or slow down.

Runs via the basemake server daemon. Add a watch with:

  basemake watch "SELECT COUNT(*) FROM orders" --every 5m
  basemake watch queries/kpi.sql --every 1h --threshold 2s
  basemake watch list
  basemake watch stop <id>
  basemake watch logs <id>`,

	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		// Proxy to watch add
		return watchAddCmd.RunE(watchAddCmd, args)
	},
}

var watchAddCmd = &cobra.Command{
	Use:   "add <sql|file.sql>",
	Short: "Add a new watch to monitor a query",
	Args:  cobra.ExactArgs(1),
	Example: `  basemake watch add "SELECT COUNT(*) FROM orders" --every 5m
  basemake watch add queries/revenue.sql --every 1h --threshold 2s
  basemake watch add "SELECT * FROM users" --every 10m --label "User count check"`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if !requireLicense(license.FeatureWatch) {
			return fmt.Errorf("license required for watch feature")
		}
		input := args[0]

		// Resolve SQL — inline string or file path
		sql, err := readSQL(input)
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}

		// Parse interval
		intervalSec := 300 // default 5 min
		if watchEvery != "" {
			d, err := parseDuration(watchEvery)
			if err != nil {
				return fmt.Errorf("invalid --every: %w", err)
			}
			intervalSec = int(d.Seconds())
		}

		// Parse threshold
		var thresholdMs int
		if watchThreshold != "" {
			d, err := parseDuration(watchThreshold)
			if err != nil {
				return fmt.Errorf("invalid --threshold: %w", err)
			}
			thresholdMs = int(d.Milliseconds())
		}

		// Set label from --label or derive from SQL
		label := watchLabel
		if label == "" {
			label = truncateForDisplay(sql, 60)
		}

		// Determine user
		user := watchUser
		if user == "" {
			hostname, _ := os.Hostname()
			user = hostname
		}

		req := server.CreateWatchRequest{
			SQL:         sql,
			Label:       label,
			IntervalSec: intervalSec,
			ThresholdMs: thresholdMs,
			DSN:         watchDSN,
			CreatedBy:   user,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		url := fmt.Sprintf("%s/api/watches", watchServerURL)
		resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
		if err != nil {
			return fmt.Errorf("post to server: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("server returned %s", resp.Status)
		}

		var result map[string]int64
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		fmt.Printf("✅ Watch created (id=%d)\n", result["id"])
		fmt.Printf("   Query: %s\n", truncateForDisplay(sql, 60))
		fmt.Printf("   Every: %s\n", watchEvery)
		if watchThreshold != "" {
			fmt.Printf("   Threshold: %s\n", watchThreshold)
		}
		fmt.Printf("   Label: %s\n", label)
		return nil
	},
}

var watchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active watches",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("%s/api/watches", watchServerURL)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("fetch watches: %w", err)
		}
		defer resp.Body.Close()

		var result struct {
			Watches []server.Watch `json:"watches"`
			Count   int            `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		if result.Count == 0 {
			fmt.Println("No watches configured.")
			fmt.Println("  Add one: basemake watch add \"SELECT COUNT(*) FROM orders\" --every 5m")
			return nil
		}

		fmt.Printf("Active watches (%d):\n\n", result.Count)
		for _, w := range result.Watches {
			status := "🟢"
			if !w.Enabled {
				status = "🔴"
			}
			lastRun := "never"
			if w.LastRunAt != nil && *w.LastRunAt != "" {
				lastRun = *w.LastRunAt
			}
			every := formatInterval(w.IntervalSec)
			fmt.Printf("  %s #%d: %s\n", status, w.ID, w.Label)
			fmt.Printf("       Every %s | Last run: %s | Created by: %s\n", every, lastRun, w.CreatedBy)
			if w.ThresholdMs > 0 {
				fmt.Printf("       Threshold: %dms\n", w.ThresholdMs)
			}
			fmt.Println()
		}
		return nil
	},
}

var watchStopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "Stop and remove a watch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid watch ID: %s", args[0])
		}

		url := fmt.Sprintf("%s/api/watches/%d", watchServerURL, id)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("delete watch: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned %s", resp.Status)
		}

		fmt.Printf("✅ Watch #%d removed\n", id)
		return nil
	},
}

var watchLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Show watch execution history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid watch ID: %s", args[0])
		}

		url := fmt.Sprintf("%s/api/watches/%d/results?limit=%d", watchServerURL, id, watchLimit)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("fetch results: %w", err)
		}
		defer resp.Body.Close()

		var result struct {
			Results []server.WatchResult `json:"results"`
			Count   int                  `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		if result.Count == 0 {
			fmt.Printf("No results for watch #%d yet.\n", id)
			return nil
		}

		fmt.Printf("Watch #%d — last %d runs:\n\n", id, result.Count)
		for _, r := range result.Results {
			alert := "✅"
			reason := ""
			if r.Alert {
				alert = "⚠️"
				reason = " " + r.AlertReason
			}
			errorInfo := ""
			if r.ErrorMsg != "" {
				errorInfo = " ❌ " + r.ErrorMsg
			}
			fmt.Printf("  %s %s | %dms | %d rows%s%s\n",
				alert, r.ExecutedAt, r.DurationMs, r.RowCount, reason, errorInfo)
		}
		return nil
	},
}

// parseDuration parses a human-friendly interval like "5m", "1h", "30s".
func parseDuration(s string) (time.Duration, error) {
	// Accept plain number as seconds
	if d, err := strconv.Atoi(s); err == nil {
		return time.Duration(d) * time.Second, nil
	}
	return time.ParseDuration(s)
}

func truncateForDisplay(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func formatInterval(sec int) string {
	switch {
	case sec < 60:
		return fmt.Sprintf("%ds", sec)
	case sec < 3600:
		return fmt.Sprintf("%dm", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%dh", sec/3600)
	default:
		return fmt.Sprintf("%dd", sec/86400)
	}
}

func init() {
	defaultServerURL := "http://localhost:9876"

	rootCmd.AddCommand(watchCmd)
	watchCmd.AddCommand(watchAddCmd)
	watchCmd.AddCommand(watchListCmd)
	watchCmd.AddCommand(watchStopCmd)
	watchCmd.AddCommand(watchLogsCmd)

	// Shared flags
	watchCmd.PersistentFlags().StringVar(&watchServerURL, "server", defaultServerURL, "Server URL")

	// Add flags
	watchAddCmd.Flags().StringVar(&watchEvery, "every", "5m", "Check interval (e.g. 5m, 1h, 30s)")
	watchAddCmd.Flags().StringVar(&watchThreshold, "threshold", "", "Alert if query exceeds this duration (e.g. 2s, 500ms)")
	watchAddCmd.Flags().StringVar(&watchLabel, "label", "", "Human-readable label")
	watchAddCmd.Flags().StringVar(&watchUser, "user", "", "User name (default: hostname)")
	watchAddCmd.Flags().StringVar(&watchDSN, "dsn", "", "Database DSN (default: active connection)")

	// Logs flags
	watchLogsCmd.Flags().IntVar(&watchLimit, "limit", 20, "Number of results to show")
}
