package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DynamicKarabo/basemake/internal/server"
	"github.com/spf13/cobra"
)

var (
	serverPort int
	serverDir  string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the basemake team server",
	Long: `Start the basemake team sync server.

The server stores query history, budget snapshots, and enables team-wide
collaboration. It runs as a lightweight daemon on your VPS or Docker host.

  basemake server start          # Start on default port 9876
  basemake server start --port 8080
  basemake server start --dir /data/basemake`,
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server daemon",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := server.NewStore(serverDir + "/basemake.db")
		if err != nil {
			return fmt.Errorf("init store: %w", err)
		}
		defer store.Close()

		svr := server.NewServer(store, serverPort, getBuildInfo().version)

		// Graceful shutdown on SIGINT/SIGTERM
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println("\nShutting down...")
			store.Close()
			os.Exit(0)
		}()

		return svr.Start()
	},
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status and statistics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := server.NewStore(serverDir + "/basemake.db")
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer store.Close()

		count, _ := store.EventCount()

		bs, _ := store.LatestBudgets()

		fmt.Printf("basemake server\n")
		fmt.Printf("  Data dir: %s\n", serverDir)
		fmt.Printf("  Events recorded: %d\n", count)
		if bs != nil {
			fmt.Printf("  Budgets last synced: %s (by %s)\n", bs.CreatedAt, bs.UserName)
		} else {
			fmt.Printf("  Budgets: not synced yet\n")
		}
		fmt.Printf("\nAPI endpoints:\n")
		fmt.Printf("  POST /api/events        Push a query event\n")
		fmt.Printf("  GET  /api/events        List recent events\n")
		fmt.Printf("  POST /api/budgets/sync  Push budgets to server\n")
		fmt.Printf("  GET  /api/budgets/latest Get latest budgets\n")
		fmt.Printf("  GET  /api/health        Health check\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStatusCmd)

	defaultDir := server.DefaultDataDir()

	serverStartCmd.Flags().IntVarP(&serverPort, "port", "p", server.DefaultPort, "HTTP port")
	serverStartCmd.Flags().StringVar(&serverDir, "dir", defaultDir, "Data directory for SQLite storage")

	serverStatusCmd.Flags().StringVar(&serverDir, "dir", defaultDir, "Data directory")
}
