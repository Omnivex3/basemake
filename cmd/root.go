package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
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

// launchedFromInteractive is set to true when the REPL is entered via `basemake` (no args).
// Suppresses the redundant header that repl.go prints on its own.
var launchedFromInteractive bool

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
			fmt.Println(BannerASCII)
			fmt.Println(strings.Repeat("─", 50))
		}
		originalHelp(cmd, args)
	})
}
func enterInteractiveMode() error {
	launchedFromInteractive = true

	fmt.Println(BannerASCII)
	fmt.Println(strings.Repeat("─", 50))

	cfg, _ := config.Load()
	hasAPIKey := hasConfiguredAPIKey(cfg)
	dsn, dsnErr := db.LoadDSN()

	var conn db.Database
	if dsnErr == nil && dsn != "" {
		var err error
		conn, err = db.Connect(dsn)
		if err != nil {
			conn = nil
		}
	}

	// Compact status
	dbStatus := "🔴 Not connected"
	if conn != nil {
		name := conn.Name()
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		dbStatus = "🟢 " + name
	}

	aiStatus := "🔴 No API key"
	if hasAPIKey {
		aiStatus = "🟢 " + strings.ToUpper(cfg.AIProvider)
	} else if cfg != nil && cfg.AIProvider == "ollama" {
		aiStatus = "🟡 Ollama (local)"
		hasAPIKey = true
	}

	fmt.Printf("  %s  ·  %s\n", dbStatus, aiStatus)
	fmt.Println()

	// First time — run onboarding
	if !hasAPIKey && conn == nil {
		fmt.Println("  👋 Welcome to basemake! Let's get you set up.")
		fmt.Println()
		runOnboarding()
		fmt.Println()
	}

	fmt.Println("  ╭──────────────────────────────────────────────╮")
	if conn != nil {
		name := conn.Name()
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		fmt.Printf("  │  🤖  basemake  ·  🟢 %-28s│\n", name)
	} else {
		fmt.Println("  │  🤖  basemake  ·  🔴 No DB                 │")
	}
	fmt.Println("  │  Ask me anything about your data.           │")
	fmt.Println("  ╰──────────────────────────────────────────────╯")
	fmt.Println()

	return replCmd.RunE(replCmd, []string{})
}

func hasConfiguredAPIKey(cfg *config.Config) bool {
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = cfg.AIProvider
	}

	switch provider {
	case "openai":
		return os.Getenv("OPENAI_API_KEY") != ""
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY") != ""
	case "ollama":
		return true
	default:
		return os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != ""
	}
}
