package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/DynamicKarabo/dbai/internal/config"
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

	default:
		return nil, fmt.Errorf("unsupported AI provider: %s (use 'openai' or 'anthropic')", provider)
	}
}

// QuestionToSQL is the main entry point for NL→SQL generation.
// It selects the configured provider and calls GenerateSQL.
// If no API key is set, it returns a placeholder SQL with instructions.
func QuestionToSQL(ctx context.Context, schemaPrompt, question string) (string, error) {
	provider, err := SelectedProvider()
	if err == ErrNoKey {
		return "-- Set OPENAI_API_KEY or ANTHROPIC_API_KEY for AI-powered queries\n" +
			"-- Schema loaded. Export the key for your provider and run again.\nSELECT 1;", nil
	}
	if err != nil {
		return "", fmt.Errorf("provider: %w", err)
	}

	systemPrompt := fmt.Sprintf(`You are a SQL expert. Given the following database schema, convert the user's natural language question into a SQL query.

Rules:
- Generate PostgreSQL-compatible SQL
- Return ONLY the SQL query — no markdown, no backticks, no explanations
- Use proper formatting with newlines
- If the question is ambiguous, make a reasonable assumption and add a comment explaining it

Schema:
%s`, schemaPrompt)

	return provider.GenerateSQL(ctx, systemPrompt, question)
}

// QuestionToSQLStream is the streaming version of QuestionToSQL.
// Returns a channel of text fragments for real-time display.
func QuestionToSQLStream(ctx context.Context, schemaPrompt, question string) (<-chan string, error) {
	provider, err := SelectedProvider()
	if err == ErrNoKey {
		ch := make(chan string, 1)
		ch <- "-- Set OPENAI_API_KEY or ANTHROPIC_API_KEY for AI-powered queries\n-- Schema loaded. Export the key for your provider and run again.\nSELECT 1;"
		close(ch)
		return ch, nil
	}
	if err != nil {
		return nil, fmt.Errorf("provider: %w", err)
	}

	systemPrompt := fmt.Sprintf(`You are a SQL expert. Given the following database schema, convert the user's natural language question into a SQL query.

Rules:
- Generate PostgreSQL-compatible SQL
- Return ONLY the SQL query — no markdown, no backticks, no explanations
- Use proper formatting with newlines
- If the question is ambiguous, make a reasonable assumption and add a comment explaining it

Schema:
%s`, schemaPrompt)

	return provider.GenerateSQLStream(ctx, systemPrompt, question)
}
