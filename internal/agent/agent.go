package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/DynamicKarabo/basemake/internal/ai"
)

// Agent runs the tool-calling loop using the Anthropic Messages API.
// It holds the client, tool specs, and iteration cap.
type Agent struct {
	client     *anthropicClient
	tools      []toolSpec
	maxIter    int
	iterations int
}

// New creates a new Agent, reading API config from env and config file.
func New() (*Agent, error) {
	// Read config the same way as the existing ai package
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	model := os.Getenv("ANTHROPIC_MODEL")

	// Fall back to config defaults from the ai package constants
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set. Set it to use the agent path.\n  export ANTHROPIC_API_KEY=sk-ant-...")
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	return &Agent{
		client:  newAnthropicClient(apiKey, baseURL, model),
		tools:   agentTools(),
		maxIter: 5,
	}, nil
}

// Run executes the agent loop for a single user question.
// It returns the final text response and an estimate of tokens used.
func (a *Agent) Run(ctx context.Context, question string) (string, *ai.ModelPricing, error) {
	messages := []anthropicMessage{
		newTextMessage(question),
	}

	// Convert tool specs to API format
	var toolDefs []toolDefinition
	for _, t := range a.tools {
		toolDefs = append(toolDefs, toDefinition(t))
	}

	a.iterations = 0

	for a.iterations < a.maxIter {
		a.iterations++

		resp, err := a.client.send(ctx, systemPrompt, messages, toolDefs)
		if err != nil {
			return "", nil, fmt.Errorf("iteration %d: %w", a.iterations, err)
		}

		// Append assistant's response to message history
		assistantContent, _ := json.Marshal(resp.Content)
		messages = append(messages, anthropicMessage{
			Role:    "assistant",
			Content: assistantContent,
		})

		switch resp.StopReason {
		case "end_turn":
			// Done — extract and return text
			text := extractText(resp)
			pricing := estimatePricing(resp)
			return text, pricing, nil

		case "tool_use":
			// Execute each tool_use block
			for _, block := range resp.Content {
				if block.Type != "tool_use" {
					continue
				}

				// Find the tool spec
				var spec *toolSpec
				for i := range a.tools {
					if a.tools[i].Name == block.Name {
						spec = &a.tools[i]
						break
					}
				}
				if spec == nil {
					return "", nil, fmt.Errorf("unknown tool called: %s", block.Name)
				}

				// Parse input
				var input map[string]any
				if block.Input != nil {
					if err := json.Unmarshal(block.Input, &input); err != nil {
						return "", nil, fmt.Errorf("tool %s: parse input: %w", block.Name, err)
					}
				} else {
					input = map[string]any{}
				}

				// Execute
				result, err := spec.Execute(ctx, input)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				// Append tool result
				messages = append(messages, newToolResultMessage(block.ID, result))
			}

		case "max_tokens":
			// Hit token limit — return what we have
			text := extractText(resp)
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
func estimatePricing(resp *anthropicResponse) *ai.ModelPricing {
	if resp.Usage == nil {
		return nil
	}
	return &ai.ModelPricing{
		InputCents:  0.3,  // claude-sonnet-4-20250514
		OutputCents: 1.5,
	}
}
