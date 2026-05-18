package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.OutputFormat != "table" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "table")
	}
	if cfg.OpenAIModel != "gpt-4" {
		t.Errorf("OpenAIModel = %q, want %q", cfg.OpenAIModel, "gpt-4")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use temp home
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := &Config{
		DefaultDSN:   "postgres://test@localhost/db",
		OutputFormat: "json",
		OpenAIModel:  "gpt-4o",
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	cfgFile := filepath.Join(tmp, ".dbai", "config.json")
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", cfgFile)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.DefaultDSN != "postgres://test@localhost/db" {
		t.Errorf("DefaultDSN = %q, want %q", loaded.DefaultDSN, "postgres://test@localhost/db")
	}
	if loaded.OutputFormat != "json" {
		t.Errorf("OutputFormat = %q, want %q", loaded.OutputFormat, "json")
	}
	if loaded.OpenAIModel != "gpt-4o" {
		t.Errorf("OpenAIModel = %q, want %q", loaded.OpenAIModel, "gpt-4o")
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OutputFormat != "table" {
		t.Errorf("expected defaults, got OutputFormat = %q", cfg.OutputFormat)
	}
}

func TestSaveOverwrites(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Save first config
	cfg1 := &Config{DefaultDSN: "first"}
	if err := cfg1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Save different config
	cfg2 := &Config{DefaultDSN: "second"}
	if err := cfg2.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.DefaultDSN != "second" {
		t.Errorf("got %q, want %q", loaded.DefaultDSN, "second")
	}
}
