package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/DynamicKarabo/basemake/internal/ai"
)

// Agent runs the tool-calling loop using an LLM provider.
type Agent struct {
	provider   provider
	tools      []toolSpec
	maxIter    int
	iterations int
}

// New creates a new Agent, reading provider config from env vars.
// Supports "anthropic" (default) and "openai" providers.
func New() (*Agent, error) {
	providerName := os.Getenv("AI_PROVIDER")
	if providerName == "" {
		providerName = "anthropic" // default
	}

	var prov provider
	var err error

	switch providerName {
	case "openai":
		prov, err = newOpenAIProvider()
	case "anthropic":
		prov, err = newAnthropicProvider()
	default:
		return nil, fmt.Errorf("unsupported agent provider: %s (use 'anthropic' or 'openai')", providerName)
	}
	if err != nil {
		return nil, err
	}

	return &Agent{
		provider: prov,
		tools:    agentTools(),
		maxIter:  5,
	}, nil
}

// Run executes the agent loop for a single user question.
// It returns the final text response.
func (a *Agent) Run(ctx context.Context, question string) (string, *ai.ModelPricing, error) {
	messages := []providerMessage{
		{Role: "user", Text: question},
	}

	// Convert tool specs to provider-agnostic tool definitions
	var toolDefs []toolDef
	for _, t := range a.tools {
		params := map[string]any{
			"type": "object",
		}
		if len(t.InputSchema.Properties) > 0 {
			props := make(map[string]any)
			for k, v := range t.InputSchema.Properties {
				p := map[string]any{
					"type": v.Type,
				}
				if v.Description != "" {
					p["description"] = v.Description
				}
				if v.Items != nil {
					p["items"] = map[string]string{"type": v.Items.Type}
				}
				props[k] = p
			}
			params["properties"] = props
		}
		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}

		toolDefs = append(toolDefs, toolDef{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		})
	}

	a.iterations = 0

	for a.iterations < a.maxIter {
		a.iterations++

		resp, err := a.provider.send(ctx, systemPrompt, messages, toolDefs)
		if err != nil {
			return "", nil, fmt.Errorf("iteration %d: %w", a.iterations, err)
		}

		// Build assistant message from response
		assistantMsg := providerMessage{Role: "assistant"}
		if resp.Text != "" {
			assistantMsg.Text = resp.Text
		}
		if len(resp.ToolCalls) > 0 {
			assistantMsg.ToolCalls = resp.ToolCalls
		}
		messages = append(messages, assistantMsg)

		switch resp.StopReason {
		case "stop", "end_turn":
			pricing := estimatePricing(resp)
			return resp.Text, pricing, nil

		case "tool_calls", "tool_use":
			// Execute each tool call
			for _, tc := range resp.ToolCalls {
				// Find the tool spec
				var spec *toolSpec
				for i := range a.tools {
					if a.tools[i].Name == tc.Name {
						spec = &a.tools[i]
						break
					}
				}
				if spec == nil {
					return "", nil, fmt.Errorf("unknown tool called: %s", tc.Name)
				}

				// Execute
				result, err := spec.Execute(ctx, tc.Input)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				// Append tool result
				messages = append(messages, providerMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Text:       result,
				})
			}

		case "max_tokens":
			text := resp.Text
			if text == "" {
				text = "Reached the maximum token limit. Try asking a more specific question."
			}
			pricing := estimatePricing(resp)
			return text + "\n\n[Reached token limit — answer may be incomplete]", pricing, nil

		default:
			return "", nil, fmt.Errorf("unexpected stop_reason: %s", resp.StopReason)
		}
	}

	return "I couldn't complete the analysis in the allowed number of steps. Try asking a more specific question.", nil, nil
}

// Iterations returns how many iterations the last Run() used.
func (a *Agent) Iterations() int {
	return a.iterations
}

// estimatePricing returns default pricing for the agent model.
func estimatePricing(resp *providerResponse) *ai.ModelPricing {
	if resp.Usage == nil {
		return nil
	}
	return &ai.ModelPricing{
		InputCents:  0.3,
		OutputCents: 1.5,
	}
}

// ── Provider constructors ──

func newAnthropicProvider() (*anthropicProvider, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set. Set it to use the agent path")
	}
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &anthropicProvider{apiKey: apiKey, baseURL: baseURL, model: model}, nil
}

func newOpenAIProvider() (*openAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set. Set it to use the agent path")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}
	return &openAIProvider{apiKey: apiKey, baseURL: baseURL, model: model}, nil
}
