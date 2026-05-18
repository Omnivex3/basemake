package cmd

import (
	"fmt"
	"os"
	"strings"

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
	rootCmd.AddCommand(connectCmd)
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
	// Straight to the charm TUI — no banners, no onboarding, no noise.
	// DSN loading and connection is handled inside replCmd.
	return replCmd.RunE(replCmd, []string{})
}

