package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Config holds persistent configuration for basemake
type Config struct {
	DefaultDSN       string            `json:"default_dsn,omitempty"`
	Connections      map[string]string `json:"connections,omitempty"`
	ActiveConnection string            `json:"active_connection,omitempty"`
	OutputFormat     string            `json:"output_format,omitempty"`
	AIProvider       string            `json:"ai_provider,omitempty"`
	OpenAIModel      string            `json:"openai_model,omitempty"`
	OpenAIBaseURL    string            `json:"openai_base_url,omitempty"`
	AnthropicModel   string            `json:"anthropic_model,omitempty"`
	AnthropicBaseURL string            `json:"anthropic_base_url,omitempty"`
	OllamaModel      string            `json:"ollama_model,omitempty"`
	OllamaBaseURL    string            `json:"ollama_base_url,omitempty"`
	OpenCodeModel    string            `json:"opencode_model,omitempty"`
	OpenCodeBaseURL  string            `json:"opencode_base_url,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		OutputFormat:   "table",
		AIProvider:     "openai",
		OpenAIModel:    "gpt-4",
		AnthropicModel: "claude-sonnet-4-20250514",
		OllamaModel:    "llama3",
		OllamaBaseURL:  "http://localhost:11434/v1",
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

	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// ── Named Connections ──

// ConnectionNames returns sorted connection names.
func (c *Config) ConnectionNames() []string {
	var names []string
	for name := range c.Connections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetConnection returns the DSN for a named connection.
func (c *Config) GetConnection(name string) (string, bool) {
	if c.Connections == nil {
		return "", false
	}
	dsn, ok := c.Connections[name]
	return dsn, ok
}

// SetConnection saves a named connection.
func (c *Config) SetConnection(name, dsn string) {
	if c.Connections == nil {
		c.Connections = make(map[string]string)
	}
	c.Connections[name] = dsn
}

// RemoveConnection deletes a named connection.
func (c *Config) RemoveConnection(name string) {
	if c.Connections != nil {
		delete(c.Connections, name)
	}
	if c.ActiveConnection == name {
		c.ActiveConnection = ""
	}
}

// ActiveDSN returns the DSN for the active connection, or default_dsn.
func (c *Config) ActiveDSN() string {
	if c.ActiveConnection != "" && c.Connections != nil {
		if dsn, ok := c.Connections[c.ActiveConnection]; ok {
			return dsn
		}
	}
	return c.DefaultDSN
}
