package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DynamicKarabo/basemake/internal/ai"
	"github.com/DynamicKarabo/basemake/internal/config"
	"github.com/DynamicKarabo/basemake/internal/db"
	"github.com/DynamicKarabo/basemake/internal/display"
	"github.com/DynamicKarabo/basemake/internal/history"
)

// ── Types ──

type replState int

const (
	stateIdle     replState = iota
	stateThinking           // AI generating SQL or query running
)

type msgKind int

const (
	msgUser msgKind = iota
	msgBot
	msgResult
	msgError
	msgCmd
)

type message struct {
	kind    msgKind
	content string
}

// ── Bubbletea Messages ──

type queryResultMsg struct {
	content string
	err     error
}

type connResultMsg struct {
	name string
	err  error
}

type introspectResultMsg struct {
	content string
	err     error
}

// ── Model ──

type Model struct {
	conn     db.Database
	format   display.Format
	state    replState
	messages []message
	input    textinput.Model
	spinner  spinner.Model
	width    int

	version string
	aiLabel string
}

// ── Constructor ──

func NewModel(conn db.Database, format display.Format, version string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type .help for commands  ·  ask your question or enter SQL"
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 72

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(Red)
	s.Spinner = spinner.Dot

	aiLabel := aiProviderLabel()

	return Model{
		conn:    conn,
		format:  format,
		state:   stateIdle,
		input:   ti,
		spinner: s,
		version: version,
		aiLabel: aiLabel,
		messages: []message{
			{kind: msgBot, content: fullStartupView(conn, aiLabel, version)},
		},
	}
}

// ── Init ──

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// ── Update ──

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.Width = max(30, msg.Width-8)

	case tea.KeyMsg:
		if m.state == stateIdle && msg.String() == "enter" {
			input := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if input == "" {
				break
			}
			if strings.HasPrefix(input, ".") {
				model, cmd := m.handleDotCommand(input)
				return model, cmd
			}
			m.messages = append(m.messages, message{kind: msgUser, content: input})
			m.state = stateThinking
			if looksLikeSQL(input) {
				return m, m.execQueryCmd(input, false)
			}
			return m, m.startNLCmd(input)
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case queryResultMsg:
		m.state = stateIdle
		if msg.err != nil {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble(msg.err.Error())})
		} else {
			m.messages = append(m.messages, message{kind: msgResult, content: msg.content})
		}

	case connResultMsg:
		if msg.err != nil {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble(msg.err.Error())})
		} else {
			conn, err := db.ActiveConnection()
			if err == nil {
				m.conn = conn
			}
			m.messages = append(m.messages, message{kind: msgCmd, content: successBubble(msg.name)})
		}

	case introspectResultMsg:
		if msg.err != nil {
			m.messages = append(m.messages, message{kind: msgError, content: errorBubble(msg.err.Error())})
		} else {
			m.messages = append(m.messages, message{kind: msgCmd, content: msg.content})
		}

	case spinner.TickMsg:
		if m.state != stateIdle {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Always feed non-control keys into the text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// ── Dot Commands ──

func (m Model) handleDotCommand(input string) (tea.Model, tea.Cmd) {
	switch {
	case input == ".quit" || input == ".exit":
		return m, tea.Quit

	case input == ".help":
		m.messages = append(m.messages, message{kind: msgCmd, content: helpBox()})
		return m, nil

	case input == ".tables":
		return m, m.introspectCmd("tables")

	case input == ".schema":
		return m, m.introspectCmd("schema")

	case strings.HasPrefix(input, ".connect "):
		dsn := strings.TrimPrefix(input, ".connect ")
		return m, m.connectCmd(dsn)

	case input == ".history":
		return m, m.historyCmd()

	default:
		m.messages = append(m.messages, message{kind: msgError, content: errorBubble("unknown command: " + input + "\n  Type .help for available commands")})
		return m, nil
	}
}

// ── Cmds (goroutine-backed) ──

func (m Model) connectCmd(dsn string) tea.Cmd {
	return func() tea.Msg {
		conn, err := db.Connect(dsn)
		if err != nil {
			return connResultMsg{err: err}
		}
		return connResultMsg{name: conn.Name()}
	}
}

func (m Model) introspectCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return introspectResultMsg{err: fmt.Errorf("no database connected — use .connect <dsn>")}
		}
		schema, err := m.conn.Introspect(context.Background())
		if err != nil {
			return introspectResultMsg{err: fmt.Errorf("introspect: %w", err)}
		}
		var b strings.Builder
		if mode == "tables" {
			b.WriteString(fmt.Sprintf("  📦 Found %d tables in %s:\n", len(schema.Tables), schema.DBName))
			for _, t := range schema.Tables {
				b.WriteString(fmt.Sprintf("    🗂  %s (%d columns)\n", t.Name, len(t.Columns)))
			}
		} else {
			for _, t := range schema.Tables {
				b.WriteString(fmt.Sprintf("  📦 %s (%d cols, %d indexes):\n", t.Name, len(t.Columns), len(t.Indexes)))
				for _, c := range t.Columns {
					pk := ""
					if c.IsPK {
						pk = " 🔑"
					}
					nullable := ""
					if c.IsNullable {
						nullable = " nullable"
					}
					b.WriteString(fmt.Sprintf("    ├─ %s %s%s%s\n", c.Name, c.Type, pk, nullable))
				}
			}
		}
		return introspectResultMsg{content: b.String()}
	}
}

func (m Model) execQueryCmd(sql string, isNL bool) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return queryResultMsg{err: fmt.Errorf("no database connected — use .connect <dsn>")}
		}

		startTime := time.Now()
		rows, err := m.conn.Query(context.Background(), sql)
		if err != nil {
			return queryResultMsg{err: fmt.Errorf("query failed: %w", err)}
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
				return queryResultMsg{err: fmt.Errorf("scan: %w", err)}
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

		_ = history.Record(history.Entry{
			SQLGenerated:       sql,
			DatabaseName:       m.conn.Name(),
			ExecutedAt:         startTime,
			ExecutionTimeMs:    elapsed,
			RowCount:           len(resultRows),
			WasNaturalLanguage: isNL,
		})

		var b strings.Builder
		if isNL {
			b.WriteString(sqlPreview(sql))
			b.WriteString("\n\n")
		}
		res := display.Result{Columns: cols, Rows: resultRows}
		if err := display.Print(&b, res, m.format); err != nil {
			return queryResultMsg{err: fmt.Errorf("print: %w", err)}
		}
		plural := "rows"
		if len(resultRows) == 1 {
			plural = "row"
		}
		b.WriteString(fmt.Sprintf("\n\n  %s %d %s in %.0fms", Dot(true, false), len(resultRows), plural, elapsed))

		return queryResultMsg{content: b.String()}
	}
}

func (m Model) startNLCmd(question string) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return queryResultMsg{err: fmt.Errorf("no database connected — use .connect <dsn>")}
		}

		schema, err := db.LoadSchema()
		if err != nil {
			return queryResultMsg{err: fmt.Errorf("no schema cache — run 'basemake connect' first: %w", err)}
		}

		provider, _ := ai.SelectedProvider()
		providerName := ""
		if provider != nil {
			providerName = provider.Name()
		}

		dialect := m.conn.Dialect()
		prompt := history.BuildPromptWithHistory(schema.SchemaForPrompt(), 5, dialect)

		ch, err := ai.QuestionToSQLStream(context.Background(), prompt, question)
		if err != nil {
			return queryResultMsg{err: fmt.Errorf("AI error: %w", err)}
		}

		var sql strings.Builder
		for token := range ch {
			sql.WriteString(token)
		}
		sqlStr := strings.TrimSpace(sql.String())

		startTime := time.Now()
		rows, err := m.conn.Query(context.Background(), sqlStr)
		if err != nil {
			return queryResultMsg{err: fmt.Errorf("query failed: %w", err)}
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
				return queryResultMsg{err: fmt.Errorf("scan: %w", err)}
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

		_ = history.Record(history.Entry{
			Question:           question,
			SQLGenerated:       sqlStr,
			DatabaseName:       m.conn.Name(),
			ExecutedAt:         startTime,
			ExecutionTimeMs:    elapsed,
			RowCount:           len(resultRows),
			WasNaturalLanguage: true,
			ProviderUsed:       providerName,
		})

		var b strings.Builder
		b.WriteString(sqlPreview(sqlStr))
		b.WriteString("\n\n")
		res := display.Result{Columns: cols, Rows: resultRows}
		if err := display.Print(&b, res, m.format); err != nil {
			return queryResultMsg{err: fmt.Errorf("print: %w", err)}
		}
		plural := "rows"
		if len(resultRows) == 1 {
			plural = "row"
		}
		b.WriteString(fmt.Sprintf("\n\n  %s %d %s in %.0fms", Dot(true, false), len(resultRows), plural, elapsed))

		return queryResultMsg{content: b.String()}
	}
}

func (m Model) historyCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := history.List(20)
		if err != nil {
			return introspectResultMsg{content: "  🤖 ⚠ " + err.Error()}
		}
		if len(entries) == 0 {
			return introspectResultMsg{content: "  🤖 No questions yet. Ask me something!"}
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("  🤖 Last %d questions:\n", len(entries)))
		for _, e := range entries {
			icon := "💬"
			if !e.WasNaturalLanguage {
				icon = "🔤"
			}
			timeStr := e.ExecutedAt.Format("15:04:05")
			q := e.Question
			if len(q) > 55 {
				q = q[:52] + "..."
			}
			b.WriteString(fmt.Sprintf("    %s [%s] %s\n", icon, timeStr, q))
		}
		return introspectResultMsg{content: b.String()}
	}
}

// ── View ──

func (m Model) View() string {
	var b strings.Builder

	for _, msg := range m.messages {
		switch msg.kind {
		case msgUser:
			b.WriteString(UserPromptStyle.Render("  You > "))
			b.WriteString(lipgloss.NewStyle().Foreground(Text).Render(msg.content))
			b.WriteString("\n")
		case msgBot:
			b.WriteString(msg.content)
			b.WriteString("\n")
		case msgResult:
			b.WriteString(SubBoxStyle.Render(msg.content))
			b.WriteString("\n")
		case msgCmd:
			b.WriteString(msg.content)
			b.WriteString("\n")
		case msgError:
			b.WriteString(msg.content)
			b.WriteString("\n")
		}
	}

	if m.state == stateThinking {
		b.WriteString("\n  " + m.spinner.View() + " " + ThinkingStyle.Render("Thinking...") + "\n")
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(DimText).Render(strings.Repeat("─", min(60, max(20, m.input.Width+4)))) + "\n")

	prompt := UserPromptStyle.Render("  You > ")
	b.WriteString(prompt + m.input.View())
	b.WriteString("\n")

	return b.String()
}

// ── Helpers ──

func looksLikeSQL(s string) bool {
	trimmed := strings.TrimSpace(s)
	upper := strings.ToUpper(trimmed)
	keywords := []string{"SELECT", "WITH", "EXPLAIN", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE"}
	for _, kw := range keywords {
		if len(upper) >= len(kw) && upper[:len(kw)] == kw {
			return true
		}
	}
	return false
}

func aiProviderLabel() string {
	cfg, _ := config.Load()
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = cfg.AIProvider
	}
	if provider == "" {
		provider = "openai"
	}

	model := ""
	switch provider {
	case "openai":
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = cfg.OpenAIModel
		}
		if model == "" {
			model = "gpt-4"
		}
	case "anthropic":
		model = os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = cfg.AnthropicModel
		}
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
	case "ollama":
		model = os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = cfg.OllamaModel
		}
		if model == "" {
			model = "llama3"
		}
	}

	if model != "" {
		return strings.ToUpper(provider) + "/" + model
	}
	return strings.ToUpper(provider)
}

// ── Render Components ──

func fullStartupView(conn db.Database, aiLabel, version string) string {
	// Extract provider and model
	provider := aiLabel
	model := ""
	if idx := strings.Index(aiLabel, "/"); idx >= 0 {
		provider = aiLabel[:idx]
		model = aiLabel[idx+1:]
	}

	dbName := ""
	connected := false
	if conn != nil {
		dbName = conn.Name()
		connected = true
	}

	// Use the spec startup screen layout
	screen := StartupScreen(logoASCII, version, provider, model, dbName, connected)

	// Wrap in the TUI welcome
	return BoxStyle.Render(screen)
}

func helpBox() string {
	lines := []string{
		HelpHeaderStyle.Render("  ⚡ Commands"),
		"",
		"    " + HelpCmdStyle.Render(".help") + "    " + HelpDescStyle.Render("Show this help") + "  " + HelpHintStyle.Render("→ you're looking at it"),
		"    " + HelpCmdStyle.Render(".quit") + "    " + HelpDescStyle.Render("Exit basemake"),
		"    " + HelpCmdStyle.Render(".tables") + "  " + HelpDescStyle.Render("List tables in the current database"),
		"    " + HelpCmdStyle.Render(".schema") + "  " + HelpDescStyle.Render("Show full database schema"),
		"    " + HelpCmdStyle.Render(".connect") + " " + HelpDescStyle.Render("<dsn> — Connect to a database"),
		"    " + HelpCmdStyle.Render(".history") + " " + HelpDescStyle.Render("Show past questions"),
		"",
		HelpHeaderStyle.Render("  💬 Queries"),
		"",
		"    " + HelpDescStyle.Render("Type a question in plain English,"),
		"    " + HelpDescStyle.Render("or write raw SQL directly."),
		"",
		"    " + HelpHintStyle.Render("Examples:") + " " + HelpDescStyle.Render("\"show me users who signed up last week\""),
		"    " + HelpDescStyle.Render("          \"SELECT * FROM orders LIMIT 5\""),
	}
	return BoxStyle.Render(strings.Join(lines, "\n"))
}

func errorBubble(msg string) string {
	return lipgloss.NewStyle().
		Foreground(Red).
		Render("  ⚠ " + msg)
}

func successBubble(msg string) string {
	return "  " + lipgloss.NewStyle().Foreground(Green).Render("✅") + " " + lipgloss.NewStyle().Foreground(Text).Render(msg)
}

func sqlPreview(sql string) string {
	if len(sql) > 80 {
		sql = sql[:77] + "..."
	}
	return lipgloss.NewStyle().
		Foreground(DimText).
		Italic(true).
		Render("  ─╴" + sql)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Same as cmd/root.go's BannerASCII.
const logoASCII = `█████                                                          █████              
▒▒███                                                          ▒▒███               
 ▒███████   ██████    █████   ██████  █████████████    ██████   ▒███ █████  ██████ 
 ▒███▒▒███ ▒▒▒▒▒███  ███▒▒   ███▒▒███▒▒███▒▒███▒▒███  ▒▒▒▒▒███  ▒███▒▒███  ███▒▒███
 ▒███ ▒███  ███████ ▒▒█████ ▒███████  ▒███ ▒███ ▒███   ███████  ▒██████▒  ▒███████ 
 ▒███ ▒███ ███▒▒███  ▒▒▒▒███▒███▒▒▒   ▒███ ▒███ ▒███  ███▒▒███  ▒███▒▒███ ▒███▒▒▒  
 ████████ ▒▒████████ ██████ ▒▒██████  █████▒███ █████▒▒████████ ████ █████▒▒██████ 
▒▒▒▒▒▒▒▒   ▒▒▒▒▒▒▒▒ ▒▒▒▒▒▒   ▒▒▒▒▒▒  ▒▒▒▒▒ ▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒▒▒ ▒▒▒▒ ▒▒▒▒▒  ▒▒▒▒▒▒`
