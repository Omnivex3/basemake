package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system,omitempty"`
	Messages  []anthropicMessage  `json:"messages"`
	Stream    bool                `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type anthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
}

func (p *anthropicProvider) Name() string { return "Anthropic" }

func (p *anthropicProvider) GenerateSQL(ctx context.Context, system, question string) (string, error) {
	body := anthropicRequest{
		Model:     p.model,
		MaxTokens: 1024,
		System:    system,
		Messages:  []anthropicMessage{{Role: "user", Content: question}},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
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

	var result anthropicResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("anthropic error: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	// Concatenate all text content blocks
	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return cleanSQL(text.String()), nil
}

func (p *anthropicProvider) GenerateSQLStream(ctx context.Context, system, question string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		body := anthropicRequest{
			Model:     p.model,
			MaxTokens: 1024,
			System:    system,
			Messages:  []anthropicMessage{{Role: "user", Content: question}},
			Stream:    true,
		}

		payload, err := json.Marshal(body)
		if err != nil {
			ch <- fmt.Sprintf("Error: %v", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(payload))
		if err != nil {
			ch <- fmt.Sprintf("Error: %v", err)
			return
		}
		req.Header.Set("x-api-key", p.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ch <- fmt.Sprintf("Error: %v", err)
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		var fullText strings.Builder

		for scanner.Scan() {
			line := scanner.Text()

			// Anthropic SSE: "event: content_block_delta" followed by "data: {...}"
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				var event struct {
					Type  string `json:"type"`
					Delta *struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"delta,omitempty"`
				}
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}

				if event.Type == "content_block_delta" && event.Delta != nil {
					if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
						fullText.WriteString(event.Delta.Text)
						ch <- event.Delta.Text
					}
				}

				if event.Type == "error" {
					ch <- fmt.Sprintf("\nError: %s", data)
					return
				}
			}
		}
	}()

	return ch, nil
}
