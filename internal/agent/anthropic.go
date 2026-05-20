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
	Model     string                  `json:"model"`
	MaxTokens int                     `json:"max_tokens"`
	System    string                  `json:"system,omitempty"`
	Messages  []anthropicReqMessage   `json:"messages"`
	Tools     []anthropicToolDef      `json:"tools,omitempty"`
}

type anthropicReqMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string or []contentBlock
}

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

type anthropicToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"input_schema"`
}

type anthropicResp struct {
	Content   []anthropicContentBlock `json:"content"`
	StopReason string                `json:"stop_reason"`
	Usage     *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ── Anthropic provider ──

type anthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func (p *anthropicProvider) modelName() string { return p.model }

func (p *anthropicProvider) send(ctx context.Context, system string, messages []providerMessage, tools []toolDef) (*providerResponse, error) {
	// Build messages in Anthropic format
	reqMsgs := make([]anthropicReqMessage, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			reqMsgs = append(reqMsgs, anthropicReqMessage{
				Role:    "user",
				Content: mustJSON(msg.Text),
			})
		case "assistant":
			var blocks []anthropicContentBlock
			if msg.Text != "" {
				blocks = append(blocks, anthropicContentBlock{Type: "text", Text: msg.Text})
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: mustJSONRaw(mapToJSON(tc.Input)),
				})
			}
			if len(blocks) == 0 {
				reqMsgs = append(reqMsgs, anthropicReqMessage{Role: "assistant", Content: mustJSON("")})
			} else {
				reqMsgs = append(reqMsgs, anthropicReqMessage{Role: "assistant", Content: mustJSON(blocks)})
			}
		case "tool":
			reqMsgs = append(reqMsgs, anthropicReqMessage{
				Role: "user",
				Content: mustJSON([]anthropicContentBlock{{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   json.RawMessage(`"` + jsonEscape(msg.Text) + `"`),
				}}),
			})
		}
	}

	// Build tool definitions from generic toolDef.Parameters
	var anthropicTools []anthropicToolDef
	for _, t := range tools {
		params := make(map[string]any)
		if t.Parameters != nil {
			params = t.Parameters
		}
		anthropicTools = append(anthropicTools, anthropicToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: inputSchema{
				Type:       "object",
				Properties: make(map[string]propertySchema),
				Required:   nil,
			},
		})
		// Copy properties if present
		if props, ok := params["properties"].(map[string]any); ok {
			for k, v := range props {
				if propMap, ok := v.(map[string]any); ok {
					prop := propertySchema{
						Type: fmt.Sprintf("%v", propMap["type"]),
					}
					if desc, ok := propMap["description"].(string); ok {
						prop.Description = desc
					}
					if items, ok := propMap["items"].(map[string]any); ok {
						if itemType, ok := items["type"].(string); ok {
							prop.Items = &itemRef{Type: itemType}
						}
					}
					anthropicTools[len(anthropicTools)-1].InputSchema.Properties[k] = prop
				}
			}
		}
		// Copy required if present
		if req, ok := params["required"].([]any); ok {
			for _, r := range req {
				if s, ok := r.(string); ok {
					anthropicTools[len(anthropicTools)-1].InputSchema.Required = append(
						anthropicTools[len(anthropicTools)-1].InputSchema.Required, s)
				}
			}
		}
	}

	body := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		System:    system,
		Messages:  reqMsgs,
		Tools:     anthropicTools,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
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

	var result anthropicResp
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", result.Error.Message)
	}

	// Convert to common response shape
	pr := &providerResponse{
		StopReason: result.StopReason,
	}
	if result.Usage != nil {
		pr.Usage = &providerUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		}
	}

	// Collect text and tool calls from content blocks
	var text string
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			text += block.Text
		case "tool_use":
			var input map[string]any
			if block.Input != nil {
				json.Unmarshal(block.Input, &input)
			}
			pr.ToolCalls = append(pr.ToolCalls, toolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: input,
			})
		}
	}
	pr.Text = text

	return pr, nil
}

// ── Helpers ──

func mustJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

func mustJSONRaw(v any) json.RawMessage {
	switch t := v.(type) {
	case json.RawMessage:
		return t
	default:
		data, _ := json.Marshal(v)
		return data
	}
}

func jsonEscape(s string) string {
	data, _ := json.Marshal(s)
	if len(data) >= 2 {
		return string(data[1 : len(data)-1])
	}
	return s
}

func mapToJSON(m map[string]any) json.RawMessage {
	data, _ := json.Marshal(m)
	return data
}
