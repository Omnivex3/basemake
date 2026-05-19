package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"

	"github.com/DynamicKarabo/basemake/internal/license"
	"github.com/DynamicKarabo/basemake/internal/server"
	"github.com/DynamicKarabo/basemake/internal/tui"
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
		if !requireLicense(license.FeatureServer) {
			os.Exit(1)
			return nil
		}
		store, err := server.NewStore(serverDir + "/basemake.db")
		if err != nil {
			return fmt.Errorf("init store: %w", err)
		}
		defer store.Close()

		svr := server.NewServer(store, serverPort, getBuildInfo().version)

		// ── Styled startup banner ──
		fmt.Println()
		fmt.Println(tui.ColoriseLogo(BannerASCII))
		fmt.Println()

		info := lipgloss.NewStyle().
			Foreground(tui.DimText).
			Render(getBuildInfo().version + "  ·  " + runtime.GOOS + "/" + runtime.GOARCH)

		addrStyle := lipgloss.NewStyle().
			Foreground(tui.Red).
			Bold(true)
		portInfo := addrStyle.Render(fmt.Sprintf("http://localhost:%d", serverPort))

		statusLine := lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Foreground(tui.Red).Render("◆"),
			"  ",
			info,
			"  │  ",
			tui.Dot(true, false),
			" Listening on ",
			portInfo,
		)
		fmt.Println(statusLine)
		fmt.Println()

		// API endpoints card
		methodStyle := lipgloss.NewStyle().Foreground(tui.Red)
		endpoints := []string{
			"  " + methodStyle.Render("GET  /api/health") + "       " + lipgloss.NewStyle().Foreground(tui.DimText).Render("Health check"),
			"  " + methodStyle.Render("POST /api/events") + "       " + lipgloss.NewStyle().Foreground(tui.DimText).Render("Push a query event"),
			"  " + methodStyle.Render("GET  /api/events") + "       " + lipgloss.NewStyle().Foreground(tui.DimText).Render("List recent events"),
			"  " + methodStyle.Render("POST /api/budgets/sync") + " " + lipgloss.NewStyle().Foreground(tui.DimText).Render("Push budgets"),
			"  " + methodStyle.Render("GET  /api/budgets/latest") + " " + lipgloss.NewStyle().Foreground(tui.DimText).Render("Latest budgets"),
			"  " + methodStyle.Render("GET  /api/watches") + "      " + lipgloss.NewStyle().Foreground(tui.DimText).Render("List watches"),
		}
		endpointBox := tui.SubBoxStyle.Render(
			lipgloss.NewStyle().Foreground(tui.Red).Bold(true).Render("  API Endpoints") + "\n" +
				strings.Join(endpoints, "\n"),
		)
		fmt.Println(endpointBox)
		fmt.Println()

		// Graceful shutdown on SIGINT/SIGTERM
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Foreground(tui.Yellow).Render("  Shutting down..."))
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

		// Styled status output
		fmt.Println()
		box := tui.BoxStyle.Render(strings.Join([]string{
			lipgloss.NewStyle().Foreground(tui.Red).Bold(true).Render("  basemake server"),
			"",
			"    " + tui.Dot(true, false) + " " + lipgloss.NewStyle().Foreground(tui.Text).Render("Data dir: ") + lipgloss.NewStyle().Foreground(tui.White).Render(serverDir),
			"    " + tui.Dot(true, false) + " " + lipgloss.NewStyle().Foreground(tui.Text).Render(fmt.Sprintf("Events: %d", count)),
			"",
			formatBudgetsLine(bs),
			"",
			lipgloss.NewStyle().Foreground(tui.DimText).Render("  API Endpoints"),
			"    " + lipgloss.NewStyle().Foreground(tui.Red).Render("http://localhost:"+fmt.Sprintf("%d", serverPort)+"/api/health"),
		}, "\n"))
		fmt.Println(box)
		fmt.Println()

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

func formatBudgetsLine(bs *server.BudgetSnapshot) string {
	if bs == nil {
		return "    " + tui.Dot(false, true) + " " + lipgloss.NewStyle().Foreground(tui.DimText).Render("Budgets: not synced yet")
	}
	return "    " + tui.Dot(true, false) + " " + lipgloss.NewStyle().Foreground(tui.Text).Render("Budgets synced: ") +
		lipgloss.NewStyle().Foreground(tui.DimText).Render(bs.CreatedAt) +
		lipgloss.NewStyle().Foreground(tui.DimText).Render(" (by ") +
		lipgloss.NewStyle().Foreground(tui.White).Render(bs.UserName) +
		lipgloss.NewStyle().Foreground(tui.DimText).Render(")")
}
