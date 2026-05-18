package db

import (
	"fmt"
	"os"
	"path/filepath"
)

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".basemake", "config")
}

// SaveDSN persists the active DSN for subsequent commands
func SaveDSN(dsn string) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return os.WriteFile(configPath(), []byte(dsn), 0644)
}

// LoadDSN reads the saved DSN
func LoadDSN() (string, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return "", fmt.Errorf("no saved connection — run 'basemake connect' first")
	}
	return string(data), nil
}
