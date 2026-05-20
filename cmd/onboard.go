package cmd

import (
	"os"
	"strings"
)

func loadAPIKeysFromEnv() {
	home, _ := os.UserHomeDir()
	envPath := home + "/.basemake/env"
	if data, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
			if len(parts) == 2 && (strings.HasPrefix(parts[0], "OPENAI") || strings.HasPrefix(parts[0], "OPENCODE") || strings.HasPrefix(parts[0], "ANTHROPIC") || parts[0] == "AI_PROVIDER") {
				if os.Getenv(parts[0]) == "" {
					_ = os.Setenv(parts[0], parts[1])
				}
			}
		}
	}
}
