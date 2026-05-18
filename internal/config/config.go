package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent configuration for basemake
type Config struct {
	DefaultDSN      string `json:"default_dsn,omitempty"`
	OutputFormat    string `json:"output_format,omitempty"`
	AIProvider      string `json:"ai_provider,omitempty"`
	OpenAIModel     string `json:"openai_model,omitempty"`
	OpenAIBaseURL   string `json:"openai_base_url,omitempty"`
	AnthropicModel  string `json:"anthropic_model,omitempty"`
	AnthropicBaseURL string `json:"anthropic_base_url,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		OutputFormat:   "table",
		AIProvider:     "openai",
		OpenAIModel:    "gpt-4",
		AnthropicModel: "claude-sonnet-4-20250514",
	}
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// Load reads the config from disk, returning defaults if not found
func Load() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// Save persists the config to disk
func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath(), data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
