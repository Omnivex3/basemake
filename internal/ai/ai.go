package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/DynamicKarabo/basemake/internal/config"
)

// Provider generates SQL from natural language questions.
type Provider interface {
	// Name returns a human-readable provider name (e.g., "OpenAI", "Anthropic").
	Name() string

	// GenerateSQL sends a prompt to the LLM and returns the generated SQL.
	GenerateSQL(ctx context.Context, systemPrompt, question string) (string, error)

	// GenerateSQLStream sends a prompt and returns a channel of text fragments.
	// The channel is closed when generation is complete.
	// Each fragment is a partial token that should be printed as it arrives.
	GenerateSQLStream(ctx context.Context, systemPrompt, question string) (<-chan string, error)
}

// ErrNoKey is returned when the required API key is not set.
var ErrNoKey = fmt.Errorf("API key not set")

// SelectedProvider returns the configured AI provider based on env vars and config.
func SelectedProvider() (Provider, error) {
	cfg, _ := config.Load()

	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = cfg.AIProvider
	}
	if provider == "" {
		provider = "openai"
	}

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = cfg.OpenAIModel
		}
		if model == "" {
			model = "gpt-4"
		}
		baseURL := os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = cfg.OpenAIBaseURL
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}

		if apiKey == "" {
			return nil, ErrNoKey
		}
		return &openAIProvider{apiKey: apiKey, model: model, baseURL: baseURL}, nil

	case "opencode":
		apiKey := os.Getenv("OPENCODE_API_KEY")
		model := os.Getenv("OPENCODE_MODEL")
		if model == "" {
			model = cfg.OpenCodeModel
		}
		if model == "" {
			model = "deepseek-chat"
		}
		baseURL := os.Getenv("OPENCODE_BASE_URL")
		if baseURL == "" {
			baseURL = cfg.OpenCodeBaseURL
		}
		if baseURL == "" {
			baseURL = "https://api.opencode.ai/v1"
		}

		if apiKey == "" {
			return nil, ErrNoKey
		}
		return &openCodeProvider{&openAIProvider{apiKey: apiKey, model: model, baseURL: baseURL}}, nil

	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		model := os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = cfg.AnthropicModel
		}
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		baseURL := os.Getenv("ANTHROPIC_BASE_URL")
		if baseURL == "" {
			baseURL = cfg.AnthropicBaseURL
		}
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}

		if apiKey == "" {
			return nil, ErrNoKey
		}
		return &anthropicProvider{apiKey: apiKey, model: model, baseURL: baseURL}, nil

	case "ollama":
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = cfg.OllamaModel
		}
		if model == "" {
			model = "llama3"
		}
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = cfg.OllamaBaseURL
		}
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		return &ollamaProvider{model: model, baseURL: baseURL}, nil

	default:
		return nil, fmt.Errorf("unsupported AI provider: %s (use 'openai', 'anthropic', or 'ollama')", provider)
	}
}

// QuestionToSQL is the main entry point for NL→SQL generation.
// It selects the configured provider and calls GenerateSQL.
// If no API key is set, it returns a placeholder SQL with instructions.
func QuestionToSQL(ctx context.Context, systemPrompt, question string) (string, error) {
	provider, err := SelectedProvider()
	if err == ErrNoKey {
		return "-- Set your API key for AI-powered queries\n" +
			"--   OpenAI:     export OPENAI_API_KEY=sk-...\n" +
			"--   Anthropic:  export ANTHROPIC_API_KEY=sk-ant-...\n" +
			"--   OpenCode:   export OPENCODE_API_KEY=sk-...\n" +
			"-- Schema loaded. Export the key for your provider and run again.\nSELECT 1;\n", nil
	}
	if err != nil {
		return "", fmt.Errorf("provider: %w", err)
	}

	// systemPrompt is already a full system prompt from BuildPromptWithHistory
	// — pass it directly, no re-wrapping
	return provider.GenerateSQL(ctx, systemPrompt, question)
}

// QuestionToSQLStream is the streaming version of QuestionToSQL.
// Returns a channel of text fragments for real-time display.
func QuestionToSQLStream(ctx context.Context, systemPrompt, question string) (<-chan string, error) {
	provider, err := SelectedProvider()
	if err == ErrNoKey {
		ch := make(chan string, 1)
		ch <- "-- Set your API key for AI-powered queries\n" +
			"--   OpenAI:     export OPENAI_API_KEY=sk-...\n" +
			"--   Anthropic:  export ANTHROPIC_API_KEY=sk-ant-...\n" +
			"--   OpenCode:   export OPENCODE_API_KEY=sk-...\n" +
			"-- Schema loaded. Export the key for your provider and run again.\nSELECT 1;"
		close(ch)
		return ch, nil
	}
	if err != nil {
		return nil, fmt.Errorf("provider: %w", err)
	}

	return provider.GenerateSQLStream(ctx, systemPrompt, question)
}

// PingProvider tests the AI provider connection by sending a minimal request.
// Returns nil if the provider responds, or an error describing the failure.
// Useful for startup health checks and diagnostics.
func PingProvider() error {
	provider, err := SelectedProvider()
	if err != nil {
		return err
	}
	// Send a minimal prompt — asking for something trivial
	_, err = provider.GenerateSQL(context.Background(),
		"You are a SQL generator. Return only SQL, no explanation.",
		"SELECT 1")
	if err != nil {
		return fmt.Errorf("%s: %w", provider.Name(), err)
	}
	return nil
}

// ─── Cost Tracking ──────────────────────────────────────────────────────────

// ModelPricing holds per-token cost estimates for a model.
type ModelPricing struct {
	InputCents  float64 // cost per 1K input tokens (cents)
	OutputCents float64 // cost per 1K output tokens (cents)
}

// Known model pricing (approximate, updated 2026-05)
var modelPricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4":       {InputCents: 3.0, OutputCents: 6.0},
	"gpt-4-turbo": {InputCents: 1.0, OutputCents: 3.0},
	"gpt-4o":      {InputCents: 0.5, OutputCents: 1.5},
	"gpt-4o-mini": {InputCents: 0.015, OutputCents: 0.06},
	// Anthropic
	"claude-sonnet-4-20250514": {InputCents: 0.3, OutputCents: 1.5},
	"claude-3-haiku":           {InputCents: 0.025, OutputCents: 0.125},
	"claude-3-opus":            {InputCents: 1.5, OutputCents: 7.5},
	// OpenCode defaults (approximate, varies by backend model)
}

// EstimateCost returns a human-readable cost estimate for a given model + token counts.
// Returns "?" when pricing is unknown for the model.
func EstimateCost(model string, inputTokens, outputTokens int) string {
	pricing, ok := modelPricing[model]
	if !ok {
		return "?"
	}
	inputCost := float64(inputTokens) / 1000.0 * pricing.InputCents
	outputCost := float64(outputTokens) / 1000.0 * pricing.OutputCents
	total := inputCost + outputCost
	if total < 0.1 {
		return "<$0.001"
	}
	return fmt.Sprintf("~$%.3f", total/100.0)
}

// EstimateTokens returns a rough token count for a string.
// Uses ~4 chars per token as a rough estimate (standard for English text).
func EstimateTokens(text string) int {
	return len(text) / 4
}

// ProviderInfo returns a human-readable summary of the configured provider.
func ProviderInfo() string {
	cfg, _ := config.Load()
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = cfg.AIProvider
	}
	if provider == "" {
		provider = "openai"
	}

	model := ""
	switch provider {
	case "openai":
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = cfg.OpenAIModel
		}
		if model == "" {
			model = "gpt-4"
		}
	case "anthropic":
		model = os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = cfg.AnthropicModel
		}
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
	case "ollama":
		model = os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = cfg.OllamaModel
		}
		if model == "" {
			model = "llama3"
		}
	case "opencode":
		model = os.Getenv("OPENCODE_MODEL")
		if model == "" {
			model = cfg.OpenCodeModel
		}
		if model == "" {
			model = "deepseek-chat"
		}
	}

	label := provider
	if model != "" {
		label += "/" + model
	}
	return label
}
