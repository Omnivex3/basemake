package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/basemake/internal/config"
)

func setupAPIKey(reader *bufio.Reader, envVar, provider string) {
	fmt.Printf("  Paste your %s key: ", strings.ReplaceAll(envVar, "_", " "))
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)

	if key == "" {
		fmt.Println("  ⚠ No key entered. You can set it later with:")
		fmt.Printf("  export %s=<your-key>\n", envVar)
		fmt.Println("  Continuing without AI...")
		return
	}

	fmt.Printf("  ✓ Key saved for %s\n", strings.ToUpper(provider))
	saveProvider(provider)
	setAPIKey(envVar, key)
}

func saveProvider(provider string) {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	cfg.AIProvider = provider
	cfg.Save()
}

func setAPIKey(envVar, key string) {
	// Save to .basemake/env file so it persists across sessions
	home, _ := os.UserHomeDir()
	envPath := home + "/.basemake/env"
	os.MkdirAll(home+"/.basemake", 0755)

	// Read existing env file
	existing := map[string]string{}
	if data, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if parts := strings.SplitN(strings.TrimSpace(line), "=", 2); len(parts) == 2 {
				existing[parts[0]] = parts[1]
			}
		}
	}
	existing[envVar] = key

	// Write back
	var lines []string
	for k, v := range existing {
		lines = append(lines, k+"="+v)
	}
	os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0600)
	os.Setenv(envVar, key)

	fmt.Printf("  ✓ Saved to ~/.basemake/env\n")
}

func setupOpenCode(reader *bufio.Reader) {
	fmt.Println("  Using your existing OpenCode API key (OpenAI-compatible).")
	fmt.Println()

	fmt.Print("  Paste your API key: ")
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)

	if key == "" {
		fmt.Println("  ⚠ No key entered. You can set it later with:")
		fmt.Println("  export OPENAI_API_KEY=<your-key>")
		return
	}

	fmt.Println()
	fmt.Print("  Base URL [https://api.openai.com/v1]: ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	fmt.Println()
	fmt.Print("  Model [deepseek-chat]: ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = "deepseek-chat"
	}

	saveProvider("openai")
	setAPIKey("OPENAI_API_KEY", key)
	setEnvVar("OPENAI_BASE_URL", baseURL)
	setEnvVar("OPENAI_MODEL", model)
	fmt.Printf("  ✓ OpenCode ready! Model: %s | %s\n", model, baseURL)
}

func setEnvVar(key, value string) {
	home, _ := os.UserHomeDir()
	envPath := home + "/.basemake/env"
	os.MkdirAll(home+"/.basemake", 0755)

	existing := map[string]string{}
	if data, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if parts := strings.SplitN(strings.TrimSpace(line), "=", 2); len(parts) == 2 {
				existing[parts[0]] = parts[1]
			}
		}
	}
	existing[key] = value

	var lines []string
	for k, v := range existing {
		lines = append(lines, k+"="+v)
	}
	os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0600)
	os.Setenv(key, value)
}

func loadAPIKeysFromEnv() {
	home, _ := os.UserHomeDir()
	envPath := home + "/.basemake/env"
	if data, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
			if len(parts) == 2 && (strings.HasPrefix(parts[0], "OPENAI") || strings.HasPrefix(parts[0], "ANTHROPIC") || parts[0] == "AI_PROVIDER") {
				if os.Getenv(parts[0]) == "" {
					os.Setenv(parts[0], parts[1])
				}
			}
		}
	}
}
