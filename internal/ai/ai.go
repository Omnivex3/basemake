package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// QuestionToSQL converts a natural language question to SQL using OpenAI
func QuestionToSQL(ctx context.Context, schemaPrompt, question string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Return a helpful placeholder when no API key is set
		return fmt.Sprintf("-- Set OPENAI_API_KEY for AI-powered queries\n-- Schema loaded. Run: export OPENAI_API_KEY=\"sk-...\"\nSELECT 1;"), nil
	}

	systemPrompt := fmt.Sprintf(`You are a SQL expert. Given the following database schema, convert the user's natural language question into a SQL query.

Rules:
- Generate PostgreSQL-compatible SQL
- Return ONLY the SQL query — no markdown, no backticks, no explanations
- Use proper formatting with newlines
- If the question is ambiguous, make a reasonable assumption and add a comment explaining it

Schema:
%s`, schemaPrompt)

	resp, err := callOpenAI(ctx, apiKey, systemPrompt, question)
	if err != nil {
		return "", fmt.Errorf("ai call: %w", err)
	}

	// Clean up response
	sql := strings.TrimSpace(resp)
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	sql = strings.TrimSpace(sql)

	return sql, nil
}

type openAIRequest struct {
	Model    string            `json:"model"`
	Messages []openAIMessage   `json:"messages"`
	Temp     float64           `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func callOpenAI(ctx context.Context, apiKey, system, user string) (string, error) {
	body := openAIRequest{
		Model: "gpt-4",
		Messages: []openAIMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temp: 0.1,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result openAIResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("openai error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}
