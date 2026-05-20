package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ── OpenAI API types ──

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMsg     `json:"messages"`
	Tools       []openAIToolDef `json:"tools,omitempty"`
	ToolChoice  string          `json:"tool_choice,omitempty"`
}

type openAIMsg struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolDef struct {
	Type     string        `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON string
	} `json:"function"`
}

type openAIResp struct {
	Choices []struct {
		FinishReason string    `json:"finish_reason"`
		Message      openAIMsg `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ── OpenAI provider ──

type openAIProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func (p *openAIProvider) modelName() string { return p.model }

func (p *openAIProvider) send(ctx context.Context, system string, messages []providerMessage, tools []toolDef) (*providerResponse, error) {
	// Build OpenAI messages — system prompt goes first
	oaMessages := []openAIMsg{
		{Role: "system", Content: system},
	}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			oaMessages = append(oaMessages, openAIMsg{Role: "user", Content: msg.Text})
		case "assistant":
			oaMsg := openAIMsg{Role: "assistant", Content: msg.Text}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					argData, _ := json.Marshal(tc.Input)
					oaMsg.ToolCalls = append(oaMsg.ToolCalls, openAIToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      tc.Name,
							Arguments: string(argData),
						},
					})
				}
				oaMsg.Content = "" // requires empty content when tool_calls present
			}
			oaMessages = append(oaMessages, oaMsg)
		case "tool":
			oaMessages = append(oaMessages, openAIMsg{
				Role:       "tool",
				ToolCallID: msg.ToolCallID,
				Content:    msg.Text,
			})
		}
	}

	// Build tool definitions
	var oaTools []openAIToolDef
	for _, t := range tools {
		oaTools = append(oaTools, openAIToolDef{
			Type: "function",
			Function: openAIFunction(t),
		})
	}

	body := openAIRequest{
		Model:      p.model,
		Messages:   oaMessages,
		Tools:      oaTools,
		ToolChoice: "auto",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
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

	var result openAIResp
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := result.Choices[0]
	pr := &providerResponse{}

	// Map finish reason
	switch choice.FinishReason {
	case "stop":
		pr.StopReason = "stop"
	case "tool_calls":
		pr.StopReason = "tool_calls"
	case "length":
		pr.StopReason = "max_tokens"
	default:
		pr.StopReason = choice.FinishReason
	}

	// Text content
	pr.Text = choice.Message.Content

	// Tool calls
	for _, tc := range choice.Message.ToolCalls {
		if tc.Type != "function" {
			continue
		}
		var input map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			input = map[string]any{"_raw": tc.Function.Arguments}
		}
		pr.ToolCalls = append(pr.ToolCalls, toolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	// Usage
	if result.Usage != nil {
		pr.Usage = &providerUsage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
		}
	}

	return pr, nil
}
