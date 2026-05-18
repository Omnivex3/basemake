package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/display"
	"github.com/DynamicKarabo/basemake/internal/tui"
	"github.com/spf13/cobra"
)

var replFormat string

func init() {
	replCmd.Flags().StringVar(&replFormat, "format", "", "Output format (table, json, csv)")
	rootCmd.AddCommand(replCmd)
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Interactive SQL shell",
	Long: `Chat with your database using natural language.

Special commands:
  .help       Show available commands
  .quit       Exit
  .tables     List tables in the current database
  .schema     Show the full schema
  .connect <dsn>  Connect to a different database

Everything else is a question or SQL query.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Config load: %v\n", err)
			cfg = config.DefaultConfig()
		}

		format := display.FormatTable
		switch replFormat {
		case "json":
			format = display.FormatJSON
		case "csv":
			format = display.FormatCSV
		default:
			if cfg.OutputFormat == "json" {
				format = display.FormatJSON
			} else if cfg.OutputFormat == "csv" {
				format = display.FormatCSV
			}
		}

		// Try active connection or default DSN
		conn, err := db.ActiveConnection()
		if err != nil {
			dsn := cfg.DefaultDSN
			if dsn == "" {
				dsn, _ = db.LoadDSN()
			}
			if dsn != "" {
				conn, err = db.Connect(dsn)
				if err != nil {
					conn = nil
				}
			}
		}

		// Get version info
		info := getBuildInfo()

		// Launch bubbletea TUI
		model := tui.NewModel(conn, format, info.version)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		return nil
	},
}
