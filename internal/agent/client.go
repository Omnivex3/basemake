package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ── Anthropic Messages API types ──

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system,omitempty"`
	Messages  []anthropicMessage  `json:"messages"`
	Tools     []toolDefinition    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string or []contentBlock
}

type contentBlock struct {
	Type   string          `json:"type"`
	Text   string          `json:"text,omitempty"`
	ID     string          `json:"id,omitempty"`
	Name   string          `json:"name,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
	// tool_result fields
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

type toolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"input_schema"`
}

type inputSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]propertySchema `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

type propertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Items       *itemRef `json:"items,omitempty"`
}

type itemRef struct {
	Type string `json:"type"`
}

type anthropicResponse struct {
	Content   []contentBlock `json:"content"`
	StopReason string        `json:"stop_reason"`
	Usage     *usageInfo     `json:"usage,omitempty"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type usageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ── Client ──

type anthropicClient struct {
	apiKey  string
	baseURL string
	model   string
}

func newAnthropicClient(apiKey, baseURL, model string) *anthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &anthropicClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

// send sends a messages request with optional tools and returns the parsed response.
func (c *anthropicClient) send(ctx context.Context, system string, messages []anthropicMessage, tools []toolDefinition) (*anthropicResponse, error) {
	body := anthropicRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result anthropicResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", result.Error.Message)
	}

	return &result, nil
}

// ── Content block helpers ──

// newTextMessage creates a user message with a simple text string.
func newTextMessage(text string) anthropicMessage {
	data, _ := json.Marshal(text)
	return anthropicMessage{Role: "user", Content: data}
}

// newToolResultMessage creates a user message with a tool_result content block.
func newToolResultMessage(toolUseID, content string) anthropicMessage {
	contentRaw := json.RawMessage(`"` + jsonEscape(content) + `"`)
	blocks := []contentBlock{
		{
			Type:      "tool_result",
			ToolUseID: toolUseID,
			Content:   contentRaw,
		},
	}
	data, _ := json.Marshal(blocks)
	return anthropicMessage{Role: "user", Content: data}
}

// jsonEscape escapes a string for use in a JSON string value.
func jsonEscape(s string) string {
	data, _ := json.Marshal(s)
	// Remove surrounding quotes
	if len(data) >= 2 {
		return string(data[1 : len(data)-1])
	}
	return s
}

// extractText concatenates all text content blocks from the response.
func extractText(resp *anthropicResponse) string {
	var result string
	for _, block := range resp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}
	return result
}
