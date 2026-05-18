package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "dbai",
	Short: "AI-powered database CLI — query, analyze, optimize",
	Long: `dbai connects to your database, learns your schema,
and lets you ask questions in plain English.

  dbai connect postgres://user:pass@localhost:5432/mydb
  dbai "show me users who ordered last month"
  dbai --explain "why is this query slow?"`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.dbai/config.yaml)")
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(analyzeCmd)
}
