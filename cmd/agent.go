package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/DynamicKarabo/basemake/internal/agent"
	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent \"your question\"",
	Short: "Ask a deep question about your database performance",
	Long: `Uses the AI agent to answer complex questions by running tools:
schema inspection, profile analysis, EXPLAIN, and observations.

Examples:
  basemake agent "why is my dashboard slow?"
  basemake agent "what changed since last deploy?"
  basemake agent "is this query normal: SELECT * FROM orders"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := args[0]

		a, err := agent.New()
		if err != nil {
			return fmt.Errorf("agent: %w", err)
		}

		fmt.Fprintf(os.Stderr, "🧠 Analyzing: %s\n\n", question)

		answer, pricing, err := a.Run(context.Background(), question)
		if err != nil {
			return fmt.Errorf("agent failed: %w", err)
		}

		fmt.Println(answer)

		if pricing != nil {
			fmt.Fprintf(os.Stderr, "\n━━━\nIterations: %d\n", a.Iterations())
			// Rough estimate: each tool call + response is ~500 tokens
			estTokens := a.Iterations() * 1000
			fmt.Fprintf(os.Stderr, "Estimated cost: %s\n", ai.EstimateCost("claude-sonnet-4-20250514", estTokens, estTokens/2))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
}
