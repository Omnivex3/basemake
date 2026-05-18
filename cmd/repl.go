package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/DynamicKarabo/dbai/internal/ai"
	"github.com/DynamicKarabo/dbai/internal/config"
	"github.com/DynamicKarabo/dbai/internal/db"
	"github.com/DynamicKarabo/dbai/internal/display"
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
	Long: `Start an interactive database shell with AI-powered query assistance.

Special commands:
  .help       Show this help
  .quit       Exit
  .tables     List tables in the current database
  .schema     Show the full schema
  .connect <dsn>  Connect to a different database

All other input is treated as SQL or natural language questions.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Config load: %v\n", err)
			cfg = config.DefaultConfig()
		}

		// Determine output format
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

		// Try active connection, or use default DSN
		conn, err := db.ActiveConnection()
		if err != nil {
			dsn := cfg.DefaultDSN
			if dsn == "" {
				dsn, _ = db.LoadDSN()
			}
			if dsn != "" {
				conn, err = db.Connect(dsn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠ Connect failed: %v\n", err)
				}
			}
		}

		fmt.Fprintf(os.Stderr, "dbai REPL — type .help for commands, .quit to exit\n")
		if conn != nil {
			fmt.Fprintf(os.Stderr, "Connected: %s\n", conn.Name())
		} else {
			fmt.Fprintf(os.Stderr, "No connection. Use .connect <dsn> to connect.\n")
		}
		fmt.Fprintln(os.Stderr)

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("dbai> ")

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			switch {
			case line == "":
				// Empty line — skip
			case line == ".quit" || line == ".exit":
				fmt.Fprintln(os.Stderr, "Bye!")
				return nil
			case line == ".help":
				printHelp()
			case line == ".tables":
				if err := showTables(conn); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case line == ".schema":
				if err := showFullSchema(conn); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			case strings.HasPrefix(line, ".connect "):
				dsn := strings.TrimPrefix(line, ".connect ")
				conn, err = db.Connect(dsn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠ Connect failed: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "✓ Connected: %s\n", conn.Name())
				}
			default:
				// Execute as query
				if err := executeREPLQuery(conn, line, format); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}

			fmt.Print("dbai> ")
		}

		return scanner.Err()
	},
}

func printHelp() {
	fmt.Fprintln(os.Stderr, `Commands:
  .help       Show this help
  .quit       Exit the REPL
  .tables     List tables in the current database
  .schema     Show the full schema with columns and indexes
  .connect <dsn>  Connect to a different database

Any other input is treated as a SQL query or natural language question.
Use --format flag to set output format (table, json, csv).`)
}

func showTables(conn db.Database) error {
	if conn == nil {
		return fmt.Errorf("no active connection — use .connect <dsn>")
	}
	schema, err := conn.Introspect(nil)
	if err != nil {
		return fmt.Errorf("introspect: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Tables in %s (%d total):\n", schema.DBName, len(schema.Tables))
	for _, t := range schema.Tables {
		fmt.Fprintf(os.Stderr, "  %s (%d columns)\n", t.Name, len(t.Columns))
	}
	return nil
}

func showFullSchema(conn db.Database) error {
	if conn == nil {
		return fmt.Errorf("no active connection — use .connect <dsn>")
	}
	schema, err := conn.Introspect(nil)
	if err != nil {
		return fmt.Errorf("introspect: %w", err)
	}
	for _, t := range schema.Tables {
		fmt.Fprintf(os.Stderr, "%s (%d columns, %d indexes):\n", t.Name, len(t.Columns), len(t.Indexes))
		for _, c := range t.Columns {
			pk := ""
			if c.IsPK {
				pk = " [PK]"
			}
			nullable := ""
			if c.IsNullable {
				nullable = " nullable"
			}
			fmt.Fprintf(os.Stderr, "  %s %s%s%s\n", c.Name, c.Type, pk, nullable)
		}
	}
	return nil
}

func executeREPLQuery(conn db.Database, input string, format display.Format) error {
	if conn == nil {
		return fmt.Errorf("no active connection — use .connect <dsn>")
	}

	sql := input
	isNL := !looksLikeSQL(sql)

	if isNL {
		s, err := db.LoadSchema()
		if err != nil {
			return fmt.Errorf("load schema: %w", err)
		}
		fmt.Fprintf(os.Stderr, "🤖 Generating SQL...\n")
		sql, err = ai.QuestionToSQL(nil, s.SchemaForPrompt(), input)
		if err != nil {
			return fmt.Errorf("ai: %w", err)
		}
		fmt.Fprintf(os.Stderr, "%s\n\n", sql)
	}

	rows, err := conn.Query(nil, sql)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

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

	plural := "rows"
	if len(resultRows) == 1 {
		plural = "row"
	}
	msg := fmt.Sprintf("(%d %s)", len(resultRows), plural)

	res := display.Result{
		Columns: cols,
		Rows:    resultRows,
		Message: msg,
	}

	if err := display.Print(os.Stdout, res, format); err != nil {
		return fmt.Errorf("print: %w", err)
	}

	if format != display.FormatTable {
		fmt.Fprintf(os.Stderr, "\n%s\n", msg)
	}

	return nil
}
