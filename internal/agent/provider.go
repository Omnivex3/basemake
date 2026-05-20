package agent

import "context"

// ── Shared types between provider implementations ──

// inputSchema mirrors the JSON Schema for tool/function parameters.
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

// ── Provider interface ──

// provider implements the wire-format for a specific LLM provider.
type provider interface {
	modelName() string
	send(ctx context.Context, system string, messages []providerMessage, tools []toolDef) (*providerResponse, error)
}

// providerMessage is a generic message in the conversation history.
type providerMessage struct {
	Role       string     // "user", "assistant", "tool"
	Text       string     // plain text content (user messages)
	ToolCallID string     // for tool result messages
	ToolCalls  []toolCall // for assistant messages with tool calls
}

// toolCall represents a single tool invocation by the model.
type toolCall struct {
	ID    string
	Name  string
	Input map[string]any
}

// toolDef is a tool definition sent to the provider.
type toolDef struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema object
}

// providerResponse is the parsed response from any provider.
type providerResponse struct {
	Text       string         // final response text (empty if tool calls)
	ToolCalls  []toolCall     // tools the model wants to call
	StopReason string         // "stop", "tool_calls", "max_tokens", or "error"
	Usage      *providerUsage
}

type providerUsage struct {
	InputTokens  int
	OutputTokens int
}
