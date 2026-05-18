package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "dbai") {
		t.Errorf("expected 'dbai' in output, got: %s", output)
	}
	if !strings.Contains(output, "Go version") {
		t.Errorf("expected 'Go version' in output, got: %s", output)
	}
}

func TestCompletionCommand(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)

			rootCmd.SetArgs([]string{"completion", shell})

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("completion %s failed: %v", shell, err)
			}

			output := stdout.String()
			if len(output) < 10 {
				t.Errorf("completion %s produced too little output (%d bytes)", shell, len(output))
			}
		})
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	rootCmd.SetArgs([]string{"completion", "invalid"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid shell, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected 'unsupported shell' error, got: %v", err)
	}
}
