package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage persistent configuration",
	Long: `View and modify basemake configuration stored in ~/.basemake/config.json.

  basemake config show                  # View all config
  basemake config get <key>             # Get a single value
  basemake config set <key> <value>     # Set a single value`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Config file: ~/.basemake/config.json\n\n")
		printField(cfg.DefaultDSN, "default_dsn")
		printField(cfg.OutputFormat, "output_format")
		printField(cfg.AIProvider, "ai_provider")
		printField(cfg.OpenAIModel, "openai_model")
		printField(cfg.OpenAIBaseURL, "openai_base_url")
		printField(cfg.AnthropicModel, "anthropic_model")
		printField(cfg.AnthropicBaseURL, "anthropic_base_url")
		printField(cfg.OllamaModel, "ollama_model")
		printField(cfg.OllamaBaseURL, "ollama_base_url")
		printField(cfg.OpenCodeModel, "opencode_model")
		printField(cfg.OpenCodeBaseURL, "opencode_base_url")
		fmt.Fprintf(os.Stderr, "\n  %-20s %s\n", "active_connection", cfg.ActiveConnection)
		if len(cfg.Connections) > 0 {
			fmt.Fprintf(os.Stderr, "  connections:\n")
			for name, dsn := range cfg.Connections {
				mark := " "
				if name == cfg.ActiveConnection {
					mark = "●"
				}
				masked := dsn
				if len(dsn) > 50 {
					masked = dsn[:47] + "..."
				}
				fmt.Fprintf(os.Stderr, "    %s %-20s %s\n", mark, name+":", masked)
			}
		}
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Long: `Get a single configuration value.

Valid keys: default_dsn, output_format, ai_provider, openai_model,
            openai_base_url, anthropic_model, anthropic_base_url,
            ollama_model, ollama_base_url, opencode_model, opencode_base_url`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		val := getField(cfg, args[0])
		if val == "" {
			return fmt.Errorf("unknown config key: %s", args[0])
		}
		cmd.Println(val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a single configuration value and save to ~/.basemake/config.json.

Valid keys: default_dsn, output_format, ai_provider, openai_model,
            openai_base_url, anthropic_model, anthropic_base_url,
            ollama_model, ollama_base_url, opencode_model, opencode_base_url

Examples:
  basemake config set ai_provider ollama
  basemake config set ollama_model llama3
  basemake config set output_format json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := setField(cfg, key, value); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "✓ Set %s = %s\n", key, value)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func printField(val, key string) {
	if val == "" {
		fmt.Printf("  %-20s (not set)\n", key)
	} else {
		fmt.Printf("  %-20s %s\n", key, val)
	}
}

func getField(cfg *config.Config, key string) string {
	switch key {
	case "default_dsn":
		return cfg.DefaultDSN
	case "output_format":
		return cfg.OutputFormat
	case "ai_provider":
		return cfg.AIProvider
	case "openai_model":
		return cfg.OpenAIModel
	case "openai_base_url":
		return cfg.OpenAIBaseURL
	case "anthropic_model":
		return cfg.AnthropicModel
	case "anthropic_base_url":
		return cfg.AnthropicBaseURL
	case "ollama_model":
		return cfg.OllamaModel
	case "ollama_base_url":
		return cfg.OllamaBaseURL
	case "opencode_model":
		return cfg.OpenCodeModel
	case "opencode_base_url":
		return cfg.OpenCodeBaseURL
	}
	return ""
}

func setField(cfg *config.Config, key, value string) error {
	switch key {
	case "default_dsn":
		cfg.DefaultDSN = value
	case "output_format":
		if value != "table" && value != "json" && value != "csv" {
			return fmt.Errorf("invalid output_format: %s (use table, json, or csv)", value)
		}
		cfg.OutputFormat = value
	case "ai_provider":
		if value != "openai" && value != "anthropic" && value != "ollama" && value != "opencode" {
			return fmt.Errorf("invalid ai_provider: %s (use openai, anthropic, ollama, or opencode)", value)
		}
		cfg.AIProvider = value
	case "openai_model":
		cfg.OpenAIModel = value
	case "openai_base_url":
		cfg.OpenAIBaseURL = value
	case "anthropic_model":
		cfg.AnthropicModel = value
	case "anthropic_base_url":
		cfg.AnthropicBaseURL = value
	case "ollama_model":
		cfg.OllamaModel = value
	case "ollama_base_url":
		cfg.OllamaBaseURL = value
	case "opencode_model":
		cfg.OpenCodeModel = value
	case "opencode_base_url":
		cfg.OpenCodeBaseURL = value
	default:
		valid := []string{"default_dsn", "output_format", "ai_provider",
			"openai_model", "openai_base_url",
			"anthropic_model", "anthropic_base_url",
			"ollama_model", "ollama_base_url",
			"opencode_model", "opencode_base_url"}
		return fmt.Errorf("unknown config key: %s\n\nValid keys:\n  %s", key, strings.Join(valid, "\n  "))
	}
	return nil
}
