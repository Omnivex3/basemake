package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DynamicKarabo/basemake/internal/config"
)

func legacyConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake", "config")
}

// SaveDSN persists the active DSN for subsequent commands.
// Writes to both the legacy DSN file and the structured JSON config.
func SaveDSN(dsn string) error {
	return SaveConnection("default", dsn)
}

// LoadDSN reads the saved DSN.
// Falls back through: legacy file → default_dsn → active connection → error.
func LoadDSN() (string, error) {
	// Try active connection name first
	cfg, cfgErr := config.Load()
	if cfgErr == nil {
		if dsn := cfg.ActiveDSN(); dsn != "" {
			return dsn, nil
		}
	}

	// Try legacy file
	data, err := os.ReadFile(legacyConfigPath())
	if err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("no saved connection — run 'basemake connect' or 'basemake init' first")
}

// ── Named Connections ──

// SaveConnection saves a named DSN to the JSON config.
func SaveConnection(name, dsn string) error {
	cfg, err := config.Load()
	if err != nil {
		// Try to create default
		cfg = config.DefaultConfig()
	}

	cfg.SetConnection(name, dsn)

	// If no active connection or this is the first one, set as active
	if cfg.ActiveConnection == "" {
		cfg.ActiveConnection = name
	}
	if name == "default" || cfg.DefaultDSN == "" {
		cfg.DefaultDSN = dsn
	}

	return cfg.Save()
}

// LoadNamedDSN returns the DSN for a named connection.
func LoadNamedDSN(name string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	dsn, ok := cfg.GetConnection(name)
	if !ok {
		return "", fmt.Errorf("connection %q not found — run 'basemake connect --list' to see saved connections", name)
	}
	return dsn, nil
}

// ListConnections returns all saved connection names with their DSNs (partially masked).
func ListConnections() (map[string]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if cfg.Connections == nil {
		return nil, nil
	}

	result := make(map[string]string, len(cfg.Connections))
	for name, dsn := range cfg.Connections {
		result[name] = maskConnDSN(dsn)
	}
	return result, nil
}

// ActiveConnectionName returns the currently active connection name.
func ActiveConnectionName() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	if cfg.ActiveConnection != "" {
		return cfg.ActiveConnection
	}
	return "default"
}

// SetActiveConnection sets the active connection by name.
func SetActiveConnection(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := cfg.GetConnection(name); !ok {
		return fmt.Errorf("connection %q not found — run 'basemake connect --save %s <dsn>' first", name, name)
	}

	cfg.ActiveConnection = name
	// Also update default_dsn for backward compatibility
	if dsn, ok := cfg.GetConnection(name); ok {
		cfg.DefaultDSN = dsn
	}
	return cfg.Save()
}

// maskConnDSN hides the password portion of a DSN for display.
func maskConnDSN(dsn string) string {
	// Try postgres://user:pass@host/db → postgres://user:***@host/db
	// Try mysql://user:pass@host/db → mysql://user:***@host/db
	for _, prefix := range []string{"postgres://", "postgresql://", "mysql://"} {
		if len(dsn) > len(prefix) {
			rest := dsn[len(prefix):]
			if atIdx := indexAt(rest); atIdx >= 0 {
				if colonIdx := stringIndex(rest, ":"); colonIdx >= 0 && colonIdx < atIdx {
					return dsn[:len(prefix)+colonIdx+1] + "***" + dsn[len(prefix)+atIdx:]
				}
			}
		}
	}
	// For SQLite or other schemes, just shorten
	if len(dsn) > 50 {
		return dsn[:47] + "..."
	}
	return dsn
}

func indexAt(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			return i
		}
	}
	return -1
}

func stringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
