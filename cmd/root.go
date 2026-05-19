package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/tui"
	"github.com/spf13/cobra"
)

// BannerASCII is the launch art shown when basemake runs without arguments.
const BannerASCII = `█████                                                          █████              
▒▒███                                                          ▒▒███               
 ▒███████   ██████    █████   ██████  █████████████    ██████   ▒███ █████  ██████ 
 ▒███▒▒███ ▒▒▒▒▒███  ███▒▒   ███▒▒███▒▒███▒▒███▒▒███  ▒▒▒▒▒███  ▒███▒▒███  ███▒▒███
 ▒███ ▒███  ███████ ▒▒█████ ▒███████  ▒███ ▒███ ▒███   ███████  ▒██████▒  ▒███████ 
 ▒███ ▒███ ███▒▒███  ▒▒▒▒███▒███▒▒▒   ▒███ ▒███ ▒███  ███▒▒███  ▒███▒▒███ ▒███▒▒▒  
 ████████ ▒▒████████ ██████ ▒▒██████  █████▒███ █████▒▒████████ ████ █████▒▒██████ 
▒▒▒▒▒▒▒▒   ▒▒▒▒▒▒▒▒ ▒▒▒▒▒▒   ▒▒▒▒▒▒  ▒▒▒▒▒ ▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒▒▒ ▒▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒`

var cfgFile string

// sharedReadOnly is the --readonly flag value for rootCmd, consumed by both the REPL and query commands.
var sharedReadOnly bool

var rootCmd = &cobra.Command{
	Use:   "basemake",
	Short: "AI-powered database CLI — query, analyze, optimize",
	Long: `basemake connects to your database, learns your schema,
and lets you ask questions in plain English.

  basemake connect postgres://user:***@localhost:5432/mydb
  basemake "show me users who ordered last month"
  basemake analyze "SELECT * FROM orders WHERE ..."`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return queryCmd.RunE(cmd, args)
		}
		return enterInteractiveMode()
	},
}

func Execute() {
	// Load persisted API keys before anything else
	loadAPIKeysFromEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.basemake/config.yaml)")
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(analyzeCmd)
}

func init() {
	originalHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			fmt.Println(tui.ColoriseLogo(BannerASCII))
			fmt.Println(strings.Repeat("─", 50))
		}
		originalHelp(cmd, args)
	})
}

func enterInteractiveMode() error {
	// Check if this looks like a first run — no config or no connections
	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg.DefaultDSN == "" && len(cfg.Connections) == 0 {
		// First run detection
		fmt.Println()
		fmt.Println("  ╭──────────────────────────────────────────────╮")
		fmt.Println("  │  Welcome to basemake 🚀                      │")
		fmt.Println("  │                                              │")
		fmt.Println("  │  Query your database in plain English.       │")
		fmt.Println("  │                                              │")
		fmt.Println("  │  Get started:  basemake init                 │")
		fmt.Println("  │  Quick demo:   basemake init --demo          │")
		fmt.Println("  │  Just connect: basemake connect --detect     │")
		fmt.Println("  ╰──────────────────────────────────────────────╯")
		fmt.Println()
		return nil
	}

	// Straight to the charm TUI — no banners, no onboarding, no noise.
	// DSN loading and connection is handled inside replCmd.
	return replCmd.RunE(replCmd, []string{})
}

