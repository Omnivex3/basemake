package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/DynamicKarabo/basemake/internal/server"
	"github.com/spf13/cobra"
)

var (
	pushServerURL string
	pushDuration  string
	pushUser      string
	historyLimit  int
	historyOffset int
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync data with the basemake team server",
}

var pushCmd = &cobra.Command{
	Use:   "push <sql>",
	Short: "Push a query event to the team server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname, _ := os.Hostname()

		if pushUser == "" {
			pushUser = hostname
		}

		var durationMs int64
		if pushDuration != "" {
			d, err := time.ParseDuration(pushDuration)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			durationMs = d.Milliseconds()
		}

		req := server.PushEventRequest{
			SQL:        args[0],
			DurationMs: durationMs,
			UserName:   pushUser,
			Hostname:   hostname,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		url := fmt.Sprintf("%s/api/events", pushServerURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("post to server: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("server returned %s", resp.Status)
		}

		fmt.Printf("✅ Event pushed to %s\n", pushServerURL)
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show team query history from the server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("%s/api/events?limit=%d&offset=%d", pushServerURL, historyLimit, historyOffset)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("fetch history: %w", err)
		}
		defer resp.Body.Close()

		var list server.ListEventsResponse
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		if list.Count == 0 {
			fmt.Println("No events yet.")
			return nil
		}

		fmt.Printf("Latest %d query events (server: %s)\n\n", list.Count, pushServerURL)
		for _, e := range list.Events {
			label := fmt.Sprintf("[%s]", truncate(e.SQL, 70))
			fmt.Printf("  %s  %s  %dms  %s\n",
				e.CreatedAt, label, e.DurationMs, e.UserName)
		}

		return nil
	},
}

var budgetPushCmd = &cobra.Command{
	Use:   "budget-push",
	Short: "Push local budgets to the team server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load budgets from .basemake/budgets.json
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		path := filepath.Join(cwd, ".basemake", "budgets.json")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		hostname, _ := os.Hostname()
		user := pushUser
		if user == "" {
			user = hostname
		}

		req := server.SyncBudgetsRequest{
			BudgetsJSON: string(data),
			UserName:    user,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		url := fmt.Sprintf("%s/api/budgets/sync", pushServerURL)
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("push budgets: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("server returned %s", resp.Status)
		}

		fmt.Printf("✅ Budgets synced to %s\n", pushServerURL)
		return nil
	},
}

var budgetPullCmd = &cobra.Command{
	Use:   "budget-pull",
	Short: "Fetch latest budgets from the team server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("%s/api/budgets/latest", pushServerURL)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("fetch budgets: %w", err)
		}
		defer resp.Body.Close()

		var bs server.BudgetSnapshot
		if err := json.NewDecoder(resp.Body).Decode(&bs); err != nil {
			return fmt.Errorf("decode: %w", err)
		}

		if bs.BudgetsJSON == "" {
			fmt.Println("No budgets synced to server yet.")
			return nil
		}

		fmt.Printf("Latest budgets (synced %s by %s):\n\n", bs.CreatedAt, bs.UserName)
		fmt.Println(bs.BudgetsJSON)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(pushCmd)
	syncCmd.AddCommand(historyCmd)
	syncCmd.AddCommand(budgetPushCmd)
	syncCmd.AddCommand(budgetPullCmd)

	defaultServerURL := "http://localhost:9876"

	pushCmd.Flags().StringVar(&pushServerURL, "server", defaultServerURL, "Server URL")
	pushCmd.Flags().StringVar(&pushDuration, "duration", "", "Query duration (e.g. 150ms)")
	pushCmd.Flags().StringVar(&pushUser, "user", "", "User name (default: hostname)")

	historyCmd.Flags().StringVar(&pushServerURL, "server", defaultServerURL, "Server URL")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "Number of events to show")
	historyCmd.Flags().IntVar(&historyOffset, "offset", 0, "Offset for pagination")

	budgetPushCmd.Flags().StringVar(&pushServerURL, "server", defaultServerURL, "Server URL")
	budgetPushCmd.Flags().StringVar(&pushUser, "user", "", "User name")

	budgetPullCmd.Flags().StringVar(&pushServerURL, "server", defaultServerURL, "Server URL")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
