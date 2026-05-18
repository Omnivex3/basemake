package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DynamicKarabo/basemake/internal/config"
)

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake", "config")
}

// SaveDSN persists the active DSN for subsequent commands.
// Writes to both the legacy DSN file and the structured JSON config.
func SaveDSN(dsn string) error {
	// Write legacy DSN file
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configPath(), []byte(dsn), 0600); err != nil {
		return fmt.Errorf("write dsn file: %w", err)
	}

	// Also sync to JSON config so config show/default_dsn stays consistent
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		return nil // non-fatal — legacy file is enough for functionality
	}
	cfg.DefaultDSN = dsn
	_ = cfg.Save() // non-fatal if this fails

	return nil
}

// LoadDSN reads the saved DSN.
// Falls back to the JSON config's default_dsn if the legacy file is missing.
func LoadDSN() (string, error) {
	// Try legacy file first
	data, err := os.ReadFile(configPath())
	if err == nil {
		return string(data), nil
	}

	// Fall back to JSON config's default_dsn
	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg.DefaultDSN != "" {
		return cfg.DefaultDSN, nil
	}

	return "", fmt.Errorf("no saved connection — run 'basemake connect' or 'basemake config set default_dsn <dsn>' first")
}
