package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/display"
	"github.com/DynamicKarabo/basemake/internal/history"
	"github.com/spf13/cobra"
)

var replFormat string

func init() {
	replCmd.Flags().StringVar(&replFormat, "format", "", "Output format (table, json, csv)")
	rootCmd.AddCommand(replCmd)
}

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Interactive SQL shell",
	Long: `Chat with your database using natural language.

Special commands:
  .help       Show available commands
  .quit       Exit
  .tables     List tables in the current database
  .schema     Show the full schema
  .connect <dsn>  Connect to a different database

Everything else is a question or SQL query.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Config load: %v\n", err)
			cfg = config.DefaultConfig()
		}

		format := display.FormatTable
		switch replFormat {
		case "json":
			format = display.FormatJSON
		case "csv":
			format = display.FormatCSV
		default:
			if cfg.OutputFormat == "json" {
				format = display.FormatJSON
			} else if cfg.OutputFormat == "csv" {
				format = display.FormatCSV
			}
		}

		// Try active connection or default DSN
		conn, err := db.ActiveConnection()
		if err != nil {
			dsn := cfg.DefaultDSN
			if dsn == "" {
				dsn, _ = db.LoadDSN()
			}
			if dsn != "" {
				conn, err = db.Connect(dsn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  ⚠ Connect failed: %v\n", err)
				}
			}
		}

		if !launchedFromInteractive {
			// Print warm welcome
			fmt.Println("  ╭──────────────────────────────────────────────╮")
			fmt.Println("  │  🤖  Welcome to basemake                     │")
			fmt.Println("  │  Chat with your database in plain English.   │")
			fmt.Println("  │                                              │")
			if conn != nil {
				fmt.Printf("  │  🟢 Connected: %-28s│\n", conn.Name())
			} else {
				fmt.Println("  │  🔴 No database connected                  │")
				fmt.Println("  │  Use  .connect <dsn>  to get started       │")
			}
			fmt.Println("  │  Type  .help  for commands                   │")
			fmt.Println("  ╰──────────────────────────────────────────────╯")
			fmt.Println()
		}

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("  You > ")

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			switch {
			case line == "":
				// Skip empty
			case line == ".quit" || line == ".exit":
				fmt.Println("  🤖 Bye! Come back anytime.")
				return nil
			case line == ".help":
				printChatHelp()
			case line == ".tables":
				showTables(conn)
			case line == ".schema":
				showFullSchema(conn)
			case strings.HasPrefix(line, ".connect "):
				dsn := strings.TrimPrefix(line, ".connect ")
				conn, err = db.Connect(dsn)
				if err != nil {
					fmt.Printf("  🤖 ⚠ Connect failed: %v\n", err)
				} else {
					fmt.Printf("  🤖 ✅ Connected: %s\n", conn.Name())
				}
			case line == ".history":
				showHistory()
			default:
				if err := executeChatQuery(conn, line, format); err != nil {
					fmt.Printf("  🤖 ⚠ %v\n", err)
				}
			}

			fmt.Print("\n  You > ")
		}

		return scanner.Err()
	},
}

func printChatHelp() {
	fmt.Println(`  🤖 Commands:
    .help              Show this
    .quit              Exit
    .tables            List tables
    .schema            Show full schema
    .connect <dsn>     Connect to a DB
    .history           Past questions

  Everything else is a question or SQL.`)
}

func showTables(conn db.Database) {
	if conn == nil {
		fmt.Print("  🤖 No database connected. Use  .connect <dsn>")
		return
	}
	schema, err := conn.Introspect(context.Background())
	if err != nil {
		fmt.Printf("  🤖 ⚠ %v", err)
		return
	}
	fmt.Printf("  🤖 Found %d tables in %s:\n", len(schema.Tables), schema.DBName)
	for _, t := range schema.Tables {
		fmt.Printf("    📦 %s (%d columns)\n", t.Name, len(t.Columns))
	}
}

func showFullSchema(conn db.Database) {
	if conn == nil {
		fmt.Print("  🤖 No database connected. Use  .connect <dsn>")
		return
	}
	schema, err := conn.Introspect(context.Background())
	if err != nil {
		fmt.Printf("  🤖 ⚠ %v", err)
		return
	}
	for _, t := range schema.Tables {
		fmt.Printf("  📦 %s (%d cols, %d indexes):\n", t.Name, len(t.Columns), len(t.Indexes))
		for _, c := range t.Columns {
			pk := ""
			if c.IsPK {
				pk = " 🔑"
			}
			nullable := ""
			if c.IsNullable {
				nullable = " nullable"
			}
			fmt.Printf("    ├─ %s %s%s%s\n", c.Name, c.Type, pk, nullable)
		}
	}
}

func executeChatQuery(conn db.Database, input string, format display.Format) error {
	if conn == nil {
		return fmt.Errorf("no database connected — use  .connect <dsn>")
	}

	sql := input
	isNL := !looksLikeSQL(sql)

	if isNL {
		s, err := db.LoadSchema()
		if err != nil {
			return fmt.Errorf("load schema: %w", err)
		}

		prompt := history.BuildPromptWithHistory(s.SchemaForPrompt(), 5)

		fmt.Print("  🤖 Thinking...\n\n  ")
		ch, err := ai.QuestionToSQLStream(context.Background(), prompt, input)
		if err != nil {
			return fmt.Errorf("ai: %w", err)
		}

		var sb strings.Builder
		for token := range ch {
			sb.WriteString(token)
			fmt.Print(token)
		}
		sql = sb.String()
		fmt.Println()
	}

	// Execute
	startTime := time.Now()
	rows, err := conn.Query(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	elapsed := time.Since(startTime).Seconds() * 1000

	cols := rows.Columns()
	var resultRows [][]string
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for rows.Next() {
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			switch val := v.(type) {
			case []byte:
				row[i] = string(val)
			case nil:
				row[i] = "NULL"
			default:
				row[i] = fmt.Sprint(val)
			}
		}
		resultRows = append(resultRows, row)
	}

	// Record history
	provider, _ := ai.SelectedProvider()
	providerName := ""
	if provider != nil {
		providerName = provider.Name()
	}
	_ = history.Record(history.Entry{
		Question:           input,
		SQLGenerated:       sql,
		DatabaseName:       conn.Name(),
		ExecutedAt:         startTime,
		ExecutionTimeMs:    elapsed,
		RowCount:           len(resultRows),
		WasNaturalLanguage: isNL,
		ProviderUsed:       providerName,
	})

	// Print results
	fmt.Println()
	res := display.Result{
		Columns: cols,
		Rows:    resultRows,
	}
	if err := display.Print(os.Stdout, res, format); err != nil {
		return fmt.Errorf("print: %w", err)
	}

	plural := "rows"
	if len(resultRows) == 1 {
		plural = "row"
	}
	fmt.Printf("\n  🤖 %d %s in %.0fms\n", len(resultRows), plural, elapsed)

	return nil
}

func showHistory() {
	entries, err := history.List(20)
	if err != nil {
		fmt.Printf("  🤖 ⚠ %v\n", err)
		return
	}
	if len(entries) == 0 {
		fmt.Print("  🤖 No questions yet. Ask me something!")
		return
	}
	fmt.Printf("  🤖 Last %d questions:\n", len(entries))
	for _, e := range entries {
		icon := "💬"
		if !e.WasNaturalLanguage {
			icon = "🔤"
		}
		timeStr := e.ExecutedAt.Format("15:04:05")
		question := e.Question
		if len(question) > 60 {
			question = question[:57] + "..."
		}
		fmt.Printf("    %s [%s] %s\n", icon, timeStr, question)
	}
}
