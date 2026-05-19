package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose your basemake setup",
	Long: `Check database connectivity, AI provider, schema cache,
configuration, and shell integration.

  basemake doctor          # Full diagnostic
  basemake doctor --quick  # Lightweight check (DB + AI only)`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		pass := "✅"
		fail := "✗"
		warn := "⚠"
		info := "ℹ"

		fmt.Println()
		fmt.Println("  basemake doctor — diagnostics")
		fmt.Println()

		// 1. Configuration
		fmt.Println("  Configuration:")
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("  %s Could not load config: %v\n", fail, err)
		} else {
			fmt.Printf("  %s Config loaded from ~/.basemake/config.json\n", pass)
			activeConns := len(cfg.Connections)
			if activeConns > 0 {
				fmt.Printf("  %s %d saved connection(s) (active: %s)\n", pass, activeConns, cfg.ActiveConnection)
			} else {
				fmt.Printf("  %s No saved connections\n", info)
			}
		}
		fmt.Println()

		// 2. Database Connection
		fmt.Println("  Database:")
		conn, err := db.ActiveConnection()
		if err == nil {
			fmt.Printf("  %s Connected: %s\n", pass, conn.Name())
			schema, schemaErr := db.LoadSchema()
			if schemaErr == nil {
				fmt.Printf("  %s Schema cached: %d tables\n", pass, len(schema.Tables))
			} else {
				fmt.Printf("  %s Schema not cached: %v\n", warn, schemaErr)
				fmt.Printf("    → Run: basemake connect <dsn>\n")
			}
		} else {
			dsn, dsnErr := db.LoadDSN()
			if dsnErr == nil {
				// Try to connect
				testConn, testErr := net.DialTimeout("tcp", extractHostPort(dsn), 3*time.Second)
				if testErr == nil {
					testConn.Close()
					fmt.Printf("  %s DSN found but not connected\n", warn)
					fmt.Printf("    → Run: basemake \"show me tables\" (auto-connects)\n")
				} else {
					fmt.Printf("  %s DSN found but host unreachable: %v\n", fail, testErr)
					fmt.Printf("    → Check if the database is running\n")
				}
			} else {
				fmt.Printf("  %s No database configured\n", info)
				fmt.Printf("    → Run: basemake init or basemake connect --detect\n")
			}

			if cfg != nil && len(cfg.Connections) > 0 {
				fmt.Printf("  %s Saved connections available: %s\n", info, strings.Join(cfg.ConnectionNames(), ", "))
				fmt.Printf("    → Switch: basemake use <name>\n")
			}
		}
		fmt.Println()

		// 3. AI Provider
		fmt.Println("  AI Provider:")
		provider, err := ai.SelectedProvider()
		if err == ai.ErrNoKey {
			fmt.Printf("  %s No API key configured\n", info)
			fmt.Printf("    → Run: basemake init (guided setup)\n")
			fmt.Printf("    → Or: basemake config set ai_provider ollama (local, no key needed)\n")
		} else if err != nil {
			fmt.Printf("  %s Provider error: %v\n", fail, err)
		} else {
			fmt.Printf("  %s Provider: %s\n", pass, provider.Name())

			// Quick reachability check
			cfg2, _ := config.Load()
			baseURL := getBaseURL(cfg2)
			if baseURL != "" {
				host := extractHostFromURL(baseURL)
				testConn, testErr := net.DialTimeout("tcp", host+":443", 3*time.Second)
				if testErr == nil {
					testConn.Close()
					fmt.Printf("  %s API reachable at %s\n", pass, baseURL)
				} else {
					fmt.Printf("  %s API unreachable: %v\n", warn, testErr)
					fmt.Printf("    → Check your internet connection\n")
				}
			}
		}
		fmt.Println()

		// 4. Shell Integration
		fmt.Println("  Shell:")
		shell := os.Getenv("SHELL")
		if strings.Contains(shell, "zsh") || strings.Contains(shell, "bash") {
			completionFile := ""
			if strings.Contains(shell, "zsh") {
				completionFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
			} else {
				completionFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
			}

			if data, err := os.ReadFile(completionFile); err == nil {
				if strings.Contains(string(data), "basemake completion") {
					fmt.Printf("  %s Tab completion installed (%s)\n", pass, filepath.Base(shell))
				} else {
					fmt.Printf("  %s Tab completion not installed\n", info)
					fmt.Printf("    → Add: basemake completion %s >> %s\n", filepath.Base(shell), completionFile)
				}
			} else {
				fmt.Printf("  %s Shell: %s\n", info, filepath.Base(shell))
			}
		} else {
			fmt.Printf("  %s Shell: %s\n", info, shell)
		}
		fmt.Println()

		// 5. Storage
		fmt.Println("  Storage:")
		home, _ := os.UserHomeDir()
		basemakeDir := filepath.Join(home, ".basemake")
		if _, err := os.Stat(basemakeDir); err == nil {
			size := formatSize(dirSize(basemakeDir))
			fmt.Printf("  %s ~/.basemake/ (%s)\n", pass, size)

			// Check key files
			envPath := filepath.Join(basemakeDir, "env")
			if _, err := os.Stat(envPath); err == nil {
				fmt.Printf("  %s API keys stored in env file\n", pass)
			}
		} else {
			fmt.Printf("  %s ~/.basemake/ not found\n", info)
			fmt.Printf("    → Run: basemake init\n")
		}

		fmt.Println()
		fmt.Println("  Done. Need help? Run: basemake init")
		fmt.Println()

		return nil
	},
}

// ── Helpers ──

func getBaseURL(cfg *config.Config) string {
	switch cfg.AIProvider {
	case "openai":
		if cfg.OpenAIBaseURL != "" {
			return cfg.OpenAIBaseURL
		}
		return "https://api.openai.com/v1"
	case "anthropic":
		if cfg.AnthropicBaseURL != "" {
			return cfg.AnthropicBaseURL
		}
		return "https://api.anthropic.com"
	case "ollama":
		if cfg.OllamaBaseURL != "" {
			return cfg.OllamaBaseURL
		}
		return "http://localhost:11434/v1"
	default:
		return ""
	}
}

func extractHostPort(dsn string) string {
	// postgres://user:pass@host:port/db → host:port
	parts := strings.SplitN(dsn, "@", 2)
	if len(parts) < 2 {
		return ""
	}
	hostPort := strings.SplitN(parts[1], "/", 2)[0]
	return hostPort
}

func extractHostFromURL(rawURL string) string {
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if idx := strings.Index(rawURL, "/"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, ":"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	return rawURL
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !fi.IsDir() {
			size += fi.Size()
		}
		return nil
	})
	return size
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
